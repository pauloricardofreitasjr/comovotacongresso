// Package agenda roda os jobs do ETL em horários fixos e avisa quando falham.
package agenda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Horario calcula a próxima execução de um job.
type Horario interface {
	// Proxima devolve o primeiro instante estritamente depois de t em que o job deve rodar.
	Proxima(t time.Time) time.Time
}

// Cada roda a intervalos fixos, contados a partir do fim da execução anterior.
type Cada time.Duration

func (c Cada) Proxima(t time.Time) time.Time { return t.Add(time.Duration(c)) }

// Semanal roda um dia da semana, em Hora:Min no fuso de Loc.
type Semanal struct {
	Dia       time.Weekday
	Hora, Min int
	Loc       *time.Location
}

func (s Semanal) Proxima(t time.Time) time.Time {
	t = t.In(s.Loc)
	c := time.Date(t.Year(), t.Month(), t.Day(), s.Hora, s.Min, 0, 0, s.Loc)
	for !c.After(t) || c.Weekday() != s.Dia {
		c = time.Date(c.Year(), c.Month(), c.Day()+1, s.Hora, s.Min, 0, 0, s.Loc)
	}
	return c
}

// Job é uma tarefa agendada.
type Job struct {
	Nome    string
	Horario Horario
	// AoIniciar roda o job assim que o agendador sobe (recupera o que um reinício atrasou).
	AoIniciar bool
	// Limite cancela uma execução que passe disso; zero = sem limite.
	Limite time.Duration
	Rodar  func(ctx context.Context) error
}

// Avisador entrega alertas a uma pessoa.
type Avisador interface {
	Avisar(ctx context.Context, msg string) error
}

// Agendador executa os jobs, um por vez. Avisa na primeira falha de uma sequência e
// quando o job volta a funcionar, para não repetir o mesmo alerta a cada tentativa.
type Agendador struct {
	Jobs     []Job
	Avisador Avisador // nil = só log
	Log      *slog.Logger
	Agora    func() time.Time                                 // padrão time.Now
	Dormir   func(ctx context.Context, d time.Duration) error // padrão espera real
}

func (a *Agendador) log() *slog.Logger {
	if a.Log != nil {
		return a.Log
	}
	return slog.Default()
}

func (a *Agendador) agora() time.Time {
	if a.Agora != nil {
		return a.Agora()
	}
	return time.Now()
}

func (a *Agendador) dormir(ctx context.Context, d time.Duration) error {
	if a.Dormir != nil {
		return a.Dormir(ctx, d)
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

// Rodar bloqueia até o contexto ser cancelado.
func (a *Agendador) Rodar(ctx context.Context) error {
	if len(a.Jobs) == 0 {
		return fmt.Errorf("agenda sem jobs")
	}
	prox := make([]time.Time, len(a.Jobs))
	falhando := make([]bool, len(a.Jobs))
	for i, j := range a.Jobs {
		if j.AoIniciar {
			prox[i] = a.agora()
		} else {
			prox[i] = j.Horario.Proxima(a.agora())
		}
		a.log().Info("job agendado", "job", j.Nome, "proxima", prox[i].Format(time.RFC3339))
	}
	for {
		i := 0
		for k := range prox {
			if prox[k].Before(prox[i]) {
				i = k
			}
		}
		if d := prox[i].Sub(a.agora()); d > 0 {
			if err := a.dormir(ctx, d); err != nil {
				return nil // cancelado: encerramento normal
			}
		}
		if ctx.Err() != nil {
			return nil
		}
		a.executar(ctx, a.Jobs[i], &falhando[i])
		prox[i] = a.Jobs[i].Horario.Proxima(a.agora())
	}
}

func (a *Agendador) executar(ctx context.Context, j Job, falhando *bool) {
	rctx := ctx
	if j.Limite > 0 {
		var cancel context.CancelFunc
		rctx, cancel = context.WithTimeout(ctx, j.Limite)
		defer cancel()
	}
	ini := a.agora()
	err := j.Rodar(rctx)
	if ctx.Err() != nil {
		return // encerrando: uma execução cortada pelo shutdown não é falha do job
	}
	if err != nil {
		a.log().Error("job falhou", "job", j.Nome, "erro", err)
		if !*falhando {
			a.avisar(ctx, fmt.Sprintf("🔴 Como Vota Congresso: job %q falhou: %v", j.Nome, err))
		}
		*falhando = true
		return
	}
	a.log().Info("job concluído", "job", j.Nome, "duracao", a.agora().Sub(ini).Round(time.Second))
	if *falhando {
		a.avisar(ctx, fmt.Sprintf("🟢 Como Vota Congresso: job %q voltou a funcionar", j.Nome))
	}
	*falhando = false
}

func (a *Agendador) avisar(ctx context.Context, msg string) {
	if a.Avisador == nil {
		return
	}
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := a.Avisador.Avisar(cctx, msg); err != nil {
		a.log().Error("não foi possível enviar o alerta", "erro", err)
	}
}

// Webhook envia o alerta por POST JSON. O corpo traz "text" (Slack, Mattermost) e
// "content" (Discord); o mesmo texto nos dois campos.
type Webhook struct {
	URL    string
	Client *http.Client
}

func (w Webhook) Avisar(ctx context.Context, msg string) error {
	body, err := json.Marshal(map[string]string{"text": msg, "content": msg})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c := w.Client
	if c == nil {
		c = http.DefaultClient
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook respondeu %s", resp.Status)
	}
	return nil
}
