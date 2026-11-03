// Package migrations embute os arquivos SQL de migração.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
