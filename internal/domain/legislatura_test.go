package domain

import (
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

var legs = []Legislatura{
	{ID: 56, Inicio: d("2019-02-01"), Fim: d("2023-01-31")},
	{ID: 57, Inicio: d("2023-02-01"), Fim: d("2027-01-31")},
	{ID: 58, Inicio: d("2027-02-01"), Fim: d("2031-01-31")},
}

func ids(ls []Legislatura) []int {
	var out []int
	for _, l := range ls {
		out = append(out, l.ID)
	}
	return out
}

func TestVigente(t *testing.T) {
	casos := []struct {
		hoje string
		id   int
		ok   bool
	}{
		{"2026-11-03", 57, true},
		{"2027-01-31", 57, true}, // último dia, inclusive
		{"2027-02-01", 58, true},
		{"2031-06-01", 58, false}, // fonte sem a próxima: devolve a mais recente
	}
	for _, c := range casos {
		l, ok := Vigente(legs, d(c.hoje))
		if l.ID != c.id || ok != c.ok {
			t.Errorf("%s: got (%d,%v), want (%d,%v)", c.hoje, l.ID, ok, c.id, c.ok)
		}
	}
	// virada ainda não publicada pela fonte: só a 57 conhecida
	l, ok := Vigente(legs[:2], d("2027-02-01"))
	if l.ID != 57 || ok {
		t.Errorf("sem a 58 publicada: got (%d,%v)", l.ID, ok)
	}
}

func TestParaSincronizar(t *testing.T) {
	if got := ids(ParaSincronizar(legs, d("2026-11-03"))); len(got) != 1 || got[0] != 57 {
		t.Errorf("meio da legislatura: %v", got)
	}
	if got := ids(ParaSincronizar(legs, d("2027-03-01"))); len(got) != 2 || got[0] != 57 || got[1] != 58 {
		t.Errorf("logo após a virada deve incluir a anterior: %v", got)
	}
	if got := ids(ParaSincronizar(legs, d("2027-06-01"))); len(got) != 1 || got[0] != 58 {
		t.Errorf("passada a janela: %v", got)
	}
	if got := ParaSincronizar(nil, d("2026-11-03")); got != nil {
		t.Errorf("sem legislaturas: %v", got)
	}
}

func TestNoPeriodo(t *testing.T) {
	if got := ids(NoPeriodo(legs, d("2023-02-01"), d("2027-03-01"))); len(got) != 2 || got[0] != 57 || got[1] != 58 {
		t.Errorf("série desde 2023 após a virada: %v", got)
	}
	if got := ids(NoPeriodo(legs, d("2023-02-01"), d("2026-11-03"))); len(got) != 1 || got[0] != 57 {
		t.Errorf("série atual: %v", got)
	}
}
