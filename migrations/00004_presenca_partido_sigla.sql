-- +goose Up
-- A lista de presença da Câmara grafa o partido de outro jeito que o histórico de exercício
-- ("Republican", "Podemos", "Solidaried" contra REPUBLICANOS, PODE, SOLIDARIEDADE). Agrupar pela
-- grafia da presença dividia cada partido em dois grupos e zerava a presença de vários na página
-- de partidos. Agora o partido é sempre o do exercício na data da sessão; sem exercício, o do
-- cadastro; só em último caso a grafia da lista de presença. Não muda a presença por parlamentar.
CREATE OR REPLACE VIEW v_elegibilidade_sessao AS
SELECT DISTINCT ON (parlamentar_id, sessao_id) parlamentar_id, sessao_id, partido
FROM (
    SELECT pr.parlamentar_id, pr.sessao_id,
           COALESCE(ex.partido, p.partido_atual, pr.partido_na_sessao) AS partido, 1 AS prio
    FROM sessoes_presencas pr
    JOIN parlamentares p ON p.id = pr.parlamentar_id
    JOIN sessoes s ON s.id = pr.sessao_id
    LEFT JOIN parlamentares_exercicio ex
           ON ex.parlamentar_id = pr.parlamentar_id AND s.inicio >= ex.inicio AND (ex.fim IS NULL OR s.inicio < ex.fim)
    UNION ALL
    SELECT e.parlamentar_id, s.id, e.partido, 2
    FROM parlamentares_exercicio e
    JOIN parlamentares p ON p.id = e.parlamentar_id
    JOIN sessoes s ON s.casa = p.casa
         AND s.inicio >= e.inicio AND (e.fim IS NULL OR s.inicio < e.fim)
) x
ORDER BY parlamentar_id, sessao_id, prio;

REFRESH MATERIALIZED VIEW mv_partido_presenca_sessao;
REFRESH MATERIALIZED VIEW mv_parlamentar_presenca_sessao;

-- +goose Down
CREATE OR REPLACE VIEW v_elegibilidade_sessao AS
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

REFRESH MATERIALIZED VIEW mv_partido_presenca_sessao;
