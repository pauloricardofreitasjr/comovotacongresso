// Comando etl: migrações e (nas próximas fases) ingestão de dados.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"comovotacongresso/internal/config"
	"comovotacongresso/internal/migrate"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: etl <migrate|migrate-down|migrate-status>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		slog.Error("falha", "erro", err)
		os.Exit(1)
	}
}

func run(cmd string) error {
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
	default:
		return fmt.Errorf("comando desconhecido: %q", cmd)
	}
}
