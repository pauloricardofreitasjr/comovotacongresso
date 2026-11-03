-- +goose Up
-- Métricas agregadas (§4.4 do doc de arquitetura, ajustadas pelo discovery).
-- Atualizadas após cada ciclo do ETL (REFRESH ... CONCURRENTLY), nunca calculadas na requisição.

-- Quais votações nominais "contam" para cada parlamentar: as ocorridas durante um período
-- de exercício dele, mais aquelas em que ele votou (dado inconsistente nunca derruba a
-- presença abaixo do real). O partido vem do voto quando existe, senão do período de exercício.
CREATE VIEW v_elegibilidade AS
SELECT DISTINCT ON (parlamentar_id, votacao_id) parlamentar_id, votacao_id, partido
FROM (
    SELECT v.parlamentar_id, v.votacao_id, v.partido_na_votacao AS partido, 1 AS prio
    FROM votos v
    JOIN votacoes vt ON vt.id = v.votacao_id AND vt.tipo = 'nominal'
    UNION ALL
    SELECT e.parlamentar_id, vt.id, e.partido, 2
    FROM parlamentares_exercicio e
    JOIN parlamentares p ON p.id = e.parlamentar_id
    JOIN votacoes vt ON vt.casa = p.casa AND vt.tipo = 'nominal'
         AND vt.data_hora >= e.inicio AND (e.fim IS NULL OR vt.data_hora < e.fim)
) x
ORDER BY parlamentar_id, votacao_id, prio;

CREATE MATERIALIZED VIEW mv_parlamentar_metricas AS
SELECT
    p.id AS parlamentar_id,
    p.id_externo,
    p.casa,
    p.nome,
    p.partido_atual,
    p.uf,
    p.url_foto,
    EXISTS (SELECT 1 FROM parlamentares_exercicio e WHERE e.parlamentar_id = p.id AND e.fim IS NULL) AS em_exercicio,
    COUNT(el.votacao_id) AS votacoes_elegiveis,
    COUNT(v.id) AS votacoes_presente,
    ROUND(100.0 * COUNT(v.id) / NULLIF(COUNT(el.votacao_id), 0), 2) AS pct_presenca,
    COUNT(v.id) FILTER (WHERE v.alinhou_governo IS NOT NULL) AS votacoes_base_governo,
    COUNT(v.id) FILTER (WHERE v.alinhou_governo) AS votacoes_alinhadas_governo,
    ROUND(
        100.0 * COUNT(v.id) FILTER (WHERE v.alinhou_governo)
        / NULLIF(COUNT(v.id) FILTER (WHERE v.alinhou_governo IS NOT NULL), 0), 2
    ) AS pct_governista,
    NOW() AS atualizado_em
FROM parlamentares p
LEFT JOIN v_elegibilidade el ON el.parlamentar_id = p.id
LEFT JOIN votos v ON v.votacao_id = el.votacao_id AND v.parlamentar_id = el.parlamentar_id
GROUP BY p.id
WITH DATA;

CREATE UNIQUE INDEX uq_mv_parlamentar_metricas ON mv_parlamentar_metricas (parlamentar_id);
CREATE INDEX idx_mv_parlamentar_presenca ON mv_parlamentar_metricas (casa, pct_presenca DESC);
CREATE INDEX idx_mv_parlamentar_governista ON mv_parlamentar_metricas (casa, pct_governista DESC);

-- Por partido: o partido é o do momento da votação (não o atual), então trocas de partido
-- contam para o partido em que o parlamentar estava.
CREATE MATERIALIZED VIEW mv_partido_metricas AS
SELECT
    p.casa,
    COALESCE(el.partido, '(sem partido)') AS partido,
    COUNT(DISTINCT el.parlamentar_id) AS parlamentares,
    COUNT(el.votacao_id) AS votacoes_elegiveis,
    COUNT(v.id) AS votacoes_presente,
    ROUND(100.0 * COUNT(v.id) / NULLIF(COUNT(el.votacao_id), 0), 2) AS pct_presenca,
    COUNT(v.id) FILTER (WHERE v.alinhou_governo IS NOT NULL) AS votacoes_base_governo,
    COUNT(v.id) FILTER (WHERE v.alinhou_governo) AS votacoes_alinhadas_governo,
    ROUND(
        100.0 * COUNT(v.id) FILTER (WHERE v.alinhou_governo)
        / NULLIF(COUNT(v.id) FILTER (WHERE v.alinhou_governo IS NOT NULL), 0), 2
    ) AS pct_governista,
    NOW() AS atualizado_em
FROM v_elegibilidade el
JOIN parlamentares p ON p.id = el.parlamentar_id
LEFT JOIN votos v ON v.votacao_id = el.votacao_id AND v.parlamentar_id = el.parlamentar_id
GROUP BY p.casa, COALESCE(el.partido, '(sem partido)')
WITH DATA;

CREATE UNIQUE INDEX uq_mv_partido_metricas ON mv_partido_metricas (casa, partido);

-- +goose Down
DROP MATERIALIZED VIEW mv_partido_metricas;
DROP MATERIALIZED VIEW mv_parlamentar_metricas;
DROP VIEW v_elegibilidade;
