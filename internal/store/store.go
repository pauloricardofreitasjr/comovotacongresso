// Package store persiste os dados no PostgreSQL. Todas as escritas são idempotentes.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"comovotacongresso/internal/domain"
)

const casaCamara = "camara"

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

type ParlamentarIn struct {
	IDExterno int
	Nome      string
	Partido   string
	UF        string
	URLFoto   string
}

type OrientacaoIn struct {
	Bancada       string
	TipoLideranca string
	Orientacao    *domain.Orientacao
}

type VotoIn struct {
	Parlamentar    ParlamentarIn
	Voto           domain.TipoVoto
	Partido        string
	AlinhouGoverno *bool
}

type VotacaoIn struct {
	ID                string
	DataHora          time.Time
	Descricao         string
	Tipo              domain.TipoVotacao
	Aprovada          *bool
	Proposicao        *domain.Prop
	Vinculo           domain.Confianca
	OrientacaoGoverno *domain.Orientacao
	Orientacoes       []OrientacaoIn
	Votos             []VotoIn
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// trunc corta em n runas (colunas VARCHAR(n)).
func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// UpsertParlamentares grava os parlamentares e devolve id_externo → id.
// atualizar=true sobrescreve os dados cadastrais (usado no sync de /deputados, que traz o
// partido atual); false só insere os novos (usado a partir dos votos, cujo partido é o da data).
func (s *Store) UpsertParlamentares(ctx context.Context, ps []ParlamentarIn, atualizar bool) (map[int]int, error) {
	var tx pgx.Tx
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	ids, err := upsertParlamentares(ctx, tx, ps, atualizar)
	if err != nil {
		return nil, err
	}
	return ids, tx.Commit(ctx)
}

func upsertParlamentares(ctx context.Context, q pgx.Tx, ps []ParlamentarIn, atualizar bool) (map[int]int, error) {
	ids := make(map[int]int, len(ps))
	ps = dedupe(ps)
	if len(ps) == 0 {
		return ids, nil
	}
	ext := make([]int32, len(ps))
	nomes, partidos, ufs, fotos := make([]string, len(ps)), make([]*string, len(ps)), make([]*string, len(ps)), make([]*string, len(ps))
	for i, p := range ps {
		ext[i] = int32(p.IDExterno)
		nomes[i] = trunc(p.Nome, 180)
		partidos[i], ufs[i], fotos[i] = nullIfEmpty(trunc(p.Partido, 30)), nullIfEmpty(trunc(p.UF, 2)), nullIfEmpty(p.URLFoto)
	}
	conflito := "DO NOTHING"
	if atualizar {
		conflito = `DO UPDATE SET nome = EXCLUDED.nome, partido_atual = EXCLUDED.partido_atual,
			uf = EXCLUDED.uf, url_foto = EXCLUDED.url_foto, updated_at = NOW()`
	}
	_, err := q.Exec(ctx, `
		INSERT INTO parlamentares (id_externo, casa, nome, partido_atual, uf, url_foto)
		SELECT t.ext, $1::casa_legislativa, t.nome, t.partido, t.uf, t.foto
		FROM unnest($2::int[], $3::text[], $4::text[], $5::text[], $6::text[]) AS t(ext, nome, partido, uf, foto)
		ON CONFLICT (id_externo, casa) `+conflito, casaCamara, ext, nomes, partidos, ufs, fotos)
	if err != nil {
		return nil, fmt.Errorf("upsert parlamentares: %w", err)
	}
	rows, err := q.Query(ctx, `SELECT id_externo, id FROM parlamentares WHERE casa = $1 AND id_externo = ANY($2)`, casaCamara, ext)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e, id int
		if err := rows.Scan(&e, &id); err != nil {
			return nil, err
		}
		ids[e] = id
	}
	return ids, rows.Err()
}

// dedupe mantém uma linha por IDExterno (a última): o mesmo parlamentar pode vir repetido
// (ex.: uma linha por partido em /deputados) e o ON CONFLICT não aceita repetir a chave.
func dedupe(ps []ParlamentarIn) []ParlamentarIn {
	idx := make(map[int]int, len(ps))
	out := make([]ParlamentarIn, 0, len(ps))
	for _, p := range ps {
		if i, ok := idx[p.IDExterno]; ok {
			out[i] = p
			continue
		}
		idx[p.IDExterno] = len(out)
		out = append(out, p)
	}
	return out
}

// SubstituirExercicio troca TODOS os intervalos de exercício do parlamentar. O ETL não deve
// usá-lo: o mesmo deputado reeleito tem exercício em mais de uma legislatura, e a
// sincronização de uma não pode apagar a outra (ver SubstituirExercicioNoPeriodo).
func (s *Store) SubstituirExercicio(ctx context.Context, parlamentarID int, ivs []domain.Intervalo) error {
	return s.SubstituirExercicioNoPeriodo(ctx, parlamentarID, ivs, time.Time{}, time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC))
}

// SubstituirExercicioNoPeriodo troca os intervalos de exercício que começam em [de, ate)
// (normalmente o período de uma legislatura), preservando os das demais. ivs vem em ordem
// cronológica. partido_atual passa a ser o partido do intervalo mais recente do parlamentar,
// em qualquer legislatura, de modo que a ordem de sincronização não importa.
func (s *Store) SubstituirExercicioNoPeriodo(ctx context.Context, parlamentarID int, ivs []domain.Intervalo, de, ate time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM parlamentares_exercicio WHERE parlamentar_id = $1 AND inicio >= $2 AND inicio < $3`,
		parlamentarID, de, ate); err != nil {
		return err
	}
	for _, iv := range ivs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO parlamentares_exercicio (parlamentar_id, inicio, fim, condicao, partido)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (parlamentar_id, inicio) DO UPDATE SET fim = EXCLUDED.fim, condicao = EXCLUDED.condicao, partido = EXCLUDED.partido`,
			parlamentarID, iv.Inicio, iv.Fim, nullIfEmpty(iv.Condicao), nullIfEmpty(trunc(iv.Partido, 30))); err != nil {
			return fmt.Errorf("inserir exercício: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE parlamentares SET updated_at = NOW(), partido_atual = COALESCE(
			(SELECT e.partido FROM parlamentares_exercicio e
			  WHERE e.parlamentar_id = parlamentares.id AND e.partido IS NOT NULL
			  ORDER BY e.inicio DESC LIMIT 1), partido_atual)
		WHERE id = $1`, parlamentarID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) VotacaoExiste(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM votacoes WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

// SalvarVotacao grava proposição, votação, orientações e votos numa única transação.
func (s *Store) SalvarVotacao(ctx context.Context, v VotacaoIn) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var propID *int
	var vinculo *string
	if v.Proposicao != nil {
		var id int
		err := tx.QueryRow(ctx, `
			INSERT INTO proposicoes (id_externo, casa, sigla_tipo, numero, ano, ementa)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id_externo, casa) DO UPDATE SET
				sigla_tipo = EXCLUDED.sigla_tipo, numero = EXCLUDED.numero, ano = EXCLUDED.ano,
				ementa = COALESCE(NULLIF(EXCLUDED.ementa, ''), proposicoes.ementa)
			RETURNING id`,
			v.Proposicao.ID, casaCamara, v.Proposicao.SiglaTipo, v.Proposicao.Numero, v.Proposicao.Ano, v.Proposicao.Ementa).Scan(&id)
		if err != nil {
			return fmt.Errorf("upsert proposição: %w", err)
		}
		propID = &id
		c := string(v.Vinculo)
		vinculo = &c
	}

	var orientGov *string
	if v.OrientacaoGoverno != nil {
		s := string(*v.OrientacaoGoverno)
		orientGov = &s
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO votacoes (id, proposicao_id, vinculo_confianca, casa, data_hora, descricao, tipo, aprovada, orientacao_governo)
		VALUES ($1, $2, $3::vinculo_confianca_enum, $4, $5, $6, $7, $8, $9::orientacao_enum)
		ON CONFLICT (id) DO UPDATE SET
			proposicao_id = EXCLUDED.proposicao_id, vinculo_confianca = EXCLUDED.vinculo_confianca,
			data_hora = EXCLUDED.data_hora, descricao = EXCLUDED.descricao, tipo = EXCLUDED.tipo,
			aprovada = EXCLUDED.aprovada, orientacao_governo = EXCLUDED.orientacao_governo`,
		v.ID, propID, vinculo, casaCamara, v.DataHora, v.Descricao, string(v.Tipo), v.Aprovada, orientGov); err != nil {
		return fmt.Errorf("upsert votação: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM votacoes_orientacoes WHERE votacao_id = $1`, v.ID); err != nil {
		return err
	}
	for _, o := range v.Orientacoes {
		var orient *string
		if o.Orientacao != nil {
			s := string(*o.Orientacao)
			orient = &s
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO votacoes_orientacoes (votacao_id, bancada, tipo_lideranca, orientacao)
			VALUES ($1, $2, $3, $4::orientacao_enum)
			ON CONFLICT (votacao_id, bancada) DO UPDATE SET tipo_lideranca = EXCLUDED.tipo_lideranca, orientacao = EXCLUDED.orientacao`,
			v.ID, trunc(o.Bancada, 50), nullIfEmpty(o.TipoLideranca), orient); err != nil {
			return fmt.Errorf("orientação %q: %w", o.Bancada, err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM votos WHERE votacao_id = $1`, v.ID); err != nil {
		return err
	}
	if len(v.Votos) > 0 {
		ps := make([]ParlamentarIn, len(v.Votos))
		for i, vt := range v.Votos {
			ps[i] = vt.Parlamentar
		}
		ids, err := upsertParlamentares(ctx, tx, ps, false)
		if err != nil {
			return err
		}
		pids := make([]int32, len(v.Votos))
		votos, partidos := make([]string, len(v.Votos)), make([]*string, len(v.Votos))
		alinhou := make([]*bool, len(v.Votos))
		for i, vt := range v.Votos {
			id, ok := ids[vt.Parlamentar.IDExterno]
			if !ok {
				return fmt.Errorf("parlamentar %d não encontrado após upsert", vt.Parlamentar.IDExterno)
			}
			pids[i], votos[i] = int32(id), string(vt.Voto)
			partidos[i] = nullIfEmpty(trunc(vt.Partido, 30))
			alinhou[i] = vt.AlinhouGoverno
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO votos (votacao_id, parlamentar_id, voto, partido_na_votacao, alinhou_governo)
			SELECT $1, t.pid, t.voto::tipo_voto_enum, t.partido, t.alinhou
			FROM unnest($2::int[], $3::text[], $4::text[], $5::bool[]) AS t(pid, voto, partido, alinhou)
			ON CONFLICT (votacao_id, parlamentar_id) DO UPDATE SET
				voto = EXCLUDED.voto, partido_na_votacao = EXCLUDED.partido_na_votacao, alinhou_governo = EXCLUDED.alinhou_governo`,
			v.ID, pids, votos, partidos, alinhou); err != nil {
			return fmt.Errorf("inserir votos: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// IniciarRun registra o início de uma execução do ETL.
func (s *Store) IniciarRun(ctx context.Context, job string, params any) (int64, error) {
	b, err := json.Marshal(params)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.pool.QueryRow(ctx, `INSERT INTO etl_runs (job, params) VALUES ($1, $2) RETURNING id`, job, b).Scan(&id)
	return id, err
}

func (s *Store) FinalizarRun(ctx context.Context, id int64, status, erro string) error {
	_, err := s.pool.Exec(ctx, `UPDATE etl_runs SET status = $2, erro = $3, finished_at = NOW() WHERE id = $1`, id, status, nullIfEmpty(erro))
	return err
}

func (s *Store) RegistrarErro(ctx context.Context, runID int64, ref, msg string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO etl_errors (run_id, ref, mensagem) VALUES ($1, $2, $3)`, runID, trunc(ref, 150), msg)
	return err
}

// metricas são as materialized views atualizadas ao fim de cada ciclo do ETL.
var metricas = []string{"mv_parlamentar_metricas", "mv_partido_metricas", "mv_parlamentar_presenca_sessao", "mv_partido_presenca_sessao"}

// AtualizarMetricas recalcula as materialized views sem bloquear leituras (CONCURRENTLY).
func (s *Store) AtualizarMetricas(ctx context.Context) error {
	for _, mv := range metricas {
		if _, err := s.pool.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY "+mv); err != nil {
			return fmt.Errorf("refresh %s: %w", mv, err)
		}
	}
	return nil
}

type SessaoIn struct {
	IDExterno int
	Inicio    time.Time
	Fim       *time.Time
	Tipo      string
	Descricao string
	Presentes []ParlamentarIn
}

func (s *Store) SessaoExiste(ctx context.Context, idExterno int) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sessoes WHERE id_externo = $1 AND casa = $2)`, idExterno, casaCamara).Scan(&ok)
	return ok, err
}

// SalvarSessao grava a sessão e substitui a lista de presença, numa única transação.
func (s *Store) SalvarSessao(ctx context.Context, in SessaoIn) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var id int
	if err := tx.QueryRow(ctx, `
		INSERT INTO sessoes (id_externo, casa, inicio, fim, tipo, descricao)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id_externo, casa) DO UPDATE SET
			inicio = EXCLUDED.inicio, fim = EXCLUDED.fim, tipo = EXCLUDED.tipo, descricao = EXCLUDED.descricao
		RETURNING id`,
		in.IDExterno, casaCamara, in.Inicio, in.Fim, trunc(in.Tipo, 60), nullIfEmpty(in.Descricao)).Scan(&id); err != nil {
		return fmt.Errorf("upsert sessão: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sessoes_presencas WHERE sessao_id = $1`, id); err != nil {
		return err
	}
	if len(in.Presentes) > 0 {
		ids, err := upsertParlamentares(ctx, tx, in.Presentes, false)
		if err != nil {
			return err
		}
		pids := make([]int32, 0, len(in.Presentes))
		partidos := make([]*string, 0, len(in.Presentes))
		visto := make(map[int]bool, len(in.Presentes))
		for _, p := range in.Presentes {
			if visto[p.IDExterno] {
				continue
			}
			visto[p.IDExterno] = true
			pid, ok := ids[p.IDExterno]
			if !ok {
				return fmt.Errorf("parlamentar %d não encontrado após upsert", p.IDExterno)
			}
			pids = append(pids, int32(pid))
			partidos = append(partidos, nullIfEmpty(trunc(p.Partido, 30)))
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO sessoes_presencas (sessao_id, parlamentar_id, partido_na_sessao)
			SELECT $1, t.pid, t.partido FROM unnest($2::int[], $3::text[]) AS t(pid, partido)`,
			id, pids, partidos); err != nil {
			return fmt.Errorf("inserir presenças: %w", err)
		}
	}
	return tx.Commit(ctx)
}
