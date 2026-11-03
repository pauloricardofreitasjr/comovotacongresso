package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrNaoEncontrado é devolvido pelas consultas por id quando o registro não existe.
var ErrNaoEncontrado = errors.New("não encontrado")

// semAcento normaliza texto para busca: minúsculas e sem acentos. A mesma transformação
// é aplicada no SQL (sqlNorm), dispensando a extensão unaccent.
func semAcento(s string) string {
	return strings.ToLower(strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i", "ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u", "ç", "c", "ñ", "n",
		"Á", "a", "À", "a", "Â", "a", "Ã", "a", "Ä", "a", "É", "e", "È", "e", "Ê", "e", "Ë", "e",
		"Í", "i", "Ì", "i", "Î", "i", "Ï", "i", "Ó", "o", "Ò", "o", "Ô", "o", "Õ", "o", "Ö", "o",
		"Ú", "u", "Ù", "u", "Û", "u", "Ü", "u", "Ç", "c", "Ñ", "n",
	).Replace(s))
}

func sqlNorm(col string) string {
	return "translate(lower(" + col + "), 'áàâãäéèêëíìîïóòôõöúùûüçñ', 'aaaaaeeeeiiiiooooouuuucn')"
}

// likeEscape escapa curingas do LIKE no termo de busca.
func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// ---------- Proposições ----------

type ProposicaoFiltro struct {
	Tipo   string // sigla (PL, PEC...)
	Ano    int
	Numero int
	Busca  string // texto livre na ementa
	Limite int
	Offset int
}

type ProposicaoResumo struct {
	ID            int
	Casa          string
	SiglaTipo     string
	Numero        int
	Ano           int
	Ementa        string
	TotalVotacoes int
	Nominais      int
	UltimaVotacao time.Time
}

func (s *Store) ListarProposicoes(ctx context.Context, f ProposicaoFiltro) ([]ProposicaoResumo, int, error) {
	var where []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if f.Tipo != "" {
		add("p.sigla_tipo = $%d", strings.ToUpper(f.Tipo))
	}
	if f.Ano > 0 {
		add("p.ano = $%d", f.Ano)
	}
	if f.Numero > 0 {
		add("p.numero = $%d", f.Numero)
	}
	if f.Busca != "" {
		add(sqlNorm("coalesce(p.ementa,'')")+" LIKE '%%' || $%d || '%%'", likeEscape(semAcento(f.Busca)))
	}
	cond := ""
	if len(where) > 0 {
		cond = "WHERE " + strings.Join(where, " AND ")
	}
	args = append(args, f.Limite, f.Offset)
	q := fmt.Sprintf(`
		SELECT p.id, p.casa::text, p.sigla_tipo, p.numero, p.ano, coalesce(p.ementa,''),
		       count(v.id), count(v.id) FILTER (WHERE v.tipo = 'nominal'), max(v.data_hora), count(*) OVER()
		FROM proposicoes p JOIN votacoes v ON v.proposicao_id = p.id
		%s
		GROUP BY p.id
		ORDER BY max(v.data_hora) DESC, p.id DESC
		LIMIT $%d OFFSET $%d`, cond, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []ProposicaoResumo
	total := 0
	for rows.Next() {
		var p ProposicaoResumo
		if err := rows.Scan(&p.ID, &p.Casa, &p.SiglaTipo, &p.Numero, &p.Ano, &p.Ementa, &p.TotalVotacoes, &p.Nominais, &p.UltimaVotacao, &total); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(out) == 0 && f.Offset > 0 { // count(*) OVER() some sem linhas: reconta para manter o total correto
		err := s.pool.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM (
			SELECT p.id FROM proposicoes p JOIN votacoes v ON v.proposicao_id = p.id %s GROUP BY p.id) t`, cond),
			args[:len(args)-2]...).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
	}
	return out, total, nil
}

// ResumoVotos conta os votos de uma votação nominal por tipo.
type ResumoVotos struct {
	Sim, Nao, Abstencao, Obstrucao, Artigo17, Outro, Total int
}

type VotacaoLinha struct {
	ID                string
	DataHora          time.Time
	Descricao         string
	Tipo              string
	Aprovada          *bool
	Vinculo           *string
	OrientacaoGoverno *string
	Resumo            *ResumoVotos // nil se não for nominal
}

type ProposicaoDetalhe struct {
	ProposicaoResumo
	Votacoes []VotacaoLinha // ordem cronológica
}

func (s *Store) GetProposicao(ctx context.Context, id int) (*ProposicaoDetalhe, error) {
	var d ProposicaoDetalhe
	err := s.pool.QueryRow(ctx, `
		SELECT p.id, p.casa::text, p.sigla_tipo, p.numero, p.ano, coalesce(p.ementa,''),
		       count(v.id), count(v.id) FILTER (WHERE v.tipo = 'nominal'), coalesce(max(v.data_hora), 'epoch'::timestamptz)
		FROM proposicoes p LEFT JOIN votacoes v ON v.proposicao_id = p.id
		WHERE p.id = $1 GROUP BY p.id`, id).
		Scan(&d.ID, &d.Casa, &d.SiglaTipo, &d.Numero, &d.Ano, &d.Ementa, &d.TotalVotacoes, &d.Nominais, &d.UltimaVotacao)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, votacoesBase+` WHERE v.proposicao_id = $1 GROUP BY v.id ORDER BY v.data_hora ASC, v.id ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		l, err := scanVotacaoLinha(rows)
		if err != nil {
			return nil, err
		}
		d.Votacoes = append(d.Votacoes, l)
	}
	return &d, rows.Err()
}

const votacoesBase = `
	SELECT v.id, v.data_hora, coalesce(v.descricao,''), v.tipo::text, v.aprovada, v.vinculo_confianca::text, v.orientacao_governo::text,
	       count(x.id), count(x.id) FILTER (WHERE x.voto='sim'), count(x.id) FILTER (WHERE x.voto='nao'),
	       count(x.id) FILTER (WHERE x.voto='abstencao'), count(x.id) FILTER (WHERE x.voto='obstrucao'),
	       count(x.id) FILTER (WHERE x.voto='artigo_17'), count(x.id) FILTER (WHERE x.voto='outro')
	FROM votacoes v LEFT JOIN votos x ON x.votacao_id = v.id`

func scanVotacaoLinha(r pgx.Rows) (VotacaoLinha, error) {
	var l VotacaoLinha
	var rs ResumoVotos
	if err := r.Scan(&l.ID, &l.DataHora, &l.Descricao, &l.Tipo, &l.Aprovada, &l.Vinculo, &l.OrientacaoGoverno,
		&rs.Total, &rs.Sim, &rs.Nao, &rs.Abstencao, &rs.Obstrucao, &rs.Artigo17, &rs.Outro); err != nil {
		return l, err
	}
	if l.Tipo == "nominal" {
		l.Resumo = &rs
	}
	return l, nil
}

// ---------- Votação ----------

type OrientacaoLinha struct {
	Bancada       string
	TipoLideranca *string
	Orientacao    *string
}

type VotoLinha struct {
	ParlamentarID  int
	Nome           string
	Partido        *string
	UF             *string
	URLFoto        *string
	Voto           string
	AlinhouGoverno *bool
}

type PropRef struct {
	ID          int
	SiglaTipo   string
	Numero, Ano int
	Ementa      string
}

type VotacaoDetalhe struct {
	VotacaoLinha
	Proposicao  *PropRef
	Orientacoes []OrientacaoLinha
	Votos       []VotoLinha // nil se não for nominal
}

func (s *Store) GetVotacao(ctx context.Context, id string) (*VotacaoDetalhe, error) {
	rows, err := s.pool.Query(ctx, votacoesBase+` WHERE v.id = $1 GROUP BY v.id`, id)
	if err != nil {
		return nil, err
	}
	var d VotacaoDetalhe
	if rows.Next() {
		if d.VotacaoLinha, err = scanVotacaoLinha(rows); err != nil {
			rows.Close()
			return nil, err
		}
	} else {
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, ErrNaoEncontrado
	}
	rows.Close()

	var pr PropRef
	err = s.pool.QueryRow(ctx, `SELECT p.id, p.sigla_tipo, p.numero, p.ano, coalesce(p.ementa,'')
		FROM votacoes v JOIN proposicoes p ON p.id = v.proposicao_id WHERE v.id = $1`, id).
		Scan(&pr.ID, &pr.SiglaTipo, &pr.Numero, &pr.Ano, &pr.Ementa)
	switch {
	case err == nil:
		d.Proposicao = &pr
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, err
	}

	orows, err := s.pool.Query(ctx, `SELECT bancada, tipo_lideranca, orientacao::text FROM votacoes_orientacoes
		WHERE votacao_id = $1 ORDER BY (tipo_lideranca = 'B') DESC, bancada`, id)
	if err != nil {
		return nil, err
	}
	defer orows.Close()
	for orows.Next() {
		var o OrientacaoLinha
		if err := orows.Scan(&o.Bancada, &o.TipoLideranca, &o.Orientacao); err != nil {
			return nil, err
		}
		d.Orientacoes = append(d.Orientacoes, o)
	}
	if err := orows.Err(); err != nil {
		return nil, err
	}

	if d.Tipo == "nominal" {
		vrows, err := s.pool.Query(ctx, `
			SELECT p.id, p.nome, x.partido_na_votacao, p.uf, p.url_foto, x.voto::text, x.alinhou_governo
			FROM votos x JOIN parlamentares p ON p.id = x.parlamentar_id
			WHERE x.votacao_id = $1 ORDER BY p.nome, p.id`, id)
		if err != nil {
			return nil, err
		}
		defer vrows.Close()
		d.Votos = []VotoLinha{}
		for vrows.Next() {
			var v VotoLinha
			if err := vrows.Scan(&v.ParlamentarID, &v.Nome, &v.Partido, &v.UF, &v.URLFoto, &v.Voto, &v.AlinhouGoverno); err != nil {
				return nil, err
			}
			d.Votos = append(d.Votos, v)
		}
		if err := vrows.Err(); err != nil {
			return nil, err
		}
	}
	return &d, nil
}

// ---------- Parlamentares ----------

type Participacao struct {
	Elegiveis, Votou int
	Pct              *float64
}
type PresencaSessao struct {
	Elegiveis, Presente int
	Pct                 *float64
}
type Governismo struct {
	Base, Alinhadas int
	Pct             *float64
}

type ParlamentarResumo struct {
	ID             int
	Casa           string
	Nome           string
	Partido        *string
	UF             *string
	URLFoto        *string
	EmExercicio    bool
	Participacao   Participacao
	PresencaSessao PresencaSessao
	Governismo     Governismo
}

type ParlamentarFiltro struct {
	Casa        string
	Partido     string
	UF          string
	EmExercicio *bool // nil = todos
	Busca       string
	OrdenarPor  string // nome | participacao | presenca_sessao | governismo
	Desc        bool
	MinVotacoes int // denominador mínimo da métrica ordenada
	Limite      int
	Offset      int
}

// colunas ordenáveis (whitelist: nunca interpolar entrada do usuário).
var ordenaveis = map[string]struct{ col, denom string }{
	"nome":            {"m.nome", ""},
	"participacao":    {"m.pct_participacao", "m.votacoes_elegiveis"},
	"presenca_sessao": {"ps.pct_presenca_sessao", "ps.sessoes_elegiveis"},
	"governismo":      {"m.pct_governista", "m.votacoes_base_governo"},
}

const parlamentarSelect = `
	SELECT m.parlamentar_id, m.casa::text, m.nome, m.partido_atual, m.uf, m.url_foto, m.em_exercicio,
	       m.votacoes_elegiveis, m.votacoes_votou, m.pct_participacao::float8,
	       coalesce(ps.sessoes_elegiveis,0), coalesce(ps.sessoes_presente,0), ps.pct_presenca_sessao::float8,
	       m.votacoes_base_governo, m.votacoes_alinhadas_governo, m.pct_governista::float8
	FROM mv_parlamentar_metricas m
	LEFT JOIN mv_parlamentar_presenca_sessao ps ON ps.parlamentar_id = m.parlamentar_id`

func scanParlamentar(r pgx.Row, extra ...any) (ParlamentarResumo, error) {
	var p ParlamentarResumo
	dest := []any{&p.ID, &p.Casa, &p.Nome, &p.Partido, &p.UF, &p.URLFoto, &p.EmExercicio,
		&p.Participacao.Elegiveis, &p.Participacao.Votou, &p.Participacao.Pct,
		&p.PresencaSessao.Elegiveis, &p.PresencaSessao.Presente, &p.PresencaSessao.Pct,
		&p.Governismo.Base, &p.Governismo.Alinhadas, &p.Governismo.Pct}
	return p, r.Scan(append(dest, extra...)...)
}

func (s *Store) ListarParlamentares(ctx context.Context, f ParlamentarFiltro) ([]ParlamentarResumo, int, error) {
	ord, ok := ordenaveis[f.OrdenarPor]
	if !ok {
		ord = ordenaveis["nome"]
	}
	var where []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if f.Casa != "" {
		add("m.casa = $%d::casa_legislativa", f.Casa)
	}
	if f.Partido != "" {
		add("upper(m.partido_atual) = upper($%d)", f.Partido)
	}
	if f.UF != "" {
		add("upper(m.uf) = upper($%d)", f.UF)
	}
	if f.EmExercicio != nil {
		add("m.em_exercicio = $%d", *f.EmExercicio)
	}
	if f.Busca != "" {
		add(sqlNorm("m.nome")+" LIKE '%%' || $%d || '%%'", likeEscape(semAcento(f.Busca)))
	}
	if ord.denom != "" && f.MinVotacoes > 0 {
		add(ord.denom+" >= $%d", f.MinVotacoes)
	}
	cond := ""
	if len(where) > 0 {
		cond = "WHERE " + strings.Join(where, " AND ")
	}
	dir := "ASC"
	if f.Desc {
		dir = "DESC"
	}
	// sem a métrica (NULL) o parlamentar vai para o fim; desempate estável por nome/id.
	order := fmt.Sprintf("ORDER BY %s %s NULLS LAST, m.nome ASC, m.parlamentar_id ASC", ord.col, dir)
	args = append(args, f.Limite, f.Offset)
	q := strings.Replace(parlamentarSelect, "SELECT ", "SELECT count(*) OVER(), ", 1) +
		" " + cond + " " + order + fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []ParlamentarResumo
	total := 0
	for rows.Next() {
		// o count(*) OVER() vem primeiro: lê numa variável e depois as colunas padrão.
		var p ParlamentarResumo
		if err := rows.Scan(&total, &p.ID, &p.Casa, &p.Nome, &p.Partido, &p.UF, &p.URLFoto, &p.EmExercicio,
			&p.Participacao.Elegiveis, &p.Participacao.Votou, &p.Participacao.Pct,
			&p.PresencaSessao.Elegiveis, &p.PresencaSessao.Presente, &p.PresencaSessao.Pct,
			&p.Governismo.Base, &p.Governismo.Alinhadas, &p.Governismo.Pct); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(out) == 0 && f.Offset > 0 { // idem: total correto mesmo além da última página
		if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM mv_parlamentar_metricas m LEFT JOIN mv_parlamentar_presenca_sessao ps ON ps.parlamentar_id = m.parlamentar_id "+cond,
			args[:len(args)-2]...).Scan(&total); err != nil {
			return nil, 0, err
		}
	}
	return out, total, nil
}

type ExercicioLinha struct {
	Inicio   time.Time
	Fim      *time.Time
	Condicao *string
	Partido  *string
}

type VotoHistorico struct {
	VotacaoID         string
	DataHora          time.Time
	Descricao         string
	Aprovada          *bool
	OrientacaoGoverno *string
	Voto              string
	AlinhouGoverno    *bool
	PartidoNaVotacao  *string
	Proposicao        *PropRef
}

type ParlamentarDetalhe struct {
	ParlamentarResumo
	Exercicio  []ExercicioLinha
	Votos      []VotoHistorico
	TotalVotos int
}

func (s *Store) GetParlamentar(ctx context.Context, id, limiteVotos, offsetVotos int) (*ParlamentarDetalhe, error) {
	p, err := scanParlamentar(s.pool.QueryRow(ctx, parlamentarSelect+` WHERE m.parlamentar_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	d := &ParlamentarDetalhe{ParlamentarResumo: p}

	erows, err := s.pool.Query(ctx, `SELECT inicio, fim, condicao, partido FROM parlamentares_exercicio
		WHERE parlamentar_id = $1 ORDER BY inicio`, id)
	if err != nil {
		return nil, err
	}
	defer erows.Close()
	for erows.Next() {
		var e ExercicioLinha
		if err := erows.Scan(&e.Inicio, &e.Fim, &e.Condicao, &e.Partido); err != nil {
			return nil, err
		}
		d.Exercicio = append(d.Exercicio, e)
	}
	if err := erows.Err(); err != nil {
		return nil, err
	}

	vrows, err := s.pool.Query(ctx, `
		SELECT v.id, v.data_hora, coalesce(v.descricao,''), v.aprovada, v.orientacao_governo::text,
		       x.voto::text, x.alinhou_governo, x.partido_na_votacao,
		       p.id, p.sigla_tipo, p.numero, p.ano, coalesce(p.ementa,''), count(*) OVER()
		FROM votos x
		JOIN votacoes v ON v.id = x.votacao_id
		LEFT JOIN proposicoes p ON p.id = v.proposicao_id
		WHERE x.parlamentar_id = $1
		ORDER BY v.data_hora DESC, v.id DESC LIMIT $2 OFFSET $3`, id, limiteVotos, offsetVotos)
	if err != nil {
		return nil, err
	}
	defer vrows.Close()
	d.Votos = []VotoHistorico{}
	for vrows.Next() {
		var h VotoHistorico
		var pid, pnum, pano *int
		var psigla, pementa *string
		if err := vrows.Scan(&h.VotacaoID, &h.DataHora, &h.Descricao, &h.Aprovada, &h.OrientacaoGoverno,
			&h.Voto, &h.AlinhouGoverno, &h.PartidoNaVotacao, &pid, &psigla, &pnum, &pano, &pementa, &d.TotalVotos); err != nil {
			return nil, err
		}
		if pid != nil {
			h.Proposicao = &PropRef{ID: *pid, SiglaTipo: *psigla, Numero: *pnum, Ano: *pano, Ementa: *pementa}
		}
		d.Votos = append(d.Votos, h)
	}
	if err := vrows.Err(); err != nil {
		return nil, err
	}
	if len(d.Votos) == 0 && offsetVotos > 0 { // página além do fim: o total não veio nas linhas
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM votos WHERE parlamentar_id = $1`, id).Scan(&d.TotalVotos); err != nil {
			return nil, err
		}
	}
	return d, nil
}

// ---------- Partidos e status ----------

type PartidoLinha struct {
	Casa           string
	Partido        string
	Parlamentares  int
	Participacao   Participacao
	PresencaSessao PresencaSessao
	Governismo     Governismo
}

func (s *Store) ListarPartidos(ctx context.Context, casa string) ([]PartidoLinha, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.casa::text, m.partido, m.parlamentares,
		       m.votacoes_elegiveis, m.votacoes_votou, m.pct_participacao::float8,
		       coalesce(ps.sessoes_elegiveis,0), coalesce(ps.sessoes_presente,0), ps.pct_presenca_sessao::float8,
		       m.votacoes_base_governo, m.votacoes_alinhadas_governo, m.pct_governista::float8
		FROM mv_partido_metricas m
		LEFT JOIN mv_partido_presenca_sessao ps ON ps.casa = m.casa AND ps.partido = m.partido
		WHERE ($1 = '' OR m.casa = $1::casa_legislativa)
		ORDER BY m.pct_governista DESC NULLS LAST, m.partido`, casa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PartidoLinha
	for rows.Next() {
		var p PartidoLinha
		if err := rows.Scan(&p.Casa, &p.Partido, &p.Parlamentares,
			&p.Participacao.Elegiveis, &p.Participacao.Votou, &p.Participacao.Pct,
			&p.PresencaSessao.Elegiveis, &p.PresencaSessao.Presente, &p.PresencaSessao.Pct,
			&p.Governismo.Base, &p.Governismo.Alinhadas, &p.Governismo.Pct); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// MediasCamara são os percentuais do conjunto da Câmara (todas as sessões e votações somadas,
// não a média simples dos deputados), para dar referência ao percentual de cada deputado.
type MediasCamara struct {
	PresencaSessao *float64
	Participacao   *float64
	Governismo     *float64
}

type Status struct {
	Medias             MediasCamara
	AtualizadoEm       *time.Time
	PrimeiraVotacao    *time.Time
	UltimaVotacao      *time.Time
	VotacoesNominais   int
	VotacoesSimbolicas int
	Sessoes            int
	Parlamentares      int
	Proposicoes        int
}

func (s *Store) GetStatus(ctx context.Context) (Status, error) {
	var st Status
	err := s.pool.QueryRow(ctx, `
		SELECT (SELECT max(atualizado_em) FROM mv_parlamentar_metricas),
		       (SELECT min(data_hora) FROM votacoes), (SELECT max(data_hora) FROM votacoes),
		       (SELECT count(*) FROM votacoes WHERE tipo = 'nominal'),
		       (SELECT count(*) FROM votacoes WHERE tipo = 'simbolica'),
		       (SELECT count(*) FROM sessoes), (SELECT count(*) FROM parlamentares),
		       (SELECT count(*) FROM proposicoes)`).
		Scan(&st.AtualizadoEm, &st.PrimeiraVotacao, &st.UltimaVotacao, &st.VotacoesNominais, &st.VotacoesSimbolicas,
			&st.Sessoes, &st.Parlamentares, &st.Proposicoes)
	if err != nil {
		return st, err
	}
	err = s.pool.QueryRow(ctx, `
		SELECT (SELECT ROUND((100.0 * SUM(m.sessoes_presente) / NULLIF(SUM(m.sessoes_elegiveis), 0))::numeric, 2)::float8
		          FROM mv_parlamentar_presenca_sessao m JOIN parlamentares p ON p.id = m.parlamentar_id WHERE p.casa = 'camara'),
		       (SELECT ROUND((100.0 * SUM(votacoes_votou) / NULLIF(SUM(votacoes_elegiveis), 0))::numeric, 2)::float8
		          FROM mv_parlamentar_metricas WHERE casa = 'camara'),
		       (SELECT ROUND((100.0 * SUM(votacoes_alinhadas_governo) / NULLIF(SUM(votacoes_base_governo), 0))::numeric, 2)::float8
		          FROM mv_parlamentar_metricas WHERE casa = 'camara')`).
		Scan(&st.Medias.PresencaSessao, &st.Medias.Participacao, &st.Medias.Governismo)
	return st, err
}
