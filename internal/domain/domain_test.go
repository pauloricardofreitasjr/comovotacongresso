package domain

import (
	"testing"
	"time"
)

func TestMapVoto(t *testing.T) {
	cases := map[string]TipoVoto{
		"Sim": VotoSim, "Não": VotoNao, "nao": VotoNao, " Abstenção ": VotoAbstencao,
		"Obstrução": VotoObstrucao, "Artigo 17": VotoArtigo17, "": VotoOutro, "Ausente": VotoOutro,
	}
	for in, want := range cases {
		if got := MapVoto(in); got != want {
			t.Errorf("MapVoto(%q)=%q, esperado %q", in, got, want)
		}
	}
}

func TestMapOrientacao(t *testing.T) {
	for in, want := range map[string]Orientacao{
		"Sim": OrientSim, "Não": OrientNao, "Liberado": OrientLiberado, "Obstrução": OrientObstrucao,
	} {
		if got := MapOrientacao(in); got == nil || *got != want {
			t.Errorf("MapOrientacao(%q)=%v, esperado %q", in, got, want)
		}
	}
	for _, in := range []string{"", "  ", "Talvez"} {
		if got := MapOrientacao(in); got != nil {
			t.Errorf("MapOrientacao(%q)=%v, esperado nil", in, *got)
		}
	}
}

func TestIsGoverno(t *testing.T) {
	for in, want := range map[string]bool{"Governo": true, " governo ": true, "GOVERNO": true, "Maioria": false, "": false} {
		if got := IsGoverno(in); got != want {
			t.Errorf("IsGoverno(%q)=%v", in, got)
		}
	}
}

func TestAlinhou(t *testing.T) {
	sim, nao, lib, obs := OrientSim, OrientNao, OrientLiberado, OrientObstrucao
	b := func(v bool) *bool { return &v }
	cases := []struct {
		nome string
		o    *Orientacao
		v    TipoVoto
		want *bool
	}{
		{"sim/sim", &sim, VotoSim, b(true)},
		{"sim/nao", &sim, VotoNao, b(false)},
		{"sim/abstencao", &sim, VotoAbstencao, b(false)},
		{"sim/obstrucao", &sim, VotoObstrucao, b(false)},
		{"nao/nao", &nao, VotoNao, b(true)},
		{"nao/sim", &nao, VotoSim, b(false)},
		{"nao/abstencao", &nao, VotoAbstencao, b(false)},
		{"nao/obstrucao", &nao, VotoObstrucao, b(false)},
		{"liberado", &lib, VotoSim, nil},
		{"sem orientação", nil, VotoSim, nil},
		{"orientação obstrução", &obs, VotoObstrucao, nil},
		{"artigo 17", &sim, VotoArtigo17, nil},
		{"outro", &sim, VotoOutro, nil},
	}
	for _, c := range cases {
		got := Alinhou(c.o, c.v)
		if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
			t.Errorf("%s: got %v want %v", c.nome, got, c.want)
		}
	}
}

func TestClassificaTipo(t *testing.T) {
	um := 1
	if tp, ok := ClassificaTipo(10, nil); !ok || tp != VotacaoNominal {
		t.Error("com votos deve ser nominal, mesmo sem aprovacao")
	}
	if tp, ok := ClassificaTipo(0, &um); !ok || tp != VotacaoSimbolica {
		t.Error("sem votos e concluída deve ser simbólica")
	}
	if _, ok := ClassificaTipo(0, nil); ok {
		t.Error("sem votos e sem resultado deve ser indeterminada")
	}
}

func TestEscolheProposicao(t *testing.T) {
	pl := Prop{ID: 1, SiglaTipo: "PL", Numero: 10, Ano: 2024}
	req := Prop{ID: 2, SiglaTipo: "REQ", Numero: 5, Ano: 2024}
	outro := Prop{ID: 3, SiglaTipo: "PLP", Numero: 1, Ano: 2024}

	cases := []struct {
		nome      string
		af, pos   []Prop
		wantID    int
		wantConf  Confianca
		wantNenhu bool
	}{
		{"uma afetada", []Prop{pl}, nil, 1, ConfDireto, false},
		{"afetada repetida", []Prop{pl, pl}, nil, 1, ConfDireto, false},
		{"matéria + REQ", []Prop{pl, req}, nil, 1, ConfDireto, false},
		{"só REQ cai no possível", []Prop{req}, []Prop{outro}, 3, ConfProvavel, false},
		{"vazia, um possível", nil, []Prop{outro}, 3, ConfProvavel, false},
		{"duas afetadas distintas, um possível", []Prop{pl, outro}, []Prop{outro}, 3, ConfProvavel, false},
		{"duas afetadas distintas, sem possível", []Prop{pl, outro}, nil, 0, "", true},
		{"vazias", nil, nil, 0, "", true},
		{"muitos possíveis", nil, []Prop{pl, outro}, 0, "", true},
	}
	for _, c := range cases {
		p, conf := EscolheProposicao(c.af, c.pos)
		if c.wantNenhu {
			if p != nil || conf != "" {
				t.Errorf("%s: esperava nil, got %v/%q", c.nome, p, conf)
			}
			continue
		}
		if p == nil || p.ID != c.wantID || conf != c.wantConf {
			t.Errorf("%s: got %v/%q, esperado id=%d/%q", c.nome, p, conf, c.wantID, c.wantConf)
		}
	}
}

func ts(s string) time.Time {
	t, err := time.Parse("2006-01-02T15:04", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestIntervalos_Suplente(t *testing.T) {
	evs := []EventoHistorico{
		{57, ts("2024-10-17T15:34"), "X", "SUPLENCIA", "Suplente"},
		{57, ts("2023-02-01T00:00"), "X", "", ""},
		{57, ts("2023-03-02T12:30"), "X", "CONVOCADO", ""},
		{57, ts("2023-03-02T12:58"), "X", "Exercício", "Suplente"},
		{57, ts("2023-11-17T10:24"), "X", "SUPLENCIA", "Suplente"},
		{57, ts("2023-12-08T16:43"), "X", "Exercício", "Suplente"},
		{56, ts("2019-02-01T11:45"), "Y", "Exercício", "Titular"}, // outra legislatura
	}
	got := Intervalos(evs, 57)
	if len(got) != 2 {
		t.Fatalf("intervalos=%+v", got)
	}
	if !got[0].Inicio.Equal(ts("2023-03-02T12:58")) || got[0].Fim == nil || !got[0].Fim.Equal(ts("2023-11-17T10:24")) {
		t.Errorf("1º intervalo=%+v", got[0])
	}
	if !got[1].Inicio.Equal(ts("2023-12-08T16:43")) || got[1].Fim == nil || !got[1].Fim.Equal(ts("2024-10-17T15:34")) {
		t.Errorf("2º intervalo=%+v", got[1])
	}
}

func TestIntervalos_TrocaDePartidoDivideIntervalo(t *testing.T) {
	evs := []EventoHistorico{
		{57, ts("2023-02-01T12:05"), "PP", "Exercício", "Titular"},
		{57, ts("2026-04-29T22:10"), "UNIÃO", "Exercício", "Titular"},
	}
	got := Intervalos(evs, 57)
	if len(got) != 2 || got[0].Partido != "PP" || got[0].Fim == nil || got[1].Partido != "UNIÃO" || got[1].Fim != nil {
		t.Fatalf("intervalos=%+v", got)
	}
	if !got[0].Fim.Equal(got[1].Inicio) {
		t.Error("o primeiro deve fechar onde o segundo abre")
	}
}

func TestIntervalos_ExercicioRepetidoMesmoPartidoNaoDuplica(t *testing.T) {
	evs := []EventoHistorico{
		{57, ts("2023-02-01T12:05"), "PP", "Exercício", "Titular"},
		{57, ts("2023-06-01T10:00"), "PP", "Exercício", "Titular"},
	}
	if got := Intervalos(evs, 57); len(got) != 1 || got[0].Fim != nil {
		t.Fatalf("intervalos=%+v", got)
	}
}

func TestIntervalos_FimDeMandatoFecha(t *testing.T) {
	evs := []EventoHistorico{
		{56, ts("2019-02-01T11:45"), "PSD", "Exercício", "Titular"},
		{56, ts("2023-01-31T23:59"), "PSD", "FIM_MANDATO", "Titular"},
	}
	got := Intervalos(evs, 56)
	if len(got) != 1 || got[0].Fim == nil || !got[0].Fim.Equal(ts("2023-01-31T23:59")) {
		t.Fatalf("intervalos=%+v", got)
	}
}

func TestIntervalos_SemExercicio(t *testing.T) {
	if got := Intervalos([]EventoHistorico{{57, ts("2023-02-01T00:00"), "X", "", ""}}, 57); len(got) != 0 {
		t.Fatalf("intervalos=%+v", got)
	}
}

func TestIntervalos_VacanciaESuspensoEncerram(t *testing.T) {
	for _, sit := range []string{"VACANCIA", "SUSPENSO"} {
		evs := []EventoHistorico{
			{57, ts("2023-03-02T12:58"), "PT", "Exercício", "Suplente"},
			{57, ts("2026-01-02T14:36"), "PT", sit, "Suplente"},
		}
		got := Intervalos(evs, 57)
		if len(got) != 1 || got[0].Fim == nil || !got[0].Fim.Equal(ts("2026-01-02T14:36")) {
			t.Errorf("%s: intervalos=%+v", sit, got)
		}
	}
}

func TestSituacaoConhecida(t *testing.T) {
	for _, s := range []string{"", "Exercício", "CONVOCADO", "Licença", "SUPLENCIA", "VACANCIA", "SUSPENSO", "FIM_MANDATO"} {
		if !SituacaoConhecida(s) {
			t.Errorf("%q deveria ser conhecida", s)
		}
	}
	if SituacaoConhecida("CASSADO") {
		t.Error("situação nova deve ser desconhecida")
	}
}
