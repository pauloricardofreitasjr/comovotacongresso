package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"comovotacongresso/internal/store"
)

// ---------- DTOs (JSON em snake_case, português) ----------

type propRef struct {
	ID            int    `json:"id"`
	Identificacao string `json:"identificacao"`
	SiglaTipo     string `json:"sigla_tipo"`
	Numero        int    `json:"numero"`
	Ano           int    `json:"ano"`
	Ementa        string `json:"ementa"`
}

func identificacao(sigla string, numero, ano int) string {
	return fmt.Sprintf("%s %d/%d", sigla, numero, ano)
}

func toPropRef(p *store.PropRef) *propRef {
	if p == nil {
		return nil
	}
	return &propRef{p.ID, identificacao(p.SiglaTipo, p.Numero, p.Ano), p.SiglaTipo, p.Numero, p.Ano, p.Ementa}
}

type resumoVotos struct {
	Sim       int `json:"sim"`
	Nao       int `json:"nao"`
	Abstencao int `json:"abstencao"`
	Obstrucao int `json:"obstrucao"`
	Artigo17  int `json:"artigo_17"`
	Outro     int `json:"outro"`
	Total     int `json:"total"`
}

func toResumo(r *store.ResumoVotos) *resumoVotos {
	if r == nil {
		return nil
	}
	return &resumoVotos{r.Sim, r.Nao, r.Abstencao, r.Obstrucao, r.Artigo17, r.Outro, r.Total}
}

type votacaoLinha struct {
	ID                string       `json:"id"`
	DataHora          time.Time    `json:"data_hora"`
	Descricao         string       `json:"descricao"`
	Tipo              string       `json:"tipo"` // nominal | simbolica
	Aprovada          *bool        `json:"aprovada"`
	VinculoConfianca  *string      `json:"vinculo_confianca"` // direto | provavel | null
	OrientacaoGoverno *string      `json:"orientacao_governo"`
	ResumoVotos       *resumoVotos `json:"resumo_votos"` // null em votação simbólica
	Principal         bool         `json:"principal,omitempty"`
}

func toVotacaoLinha(l store.VotacaoLinha) votacaoLinha {
	return votacaoLinha{l.ID, l.DataHora, l.Descricao, l.Tipo, l.Aprovada, l.Vinculo, l.OrientacaoGoverno, toResumo(l.Resumo), false}
}

// ---------- /status ----------

// mediasDTO: percentuais do conjunto da Câmara (todas as sessões e votações somadas), referência para cada deputado.
type mediasDTO struct {
	PresencaSessao *float64 `json:"presenca_sessao"`
	Participacao   *float64 `json:"participacao_votacoes_nominais"`
	Governismo     *float64 `json:"alinhamento_governo"`
}

type statusDTO struct {
	AtualizadoEm       *time.Time `json:"atualizado_em"`
	PrimeiraVotacao    *time.Time `json:"primeira_votacao"`
	UltimaVotacao      *time.Time `json:"ultima_votacao"`
	VotacoesNominais   int        `json:"votacoes_nominais"`
	VotacoesSimbolicas int        `json:"votacoes_simbolicas"`
	Sessoes            int        `json:"sessoes_deliberativas"`
	Parlamentares      int        `json:"parlamentares"`
	Proposicoes        int        `json:"proposicoes"`
	Medias             mediasDTO  `json:"medias_camara"`
}

func (a *API) status(w http.ResponseWriter, r *http.Request) {
	st, err := a.db.GetStatus(r.Context())
	if err != nil {
		a.falha(w, r, err)
		return
	}
	escrever(w, http.StatusOK, statusDTO{st.AtualizadoEm, st.PrimeiraVotacao, st.UltimaVotacao, st.VotacoesNominais,
		st.VotacoesSimbolicas, st.Sessoes, st.Parlamentares, st.Proposicoes,
		mediasDTO{st.Medias.PresencaSessao, st.Medias.Participacao, st.Medias.Governismo}})
}

// ---------- /proposicoes ----------

type proposicaoResumo struct {
	propRef
	Casa             string    `json:"casa"`
	TotalVotacoes    int       `json:"total_votacoes"`
	VotacoesNominais int       `json:"votacoes_nominais"`
	UltimaVotacao    time.Time `json:"ultima_votacao"`
	// só na listagem, e só para proposições com votação nominal
	UltimaNominal *ultimaNominalDTO `json:"ultima_nominal,omitempty"`
}

type ultimaNominalDTO struct {
	VotacaoID string    `json:"votacao_id"`
	DataHora  time.Time `json:"data_hora"`
	Aprovada  *bool     `json:"aprovada"`
	Sim       int       `json:"sim"`
	Nao       int       `json:"nao"`
}

func toProposicaoResumo(p store.ProposicaoResumo) proposicaoResumo {
	var un *ultimaNominalDTO
	if p.UltimaNominal != nil {
		un = &ultimaNominalDTO{p.UltimaNominal.VotacaoID, p.UltimaNominal.DataHora, p.UltimaNominal.Aprovada, p.UltimaNominal.Sim, p.UltimaNominal.Nao}
	}
	return proposicaoResumo{
		UltimaNominal:    un,
		propRef:          propRef{p.ID, identificacao(p.SiglaTipo, p.Numero, p.Ano), p.SiglaTipo, p.Numero, p.Ano, p.Ementa},
		Casa:             p.Casa,
		TotalVotacoes:    p.TotalVotacoes,
		VotacoesNominais: p.Nominais,
		UltimaVotacao:    p.UltimaVotacao,
	}
}

func (a *API) listarProposicoes(w http.ResponseWriter, r *http.Request) {
	pagina, limite, offset, err := paginacao(r, "pagina", "limite")
	if err != nil {
		erro(w, http.StatusBadRequest, "parametro_invalido", err.Error())
		return
	}
	f := store.ProposicaoFiltro{Limite: limite, Offset: offset}
	if f.Ano, err = inteiro(r, "ano", 0, 1900, 2200); err == nil {
		f.Numero, err = inteiro(r, "numero", 0, 1, 10_000_000)
	}
	if err == nil {
		f.Tipo, err = texto(r, "tipo", 15)
	}
	if err == nil {
		f.Busca, err = texto(r, "busca", 100)
	}
	if err == nil {
		f.SoNominais, err = booleano(r, "so_nominais")
	}
	if err != nil {
		erro(w, http.StatusBadRequest, "parametro_invalido", err.Error())
		return
	}
	itens, total, err := a.db.ListarProposicoes(r.Context(), f)
	if err != nil {
		a.falha(w, r, err)
		return
	}
	dados := make([]proposicaoResumo, len(itens))
	for i, p := range itens {
		dados[i] = toProposicaoResumo(p)
	}
	escrever(w, http.StatusOK, novaPagina(dados, pagina, limite, total))
}

type proposicaoDetalhe struct {
	proposicaoResumo
	Votacoes []votacaoLinha `json:"votacoes"`
}

func (a *API) getProposicao(w http.ResponseWriter, r *http.Request) {
	id, ok := idInt(r)
	if !ok {
		erro(w, http.StatusBadRequest, "parametro_invalido", "id deve ser um inteiro positivo")
		return
	}
	d, err := a.db.GetProposicao(r.Context(), id)
	if err != nil {
		a.falha(w, r, err)
		return
	}
	out := proposicaoDetalhe{proposicaoResumo: toProposicaoResumo(d.ProposicaoResumo), Votacoes: make([]votacaoLinha, len(d.Votacoes))}
	for i, v := range d.Votacoes {
		out.Votacoes[i] = toVotacaoLinha(v)
	}
	if i := indicePrincipal(d.Votacoes); i >= 0 {
		out.Votacoes[i].Principal = true
	}
	escrever(w, http.StatusOK, out)
}

// indicePrincipal escolhe a votação em destaque da linha do tempo (as votações vêm em ordem
// cronológica). Regra: entre as nominais com vínculo direto, a de mais votos (empate: a mais
// recente); sem nominais diretas, a direta mais recente; senão a mais recente. É uma heurística
// de apresentação, não uma afirmação sobre qual votação "decidiu" a matéria.
func indicePrincipal(vs []store.VotacaoLinha) int {
	if len(vs) == 0 {
		return -1
	}
	direto := func(v store.VotacaoLinha) bool { return v.Vinculo != nil && *v.Vinculo == "direto" }
	melhor, maxVotos := -1, -1
	for i, v := range vs {
		if v.Tipo == "nominal" && direto(v) && v.Resumo != nil && v.Resumo.Total >= maxVotos {
			melhor, maxVotos = i, v.Resumo.Total
		}
	}
	if melhor >= 0 {
		return melhor
	}
	for i := len(vs) - 1; i >= 0; i-- {
		if direto(vs[i]) {
			return i
		}
	}
	return len(vs) - 1
}

// ---------- /votacoes/{id} ----------

type orientacaoDTO struct {
	Bancada       string  `json:"bancada"`
	TipoLideranca *string `json:"tipo_lideranca"` // P = partido, B = bloco/liderança
	Orientacao    *string `json:"orientacao"`     // null = sem orientação
}

type votoDTO struct {
	ParlamentarID  int     `json:"parlamentar_id"`
	Nome           string  `json:"nome"`
	Partido        *string `json:"partido"` // partido na data da votação
	UF             *string `json:"uf"`
	URLFoto        *string `json:"url_foto"`
	Voto           string  `json:"voto"`
	AlinhouGoverno *bool   `json:"alinhou_governo"`
}

type votacaoDetalhe struct {
	votacaoLinha
	Proposicao       *propRef        `json:"proposicao"`
	Orientacoes      []orientacaoDTO `json:"orientacoes"`
	VotosIndividuais bool            `json:"votos_individuais"` // false em votação simbólica
	Votos            []votoDTO       `json:"votos"`             // null quando não há votos individuais
}

func (a *API) getVotacao(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chiParam(r, "id"))
	if id == "" || len(id) > 100 {
		erro(w, http.StatusBadRequest, "parametro_invalido", "id da votação inválido")
		return
	}
	d, err := a.db.GetVotacao(r.Context(), id)
	if err != nil {
		a.falha(w, r, err)
		return
	}
	out := votacaoDetalhe{
		votacaoLinha:     toVotacaoLinha(d.VotacaoLinha),
		Proposicao:       toPropRef(d.Proposicao),
		Orientacoes:      make([]orientacaoDTO, len(d.Orientacoes)),
		VotosIndividuais: d.Votos != nil,
	}
	for i, o := range d.Orientacoes {
		out.Orientacoes[i] = orientacaoDTO{o.Bancada, o.TipoLideranca, o.Orientacao}
	}
	if d.Votos != nil {
		out.Votos = make([]votoDTO, len(d.Votos))
		for i, v := range d.Votos {
			out.Votos[i] = votoDTO{v.ParlamentarID, v.Nome, v.Partido, v.UF, v.URLFoto, v.Voto, v.AlinhouGoverno}
		}
	}
	escrever(w, http.StatusOK, out)
}
