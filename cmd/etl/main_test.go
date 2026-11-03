package main

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

func TestJanelas(t *testing.T) {
	got := janelas(d("2023-02-01"), d("2023-04-05"), 30)
	want := [][2]string{{"2023-02-01", "2023-03-02"}, {"2023-03-03", "2023-04-01"}, {"2023-04-02", "2023-04-05"}}
	if len(got) != len(want) {
		t.Fatalf("janelas=%v", got)
	}
	for i, w := range want {
		if !got[i][0].Equal(d(w[0])) || !got[i][1].Equal(d(w[1])) {
			t.Errorf("janela %d = %v..%v, esperado %v..%v", i, got[i][0], got[i][1], w[0], w[1])
		}
	}
	if g := janelas(d("2023-02-01"), d("2023-02-01"), 30); len(g) != 1 || !g[0][1].Equal(d("2023-02-01")) {
		t.Errorf("um dia: %v", g)
	}
	if g := janelas(d("2023-02-02"), d("2023-02-01"), 30); len(g) != 0 {
		t.Errorf("intervalo invertido: %v", g)
	}
}

func TestIntervalo(t *testing.T) {
	hoje := d("2026-11-02")
	if ini, fim, err := intervalo("2026-10-01", "", hoje); err != nil || !ini.Equal(d("2026-10-01")) || !fim.Equal(hoje) {
		t.Errorf("padrão: %v %v %v", ini, fim, err)
	}
	for _, c := range [][2]string{{"", ""}, {"abc", ""}, {"2026-10-01", "xyz"}, {"2026-10-05", "2026-10-01"}} {
		if _, _, err := intervalo(c[0], c[1], hoje); err == nil {
			t.Errorf("esperava erro para %v", c)
		}
	}
}

func TestDia(t *testing.T) {
	loc := brasilia()
	got := dia(time.Date(2026, 11, 2, 23, 59, 0, 0, loc))
	if got.Hour() != 0 || got.Day() != 2 || got.Location() != loc {
		t.Errorf("dia=%v", got)
	}
}
