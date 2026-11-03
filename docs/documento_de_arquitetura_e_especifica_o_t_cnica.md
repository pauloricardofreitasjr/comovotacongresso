# Documento de Arquitetura e Especificação Técnica (PRD / Knowledge Base)

Este documento define os requisitos de produto, modelo de dados relacional, contratos de APIs governamentais, algoritmos de cálculo e fluxos de ingestão para a implementação do sistema **Monitor de Votações do Congresso Nacional** (Câmara dos Deputados e Senado Federal).

---

## 1. Visão Geral do Sistema

O objetivo da plataforma é consolidar e auditar as votações legislativas federais brasileiras, disponibilizando:
1. **Catálogo de Matérias Legislativas**: Listagem de proposições (PL, PEC, MPV, etc.) com histórico temporal de deliberações.
2. **Diferenciação Metodológica**: Separação inequívoca entre votações **nominais** e **simbólicas**.
3. **Detalhamento Nominal**: Registro do voto individual de cada parlamentar por votação.
4. **Métricas Agregadas por Parlamentar e Bancada**:
   - **Índice de Presença em Plenário** (calculado sobre votações nominais).
   - **Taxa de Alinhamento com o Governo** (% governista / fidelidade com o Palácio do Planalto).
   - **Taxa de Fidelidade Partidária** (% de convergência com a liderança do próprio partido/bloco).

---

## 2. Contratos das APIs Externas

### 2.1 Câmara dos Deputados
- **Base URL:** `https://dadosabertos.camara.leg.br/api/v2`
- **Formato:** JSON (`Accept: application/json`)
- **Autenticação:** Aberta / Sem token.
- **Rate Limit:** Tolerante, recomenda-se delay de 100ms a 200ms entre requisições sequenciais.
- **Endpoints Chave:**
  - `GET /votacoes?dataInicio={YYYY-MM-DD}&dataFim={YYYY-MM-DD}`: Lista votações realizadas no intervalo.
  - `GET /votacoes/{id}`: Detalhes da votação, aprovação e matéria vinculada.
  - `GET /votacoes/{id}/orientacoes`: Posições orientadas pelas lideranças partidárias e pela liderança do Governo (registro com `siglaPartidoBloco = 'Governo'` e `codTipoLideranca = 'B'`). `orientacaoVoto` pode vir vazio (sem orientação). Ver `docs/discovery/README.md`.
  - `GET /votacoes/{id}/votos`: Votos individuais dos deputados. Quando o retorno for um array vazio em deliberação concluída, a votação foi **simbólica**.
  - `GET /deputados?idLegislatura={id}&ordem=ASC&ordenarPor=nome`: Listagem de parlamentares ativos.

### 2.2 Senado Federal
- **Base URL:** `https://legis.senado.leg.br/dadosabertos`
- **Formato:** JSON (header `Accept: application/json` obrigatório)
- **Autenticação:** Aberta / Sem token.
- **Endpoints Chave:**
  - `GET /votacao?dataInicio=&dataFim=`: Votações com votos nominais individuais (`votos[]`) e matéria (`identificacao`, `codigoMateria`). `votacaoSecreta = 'S'` não traz voto individual.
  - `GET /plenario/votacao/orientacaoBancada/{YYYYMMDD}/{YYYYMMDD}`: Orientações de liderança (`orientacoesLideranca[]`; o Governo aparece como `partido = 'Governo'`).
  - `GET /senador/lista/atual` e `/senador/lista/legislatura/{ini}/{fim}`: Senadores.
  - Os endpoints `/materia/{codigo}/votacoes` e `/plenario/lista/votacao/...` estão **deprecated**.
  - Códigos de voto próprios: `Sim`, `Não`, `Abstenção`, `Presidente (art. 51 RISF)`, `P-NRV`, `AP`, `LS`, `NCom`.

---

## 3. Modelo de Banco de Dados Relacional (PostgreSQL)

O schema normalizado a seguir unifica os registros da Câmara e do Senado.

```sql
-- Extensões e Tipos Enumerados
CREATE TYPE casa_legislativa AS ENUM ('camara', 'senado');
CREATE TYPE tipo_votacao_enum AS ENUM ('nominal', 'simbolica', 'secreta');
CREATE TYPE tipo_voto_enum AS ENUM ('sim', 'nao', 'abstencao', 'obstrucao', 'artigo_17', 'outro');
CREATE TYPE orientacao_enum AS ENUM ('sim', 'nao', 'liberado', 'obstrucao');

-- 1. Parlamentares
CREATE TABLE parlamentares (
    id SERIAL PRIMARY KEY,
    id_externo INT NOT NULL,
    casa casa_legislativa NOT NULL,
    nome VARCHAR(180) NOT NULL,
    nome_civil VARCHAR(180),
    partido_atual VARCHAR(30),
    uf VARCHAR(2),
    url_foto TEXT,
    ativo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(id_externo, casa)
);

-- 2. Proposições Legislativas
CREATE TABLE proposicoes (
    id SERIAL PRIMARY KEY,
    id_externo INT NOT NULL,
    casa casa_legislativa NOT NULL,
    sigla_tipo VARCHAR(15) NOT NULL,
    numero INT NOT NULL,
    ano INT NOT NULL,
    ementa TEXT,
    ementa_detalhada TEXT,
    url_documento TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(id_externo, casa)
);

-- 3. Votações
CREATE TABLE votacoes (
    id VARCHAR(100) PRIMARY KEY,
    proposicao_id INT REFERENCES proposicoes(id) ON DELETE SET NULL,
    casa casa_legislativa NOT NULL,
    data_hora TIMESTAMP WITH TIME ZONE NOT NULL,
    descricao TEXT,
    tipo tipo_votacao_enum NOT NULL,
    aprovada BOOLEAN,
    orientacao_governo orientacao_enum,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 4. Orientações de Bancada
CREATE TABLE votacoes_orientacoes (
    id SERIAL PRIMARY KEY,
    votacao_id VARCHAR(100) REFERENCES votacoes(id) ON DELETE CASCADE,
    bancada VARCHAR(50) NOT NULL,
    orientacao orientacao_enum NOT NULL,
    UNIQUE(votacao_id, bancada)
);

-- 5. Votos Individuais (Votações Nominais)
CREATE TABLE votos (
    id BIGSERIAL PRIMARY KEY,
    votacao_id VARCHAR(100) REFERENCES votacoes(id) ON DELETE CASCADE,
    parlamentar_id INT REFERENCES parlamentares(id) ON DELETE RESTRICT,
    voto tipo_voto_enum NOT NULL,
    partido_na_votacao VARCHAR(30),
    alinhou_governo BOOLEAN,
    alinhou_partido BOOLEAN,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(votacao_id, parlamentar_id)
);

-- Índices de Performance
CREATE INDEX idx_votos_parlamentar ON votos(parlamentar_id);
CREATE INDEX idx_votos_votacao ON votos(votacao_id);
CREATE INDEX idx_votacoes_proposicao ON votacoes(proposicao_id);
CREATE INDEX idx_votacoes_data ON votacoes(data_hora DESC);
CREATE INDEX idx_proposicoes_busca ON proposicoes(sigla_tipo, numero, ano);
```

---

## 4. Algoritmos de Normalização e Cálculo

### 4.1 Identificação do Tipo de Votação
1. Realizar requisição ao endpoint de votos individuais (`/votacoes/{id}/votos`).
2. Se `len(votos) > 0`: registrar `tipo = 'nominal'`.
3. Se `len(votos) == 0` e o status da votação for concluído: registrar `tipo = 'simbolica'`.

### 4.2 Mapeamento de Voto
Os termos brutos retornados pela API devem ser mapeados para os tipos canônicos:

| Retorno da API | `tipo_voto_enum` | Conta como Presença em Plenário? |
|---|---|---|
| `Sim` | `sim` | Sim |
| `Não` | `nao` | Sim |
| `Abstenção` | `abstencao` | Sim |
| `Obstrução` | `obstrucao` | Sim |
| `Artigo 17` / Voto Presidência | `artigo_17` | Sim |
| *(Ausente na listagem de votos — Câmara)* | *(Sem registro)* | Não |
| `P-NRV` (Senado) | `outro` | Sim (presente, sem voto registrado) |
| `AP` / `LS` / `NCom` (Senado) | *(Sem registro de voto)* | Não (confirmar com `/plenario/lista/tiposComparecimento`) |
| `Presidente (art. 51 RISF)` (Senado) | `artigo_17` | Sim |

### 4.3 Regra de Alinhamento Governamental
Para cada voto individual:
1. Localizar na tabela `votacoes_orientacoes` o registro onde `lower(bancada) = 'governo'` para a votação em questão.
2. Se a orientação do governo for `liberado`, vazia ou inexistente: `alinhou_governo = NULL`.
3. Se a orientação for `sim`:
   - Voto `sim` $\rightarrow$ `alinhou_governo = TRUE`
   - Voto `nao` $\rightarrow$ `alinhou_governo = FALSE`
   - Voto `abstencao` ou `obstrucao` $\rightarrow$ `alinhou_governo = FALSE`
4. Se a orientação for `nao`:
   - Voto `nao` $\rightarrow$ `alinhou_governo = TRUE`
   - Voto `sim` $\rightarrow$ `alinhou_governo = FALSE`
   - Voto `abstencao` ou `obstrucao` $\rightarrow$ `alinhou_governo = FALSE`

### 4.4 Fórmulas Agregadas (Consultas SQL)

- **Taxa de Fidelidade ao Governo por Parlamentar:**
$$Fidelidade = \frac{\sum (\text{votos com } alinhou\_governo = TRUE)}{\sum (\text{votos com } alinhou\_governo \text{ IS NOT NULL})} \times 100$$

```sql
SELECT 
    p.id,
    p.nome,
    p.partido_atual,
    p.uf,
    COUNT(v.id) FILTER (WHERE v.alinhou_governo IS NOT NULL) AS votacoes_base_governo,
    ROUND(
        100.0 * COUNT(v.id) FILTER (WHERE v.alinhou_governo = TRUE) / 
        NULLIF(COUNT(v.id) FILTER (WHERE v.alinhou_governo IS NOT NULL), 0), 
        2
    ) AS pct_governista
FROM parlamentares p
LEFT JOIN votos v ON p.id = v.parlamentar_id
GROUP BY p.id;
```

- **Taxa de Presença em Votações Nominais:**
```sql
SELECT 
    p.id,
    p.nome,
    COUNT(v.id) AS votacoes_presente,
    (SELECT COUNT(*) FROM votacoes WHERE tipo = 'nominal' AND casa = p.casa) AS total_votacoes_nominais,
    ROUND(
        100.0 * COUNT(v.id) / 
        NULLIF((SELECT COUNT(*) FROM votacoes WHERE tipo = 'nominal' AND casa = p.casa), 0),
        2
    ) AS pct_presenca
FROM parlamentares p
LEFT JOIN votos v ON p.id = v.parlamentar_id
GROUP BY p.id;
```

---

## 5. Pipeline de Ingestão de Dados (ETL)

O processo de ingestão deve ser executado de forma assíncrona (ex.: via Celery, BullMQ ou Cron Job diário):

```text
[Pipeline Diário / Horário]
         │
         ▼
[1] Fetch /votacoes?dataInicio={D-1}&dataFim={D}
         │
         ▼
[2] Para cada Votação:
         ├── (a) Upsert Proposição em 'proposicoes'
         ├── (b) Fetch e Upsert Orientações em 'votacoes_orientacoes'
         ├── (c) Fetch Votos em '/votacoes/{id}/votos'
         │       ├── Se votos vazios: Define tipo = 'simbolica'
         │       └── Se votos presentes:
         │             - Define tipo = 'nominal'
         │             - Computa alinhou_governo para cada voto
         │             - Batch upsert na tabela 'votos'
         └── (d) Upsert registro em 'votacoes'
         │
         ▼
[3] Refresh em Materialized Views de Métricas Agregadas
```

---

## 6. Especificação de Endpoints da API (Backend)

| Método | Rota | Descrição | Parâmetros Chave |
|---|---|---|---|
| `GET` | `/api/v1/proposicoes` | Listagem paginada de projetos com votações | `page`, `limit`, `tipo`, `ano`, `busca` |
| `GET` | `/api/v1/proposicoes/{id}` | Detalhes da matéria e lista de votações | `id` (interno) |
| `GET` | `/api/v1/votacoes/{id}` | Metadados, orientações de bancada e lista de votos | `id` (chave da votação) |
| `GET` | `/api/v1/parlamentares` | Lista parlamentares com filtros e ranking | `casa`, `partido`, `uf`, `ordenarPor` (`presenca`, `governismo`) |
| `GET` | `/api/v1/parlamentares/{id}` | Perfil detalhado com KPIs calculados e histórico de votos | `id` (interno), `page_votos` |

---

## 7. Diretrizes Técnicas para o Claude / Agente Desenvolvedor

1. **Idempotência**: Todos os scripts de carga e endpoints de persistência devem usar `UPSERT` (`ON CONFLICT DO UPDATE`) para permitir re-execução sem duplicação de dados.
2. **Resiliência de Rede**: Implementar estratégia de *retry* com *exponential backoff* nas chamadas HTTP para as APIs da Câmara e do Senado.
3. **Desacoplamento de Visualização**: Votações nominais e simbólicas devem ter componentes visuais distintos na interface (ex.: badge informativo e ocultação da listagem individual quando simbólica).
4. **Performance**: Consultas de agregação de fidelidade governista e presença não devem ser computadas *on the fly* em cada requisição de listagem; utilizar tabelas agregadas ou *Materialized Views* atualizadas após cada ciclo do ETL.