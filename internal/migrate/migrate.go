// Package migrate aplica as migrações SQL embutidas (goose).
package migrate

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"

	"comovotacongresso/migrations"
)

// Run executa "up", "down" ou "status" contra o banco em dsn.
func Run(dsn, cmd string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	switch cmd {
	case "up":
		return goose.Up(db, ".")
	case "down":
		return goose.Down(db, ".")
	case "status":
		return goose.Status(db, ".")
	default:
		return fmt.Errorf("comando de migração desconhecido: %q", cmd)
	}
}
