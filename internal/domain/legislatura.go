package domain

import (
	"sort"
	"time"
)

// Legislatura é o período de uma legislatura. Fim é o último dia (inclusive).
type Legislatura struct {
	ID     int
	Inicio time.Time
	Fim    time.Time
}

// JanelaRecente é por quanto tempo após o início de uma legislatura a anterior continua
// sendo sincronizada, para fechar os mandatos que a fonte encerra com atraso.
const JanelaRecente = 90 * 24 * time.Hour

// Vigente devolve a legislatura que contém hoje. Se a fonte ainda não publicou a nova
// (hoje depois do fim da última conhecida), devolve a mais recente e ok=false.
func Vigente(legs []Legislatura, hoje time.Time) (l Legislatura, ok bool) {
	ord := ordenadas(legs)
	for _, c := range ord {
		if !hoje.Before(c.Inicio) && hoje.Before(c.Fim.AddDate(0, 0, 1)) {
			return c, true
		}
	}
	if len(ord) == 0 {
		return Legislatura{}, false
	}
	return ord[len(ord)-1], false
}

// ParaSincronizar lista as legislaturas que o sync periódico deve cobrir, da mais antiga
// para a mais recente: a vigente e, nos primeiros JanelaRecente dias dela, a anterior.
// A ordem não afeta o resultado (partido_atual vem do intervalo mais recente), mas mantém o log legível.
func ParaSincronizar(legs []Legislatura, hoje time.Time) []Legislatura {
	v, _ := Vigente(legs, hoje)
	if v.ID == 0 {
		return nil
	}
	out := []Legislatura{v}
	if hoje.Sub(v.Inicio) < JanelaRecente {
		for _, c := range ordenadas(legs) {
			if c.ID == v.ID-1 {
				out = append([]Legislatura{c}, out...)
			}
		}
	}
	return out
}

// NoPeriodo lista as legislaturas que se sobrepõem a [ini, fim] (backfill), da mais antiga
// para a mais recente.
func NoPeriodo(legs []Legislatura, ini, fim time.Time) []Legislatura {
	var out []Legislatura
	for _, c := range ordenadas(legs) {
		if !c.Inicio.After(fim) && !c.Fim.Before(ini) {
			out = append(out, c)
		}
	}
	return out
}

func ordenadas(legs []Legislatura) []Legislatura {
	out := append([]Legislatura(nil), legs...)
	sort.Slice(out, func(i, j int) bool { return out[i].Inicio.Before(out[j].Inicio) })
	return out
}
