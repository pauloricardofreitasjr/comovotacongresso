package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"
)

// Roda up → down → up num schema temporário do Postgres apontado por DATABASE_URL.
// Sem DATABASE_URL o teste é pulado (ex.: rode `make up` antes).
func TestMigracoes_UpDownUp(t *testing.T) {
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Skip("DATABASE_URL não definido")
	}
	schema := fmt.Sprintf("teste_%d", time.Now().UnixNano())

	admin, err := sql.Open("pgx", base)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	ctx := context.Background()
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Skipf("banco indisponível: %v", err)
	}
	t.Cleanup(func() { admin.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE") })

	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	dsn := u.String()

	tabelas := func() int {
		var n int
		err := admin.QueryRowContext(ctx,
			`SELECT count(*) FROM information_schema.tables WHERE table_schema=$1 AND table_name <> 'goose_db_version'`,
			schema).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}

	if err := Run(dsn, "up"); err != nil {
		t.Fatalf("up: %v", err)
	}
	if n := tabelas(); n != 8 {
		t.Fatalf("após up: %d tabelas, esperado 8", n)
	}
	if err := Run(dsn, "down"); err != nil {
		t.Fatalf("down: %v", err)
	}
	if n := tabelas(); n != 0 {
		t.Fatalf("após down: %d tabelas, esperado 0", n)
	}
	if err := Run(dsn, "up"); err != nil {
		t.Fatalf("up de novo: %v", err)
	}

	// invariante do schema: proposicao_id e vinculo_confianca andam juntos.
	_, err = admin.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s.votacoes(id,casa,data_hora,tipo,proposicao_id) VALUES('x','camara',now(),'nominal',1)`, schema))
	if err == nil {
		t.Fatal("esperava violação do CHECK de vínculo")
	}
}

func TestRun_ComandoDesconhecido(t *testing.T) {
	if err := Run("postgres://invalido", "xyz"); err == nil {
		t.Fatal("esperava erro")
	}
}
