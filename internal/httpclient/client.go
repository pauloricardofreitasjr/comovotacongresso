// Package httpclient é o cliente HTTP compartilhado pelos providers da Câmara e do Senado:
// JSON, retry com backoff exponencial e jitter, e intervalo mínimo entre requisições.
package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Options configura o Client. Zero values recebem defaults razoáveis.
type Options struct {
	HTTPClient  *http.Client
	UserAgent   string
	MinInterval time.Duration // intervalo mínimo entre requisições (rate limiting educado)
	MaxRetries  int           // tentativas adicionais após a primeira
	BaseBackoff time.Duration
	MaxBackoff  time.Duration

	// sleep é injetável para testes.
	sleep func(context.Context, time.Duration) error
}

// Client faz GETs JSON com retry.
type Client struct {
	opt Options

	mu   sync.Mutex
	next time.Time // quando a próxima requisição pode sair
}

// StatusError representa uma resposta HTTP não-2xx.
type StatusError struct {
	URL    string
	Status int
}

func (e *StatusError) Error() string { return fmt.Sprintf("GET %s: status %d", e.URL, e.Status) }

func New(opt Options) *Client {
	if opt.HTTPClient == nil {
		opt.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if opt.UserAgent == "" {
		opt.UserAgent = "comovotacongresso/1.0 (+https://www.comovotacongresso.com.br)"
	}
	if opt.MaxRetries == 0 {
		opt.MaxRetries = 4
	}
	if opt.BaseBackoff == 0 {
		opt.BaseBackoff = 500 * time.Millisecond
	}
	if opt.MaxBackoff == 0 {
		opt.MaxBackoff = 30 * time.Second
	}
	if opt.sleep == nil {
		opt.sleep = sleepCtx
	}
	return &Client{opt: opt}
}

// GetJSON faz GET em url e decodifica o corpo em out.
func (c *Client) GetJSON(ctx context.Context, url string, out any) error {
	var lastErr error
	for attempt := 0; attempt <= c.opt.MaxRetries; attempt++ {
		if err := c.wait(ctx); err != nil {
			return err
		}
		body, retryAfter, err := c.do(ctx, url)
		if err == nil {
			if err := json.Unmarshal(body, out); err != nil {
				return fmt.Errorf("GET %s: decodificando JSON: %w", url, err)
			}
			return nil
		}
		lastErr = err
		if !retryable(err) || attempt == c.opt.MaxRetries {
			break
		}
		delay := c.backoff(attempt)
		if retryAfter > delay {
			delay = retryAfter
		}
		if err := c.opt.sleep(ctx, delay); err != nil {
			return err
		}
	}
	return lastErr
}

func (c *Client) do(ctx context.Context, url string) (body []byte, retryAfter time.Duration, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.opt.UserAgent)

	resp, err := c.opt.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, parseRetryAfter(resp.Header.Get("Retry-After")), &StatusError{URL: url, Status: resp.StatusCode}
	}
	body, err = io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	return body, 0, err
}

// wait respeita MinInterval entre requisições de qualquer goroutine.
func (c *Client) wait(ctx context.Context) error {
	if c.opt.MinInterval <= 0 {
		return ctx.Err()
	}
	c.mu.Lock()
	now := time.Now()
	at := c.next
	if at.Before(now) {
		at = now
	}
	c.next = at.Add(c.opt.MinInterval)
	c.mu.Unlock()
	return c.opt.sleep(ctx, time.Until(at))
}

// backoff: exponencial com jitter total (0..base*2^attempt), limitado a MaxBackoff.
func (c *Client) backoff(attempt int) time.Duration {
	d := c.opt.BaseBackoff << attempt
	if d <= 0 || d > c.opt.MaxBackoff {
		d = c.opt.MaxBackoff
	}
	return time.Duration(rand.Int64N(int64(d)) + 1)
}

func retryable(err error) bool {
	var se *StatusError
	if errors.As(err, &se) {
		return se.Status == http.StatusTooManyRequests || se.Status >= 500
	}
	// erros de contexto nunca são retentados; demais erros de rede, sim.
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if s, err := strconv.Atoi(v); err == nil && s >= 0 {
		return time.Duration(s) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(time.Until(t), 0)
	}
	return 0
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
