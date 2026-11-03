package store

import (
	"context"
	"testing"

	"comovotacongresso/internal/domain"
	"comovotacongresso/internal/testdb"
)

// A tem votação nominal (duas, a mais recente rejeitada) e uma simbólica; B só tem simbólica.
func TestListarProposicoes_SoNominaisEUltimaNominal(t *testing.T) {
	pool := testdb.Novo(t)
	s := New(pool)
	ctx := context.Background()

	a := &domain.Prop{ID: 1, SiglaTipo: "PL", Numero: 1, Ano: 2025, Ementa: "A"}
	b := &domain.Prop{ID: 2, SiglaTipo: "PL", Numero: 2, Ano: 2025, Ementa: "B"}
	voto := func(ext int, v domain.TipoVoto) VotoIn {
		return VotoIn{Parlamentar: ParlamentarIn{IDExterno: ext, Nome: "x"}, Voto: v, Partido: "PT"}
	}
	salvar := func(v VotacaoIn) {
		if err := s.SalvarVotacao(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	salvar(VotacaoIn{ID: "A1", DataHora: utc("2025-03-01 12:00"), Tipo: domain.VotacaoNominal, Aprovada: ptr(true), Proposicao: a, Vinculo: domain.ConfDireto,
		Votos: []VotoIn{voto(1, domain.VotoSim), voto(2, domain.VotoSim), voto(3, domain.VotoNao)}})
	salvar(VotacaoIn{ID: "A2", DataHora: utc("2025-03-05 12:00"), Tipo: domain.VotacaoNominal, Aprovada: ptr(false), Proposicao: a, Vinculo: domain.ConfDireto,
		Votos: []VotoIn{voto(1, domain.VotoSim), voto(2, domain.VotoNao), voto(3, domain.VotoNao), voto(4, domain.VotoAbstencao)}})
	salvar(VotacaoIn{ID: "A3", DataHora: utc("2025-03-09 12:00"), Tipo: domain.VotacaoSimbolica, Aprovada: ptr(true), Proposicao: a, Vinculo: domain.ConfDireto})
	salvar(VotacaoIn{ID: "B1", DataHora: utc("2025-03-10 12:00"), Tipo: domain.VotacaoSimbolica, Aprovada: ptr(true), Proposicao: b, Vinculo: domain.ConfDireto})

	todas, total, err := s.ListarProposicoes(ctx, ProposicaoFiltro{Limite: 10})
	if err != nil || total != 2 || len(todas) != 2 {
		t.Fatalf("sem filtro: total=%d len=%d err=%v", total, len(todas), err)
	}
	if todas[0].Numero != 2 || todas[0].UltimaNominal != nil { // B é a mais recente e só tem simbólica
		t.Errorf("B não deveria ter UltimaNominal: %+v", todas[0])
	}
	un := todas[1].UltimaNominal
	// a nominal mais recente é A2 (não a simbólica A3): rejeitada, 1 Sim x 2 Não
	if un == nil || un.VotacaoID != "A2" || un.Aprovada == nil || *un.Aprovada || un.Sim != 1 || un.Nao != 2 {
		t.Errorf("UltimaNominal de A = %+v", un)
	}

	so, total, err := s.ListarProposicoes(ctx, ProposicaoFiltro{Limite: 10, SoNominais: true})
	if err != nil || total != 1 || len(so) != 1 || so[0].Numero != 1 {
		t.Fatalf("SoNominais: total=%d %+v err=%v", total, so, err)
	}
	// página além do fim: o total continua correto com o filtro (reconta com o mesmo HAVING)
	vazio, total, err := s.ListarProposicoes(ctx, ProposicaoFiltro{Limite: 10, Offset: 10, SoNominais: true})
	if err != nil || len(vazio) != 0 || total != 1 {
		t.Errorf("além do fim: len=%d total=%d err=%v", len(vazio), total, err)
	}
}
