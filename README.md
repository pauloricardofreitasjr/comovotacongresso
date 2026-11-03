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

Estrutura: `cmd/api` (HTTP), `cmd/etl` (migrations e ingestão), `internal/` (httpclient, config, ...), `migrations/` (SQL, goose).
