# Usa o Go local se existir; senão roda em container. Força um modo: make GO=go <alvo>
GO ?= $(if $(shell command -v go 2>/dev/null),go,docker run --rm -v $(CURDIR):/src -v comovota-gomod:/go/pkg/mod -v comovota-gocache:/root/.cache/go-build -w /src golang:1.26-alpine go)

.DEFAULT_GOAL := help
.PHONY: help test cover vet fmt check up down migrate psql

help: ## Lista os alvos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

test: ## Roda os testes
	$(GO) test ./...

cover: ## Cobertura de testes (use DATABASE_URL para incluir as migrations)
	$(GO) test -count=1 -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -1

vet: ## go vet
	$(GO) vet ./...

fmt: ## Formata o código
	$(GO) fmt ./...

check: vet test ## vet + testes (o que a CI roda)
	@test -z "$$(gofmt -l .)" || (echo "gofmt pendente: rode make fmt"; exit 1)

up: ## Sobe o Postgres e aplica as migrations
	docker compose up -d --build db migrate

down: ## Derruba os containers (mantém o volume do banco)
	docker compose down

migrate: ## Reaplica as migrations
	docker compose run --rm --build migrate

psql: ## Abre o psql no banco
	docker compose exec db psql -U comovota comovota
