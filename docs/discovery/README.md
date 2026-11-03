# Discovery — APIs da Câmara e do Senado

Feito em 2026-11-02 a partir dos Swaggers oficiais e de chamadas reais. Specs completas em `raw/`, amostras de resposta em `samples/` (usar como fixtures do ETL).

- Câmara: spec em `https://dadosabertos.camara.leg.br/api/v2/api-docs` (OpenAPI 3.0.1; a URL `swagger/api.html` só renderiza o Swagger UI).
- Senado: spec em `https://legis.senado.leg.br/dadosabertos/v3/api-docs` (OpenAPI 3.1, v4.1.3).

## 1. Câmara dos Deputados

| Fato | Evidência / implicação |
|---|---|
| `GET /votacoes` traz **todos os órgãos** (comissões incluídas). Em outubro/2026, a maioria era CCULT, CSAUDE etc. | Filtrar `idOrgao=180` (PLEN) para o escopo do produto. Decisão de produto: incluir comissões? |
| Id da votação é string `"2635476-36"` (`{idEvento/ordem}-{seq}`). | `votacoes.id VARCHAR` do doc está correto. |
| `GET /votacoes` pagina (`pagina`, `itens` ≤ 100, links `next`/`last`). `/votos` e `/orientacoes` **não** paginam (466 votos numa resposta). | Cliente precisa seguir `links.next` só na listagem. |
| Votação **simbólica**: `/votos` retorna `dados: []`, mas `aprovacao` vem preenchido (1/0). | Regra §4.1 do doc confirmada. Amostras: `camara_v_2620905-53*`. |
| Votação nominal: `/votos` com `tipoVoto` e o próprio `deputado_` (inclui **`siglaPartido` na data do voto**, `siglaUf`, `idLegislatura`, `urlFoto`). | Resolve a lacuna #2: guardar `partido_na_votacao` a partir daqui. Dá também para criar parlamentar sem chamar `/deputados`. |
| Valores de `tipoVoto` observados: `Sim`, `Não`, `Abstenção`, `Artigo 17`. (`Obstrução` não apareceu na amostra; manter no mapa e logar valores desconhecidos como `outro`.) | §4.2 ok. |
| `/orientacoes` retorna `siglaPartidoBloco`, `codTipoLideranca` (`P` = partido, `B` = bloco/federação/liderança), `orientacaoVoto`. | Ver itens abaixo. |
| **Governo aparece como `siglaPartidoBloco = "Governo"` com `codTipoLideranca = "B"`**, não `GOV.`/`Gov.` como diz o doc. | Corrigir §2.1 e §4.3: casar `lower(siglaPartidoBloco) = 'governo'`. Deixar configurável. |
| `orientacaoVoto` observados: `Sim`, `Não`, `Liberado` e **`""` (vazio)**. | Vazio = sem orientação; tratar como `liberado`/NULL. `obstrucao` não apareceu, mas existe no enum. |
| Entradas `B` incluem `Maioria`, `Minoria`, `Oposição`, `Governo` (lideranças) e federações/blocos (`Fdr PT-PCdoB-PV`, `Fdr PSOL-REDE`, `Bl UniPpPsd...` — nome truncado). Entradas `P` são partidos isolados. | Lacuna #3 confirmada: **não há mapa explícito partido→bloco** na resposta e o nome do bloco vem truncado. Fidelidade partidária exige tabela de composição de blocos por legislatura (`/blocos`, `/partidos/{id}`) ou derivar de sigla. **Spike restante.** |
| `/votacoes/{id}` não traz `proposicaoObjeto` na listagem (veio `null`); a proposição vem em `ultimaApresentacaoProposicao`, `proposicoesAfetadas[]` (com `siglaTipo`, `numero`, `ano`, `ementa`) e `objetosPossiveis[]`. Há também `/proposicoes/{id}/votacoes`. | Vincular votação→proposição via `proposicoesAfetadas` (e fallback `objetosPossiveis`). Nem toda votação tem proposição (requerimentos, pareceres `SSP`) → `proposicao_id` nulo é normal. |
| Em votações nominais, a descrição já traz o placar (`Sim: 461; Não: 4; Total: 465.`). | Útil para validar totais no ETL (contagem de votos == placar). |

## 2. Senado Federal

O doc citava `/materia/{codigo}/votacoes` e `/plenario/votacao/{codigoSessao}`. **O primeiro está marcado DEPRECATED** na spec atual; o contrato recomendado é outro:

| Necessidade | Endpoint atual | Observações |
|---|---|---|
| Votações + votos individuais | `GET /dadosabertos/votacao?dataInicio=&dataFim=` (também `codigoMateria`, `sigla/numero/ano`, `codigoParlamentar`, `codigoSessao`) | Retorna lista; cada item já inclui `votos[]` (`codigoParlamentar`, `siglaVotoParlamentar`, `siglaPartidoParlamentar`, `siglaUFParlamentar`), `identificacao` ("PEC 221/2019"), `codigoMateria`, `resultadoVotacao` (`A`), `votacaoSecreta` (`S/N`). Uma chamada substitui `/votos` + proposição. |
| Orientações de bancada | `GET /dadosabertos/plenario/votacao/orientacaoBancada/{ini}/{fim}` (datas `YYYYMMDD`) | `orientacoesLideranca[]` com `partido` e `voto` (`SIM`/`NAO`/`LIVRE`). **`Governo` aparece como "partido"**, igual à Câmara. Casa com `/votacao` por `codigoVotacaoSve`/`sequencialVotacao` (confirmar chave). |
| Senadores | `GET /dadosabertos/senador/lista/atual` e `/senador/lista/legislatura/{ini}/{fim}` | JSON aninhado (`ListaParlamentarEmExercicio.Parlamentares.Parlamentar[].IdentificacaoParlamentar`), ids como **string**. Inclui bloco e foto. |
| Votos de veto/CN | `/plenario/resultado/veto/...`, `/plenario/resultado/cn/{data}` | Fora do escopo do MVP. |

Particularidades:
- Header `Accept: application/json` é necessário (default é XML).
- **Códigos de voto do Senado são diferentes**: `Sim`, `Não`, `Presidente (art. 51 RISF)`, `P-NRV` (presente, não registrou voto), `AP` (ausência/ausente?), `LS` (licença), `NCom` (não compareceu), possivelmente `Abstenção`. Precisa de **mapa próprio**, e *presença* no Senado deve usar `/plenario/lista/tiposComparecimento` para saber o que conta como presente — **a regra "ausente na lista = sem registro" do doc não vale no Senado**: ausentes aparecem com código.
- `totalVotosSim/Nao` vieram `null` em `/votacao`; os totais corretos estão em `qtdVotosSim/Nao` no endpoint de orientações.
- Votos secretos (`votacaoSecreta = S`) não têm voto individual → tratar como categoria própria (nem nominal nem simbólica).
- Votação de mérito só tem `votos[]` quando nominal; é preciso verificar o caso simbólico no Senado (não coberto pela amostra).

## 3. Impacto no plano e no doc de arquitetura

1. **Corrigir** §2.1/§4.3: Governo = `siglaPartidoBloco "Governo"` (Câmara) e `partido "Governo"` (Senado); `orientacaoVoto` vazio = sem orientação.
2. **Corrigir** §2.2: usar `/votacao` e `/orientacaoBancada`; os endpoints citados no doc estão deprecated.
3. **Adicionar** ao schema: `votos.partido_na_votacao`, `votos.alinhou_partido`, e `tipo_votacao_enum` com `secreta`. Mapa de votos por casa (o Senado tem `P-NRV`, `AP`, `LS`, `NCom`).
4. **Presença no Senado** precisa de regra própria (ausência vem com código).
5. **Escopo**: decidir Plenário apenas (`idOrgao=180` / Plenário do Senado) vs. comissões.
6. A Fase 5 (Senado) fica menor do que o previsto: um endpoint já traz votos + matéria; o risco passa a ser o casamento com orientações e a regra de presença.

## 3.1 Período de exercício (Câmara) — resolvido

`GET /deputados/{id}/historico` devolve a linha do tempo do mandato (amostras: `samples/camara_dep_historico.json`, `camara_dep_historico_suplente.json`). Campos: `idLegislatura`, `dataHora`, `siglaPartido`, `situacao`, `condicaoEleitoral`, `descricaoStatus`.

- Valores de `situacao` observados: `Exercício`, `Licença`, `SUPLENCIA`, `CONVOCADO`, `FIM_MANDATO` e `null` (linha informativa "Nome/Partido no início da legislatura").
- **Regra para montar intervalos de exercício:** abre em `situacao = 'Exercício'` e fecha na próxima linha com `Licença`, `SUPLENCIA` ou `FIM_MANDATO`. `CONVOCADO` e `null` não abrem nem fecham.
- Titulares e suplentes seguem o mesmo padrão, então um único algoritmo cobre os dois (ex.: suplente 103758 teve dois mandatos de exercício em 2023–2024).
- O histórico mistura legislaturas; filtrar por `idLegislatura`. Há linhas espúrias (ex.: `idLegislatura = 58` com data de 2023) que devem ser ignoradas.
- Troca de partido aparece como linha `Exercício` com `descricaoStatus = 'Alteração de partido'`, o que dá também o histórico de partido (útil para a v2 de fidelidade).
- Custo: 1 chamada por deputado (~800 por legislatura, contando suplentes), feita uma vez e atualizada incrementalmente.
- Legislatura atual: 57 (2023-02-01 a 2027-01-31), via `/legislaturas`.
- **Presença** = votos do deputado / nominais do Plenário com `data_hora` dentro de algum intervalo de exercício dele. Tabela nova: `parlamentares_exercicio(parlamentar_id, inicio, fim, condicao, partido)`.
- Ressalva: o horário da votação (`dataHoraRegistro`) é comparado com `dataHora` do histórico; licenças no mesmo dia da votação podem gerar bordas, por isso validar contra uma amostra de presença conhecida do site da Câmara.
- **Senado:** equivalente ainda não investigado (v1.1).

## 3.2 Vínculo votação → proposição (Câmara, Plenário)

Amostra: 5 votações de um evento (`idEvento=83057`) + 50 recentes + 75 sorteadas de 2023, 2024 e 2025 (`samples/camara_evento_83057_resumo.json`, `camara_vot_*.json`).

- `proposicaoObjeto` da listagem vem sempre `null`; o vínculo vem do **detalhe** `/votacoes/{id}` (1 chamada extra por votação).
- `proposicoesAfetadas[]` resolve a maioria: **~85% das votações têm exatamente 1 proposição** (inclusive requerimentos, que já apontam para a matéria — ex.: o requerimento `2635476-23` aponta para o PLP 186/2026, a mesma do mérito `2635476-36`). Sem precisar de heurística por evento.
- **~10% vêm com `proposicoesAfetadas` vazio**, quase todas "Alteração do Regime de Tramitação…" e "Aprovado o requerimento nº X/AAAA… urgência (art. 155)". Nesses casos `objetosPossiveis[]` tem 1 item (ex.: PL 2834/2023), mas ele **pode ser uma proposição apensada, não a citada na descrição** (a descrição citava o PL 4670/2020) → usar como vínculo de menor confiança.
- **~2% vêm com 2 proposições** (matéria + o `REQ` do requerimento de urgência). Regra: matéria principal = a proposição de tipo diferente de `REQ`; o `REQ` vira atributo da votação.
- Não observei votações sem nenhum vínculo na amostra do Plenário; mesmo assim manter `proposicao_id` anulável.

**Algoritmo adotado:** (1) `proposicoesAfetadas` sem `REQ`, se houver exatamente uma → vínculo `direto`; (2) senão, `objetosPossiveis` de tamanho 1 → vínculo `provavel`; (3) senão `proposicao_id = NULL` → lista "Outras votações". Guardar `vinculo_confianca` em `votacoes` e mostrar o selo só quando `provavel`.

## 3.3 Licença e fotos (pesquisa em 2026-11-02; não é parecer jurídico)

**Câmara** — [Termos gerais do portal](https://www2.camara.leg.br/termo-de-uso-e-politica-de-privacidade/termos-gerais), item 2: *"Todas as informações publicadas no Portal da Câmara são de domínio público, gratuitas e podem ser reproduzidas desde que citada a fonte 'Câmara dos Deputados' e a autoria da informação, quando houver."*
- Uma busca citou uma versão do texto que inclui "dados, imagens e infografias"; ao ler a página diretamente, o trecho **não menciona imagens nem fotos**. Os resultados divergem, então **não há cláusula explícita sobre fotos de deputados**.
- Não trata de uso comercial de forma explícita (só diz "gratuitas"); isenta a Câmara de responsabilidade por imprecisões e dados de terceiros.
- A página de termos específica dos dados abertos (`dadosabertos.camara.leg.br/termosdeuso.html`) retornou "Página não encontrada".

**Senado** — a página do Senado Fotos diz que as imagens são protegidas por direitos autorais e "podem ser livremente utilizadas, desde que dados os créditos"; a [Política de Uso](https://www12.senado.leg.br/assessoria-de-imprensa/politica-de-uso) permite reproduzir com citação de fonte e sem descaracterizar o conteúdo. Não identifiquei uma licença nomeada (ex.: CC) para dados ou fotos. Contato indicado para dúvidas de vídeos/imagens: seacer@senado.leg.br.

**Conclusão:**
- Usar os **dados** é seguro com atribuição ("Fonte: Câmara dos Deputados / Senado Federal").
- Para **copiar fotos** para storage próprio, a base é a regra geral de reprodução com crédito, mas **sem cláusula explícita** na Câmara. Risco baixo, não zero.
- Plano: (1) crédito de fonte visível no rodapé e em cada perfil; (2) pedir confirmação por escrito à Câmara (canal de atendimento do portal) e ao Senado antes do lançamento; (3) guardar `url_foto` original e servir o espelho com fallback para o link oficial; (4) não descaracterizar nem editar as imagens além de redimensionar.

## 3.4 Achados da implementação da Fase 1 (API real, 2026-11-02)

- **`/deputados?idLegislatura=57` repete deputados**: vem uma linha por (deputado, partido). 882 linhas para 648 deputados (197 com troca de partido). É preciso deduplicar por `id`; **`siglaPartido` da listagem não é confiável como partido atual** — usamos o partido do último intervalo de exercício.
- **Situações de exercício na 57ª legislatura:** `Exercício` (abre), `CONVOCADO` e `null` (neutras), e que **encerram**: `Licença`, `SUPLENCIA`, `VACANCIA` (renúncia etc., 24 casos), `SUSPENSO` (3 casos). Outras legislaturas têm `FIM_MANDATO` e `Afastado`. O ETL avisa quando aparece situação desconhecida na legislatura processada.
- **Validação:** com essas regras, **513 deputados** ficam com exercício em aberto hoje, igual ao número de cadeiras.
- **Placar:** nas 3 votações nominais de 13–15/out/2026, os votos gravados batem exatamente com o placar escrito na descrição oficial (Sim/Não), mais o voto do presidente (`Artigo 17`).
- **Condição eleitoral** inclui `EFETIVADO` (suplente efetivado), além de `Titular` e `Suplente`.
- **Ritmo:** com 150 ms entre requisições e 4 votações em paralelo, ~0,6 s por votação; o sync de parlamentares (648 históricos) leva ~1m40s.

### Backfill completo da Câmara (fev/2023 a 02/nov/2026, só Plenário)

- **6.251 votações gravadas** (1.079 nominais, 5.172 simbólicas) e **436.085 votos individuais**; ~55 min com 150 ms entre requisições. 996 votações em 2023, 2.074 em 2024, 1.938 em 2025, 1.243 em 2026.
- **83 votações (1,3%) listadas pela API não existem** no detalhe nem nos votos (404). Tratadas como "indisponíveis na fonte" (registradas em `etl_errors` com ref `indisponivel/…`, fora da contagem de erros).
- **91 "adiadas"**: sem votos e sem resultado (`aprovacao` nulo). Não são gravadas; as recentes são reavaliadas pelo `daily`, as antigas provavelmente nunca ganham resultado.
- **Defeito da fonte:** em 2 votações nominais (`2576389-4`, `2645346-18`) **todos os votos vêm com `tipoVoto: null`** (887 votos, gravados como `outro`). A presença é preservada; o alinhamento fica de fora. O placar só existe no texto da descrição.
- **Orientação do Governo:** 490 nominais com "Sim", 344 com "Não", 28 "Liberado", 3 "Obstrução" e 214 sem orientação do Governo registrada. Logo, 337.775 dos 436.085 votos (77%) entram no cálculo de alinhamento.
- **Vínculo com proposição:** 12 votações sem proposição; 1.650 com vínculo `provavel`, quase todas votações de procedimento ("Alteração do Regime de Tramitação…", "requerimento de urgência (art. 155)") ligadas à matéria a que se aplicam.

## 3.5 Métricas (Fase 2) — resultados e validação da "presença"

- **Participação em votações nominais** (antes chamada "presença"): denominador = votações nominais ocorridas dentro de um período de exercício do deputado **mais** as em que ele votou; numerador = votos registrados. Nenhum deputado passou de 100%.
- **Governismo por partido** (partido do momento do voto): PT 97,8%, PCdoB 97,0%, PSB 92,3%, PSD 77,0%, UNIÃO 65,2%, PL 30,2%, NOVO 17,3%. Ordem coerente com o espectro político, o que valida a regra de alinhamento.

### Validação contra a presença oficial em sessão (2025)

A presença oficial da Câmara é por **sessão** (registro de presença), disponível em `GET /eventos/{id}/deputados` (lista quem registrou presença; ex.: 502 deputados na sessão de 14/10/2026; não informa ausência justificada). Comparei, para 2025, as 121 sessões deliberativas do Plenário com os votos nas 408 votações nominais:

| | Resultado |
|---|---|
| Presença média em sessão | 88,4% |
| Participação média nas nominais | 75,0% |
| Correlação entre as duas | 0,79 |

- As duas métricas são **diferentes**. Há deputados com 96–99% de presença em sessão e só 22–26% de votos nas nominais (ex.: Antonio Brito, Pinheirinho), e líderes como Elmar Nascimento (81% sessão, 29% nominais). Portanto os números baixos **não são erro de cálculo**: o deputado registra presença, mas não vota em muitas nominais.
- **Decisão:** a métrica foi renomeada para **participação em votações nominais** (`pct_participacao`, `votacoes_votou`) e não deve ser apresentada como "presença em plenário". A presença oficial em sessão é uma métrica separada, possível via `/eventos/{id}/deputados` (≈120 sessões/ano × 1 chamada), a decidir se entra no produto.
- A Aos Fatos documenta erros frequentes na leitura dos dados de presença da Câmara; reforça a necessidade de nomear e explicar cada métrica.
- 4 parlamentares ficaram sem nenhuma votação elegível (entraram e saíram sem nominal no período).

## 3.6 Presença oficial em sessão (implementada)

- **Fonte:** `GET /eventos?idOrgao=180` (filtra `descricaoTipo = "Sessão Deliberativa"` e `situacao = "Encerrada"`) + `GET /eventos/{id}/deputados` (quem registrou presença). Só lista presentes; não informa ausência justificada.
- **Carga real (fev/2023 a nov/2026):** 393 sessões deliberativas, ~1m20s, sem erros; 2 não encerradas ficam de fora até terminarem (o `daily` reprocessa D-1 e D).
- **Elegibilidade:** sessões iniciadas durante um período de exercício do deputado, mais aquelas em que registrou presença. Mesmo desenho da participação nominal.
- **Conferência:** para 2025, os números do banco coincidem com o cálculo independente anterior (ex.: Isnaldo Bulhões Jr. 58 presenças, Bivar 59, Elmar 98; 120 sessões encerradas contra 121 sem o filtro de situação).
- **Presença média (período inteiro): 92,3%**, contra 79,1% de participação nas nominais. Exemplo extremo: Antonio Brito, 100% de presença em sessão e 35% de participação nas nominais.
- As duas métricas são mantidas separadas, com nomes próprios (`pct_presenca_sessao` e `pct_participacao`).
- **Siglas de partido na lista de presença divergem das do exercício** ("Republican", "Podemos", "Solidaried", "União" contra REPUBLICANOS, PODE, SOLIDARIEDADE, UNIÃO). Agrupar por partido pela grafia da presença dividia cada partido em dois grupos e mostrava 0% de presença para vários na página de partidos (achado em revisão de UX, nov/2026). A migration `00004` passou a agrupar pela sigla do exercício; a presença por parlamentar não mudou (média 92,3%).

### Placar da descrição oficial × votos registrados (nov/2026)

- **1.073 das 1.079 votações nominais** trazem "Sim: N; Não: N" na descrição; em **1.069** o placar bate exatamente com os votos gravados. As 4 que divergem: **PDL 995/2026** e **OBJ 2/2025** (votos sem posição, `tipoVoto` nulo, já conhecidos), **MPV 1151/2022** (337 × 27 gravado, 327 × 27 na descrição) e **PL 5230/2023, Emenda 40** (315 × 87 gravado, 35 × 375 na descrição, resultado "Rejeitada", coerente com a descrição). Conferido direto na API da Câmara: **o banco reproduz a fonte** (315 × 87 e 337 × 27); a inconsistência está nos dados da Câmara. O site avisa quando isso acontece.
- **Resultado oficial que não segue "mais Sim que Não":** 10 votações rejeitadas com mais Sim que Não (7 PL, 1 PLP, 1 PDL, 1 REP; 6 delas são requerimentos de urgência, 236 a 254 Sim, abaixo dos 257 da maioria absoluta) e 1 aprovada em empate (PL 1625/2026, 182 × 182, descrição "Resultado."). O site mostra uma nota explicando que existem regras de maioria especial e desempate, sem afirmar qual valeu em cada caso.

## 4. Ainda em aberto (não resolvido por Swagger nem por amostra)

- Mapa partido→bloco/federação por votação e por legislatura (Câmara: ver `/blocos` e `/partidos/{id}`; Senado: `Bloco` em `senador/lista/atual` + `composicao/lista/partidos`).
- Definição de fidelidade partidária (decisão de produto).
- Comportamento de votação simbólica e de `/votacao` no Senado fora do caso nominal.
- Rate limits reais (não observamos bloqueio com 100–200 ms de delay) e política de backfill.
- Itens 5–10 da lista de discovery anterior (escopo temporal, hospedagem, licença, metodologia pública, público-alvo).
