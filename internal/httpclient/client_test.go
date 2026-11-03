package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type payload struct {
	Ok bool `json:"ok"`
}

// novo devolve um client que não dorme de verdade e registra os delays pedidos.
func novo(delays *[]time.Duration) *Client {
	return New(Options{
		MaxRetries:  3,
		BaseBackoff: 100 * time.Millisecond,
		MaxBackoff:  time.Second,
		sleep: func(_ context.Context, d time.Duration) error {
			if delays != nil {
				*delays = append(*delays, d)
			}
			return nil
		},
	})
}

func TestGetJSON_Sucesso(t *testing.T) {
	var gotAccept, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept, gotUA = r.Header.Get("Accept"), r.Header.Get("User-Agent")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var p payload
	if err := novo(nil).GetJSON(context.Background(), srv.URL, &p); err != nil {
		t.Fatal(err)
	}
	if !p.Ok || gotAccept != "application/json" || gotUA == "" {
		t.Fatalf("p=%+v accept=%q ua=%q", p, gotAccept, gotUA)
	}
}

func TestGetJSON_RetentaEmCincoXXEDepoisSucesso(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var delays []time.Duration
	var p payload
	if err := novo(&delays).GetJSON(context.Background(), srv.URL, &p); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 3 || len(delays) != 2 {
		t.Fatalf("chamadas=%d delays=%v", n.Load(), delays)
	}
}

func TestGetJSON_NaoRetentaEm404(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := novo(nil).GetJSON(context.Background(), srv.URL, &payload{})
	var se *StatusError
	if !errors.As(err, &se) || se.Status != 404 || n.Load() != 1 {
		t.Fatalf("err=%v chamadas=%d", err, n.Load())
	}
}

func TestGetJSON_EsgotaTentativas(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := novo(nil).GetJSON(context.Background(), srv.URL, &payload{})
	var se *StatusError
	if !errors.As(err, &se) || se.Status != 500 || n.Load() != 4 { // 1 + 3 retries
		t.Fatalf("err=%v chamadas=%d", err, n.Load())
	}
}

func TestGetJSON_RespeitaRetryAfter(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) == 1 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var delays []time.Duration
	if err := novo(&delays).GetJSON(context.Background(), srv.URL, &payload{}); err != nil {
		t.Fatal(err)
	}
	if len(delays) != 1 || delays[0] != 7*time.Second {
		t.Fatalf("delays=%v", delays)
	}
}

func TestGetJSON_ContextoCancelado(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := New(Options{MinInterval: time.Millisecond})
	if err := c.GetJSON(ctx, srv.URL, &payload{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetJSON_JSONInvalidoNaoRetenta(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.Write([]byte(`<html>`))
	}))
	defer srv.Close()

	if err := novo(nil).GetJSON(context.Background(), srv.URL, &payload{}); err == nil || n.Load() != 1 {
		t.Fatalf("err=%v chamadas=%d", err, n.Load())
	}
}

func TestMinInterval_EspacaRequisicoes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var delays []time.Duration
	c := New(Options{
		MinInterval: 200 * time.Millisecond,
		sleep: func(_ context.Context, d time.Duration) error {
			delays = append(delays, d)
			return nil
		},
	})
	for i := 0; i < 3; i++ {
		if err := c.GetJSON(context.Background(), srv.URL, &payload{}); err != nil {
			t.Fatal(err)
		}
	}
	// a 1ª sai imediatamente (~0); 2ª e 3ª são agendadas ~200ms e ~400ms à frente.
	if len(delays) != 3 || delays[1] < 100*time.Millisecond || delays[2] < 300*time.Millisecond {
		t.Fatalf("delays=%v", delays)
	}
}

func TestBackoff_CrescenteELimitado(t *testing.T) {
	c := novo(nil)
	for attempt := 0; attempt < 10; attempt++ {
		d := c.backoff(attempt)
		if d <= 0 || d > c.opt.MaxBackoff {
			t.Fatalf("attempt=%d d=%v", attempt, d)
		}
	}
}

func TestParseRetryAfter(t *testing.T) {
	futuro := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
	passado := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	cases := []struct {
		in       string
		min, max time.Duration
	}{
		{"", 0, 0},
		{"5", 5 * time.Second, 5 * time.Second},
		{"lixo", 0, 0},
		{"-3", 0, 0},
		{futuro, 20 * time.Second, 30 * time.Second},
		{passado, 0, 0},
	}
	for _, c := range cases {
		if got := parseRetryAfter(c.in); got < c.min || got > c.max {
			t.Errorf("parseRetryAfter(%q)=%v, esperado em [%v,%v]", c.in, got, c.min, c.max)
		}
	}
}

func TestSleepCtx(t *testing.T) {
	if err := sleepCtx(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if err := sleepCtx(context.Background(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepCtx(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestStatusError_Mensagem(t *testing.T) {
	e := &StatusError{URL: "http://x/y", Status: 502}
	if got := e.Error(); got != "GET http://x/y: status 502" {
		t.Fatalf("msg=%q", got)
	}
}

func TestRetryable(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{&StatusError{Status: 429}, true},
		{&StatusError{Status: 500}, true},
		{&StatusError{Status: 404}, false},
		{context.Canceled, false},
		{context.DeadlineExceeded, false},
		{errors.New("connection reset"), true},
	}
	for _, c := range cases {
		if got := retryable(c.err); got != c.want {
			t.Errorf("retryable(%v)=%v, esperado %v", c.err, got, c.want)
		}
	}
}
