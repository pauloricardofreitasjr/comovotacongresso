package store

import (
	"context"
	"testing"
	"time"

	"comovotacongresso/internal/domain"
	"comovotacongresso/internal/testdb"
)

func ptr[T any](v T) *T { return &v }

func utc(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04", s)
	if err != nil {
		panic(err)
	}
	return t
}

type linha struct {
	elegiveis, presente, base, alinhadas int
	pctPresenca, pctGov                  *float64
	emExercicio                          bool
}

// Dataset montado à mão. Votações nominais V1..V4 (1 por dia, 12h) e uma simbólica V5.
//
//	A: em exercício desde jan/2025 (aberto), partido PT.  Vota V1 (sim, alinhou) e V2 (não, não alinhou).
//	B: suplente, exercício de 02/03 00:00 a 03/03 23:00.  Vota V2 (alinhou) e V3 (sem orientação).
//	C: exercício [jan, 02/03 00:00) e [04/03 00:00, aberto), PSD. Nunca vota.
//	D: sem período de exercício (dado inconsistente), PL. Vota V1 (alinhou).
func TestMetricas_DatasetSintetico(t *testing.T) {
	pool := testdb.Novo(t)
	s := New(pool)
	ctx := context.Background()

	ids, err := s.UpsertParlamentares(ctx, []ParlamentarIn{
		{IDExterno: 1, Nome: "A", Partido: "PT", UF: "SP"},
		{IDExterno: 2, Nome: "B", Partido: "PT", UF: "RJ"},
		{IDExterno: 3, Nome: "C", Partido: "PSD", UF: "MG"},
		{IDExterno: 4, Nome: "D", Partido: "PL", UF: "BA"},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	mustExerc := func(ext int, ivs ...domain.Intervalo) {
		if err := s.SubstituirExercicio(ctx, ids[ext], ivs); err != nil {
			t.Fatal(err)
		}
	}
	mustExerc(1, domain.Intervalo{Inicio: utc("2025-01-01 00:00"), Condicao: "Titular", Partido: "PT"})
	mustExerc(2, domain.Intervalo{Inicio: utc("2025-03-02 00:00"), Fim: ptr(utc("2025-03-03 23:00")), Condicao: "Suplente", Partido: "PT"})
	mustExerc(3,
		domain.Intervalo{Inicio: utc("2025-01-01 00:00"), Fim: ptr(utc("2025-03-02 00:00")), Condicao: "Titular", Partido: "PSD"},
		domain.Intervalo{Inicio: utc("2025-03-04 00:00"), Condicao: "Titular", Partido: "PSD"})

	voto := func(ext int, v domain.TipoVoto, partido string, alinhou *bool) VotoIn {
		return VotoIn{Parlamentar: ParlamentarIn{IDExterno: ext, Nome: "x"}, Voto: v, Partido: partido, AlinhouGoverno: alinhou}
	}
	salvar := func(id, quando string, tipo domain.TipoVotacao, votos ...VotoIn) {
		if err := s.SalvarVotacao(ctx, VotacaoIn{ID: id, DataHora: utc(quando), Tipo: tipo, Votos: votos}); err != nil {
			t.Fatal(err)
		}
	}
	salvar("V1", "2025-03-01 12:00", domain.VotacaoNominal,
		voto(1, domain.VotoSim, "PT", ptr(true)), voto(4, domain.VotoSim, "PL", ptr(true)))
	salvar("V2", "2025-03-02 12:00", domain.VotacaoNominal,
		voto(1, domain.VotoNao, "PT", ptr(false)), voto(2, domain.VotoSim, "PT", ptr(true)))
	salvar("V3", "2025-03-03 12:00", domain.VotacaoNominal,
		voto(2, domain.VotoSim, "PT", nil))
	salvar("V4", "2025-03-04 12:00", domain.VotacaoNominal)
	salvar("V5", "2025-03-02 13:00", domain.VotacaoSimbolica)

	ler := func() map[int]linha {
		rows, err := pool.Query(ctx, `SELECT id_externo, votacoes_elegiveis, votacoes_presente, votacoes_base_governo,
			votacoes_alinhadas_governo, pct_presenca::float8, pct_governista::float8, em_exercicio FROM mv_parlamentar_metricas`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[int]linha{}
		for rows.Next() {
			var ext int
			var l linha
			if err := rows.Scan(&ext, &l.elegiveis, &l.presente, &l.base, &l.alinhadas, &l.pctPresenca, &l.pctGov, &l.emExercicio); err != nil {
				t.Fatal(err)
			}
			out[ext] = l
		}
		return out
	}

	// Antes do refresh a view reflete o estado de quando foi criada (vazia): prova que o refresh é necessário.
	if got := ler(); len(got) != 0 {
		t.Fatalf("view deveria estar vazia antes do refresh, tem %d linhas", len(got))
	}
	if err := s.AtualizarMetricas(ctx); err != nil {
		t.Fatal(err)
	}

	f := func(v float64) *float64 { return &v }
	want := map[int]linha{
		1: {4, 2, 2, 1, f(50), f(50), true},
		2: {2, 2, 1, 1, f(100), f(100), false},
		3: {2, 0, 0, 0, f(0), nil, true},
		4: {1, 1, 1, 1, f(100), f(100), false},
	}
	got := ler()
	for ext, w := range want {
		g, ok := got[ext]
		if !ok {
			t.Errorf("parlamentar %d ausente", ext)
			continue
		}
		if g.elegiveis != w.elegiveis || g.presente != w.presente || g.base != w.base || g.alinhadas != w.alinhadas ||
			g.emExercicio != w.emExercicio || !igual(g.pctPresenca, w.pctPresenca) || !igual(g.pctGov, w.pctGov) {
			t.Errorf("parlamentar %d: got %+v / pct=%v,%v, esperado %+v / pct=%v,%v", ext, g, deref(g.pctPresenca), deref(g.pctGov), w, deref(w.pctPresenca), deref(w.pctGov))
		}
	}

	// Por partido (partido do momento): PT = A(4 elegíveis) + B(2) = 6; presentes 4; base 3; alinhadas 2.
	type part struct {
		parl, eleg, pres, base int
		pctPres, pctGov        *float64
	}
	wantP := map[string]part{
		"PT":  {2, 6, 4, 3, f(66.67), f(66.67)},
		"PSD": {1, 2, 0, 0, f(0), nil},
		"PL":  {1, 1, 1, 1, f(100), f(100)},
	}
	rows, err := pool.Query(ctx, `SELECT partido, parlamentares, votacoes_elegiveis, votacoes_presente, votacoes_base_governo,
		pct_presenca::float8, pct_governista::float8 FROM mv_partido_metricas`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var nome string
		var p part
		if err := rows.Scan(&nome, &p.parl, &p.eleg, &p.pres, &p.base, &p.pctPres, &p.pctGov); err != nil {
			t.Fatal(err)
		}
		n++
		w, ok := wantP[nome]
		if !ok || p.parl != w.parl || p.eleg != w.eleg || p.pres != w.pres || p.base != w.base || !igual(p.pctPres, w.pctPres) || !igual(p.pctGov, w.pctGov) {
			t.Errorf("partido %s: got %+v / %v,%v, esperado %+v", nome, p, deref(p.pctPres), deref(p.pctGov), w)
		}
	}
	if n != len(wantP) {
		t.Errorf("partidos=%d, esperado %d", n, len(wantP))
	}

	// Novo voto + refresh: a view acompanha (CONCURRENTLY funciona na 2ª vez).
	salvar("V4", "2025-03-04 12:00", domain.VotacaoNominal, voto(3, domain.VotoSim, "PSD", ptr(true)))
	if err := s.AtualizarMetricas(ctx); err != nil {
		t.Fatal(err)
	}
	if c := ler()[3]; c.presente != 1 || c.pctPresenca == nil || *c.pctPresenca != 50 || c.pctGov == nil || *c.pctGov != 100 {
		t.Errorf("C após voto em V4: %+v", c)
	}
}

func igual(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	d := *a - *b
	return d < 0.005 && d > -0.005
}

func deref(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
