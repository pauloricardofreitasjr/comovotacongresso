package store

import (
	"context"
	"testing"
	"time"

	"comovotacongresso/internal/domain"
	"comovotacongresso/internal/testdb"
)

// Mesmo desenho do dataset de votações; sessões S1..S4 (1 por dia, 14h).
//
//	A: exercício aberto desde jan/2025.            Presente em S1, S2, S3.      → 3/4 = 75%
//	B: suplente, de 02/03 00:00 a 03/03 23:00.     Presente em S2.              → 1/2 = 50%
//	C: [jan, 02/03 00:00) e [04/03 00:00, aberto). Nunca presente.              → 0/2 = 0%  (S1 e S4)
//	D: sem período de exercício (inconsistente).   Presente em S1.              → 1/1 = 100%
func TestMetricas_PresencaSessao(t *testing.T) {
	pool := testdb.Novo(t)
	s := New(pool)
	ctx := context.Background()

	ids, err := s.UpsertParlamentares(ctx, []ParlamentarIn{
		{IDExterno: 1, Nome: "A", Partido: "PT"}, {IDExterno: 2, Nome: "B", Partido: "PT"},
		{IDExterno: 3, Nome: "C", Partido: "PSD"}, {IDExterno: 4, Nome: "D", Partido: "PL"},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	ex := func(ext int, ivs ...domain.Intervalo) {
		if err := s.SubstituirExercicio(ctx, ids[ext], ivs); err != nil {
			t.Fatal(err)
		}
	}
	ex(1, domain.Intervalo{Inicio: utc("2025-01-01 00:00"), Partido: "PT"})
	ex(2, domain.Intervalo{Inicio: utc("2025-03-02 00:00"), Fim: ptr(utc("2025-03-03 23:00")), Partido: "PT"})
	ex(3, domain.Intervalo{Inicio: utc("2025-01-01 00:00"), Fim: ptr(utc("2025-03-02 00:00")), Partido: "PSD"},
		domain.Intervalo{Inicio: utc("2025-03-04 00:00"), Partido: "PSD"})

	p := func(ext int, partido string) ParlamentarIn {
		return ParlamentarIn{IDExterno: ext, Nome: "x", Partido: partido}
	}
	sessao := func(id int, dia string, presentes ...ParlamentarIn) {
		if err := s.SalvarSessao(ctx, SessaoIn{IDExterno: id, Inicio: utc(dia + " 14:00"), Tipo: "Sessão Deliberativa", Presentes: presentes}); err != nil {
			t.Fatal(err)
		}
	}
	sessao(1, "2025-03-01", p(1, "PT"), p(4, "PL"))
	sessao(2, "2025-03-02", p(1, "PT"), p(2, "PT"))
	sessao(3, "2025-03-03", p(1, "PT"))
	sessao(4, "2025-03-04")

	if err := s.AtualizarMetricas(ctx); err != nil {
		t.Fatal(err)
	}

	type l struct {
		eleg, pres int
		pct        float64
	}
	want := map[int]l{1: {4, 3, 75}, 2: {2, 1, 50}, 3: {2, 0, 0}, 4: {1, 1, 100}}
	rows, err := pool.Query(ctx, `SELECT p.id_externo, m.sessoes_elegiveis, m.sessoes_presente, m.pct_presenca_sessao::float8
		FROM mv_parlamentar_presenca_sessao m JOIN parlamentares p ON p.id = m.parlamentar_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var ext int
		var g l
		if err := rows.Scan(&ext, &g.eleg, &g.pres, &g.pct); err != nil {
			t.Fatal(err)
		}
		n++
		if w := want[ext]; g.eleg != w.eleg || g.pres != w.pres || g.pct != w.pct {
			t.Errorf("parlamentar %d: got %+v, esperado %+v", ext, g, w)
		}
	}
	if n != 4 {
		t.Errorf("linhas=%d", n)
	}

	// PT = A(4 elegíveis, 3 presentes) + B(2, 1) → 4/6 = 66,67%
	var pct float64
	var parl int
	if err := pool.QueryRow(ctx, `SELECT parlamentares, pct_presenca_sessao::float8 FROM mv_partido_presenca_sessao WHERE partido='PT'`).Scan(&parl, &pct); err != nil {
		t.Fatal(err)
	}
	if parl != 2 || pct < 66.66 || pct > 66.68 {
		t.Errorf("PT: parlamentares=%d pct=%v", parl, pct)
	}

	// regravar a sessão substitui a lista (idempotente, sem acumular)
	sessao(1, "2025-03-01", p(1, "PT"))
	var total int
	pool.QueryRow(ctx, `SELECT count(*) FROM sessoes_presencas sp JOIN sessoes s ON s.id=sp.sessao_id WHERE s.id_externo=1`).Scan(&total)
	if total != 1 {
		t.Errorf("presenças da sessão 1 após regravar = %d, esperado 1", total)
	}
}

// Regressão: o exercício de uma legislatura não pode apagar o de outra, e partido_atual vem do
// intervalo mais recente mesmo quando a legislatura antiga é sincronizada por último.
func TestSubstituirExercicioNoPeriodo_PreservaOutrasLegislaturas(t *testing.T) {
	pool := testdb.Novo(t)
	s := New(pool)
	ctx := context.Background()
	ids, err := s.UpsertParlamentares(ctx, []ParlamentarIn{{IDExterno: 1, Nome: "A", Partido: "PT"}}, true)
	if err != nil {
		t.Fatal(err)
	}
	id := ids[1]
	l57de, l57ate := utc("2023-02-01 00:00"), utc("2027-02-01 00:00")
	l58de, l58ate := utc("2027-02-01 00:00"), utc("2031-02-01 00:00")

	iv57 := []domain.Intervalo{{Inicio: utc("2023-02-01 12:00"), Fim: ptr(utc("2027-01-31 12:00")), Partido: "PT"}}
	iv58 := []domain.Intervalo{{Inicio: utc("2027-02-01 12:00"), Partido: "PL"}}
	for _, c := range []struct {
		ivs     []domain.Intervalo
		de, ate time.Time
	}{{iv58, l58de, l58ate}, {iv57, l57de, l57ate}} { // 58 primeiro, 57 por último
		if err := s.SubstituirExercicioNoPeriodo(ctx, id, c.ivs, c.de, c.ate); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM parlamentares_exercicio WHERE parlamentar_id=$1`, id).Scan(&n); err != nil || n != 2 {
		t.Fatalf("intervalos=%d err=%v, want 2", n, err)
	}
	var partido string
	if err := pool.QueryRow(ctx, `SELECT partido_atual FROM parlamentares WHERE id=$1`, id).Scan(&partido); err != nil || partido != "PL" {
		t.Errorf("partido_atual=%q err=%v, want PL (intervalo mais recente)", partido, err)
	}
}
