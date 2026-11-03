package domain

import (
	"sort"
	"time"
)

// EventoHistorico é uma linha de /deputados/{id}/historico.
type EventoHistorico struct {
	IDLegislatura int
	DataHora      time.Time
	Partido       string
	Situacao      string // Exercício, Licença, SUPLENCIA, CONVOCADO, FIM_MANDATO ou ""
	Condicao      string // Titular, Suplente ou ""
}

// Intervalo é um período contínuo de exercício. Fim nil = em aberto.
type Intervalo struct {
	Inicio   time.Time
	Fim      *time.Time
	Condicao string
	Partido  string
}

// SituacaoConhecida diz se a situação do histórico é uma das tratadas por Intervalos.
// Situações novas na API devem ser logadas pelo chamador, pois podem encerrar exercício.
func SituacaoConhecida(s string) bool {
	switch s {
	case "", "Exercício", "CONVOCADO", "Licença", "SUPLENCIA", "VACANCIA", "SUSPENSO", "FIM_MANDATO":
		return true
	}
	return false
}

// Intervalos monta os períodos de exercício de uma legislatura (discovery §3.1):
// abre em "Exercício"; fecha em Licença, SUPLENCIA, VACANCIA (renúncia, cassação...),
// SUSPENSO ou FIM_MANDATO. Uma troca de
// partido durante o exercício fecha o intervalo e abre outro (histórico de partido).
// CONVOCADO e linhas sem situação são ignoradas.
func Intervalos(eventos []EventoHistorico, legislatura int) []Intervalo {
	var evs []EventoHistorico
	for _, e := range eventos {
		if e.IDLegislatura == legislatura {
			evs = append(evs, e)
		}
	}
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].DataHora.Before(evs[j].DataHora) })

	var out []Intervalo
	var cur *Intervalo
	fechar := func(t time.Time) {
		if cur != nil {
			f := t
			cur.Fim = &f
			out = append(out, *cur)
			cur = nil
		}
	}
	for _, e := range evs {
		switch e.Situacao {
		case "Exercício":
			if cur != nil && cur.Partido != e.Partido {
				fechar(e.DataHora)
			}
			if cur == nil {
				cur = &Intervalo{Inicio: e.DataHora, Condicao: e.Condicao, Partido: e.Partido}
			}
		case "Licença", "SUPLENCIA", "VACANCIA", "SUSPENSO", "FIM_MANDATO":
			fechar(e.DataHora)
		}
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}
