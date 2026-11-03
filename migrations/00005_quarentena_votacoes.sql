-- +goose Up
-- Quarentena de votações cuja lista de votos não é confiável ou repete a de outra (achado de nov/2026:
-- a API da Câmara devolve a mesma lista de votos para votações diferentes; ver docs/discovery/README.md §3.5).
--   lista_duplicada    a lista de votos é idêntica à de outra votação e a descrição oficial NÃO bate com ela:
--                      os votos individuais provavelmente são de outra votação. Fica oculta e fora das métricas.
--   registro_duplicado a lista é idêntica à de outra votação e a descrição bate: é o mesmo ato registrado duas vezes.
--                      Os votos são válidos, mas contar os dois dobraria o histórico e a participação.
-- A marcação é recalculada pelo ETL (Store.AtualizarQuarentena), nunca à mão.
ALTER TABLE votacoes
    ADD COLUMN quarentena VARCHAR(30) CHECK (quarentena IN ('lista_duplicada', 'registro_duplicado')),
    ADD COLUMN quarentena_ref VARCHAR(100) REFERENCES votacoes(id) ON DELETE SET NULL;

-- Votações em quarentena não contam para a elegibilidade (e, por consequência, nem para participação e alinhamento).
CREATE OR REPLACE VIEW v_elegibilidade AS
SELECT DISTINCT ON (parlamentar_id, votacao_id) parlamentar_id, votacao_id, partido
FROM (
    SELECT v.parlamentar_id, v.votacao_id, v.partido_na_votacao AS partido, 1 AS prio
    FROM votos v
    JOIN votacoes vt ON vt.id = v.votacao_id AND vt.tipo = 'nominal' AND vt.quarentena IS NULL
    UNION ALL
    SELECT e.parlamentar_id, vt.id, e.partido, 2
    FROM parlamentares_exercicio e
    JOIN parlamentares p ON p.id = e.parlamentar_id
    JOIN votacoes vt ON vt.casa = p.casa AND vt.tipo = 'nominal' AND vt.quarentena IS NULL
         AND vt.data_hora >= e.inicio AND (e.fim IS NULL OR vt.data_hora < e.fim)
) x
ORDER BY parlamentar_id, votacao_id, prio;

REFRESH MATERIALIZED VIEW mv_parlamentar_metricas;
REFRESH MATERIALIZED VIEW mv_partido_metricas;

-- +goose Down
CREATE OR REPLACE VIEW v_elegibilidade AS
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

REFRESH MATERIALIZED VIEW mv_parlamentar_metricas;
REFRESH MATERIALIZED VIEW mv_partido_metricas;

ALTER TABLE votacoes DROP COLUMN quarentena_ref, DROP COLUMN quarentena;
