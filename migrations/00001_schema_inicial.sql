-- +goose Up
-- Schema inicial (ver docs/documento_de_arquitetura_e_especifica_o_t_cnica.md §3 e docs/discovery/README.md).

CREATE TYPE casa_legislativa AS ENUM ('camara', 'senado');
CREATE TYPE tipo_votacao_enum AS ENUM ('nominal', 'simbolica', 'secreta');
CREATE TYPE tipo_voto_enum AS ENUM ('sim', 'nao', 'abstencao', 'obstrucao', 'artigo_17', 'outro');
CREATE TYPE orientacao_enum AS ENUM ('sim', 'nao', 'liberado', 'obstrucao');
CREATE TYPE vinculo_confianca_enum AS ENUM ('direto', 'provavel');

CREATE TABLE parlamentares (
    id SERIAL PRIMARY KEY,
    id_externo INT NOT NULL,
    casa casa_legislativa NOT NULL,
    nome VARCHAR(180) NOT NULL,
    nome_civil VARCHAR(180),
    partido_atual VARCHAR(30),
    uf VARCHAR(2),
    url_foto TEXT,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id_externo, casa)
);

-- Intervalos em que o parlamentar esteve em exercício (base da presença).
-- fim NULL = exercício em aberto.
CREATE TABLE parlamentares_exercicio (
    id SERIAL PRIMARY KEY,
    parlamentar_id INT NOT NULL REFERENCES parlamentares(id) ON DELETE CASCADE,
    inicio TIMESTAMPTZ NOT NULL,
    fim TIMESTAMPTZ,
    condicao VARCHAR(20),
    partido VARCHAR(30),
    UNIQUE (parlamentar_id, inicio),
    CHECK (fim IS NULL OR fim >= inicio)
);

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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id_externo, casa)
);

CREATE TABLE votacoes (
    id VARCHAR(100) PRIMARY KEY,
    proposicao_id INT REFERENCES proposicoes(id) ON DELETE SET NULL,
    vinculo_confianca vinculo_confianca_enum,
    casa casa_legislativa NOT NULL,
    data_hora TIMESTAMPTZ NOT NULL,
    descricao TEXT,
    tipo tipo_votacao_enum NOT NULL,
    aprovada BOOLEAN,
    -- denormalizado a partir de votacoes_orientacoes (NULL = sem orientação do Governo)
    orientacao_governo orientacao_enum,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((proposicao_id IS NULL) = (vinculo_confianca IS NULL))
);

CREATE TABLE votacoes_orientacoes (
    id SERIAL PRIMARY KEY,
    votacao_id VARCHAR(100) NOT NULL REFERENCES votacoes(id) ON DELETE CASCADE,
    bancada VARCHAR(50) NOT NULL,
    tipo_lideranca CHAR(1),
    -- NULL = bancada sem orientação (campo vazio na API)
    orientacao orientacao_enum,
    UNIQUE (votacao_id, bancada)
);

CREATE TABLE votos (
    id BIGSERIAL PRIMARY KEY,
    votacao_id VARCHAR(100) NOT NULL REFERENCES votacoes(id) ON DELETE CASCADE,
    parlamentar_id INT NOT NULL REFERENCES parlamentares(id) ON DELETE RESTRICT,
    voto tipo_voto_enum NOT NULL,
    partido_na_votacao VARCHAR(30),
    alinhou_governo BOOLEAN,
    alinhou_partido BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (votacao_id, parlamentar_id)
);

-- Execuções do ETL (observabilidade e retomada do backfill).
CREATE TABLE etl_runs (
    id BIGSERIAL PRIMARY KEY,
    job VARCHAR(50) NOT NULL,
    params JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    erro TEXT
);

CREATE TABLE etl_errors (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES etl_runs(id) ON DELETE CASCADE,
    ref VARCHAR(150) NOT NULL,
    mensagem TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_votos_parlamentar ON votos(parlamentar_id);
CREATE INDEX idx_votos_votacao ON votos(votacao_id);
CREATE INDEX idx_votacoes_proposicao ON votacoes(proposicao_id);
CREATE INDEX idx_votacoes_data ON votacoes(data_hora DESC);
CREATE INDEX idx_votacoes_tipo_casa_data ON votacoes(casa, tipo, data_hora);
CREATE INDEX idx_proposicoes_busca ON proposicoes(sigla_tipo, numero, ano);
CREATE INDEX idx_exercicio_parlamentar ON parlamentares_exercicio(parlamentar_id, inicio);
CREATE INDEX idx_etl_errors_run ON etl_errors(run_id);

-- +goose Down
DROP TABLE etl_errors, etl_runs, votos, votacoes_orientacoes, votacoes, proposicoes,
    parlamentares_exercicio, parlamentares;
DROP TYPE vinculo_confianca_enum, orientacao_enum, tipo_voto_enum, tipo_votacao_enum, casa_legislativa;
