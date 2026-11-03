-- +goose Up
-- Presença OFICIAL em sessão deliberativa do Plenário (registro de presença da Câmara).
-- Diferente de "participação em votações nominais" (00002): ver docs/discovery/README.md §3.5.

CREATE TABLE sessoes (
    id SERIAL PRIMARY KEY,
    id_externo INT NOT NULL,
    casa casa_legislativa NOT NULL,
    inicio TIMESTAMPTZ NOT NULL,
    fim TIMESTAMPTZ,
    tipo VARCHAR(60) NOT NULL,
    descricao TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id_externo, casa)
);

CREATE TABLE sessoes_presencas (
    sessao_id INT NOT NULL REFERENCES sessoes(id) ON DELETE CASCADE,
    parlamentar_id INT NOT NULL REFERENCES parlamentares(id) ON DELETE RESTRICT,
    partido_na_sessao VARCHAR(30),
    PRIMARY KEY (sessao_id, parlamentar_id)
);

CREATE INDEX idx_sessoes_inicio ON sessoes(casa, inicio);
CREATE INDEX idx_sessoes_presencas_parlamentar ON sessoes_presencas(parlamentar_id);

-- Sessões que "contam" para cada parlamentar: as iniciadas durante um período de exercício
-- dele, mais aquelas em que registrou presença. Partido: o da lista de presença, senão o do período.
CREATE VIEW v_elegibilidade_sessao AS
SELECT DISTINCT ON (parlamentar_id, sessao_id) parlamentar_id, sessao_id, partido
FROM (
    SELECT pr.parlamentar_id, pr.sessao_id, pr.partido_na_sessao AS partido, 1 AS prio
    FROM sessoes_presencas pr
    UNION ALL
    SELECT e.parlamentar_id, s.id, e.partido, 2
    FROM parlamentares_exercicio e
    JOIN parlamentares p ON p.id = e.parlamentar_id
    JOIN sessoes s ON s.casa = p.casa
         AND s.inicio >= e.inicio AND (e.fim IS NULL OR s.inicio < e.fim)
) x
ORDER BY parlamentar_id, sessao_id, prio;

CREATE MATERIALIZED VIEW mv_parlamentar_presenca_sessao AS
SELECT
    p.id AS parlamentar_id,
    COUNT(el.sessao_id) AS sessoes_elegiveis,
    COUNT(pr.sessao_id) AS sessoes_presente,
    ROUND(100.0 * COUNT(pr.sessao_id) / NULLIF(COUNT(el.sessao_id), 0), 2) AS pct_presenca_sessao,
    NOW() AS atualizado_em
FROM parlamentares p
LEFT JOIN v_elegibilidade_sessao el ON el.parlamentar_id = p.id
LEFT JOIN sessoes_presencas pr ON pr.sessao_id = el.sessao_id AND pr.parlamentar_id = el.parlamentar_id
GROUP BY p.id
WITH DATA;

CREATE UNIQUE INDEX uq_mv_parlamentar_presenca_sessao ON mv_parlamentar_presenca_sessao (parlamentar_id);
CREATE INDEX idx_mv_presenca_sessao_pct ON mv_parlamentar_presenca_sessao (pct_presenca_sessao DESC);

CREATE MATERIALIZED VIEW mv_partido_presenca_sessao AS
SELECT
    p.casa,
    COALESCE(el.partido, '(sem partido)') AS partido,
    COUNT(DISTINCT el.parlamentar_id) AS parlamentares,
    COUNT(el.sessao_id) AS sessoes_elegiveis,
    COUNT(pr.sessao_id) AS sessoes_presente,
    ROUND(100.0 * COUNT(pr.sessao_id) / NULLIF(COUNT(el.sessao_id), 0), 2) AS pct_presenca_sessao,
    NOW() AS atualizado_em
FROM v_elegibilidade_sessao el
JOIN parlamentares p ON p.id = el.parlamentar_id
LEFT JOIN sessoes_presencas pr ON pr.sessao_id = el.sessao_id AND pr.parlamentar_id = el.parlamentar_id
GROUP BY p.casa, COALESCE(el.partido, '(sem partido)')
WITH DATA;

CREATE UNIQUE INDEX uq_mv_partido_presenca_sessao ON mv_partido_presenca_sessao (casa, partido);

-- +goose Down
DROP MATERIALIZED VIEW mv_partido_presenca_sessao;
DROP MATERIALIZED VIEW mv_parlamentar_presenca_sessao;
DROP VIEW v_elegibilidade_sessao;
DROP TABLE sessoes_presencas, sessoes;
