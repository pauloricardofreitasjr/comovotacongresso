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

## API

`DATABASE_URL=... HTTP_ADDR=:8080 go run ./cmd/api` — somente leitura, JSON, em `/api/v1`. Especificação em `/api/v1/openapi.yaml` ([fonte](internal/api/openapi.yaml)).

| Rota | O que devolve |
|---|---|
| `GET /api/v1/status` | última atualização, período e totais |
| `GET /api/v1/proposicoes` | proposições com votação (`tipo`, `ano`, `numero`, `busca`, `pagina`, `limite`) |
| `GET /api/v1/proposicoes/{id}` | proposição + linha do tempo das votações (`principal` em destaque) |
| `GET /api/v1/votacoes/{id}` | votação, orientações e votos individuais (só nominais) |
| `GET /api/v1/parlamentares` | ranking (`partido`, `uf`, `busca`, `em_exercicio`, `ordenarPor`, `ordem`, `min_votacoes`) |
| `GET /api/v1/parlamentares/{id}` | perfil, métricas, exercício e histórico de votos (`page_votos`, `limite_votos`) |
| `GET /api/v1/partidos` | métricas por partido |

Cada parlamentar traz três métricas **distintas**, sempre com as contagens: `participacao_votacoes_nominais`, `presenca_sessao` (oficial) e `alinhamento_governo`. `{id}` de proposição e parlamentar é o id interno; o de votação é o da Câmara (`2635476-36`). Erros: `{"erro":{"codigo","mensagem"}}`. Respostas levam `Cache-Control` (dados mudam 1x/dia) e CORS aberto; rate limiting fica no proxy/CDN.

## Site (Next.js)

Em `web/` (Next.js 16, React 19, TypeScript). Renderiza no servidor a partir da API Go (`API_URL`, padrão `http://localhost:8080`; o navegador nunca fala direto com ela) e reaproveita as respostas por 5 min.

```bash
cd web && npm ci
API_URL=http://localhost:8080 npm run dev      # http://localhost:3000
npm run lint && npm run typecheck && npm test  # o que a CI roda, mais npm run build
```

Telas: início, leis e projetos (busca e filtros), linha do tempo da proposição, votação (nominal ou simbólica, com placar, orientações, partidos e deputados filtráveis), deputados (rankings), perfil, partidos e metodologia (renderiza `docs/metodologia.md`; `METODOLOGIA_PATH` aponta o arquivo em produção). Sem JavaScript de cliente além do destaque da navegação: filtros e paginação são formulários e links.

Stack completa em containers: `docker compose --profile app up --build` (site em `:3000`, API em `:8080`).

## Ingestão (ETL da Câmara)

Variáveis: `DATABASE_URL` (obrigatória), `CAMARA_BASE_URL` e `CAMARA_MIN_INTERVAL_MS` (padrão 150) opcionais. Com `go run ./cmd/etl <comando>` ou `bin/etl`:

```bash
etl sync-parlamentares                         # deputados da legislatura vigente (e da anterior nos 90 dias após a virada) + períodos de exercício
etl votacoes -from 2026-10-01 -to 2026-10-31   # votações do Plenário (pula as já gravadas; -force rebusca)
etl daily                                      # ontem e hoje (rodar 1x/dia)
etl backfill                                   # desde 2023-02-01: sync-parlamentares + votações em janelas de 30 dias
etl sessoes -from 2023-02-01                    # presença oficial em sessões deliberativas do Plenário
etl metricas                                   # recalcula as materialized views (roda sozinho ao fim dos demais)
```

Participação em votações nominais e alinhamento ao governo (não é a presença oficial em sessão; ver `docs/discovery/README.md` §3.5) ficam em `mv_parlamentar_metricas` e `mv_partido_metricas`; a presença oficial em sessão, em `mv_parlamentar_presenca_sessao` e `mv_partido_presenca_sessao`. Todas são recalculadas (`REFRESH ... CONCURRENTLY`) ao fim de cada ciclo; a API só lê delas.

Tudo é idempotente; cada execução fica registrada em `etl_runs`/`etl_errors`. O `backfill` é retomável (pula votações já gravadas).

Estrutura: `cmd/api` (HTTP), `cmd/etl` (migrations e ingestão), `internal/` (`domain` regras puras, `camara` cliente, `etl` orquestração, `store` Postgres, `httpclient`, ...), `migrations/` (SQL, goose).
