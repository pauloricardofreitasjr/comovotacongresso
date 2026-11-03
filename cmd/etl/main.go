// Comando etl: migrações e ingestão de dados da Câmara.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"comovotacongresso/internal/camara"
	"comovotacongresso/internal/config"
	"comovotacongresso/internal/etl"
	"comovotacongresso/internal/httpclient"
	"comovotacongresso/internal/migrate"
	"comovotacongresso/internal/store"
)

const (
	usage = `uso: etl <comando> [flags]

comandos:
  migrate | migrate-down | migrate-status
  metricas                   recalcula presença e alinhamento (também roda ao fim dos demais)
  sync-parlamentares [-legislatura 57]
  votacoes -from AAAA-MM-DD -to AAAA-MM-DD [-force]
  sessoes  -from AAAA-MM-DD -to AAAA-MM-DD [-force]   presença oficial em sessões deliberativas
  daily                      votações de ontem e hoje (horário de Brasília)
  backfill [-from 2023-02-01] [-to hoje] [-legislatura 57]
`
	inicioLegislatura57 = "2023-02-01"
	janelaBackfillDias  = 30
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1], os.Args[2:]); err != nil {
		slog.Error("falha", "erro", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd string, args []string) error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	switch cmd {
	case "migrate":
		return migrate.Run(cfg.DatabaseURL, "up")
	case "migrate-down":
		return migrate.Run(cfg.DatabaseURL, "down")
	case "migrate-status":
		return migrate.Run(cfg.DatabaseURL, "status")
	case "sync-parlamentares", "votacoes", "daily", "backfill", "metricas", "sessoes":
		return ingestao(ctx, cfg, cmd, args)
	default:
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("comando desconhecido: %q", cmd)
	}
}

func ingestao(ctx context.Context, cfg config.Config, cmd string, args []string) error {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	legislatura := fs.Int("legislatura", 57, "legislatura (sync-parlamentares/backfill)")
	from := fs.String("from", "", "data inicial AAAA-MM-DD")
	to := fs.String("to", "", "data final AAAA-MM-DD")
	force := fs.Bool("force", false, "rebuscar votações já gravadas")
	if err := fs.Parse(args); err != nil {
		return err
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	e := &etl.ETL{
		Fonte: camara.New(httpclient.New(httpclient.Options{MinInterval: cfg.CamaraMinInterval}), cfg.CamaraBaseURL),
		Store: store.New(pool),
		Log:   slog.Default(),
	}
	hoje := dia(time.Now().In(brasilia()))

	switch cmd {
	case "metricas":
		return e.AtualizarMetricas(ctx)

	case "sync-parlamentares":
		if _, err = e.SyncParlamentares(ctx, *legislatura); err != nil {
			return err
		}
		return e.AtualizarMetricas(ctx)

	case "votacoes":
		ini, fim, err := intervalo(*from, *to, hoje)
		if err != nil {
			return err
		}
		if _, err = e.IngerirVotacoes(ctx, ini, fim, !*force); err != nil {
			return err
		}
		return e.AtualizarMetricas(ctx)

	case "sessoes":
		ini, fim, err := intervalo(*from, *to, hoje)
		if err != nil {
			return err
		}
		if _, err = e.IngerirSessoes(ctx, ini, fim, !*force); err != nil {
			return err
		}
		return e.AtualizarMetricas(ctx)

	case "daily":
		// reprocessa D-1 e D: votações recentes podem ganhar resultado/ajustes depois.
		if _, err = e.IngerirVotacoes(ctx, hoje.AddDate(0, 0, -1), hoje, false); err != nil {
			return err
		}
		if _, err = e.IngerirSessoes(ctx, hoje.AddDate(0, 0, -1), hoje, false); err != nil {
			return err
		}
		return e.AtualizarMetricas(ctx)

	case "backfill":
		if *from == "" {
			*from = inicioLegislatura57
		}
		ini, fim, err := intervalo(*from, *to, hoje)
		if err != nil {
			return err
		}
		if _, err := e.SyncParlamentares(ctx, *legislatura); err != nil {
			return err
		}
		var total etl.Resumo
		for _, j := range janelas(ini, fim, janelaBackfillDias) {
			r, err := e.IngerirVotacoes(ctx, j[0], j[1], !*force)
			if err != nil {
				return err
			}
			if _, err := e.IngerirSessoes(ctx, j[0], j[1], !*force); err != nil {
				return err
			}
			total.Total += r.Total
			total.Processadas += r.Processadas
			total.Nominais += r.Nominais
			total.Simbolicas += r.Simbolicas
			total.Indisponiveis += r.Indisponiveis
			total.Adiadas += r.Adiadas
			total.Puladas += r.Puladas
			total.Erros += r.Erros
		}
		slog.Info("backfill concluído", "resumo", total)
		return e.AtualizarMetricas(ctx)
	}
	return nil
}

func brasilia() *time.Location {
	l, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.FixedZone("BRT", -3*3600)
	}
	return l
}

// dia zera a hora mantendo o fuso.
func dia(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func intervalo(from, to string, hoje time.Time) (ini, fim time.Time, err error) {
	if from == "" {
		return ini, fim, errors.New("informe -from AAAA-MM-DD")
	}
	if ini, err = time.ParseInLocation("2006-01-02", from, hoje.Location()); err != nil {
		return ini, fim, fmt.Errorf("-from inválido: %w", err)
	}
	fim = hoje
	if to != "" {
		if fim, err = time.ParseInLocation("2006-01-02", to, hoje.Location()); err != nil {
			return ini, fim, fmt.Errorf("-to inválido: %w", err)
		}
	}
	if fim.Before(ini) {
		return ini, fim, errors.New("-to anterior a -from")
	}
	return ini, fim, nil
}

// janelas divide [ini, fim] em blocos de até n dias, sem sobreposição.
func janelas(ini, fim time.Time, n int) [][2]time.Time {
	var out [][2]time.Time
	for cur := ini; !cur.After(fim); cur = cur.AddDate(0, 0, n) {
		end := cur.AddDate(0, 0, n-1)
		if end.After(fim) {
			end = fim
		}
		out = append(out, [2]time.Time{cur, end})
	}
	return out
}
