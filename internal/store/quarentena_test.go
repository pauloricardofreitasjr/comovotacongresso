package store

import (
	"context"
	"fmt"
	"testing"

	"comovotacongresso/internal/domain"
	"comovotacongresso/internal/testdb"
)

// Cenários (25 deputados):
//
//	QA  placar da descrição bate com a lista (20 Sim x 5 Não)
//	QB  MESMA lista de QA, mas a descrição diz 3 x 22  → lista trocada: lista_duplicada (ref QA)
//	QC  lista própria (10 x 15), descrição bate
//	QD  MESMA lista de QC, descrição sem placar       → mesmo ato registrado duas vezes: registro_duplicado (ref QC)
//	QE  lista própria (25 Sim)
//	F,G listas idênticas, mas com só 5 votos           → abaixo do mínimo: nada a marcar
func TestQuarentena_ListasIdenticasEEfeitos(t *testing.T) {
	pool := testdb.Novo(t)
	s := New(pool)
	ctx := context.Background()

	var ps []ParlamentarIn
	for i := 1; i <= 25; i++ {
		ps = append(ps, ParlamentarIn{IDExterno: i, Nome: fmt.Sprintf("Dep %02d", i), Partido: "PT"})
	}
	ids, err := s.UpsertParlamentares(ctx, ps, true)
	if err != nil {
		t.Fatal(err)
	}

	lista := func(n int, simAte int) []VotoIn { // n deputados; os primeiros simAte votam Sim, os demais Não
		var vs []VotoIn
		for i := 1; i <= n; i++ {
			v := domain.VotoNao
			if i <= simAte {
				v = domain.VotoSim
			}
			vs = append(vs, VotoIn{Parlamentar: ParlamentarIn{IDExterno: i, Nome: "x"}, Voto: v, Partido: "PT"})
		}
		return vs
	}
	salvar := func(id, quando, desc string, votos []VotoIn) {
		t.Helper()
		if err := s.SalvarVotacao(ctx, VotacaoIn{ID: id, DataHora: utc(quando), Descricao: desc, Tipo: domain.VotacaoNominal, Votos: votos}); err != nil {
			t.Fatal(err)
		}
	}
	salvar("QA", "2025-03-01 12:00", "Mantido o texto. Sim: 20; não: 5; total: 25.", lista(25, 20))
	salvar("QB", "2025-03-01 22:00", "Rejeitada a Emenda de Plenário nº 40. Sim: 3; não: 22; total: 25.", lista(25, 20))
	salvar("QC", "2025-03-05 12:00", "Rejeitado o Recurso. Sim: 10; não: 15; total: 25.", lista(25, 10))
	salvar("QD", "2025-03-05 12:01", "Rejeitado o Recurso nº 3/2025.", lista(25, 10))
	salvar("QE", "2025-03-09 12:00", "Aprovado. Sim: 25; não: 0; total: 25.", lista(25, 25))
	salvar("QF", "2025-03-10 12:00", "Aprovado o requerimento.", lista(5, 5))
	salvar("QG", "2025-03-10 12:05", "Aprovado o requerimento (outro).", lista(5, 5))

	if err := s.AtualizarMetricas(ctx); err != nil {
		t.Fatal(err)
	}

	estado := func() map[string]string {
		rows, err := pool.Query(ctx, `SELECT id, coalesce(quarentena,'') || '>' || coalesce(quarentena_ref,'') FROM votacoes`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[string]string{}
		for rows.Next() {
			var id, v string
			if err := rows.Scan(&id, &v); err != nil {
				t.Fatal(err)
			}
			out[id] = v
		}
		return out
	}
	want := map[string]string{"QA": ">", "QB": "lista_duplicada>QA", "QC": ">", "QD": "registro_duplicado>QC", "QE": ">", "QF": ">", "QG": ">"}
	for id, w := range want {
		if got := estado()[id]; got != w {
			t.Errorf("%s: quarentena=%q, esperado %q", id, got, w)
		}
	}

	// Efeito nas métricas: o deputado 1 votou em tudo (7 votações), mas só QA, QC, QE, QF e QG contam.
	var eleg, votou int
	if err := pool.QueryRow(ctx, `SELECT votacoes_elegiveis, votacoes_votou FROM mv_parlamentar_metricas WHERE parlamentar_id = $1`, ids[1]).Scan(&eleg, &votou); err != nil {
		t.Fatal(err)
	}
	if eleg != 5 || votou != 5 {
		t.Errorf("deputado 1: elegíveis=%d votou=%d, esperado 5 e 5 (QB e QD fora)", eleg, votou)
	}

	// Efeito na leitura: QB sem votos nem resumo; QD mantém os votos; histórico sem as duas.
	b, err := s.GetVotacao(ctx, "QB")
	if err != nil || b.Resumo != nil || b.Votos != nil || !b.ListaOculta() || b.QuarentenaRef == nil || *b.QuarentenaRef != "QA" {
		t.Errorf("QB: resumo=%v votos=%d ref=%v err=%v", b.Resumo, len(b.Votos), b.QuarentenaRef, err)
	}
	d, err := s.GetVotacao(ctx, "QD")
	if err != nil || d.Resumo == nil || len(d.Votos) != 25 || d.ListaOculta() || d.Quarentena == nil {
		t.Errorf("QD: resumo=%v votos=%d err=%v", d.Resumo, len(d.Votos), err)
	}
	h, err := s.GetParlamentar(ctx, ids[1], FiltroVotos{}, 50, 0)
	if err != nil || h.TotalVotos != 5 {
		t.Fatalf("histórico do deputado 1: total=%d err=%v, esperado 5", h.TotalVotos, err)
	}
	for _, v := range h.Votos {
		if v.VotacaoID == "QB" || v.VotacaoID == "QD" {
			t.Errorf("histórico não deveria conter %s", v.VotacaoID)
		}
	}

	// Idempotente, e a marca some quando a causa some (a Câmara corrige a lista de QB).
	if err := s.AtualizarMetricas(ctx); err != nil {
		t.Fatal(err)
	}
	for id, w := range want {
		if got := estado()[id]; got != w {
			t.Errorf("após reexecutar, %s: %q, esperado %q", id, got, w)
		}
	}
	salvar("QB", "2025-03-01 22:00", "Rejeitada a Emenda de Plenário nº 40. Sim: 3; não: 22; total: 25.", lista(25, 3))
	if err := s.AtualizarMetricas(ctx); err != nil {
		t.Fatal(err)
	}
	if got := estado()["QB"]; got != ">" {
		t.Errorf("QB corrigida deveria sair da quarentena, está %q", got)
	}
}

// Quando nenhuma votação do grupo bate com a descrição, nenhuma lista é confiável.
func TestQuarentena_NenhumaConsistenteTodasOcultas(t *testing.T) {
	pool := testdb.Novo(t)
	s := New(pool)
	ctx := context.Background()
	var ps []ParlamentarIn
	var vs []VotoIn
	for i := 1; i <= 22; i++ {
		ps = append(ps, ParlamentarIn{IDExterno: i, Nome: "x", Partido: "PT"})
		vs = append(vs, VotoIn{Parlamentar: ParlamentarIn{IDExterno: i, Nome: "x"}, Voto: domain.VotoSim, Partido: "PT"})
	}
	if _, err := s.UpsertParlamentares(ctx, ps, true); err != nil {
		t.Fatal(err)
	}
	for i, id := range []string{"X1", "X2"} {
		err := s.SalvarVotacao(ctx, VotacaoIn{ID: id, DataHora: utc(fmt.Sprintf("2025-03-0%d 12:00", i+1)), Descricao: "Sim: 1; não: 21.", Tipo: domain.VotacaoNominal, Votos: vs})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := s.AtualizarQuarentena(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM votacoes WHERE quarentena = 'lista_duplicada'`).Scan(&n); err != nil || n != 2 {
		t.Errorf("lista_duplicada=%d err=%v, esperado 2", n, err)
	}
}
