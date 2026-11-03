# Como Vota Congresso

Monitor de votações do Congresso Nacional (`www.comovotacongresso.com.br`). Começa pela Câmara dos Deputados (Plenário); Senado na v1.1.

Documentação: [`docs/plano_de_implementacao.md`](docs/plano_de_implementacao.md), [arquitetura](docs/documento_de_arquitetura_e_especifica_o_t_cnica.md), [discovery das APIs](docs/discovery/README.md), [metodologia (rascunho)](docs/metodologia.md).

## Desenvolvimento

Requer Docker. Go roda em container (ou use `make GO=go ...` com Go 1.26 local).

```bash
make up        # sobe Postgres e aplica as migrations
make test      # go test ./...
make psql      # shell no banco
make down
```

## Ingestão (ETL da Câmara)

Variáveis: `DATABASE_URL` (obrigatória), `CAMARA_BASE_URL` e `CAMARA_MIN_INTERVAL_MS` (padrão 150) opcionais. Com `go run ./cmd/etl <comando>` ou `bin/etl`:

```bash
etl sync-parlamentares                         # deputados da 57ª legislatura + períodos de exercício
etl votacoes -from 2026-10-01 -to 2026-10-31   # votações do Plenário (pula as já gravadas; -force rebusca)
etl daily                                      # ontem e hoje (rodar 1x/dia)
etl backfill                                   # desde 2023-02-01: sync-parlamentares + votações em janelas de 30 dias
etl sessoes -from 2023-02-01                    # presença oficial em sessões deliberativas do Plenário
etl metricas                                   # recalcula as materialized views (roda sozinho ao fim dos demais)
```

Participação em votações nominais e alinhamento ao governo (não é a presença oficial em sessão; ver `docs/discovery/README.md` §3.5) ficam em `mv_parlamentar_metricas` e `mv_partido_metricas`; a presença oficial em sessão, em `mv_parlamentar_presenca_sessao` e `mv_partido_presenca_sessao`. Todas são recalculadas (`REFRESH ... CONCURRENTLY`) ao fim de cada ciclo; a API só lê delas.

Tudo é idempotente; cada execução fica registrada em `etl_runs`/`etl_errors`. O `backfill` é retomável (pula votações já gravadas).

Estrutura: `cmd/api` (HTTP), `cmd/etl` (migrations e ingestão), `internal/` (`domain` regras puras, `camara` cliente, `etl` orquestração, `store` Postgres, `httpclient`, ...), `migrations/` (SQL, goose).
