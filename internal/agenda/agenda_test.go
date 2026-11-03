package agenda

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

var brt = time.FixedZone("BRT", -3*3600)

func TestSemanal_Proxima(t *testing.T) {
	s := Semanal{Dia: time.Sunday, Hora: 4, Min: 0, Loc: brt}
	casos := []struct{ de, quer string }{
		{"2026-11-03T10:00:00-03:00", "2026-11-08T04:00:00-03:00"}, // terça → domingo
		{"2026-11-08T03:59:00-03:00", "2026-11-08T04:00:00-03:00"}, // mesmo dia, antes da hora
		{"2026-11-08T04:00:00-03:00", "2026-11-15T04:00:00-03:00"}, // estritamente depois
		{"2026-11-08T08:00:00Z", "2026-11-15T04:00:00-03:00"},      // 05:00 BRT já passou; UTC é convertido
	}
	for _, c := range casos {
		de, _ := time.Parse(time.RFC3339, c.de)
		if got := s.Proxima(de).Format(time.RFC3339); got != c.quer {
			t.Errorf("de %s: got %s, want %s", c.de, got, c.quer)
		}
	}
}

type avisos struct {
	mu   sync.Mutex
	msgs []string
}

func (a *avisos) Avisar(_ context.Context, m string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.msgs = append(a.msgs, m)
	return nil
}

// relógio falso: Dormir avança o tempo e cancela depois de n esperas.
func falso(n int, cancel context.CancelFunc) (func() time.Time, func(context.Context, time.Duration) error) {
	t := time.Date(2026, 11, 3, 10, 0, 0, 0, brt)
	esperas := 0
	return func() time.Time { return t }, func(ctx context.Context, d time.Duration) error {
		esperas++
		if esperas > n {
			cancel()
			return ctx.Err()
		}
		t = t.Add(d)
		return nil
	}
}

func TestRodar_AvisaSoNaPrimeiraFalhaENaRecuperacao(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agora, dormir := falso(4, cancel)
	resultados := []error{errors.New("boom"), errors.New("boom de novo"), nil, errors.New("outra"), nil}
	n := 0
	av := &avisos{}
	a := &Agendador{
		Jobs: []Job{{Nome: "daily", Horario: Cada(time.Hour), AoIniciar: true, Rodar: func(context.Context) error {
			r := resultados[n]
			n++
			return r
		}}},
		Avisador: av, Agora: agora, Dormir: dormir,
	}
	if err := a.Rodar(ctx); err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("execuções=%d, want 5", n)
	}
	// falha, falha (sem novo alerta), ok (recuperou), falha, ok (recuperou)
	if len(av.msgs) != 4 {
		t.Fatalf("alertas=%d: %v", len(av.msgs), av.msgs)
	}
	for i, quer := range []string{"falhou", "voltou", "falhou", "voltou"} {
		if !strings.Contains(av.msgs[i], quer) {
			t.Errorf("alerta %d=%q, want conter %q", i, av.msgs[i], quer)
		}
	}
}

func TestRodar_ExecutaOJobMaisProximoPrimeiro(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agora, dormir := falso(3, cancel)
	var ordem []string
	job := func(nome string, h Horario) Job {
		return Job{Nome: nome, Horario: h, Rodar: func(context.Context) error { ordem = append(ordem, nome); return nil }}
	}
	a := &Agendador{
		Jobs:  []Job{job("lento", Cada(150*time.Minute)), job("rapido", Cada(time.Hour))},
		Agora: agora, Dormir: dormir,
	}
	if err := a.Rodar(ctx); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(ordem, ","); got != "rapido,rapido,lento" {
		t.Errorf("ordem=%s", got)
	}
}

func TestRodar_ExecucaoCortadaPeloShutdownNaoAlerta(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	av := &avisos{}
	a := &Agendador{
		Jobs: []Job{{Nome: "daily", Horario: Cada(time.Hour), AoIniciar: true, Rodar: func(ctx context.Context) error {
			cancel()
			return ctx.Err()
		}}},
		Avisador: av,
	}
	if err := a.Rodar(ctx); err != nil {
		t.Fatal(err)
	}
	if len(av.msgs) != 0 {
		t.Errorf("alertas indevidos: %v", av.msgs)
	}
}

func TestWebhook(t *testing.T) {
	var corpo string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		corpo = string(b)
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("content-type")
		}
	}))
	defer srv.Close()
	if err := (Webhook{URL: srv.URL}).Avisar(context.Background(), "oi"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(corpo, `"text":"oi"`) || !strings.Contains(corpo, `"content":"oi"`) {
		t.Errorf("corpo=%s", corpo)
	}
	ruim := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "x", 500) }))
	defer ruim.Close()
	if err := (Webhook{URL: ruim.URL}).Avisar(context.Background(), "oi"); err == nil {
		t.Error("esperava erro com status 500")
	}
}
