# Plano de Implementação — Monitor de Votações do Congresso

Baseado em `documento_de_arquitetura_e_especifica_o_t_cnica.md`. O repositório está vazio (só README), então é um projeto novo.

## 0. Premissas de stack (o doc não fixa; ajuste se discordar)

- **Backend em Go** (módulo único): `cmd/api` (HTTP, `chi` + `net/http`), `cmd/etl` (CLI/worker com cron, concorrência via goroutines e `errgroup`), `internal/` (domínio, providers Câmara/Senado, repositório).
- **Frontend separado** em Next.js (`web/`), consumindo a API via OpenAPI (cliente tipado gerado).
- **PostgreSQL 16** com `pgx` + `sqlc` (queries SQL tipadas) e migrations em SQL puro via `goose` (o schema do doc já está pronto). Ingestão em lote com `COPY`/`INSERT ... ON CONFLICT`.
- **Cache:** CDN gratuita na frente (Redis só se as métricas pedirem) com `Cache-Control`/`stale-while-revalidate`.
- **Docker Compose** para dev (postgres + redis). Agendamento com cron/`robfig/cron` ou job do orquestrador.
- Testes: `testing` + `testify`, `testcontainers-go` para Postgres; fixtures JSON gravadas das APIs reais para testar o ETL offline.

## 0.1 Decisões de produto (2026-11-02)

- **Escopo:** apenas Plenário (Câmara `idOrgao=180` e Plenário do Senado). Comissões ficam fora.
- **Fidelidade partidária:** **fora do MVP**. Mantidos `partido_na_votacao` e `alinhou_partido` no schema (barato agora, caro de reconstruir), mas sem cálculo, MV, endpoint ou UI até a v2. Isso dispensa o spike de blocos/federações no MVP.
- **Backfill:** a partir da 57ª legislatura (2023).
- **Público:** MVP voltado ao **eleitor/cidadão** (linguagem simples, busca por parlamentar e matéria, perfis e rankings). Recursos para jornalistas e pesquisadores (filtros avançados, exportação CSV, API pública documentada, metodologia detalhada) ficam documentados como **v2**.

- **Senado:** fora do MVP; entra na v1.1, depois da Câmara.
- **Participação em votações nominais** (antes "presença"; renomeada após validação, ver `discovery/README.md` §3.5): denominador = votações nominais ocorridas enquanto o parlamentar estava em exercício (exige datas de exercício/suplência por mandato).
- **Governismo:** mantida a regra do §4.3 (abstenção/obstrução com orientação Sim ou Não contam como não alinhado).
- **Hospedagem:** VPS/PaaS barato (até ~US$30/mês): um container Go, Postgres pequeno, CDN gratuita na frente, **sem Redis no início**.

- **Votações sem proposição:** agrupar sob a matéria principal; as sem vínculo ficam numa lista "Outras votações". O vínculo vem de `proposicoesAfetadas` (direto) com fallback em `objetosPossiveis` (provável), sem heurística por evento — ver `discovery/README.md` §3.2.
- **Destaques:** a lista mostra a matéria; o detalhe traz a linha do tempo cronológica das votações, com a principal em destaque.
- **Atualização:** ciclo diário (madrugada, D-1 e D).
- **Fotos:** copiar para o nosso storage (objeto estático atrás da CDN, baixado no sync de parlamentares). **Pré-requisito:** confirmar licença/termos de reuso das fotos oficiais antes do lançamento; se não permitir, cair para link direto.

- **Produto:** nome **Como Vota Congresso**; domínio `www.comovotacongresso.com.br` (decisões para a Fase 6: DNS, HTTPS, CDN, `canonical` sem/com `www`).
- **Metodologia pública:** rascunho em `docs/metodologia.md` (aguardando revisão; deve estar publicada antes do lançamento e linkada no rodapé).

- **A decidir:** incluir a presença oficial em sessão (`/eventos/{id}/deputados`) como métrica separada. Custo baixo (~120 chamadas/ano); valor alto para o eleitor, que entende "presença" como a medida oficial.

## 1. Lacunas e decisões a fechar antes/durante (encontradas no doc)

> Discovery feito em 2026-11-02: ver `docs/discovery/README.md`. Os itens abaixo marcam o que mudou.

| # | Lacuna | Proposta |
|---|---|---|
| 1 | **[Fora do MVP]** **Fidelidade partidária** está nos objetivos (§1) mas não tem fórmula nem tabela de apoio (§4.4 só cobre governo e presença). | Definir: voto == orientação da bancada/partido do parlamentar *na data da votação*, ignorando `liberado`. Fórmula análoga à governista. |
| 2 | `parlamentares.partido_atual` não serve para fidelidade histórica (troca de partido). | **Aplicado ao schema do doc:** `votos.partido_na_votacao` (a Câmara devolve `siglaPartido` no voto) e `alinhou_partido`. |
| 3 | **[v2, junto com a fidelidade]** Orientações vêm por **bancada/bloco**, com nomes truncados (`Bl UniPpPsd...`) e sem mapa partido→bloco no payload. |  montar composição de blocos por legislatura via `/blocos` e `/partidos/{id}` (Câmara). Spike antes da Fase 2. |
| 4 | **Presença**: denominador conta todas as nominais da casa, sem considerar o período do mandato. | **Resolvido no discovery (Câmara):** intervalos de exercício via `/deputados/{id}/historico` (ver `discovery/README.md` §3.1); nova tabela `parlamentares_exercicio`. |
| 5 | Mapeamento de voto sem regra para `artigo_17`/`outro` no alinhamento (§4.3). | `alinhou_governo = NULL` para ambos (presidente não vota por orientação). |
| 6 | `votacoes.orientacao_governo` duplica `votacoes_orientacoes`. | Manter só como coluna denormalizada preenchida pelo ETL (ou remover). |
| 7 | ~~Senado sem endpoints mapeados.~~ **Resolvido no discovery:** `/votacao` (votos), `/plenario/votacao/orientacaoBancada` e `/senador/lista/atual`. Ficam: regra de presença própria (ausentes vêm com código), votação secreta e votação simbólica. | Entregar Câmara primeiro. |
| 8 | Votação sem votos ≠ sempre simbólica (pode estar em andamento ou ser sem registro). | Só classificar `simbolica` se a votação estiver concluída (`aprovacao` definida); senão reprocessar depois. |
| 10 | **Escopo:** `/votacoes` da Câmara inclui comissões. | **Decidido:** só Plenário (`idOrgao=180`). |
| 11 | Fotos copiadas exigem storage. Licença: regra geral de reprodução com crédito, **sem cláusula explícita para fotos** (ver `discovery/README.md` §3.3). | Bucket/volume + job idempotente; crédito de fonte visível; guardar `url_foto` original com fallback; pedir confirmação por escrito à Câmara/Senado antes do lançamento. |
| 9 | Falta política de histórico inicial (backfill) e de legislaturas. | Backfill a partir da 57ª legislatura (2023); parametrizável. |

## 2. Fases

### Fase 0 — Fundação (≈1–2 dias)
- Estrutura Go (`go.mod`, `cmd/`, `internal/`), `golangci-lint`, Docker Compose, CI (lint+test).
- `migrations/`: migration inicial com o schema do §3 + ajustes #2, #6; runner de migrations.
- Cliente HTTP compartilhado (`net/http` + `context`) com **retry + exponential backoff + jitter**, delay de 100–200 ms (§7.2), timeout e logging.
- **Entrega:** `docker compose up` sobe banco migrado; testes do cliente HTTP passam.

### Fase 1 — ETL Câmara, núcleo (≈3–4 dias) — **concluída**; ver README e `discovery/README.md` §3.4
0. Listagem de votações filtrada por `idOrgao=180`, seguindo `links.next`; `/votos` e `/orientacoes` não paginam.
1. Sync de parlamentares e **histórico de exercício** (`/deputados/{id}/historico` → `parlamentares_exercicio`), a partir de `/deputados?idLegislatura=`; upsert por `(id_externo, casa)`; o `deputado_` do voto também alimenta o cadastro.
2. Ingestão de votações por janela de datas (o detalhe `/votacoes/{id}` é obrigatório: é ele que traz as proposições; salvar `vinculo_confianca`) (`/votacoes?dataInicio&dataFim`):
   - (a) proposição → `proposicoes` (upsert)
   - (b) `/orientacoes` → `votacoes_orientacoes`
   - (c) `/votos` → classificar nominal/simbólica (§4.1, com ressalva #8), mapear voto (§4.2)
   - (d) calcular `alinhou_governo` (§4.3) e `alinhou_partido`, batch upsert em `votos`
   - (e) upsert em `votacoes`
3. Tudo idempotente (`ON CONFLICT DO UPDATE`), transação por votação, falha de uma não derruba o lote (registrar em tabela `etl_runs`/`etl_errors`).
4. CLI: `etl backfill --from --to` e `etl daily` (D-1..D).
- **Testes:** funções puras (`mapearVoto`, `calcularAlinhamento`) com tabela de casos do §4.3; ETL contra fixtures; re-execução não duplica.
- **Entrega:** banco populado com 1 mês real de votações da Câmara.

### Fase 2 — Métricas agregadas (≈1–2 dias) — **concluída**; ver migration `00002_metricas.sql`
- Presença usa `parlamentares_exercicio` (votações nominais dentro do intervalo de exercício).
- Materialized views: `mv_parlamentar_governismo`, `mv_parlamentar_presenca`, e agregado por partido/bancada. Índice único para `REFRESH ... CONCURRENTLY`.
- Refresh ao fim de cada ciclo do ETL (§5 passo 3).
- **Testes:** dataset sintético com resultados esperados calculados à mão (inclui `liberado`, abstenção, parlamentar ausente, troca de partido).

### Fase 3 — API (≈2–3 dias)
- Endpoints do §6 sob `/api/v1`, com validação (`go-playground/validator`), paginação e filtros; `ordenarPor=presenca|governismo` lê das MVs (nunca agrega on the fly, §7.4).
- `/votacoes/{id}`: para `simbolica`, **não** retorna lista de votos, só metadados + orientações + flag `tipo`.
- OpenAPI gerado, CORS, rate limit básico, healthcheck.
- **Testes:** integração com Postgres de teste.

### Fase 4 — Frontend (≈4–5 dias)
- Páginas: listagem de proposições (busca/filtros), detalhe de proposição (linha do tempo de votações), detalhe de votação, ranking de parlamentares, perfil do parlamentar (KPIs + histórico paginado).
- Componentes distintos para nominal vs simbólica: badge e ocultação da lista individual na simbólica (§7.3).
- Estados de carregamento/vazio/erro, responsivo, acessibilidade básica.

### Fase 5 — Senado (≈3–5 dias)
- Contrato já mapeado em `docs/discovery/README.md`; falta validar casamento `/votacao` × `orientacaoBancada` (chave `codigoVotacaoSve`/`sequencialVotacao`) e o caso simbólico/secreto.
- Mapa de votos e regra de presença próprios do Senado (`P-NRV`, `AP`, `LS`, `NCom`).
- Adapter Go com a mesma interface do da Câmara (interface `ProviderLegislativo`), reusando normalização, cálculo e MVs. Governo no Senado: entrada `partido = 'Governo'` em `orientacoesLideranca`.
- Filtro `casa` já existe no schema/API.

### Fase 6 — Operação (≈2 dias)
- Agendamento diário/horário, alertas de falha do ETL, logs estruturados.
- Deploy (container único para API/ETL + Postgres gerenciado + frontend), backups, variáveis de ambiente documentadas.
- README: como rodar, backfill, arquitetura.

## 3. Ordem de valor / marcos

1. **MVP (Fases 0–3):** API da Câmara funcionando com métricas corretas.
2. **v1 (Fase 4):** produto navegável.
3. **v1.1 (Fases 5–6):** Senado e produção.

## 4. Riscos

- Mudanças/instabilidade nas APIs governamentais → fixtures + testes de contrato, retry/backoff.
- Volume de chamadas no backfill (1 request de votos + 1 de orientações por votação) → processar com concorrência baixa, retomada por checkpoint em `etl_runs`.
- Ambiguidade de orientação por bloco/federação → validar amostra manualmente contra o site da Câmara antes de publicar números.
- Neutralidade: publicar a metodologia (como "alinhamento" é calculado e o que é excluído) na própria interface.

## 5. Próximo passo sugerido

Iniciar pela Fase 0. O spike de blocos/federações (fidelidade) fica para a v2.
