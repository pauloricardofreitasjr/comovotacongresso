package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"comovotacongresso/internal/store"
)

func chiParam(r *http.Request, nome string) string { return chi.URLParam(r, nome) }

// Métricas, sempre acompanhadas das contagens que as sustentam (um percentual sem a base engana).

type metricaParticipacao struct {
	Elegiveis int      `json:"votacoes_elegiveis"`
	Votou     int      `json:"votacoes_votou"`
	Pct       *float64 `json:"pct"`
}
type metricaPresenca struct {
	Elegiveis int      `json:"sessoes_elegiveis"`
	Presente  int      `json:"sessoes_presente"`
	Pct       *float64 `json:"pct"`
}
type metricaGoverno struct {
	Base      int      `json:"votacoes_com_orientacao"`
	Alinhadas int      `json:"votacoes_alinhadas"`
	Pct       *float64 `json:"pct"`
}

type metricas struct {
	Participacao       metricaParticipacao `json:"participacao_votacoes_nominais"`
	PresencaSessao     metricaPresenca     `json:"presenca_sessao"`
	AlinhamentoGoverno metricaGoverno      `json:"alinhamento_governo"`
}

func toMetricas(p store.Participacao, s store.PresencaSessao, g store.Governismo) metricas {
	return metricas{
		metricaParticipacao{p.Elegiveis, p.Votou, p.Pct},
		metricaPresenca{s.Elegiveis, s.Presente, s.Pct},
		metricaGoverno{g.Base, g.Alinhadas, g.Pct},
	}
}

type parlamentarResumo struct {
	ID          int      `json:"id"`
	Casa        string   `json:"casa"`
	Nome        string   `json:"nome"`
	Partido     *string  `json:"partido"` // partido atual
	UF          *string  `json:"uf"`
	URLFoto     *string  `json:"url_foto"`
	EmExercicio bool     `json:"em_exercicio"`
	Metricas    metricas `json:"metricas"`
}

func toParlamentarResumo(p store.ParlamentarResumo) parlamentarResumo {
	return parlamentarResumo{p.ID, p.Casa, p.Nome, p.Partido, p.UF, p.URLFoto, p.EmExercicio,
		toMetricas(p.Participacao, p.PresencaSessao, p.Governismo)}
}

func (a *API) listarParlamentares(w http.ResponseWriter, r *http.Request) {
	pagina, limite, offset, err := paginacao(r, "pagina", "limite")
	if err != nil {
		erro(w, http.StatusBadRequest, "parametro_invalido", err.Error())
		return
	}
	f := store.ParlamentarFiltro{Limite: limite, Offset: offset}
	q := r.URL.Query()

	if f.Casa, err = casaParam(r); err == nil {
		f.Partido, err = texto(r, "partido", 30)
	}
	if err == nil {
		f.UF, err = texto(r, "uf", 2)
	}
	if err == nil {
		f.Busca, err = texto(r, "busca", 100)
	}
	if err == nil {
		f.MinVotacoes, err = inteiro(r, "min_votacoes", 0, 0, 100_000)
	}
	if err == nil {
		switch q.Get("em_exercicio") {
		case "", "true":
			t := true
			f.EmExercicio = &t
		case "false":
			t := false
			f.EmExercicio = &t
		case "todos":
		default:
			err = paramErr("parâmetro 'em_exercicio' deve ser 'true', 'false' ou 'todos'")
		}
	}
	if err == nil {
		switch f.OrdenarPor = q.Get("ordenarPor"); f.OrdenarPor {
		case "":
			f.OrdenarPor = "nome"
		case "nome", "participacao", "presenca_sessao", "governismo":
		default:
			err = paramErr("parâmetro 'ordenarPor' deve ser 'nome', 'participacao', 'presenca_sessao' ou 'governismo'")
		}
	}
	if err == nil {
		// padrão: nome A→Z; métricas do maior para o menor.
		f.Desc = f.OrdenarPor != "nome"
		switch q.Get("ordem") {
		case "":
		case "asc":
			f.Desc = false
		case "desc":
			f.Desc = true
		default:
			err = paramErr("parâmetro 'ordem' deve ser 'asc' ou 'desc'")
		}
	}
	if err != nil {
		erro(w, http.StatusBadRequest, "parametro_invalido", err.Error())
		return
	}

	itens, total, err := a.db.ListarParlamentares(r.Context(), f)
	if err != nil {
		a.falha(w, r, err)
		return
	}
	dados := make([]parlamentarResumo, len(itens))
	for i, p := range itens {
		dados[i] = toParlamentarResumo(p)
	}
	escrever(w, http.StatusOK, novaPagina(dados, pagina, limite, total))
}

type exercicioDTO struct {
	Inicio   time.Time  `json:"inicio"`
	Fim      *time.Time `json:"fim"` // null = em exercício
	Condicao *string    `json:"condicao"`
	Partido  *string    `json:"partido"`
}

type votoHistoricoDTO struct {
	VotacaoID         string    `json:"votacao_id"`
	DataHora          time.Time `json:"data_hora"`
	Descricao         string    `json:"descricao"`
	Aprovada          *bool     `json:"aprovada"`
	Voto              string    `json:"voto"`
	PartidoNaVotacao  *string   `json:"partido_na_votacao"`
	OrientacaoGoverno *string   `json:"orientacao_governo"`
	AlinhouGoverno    *bool     `json:"alinhou_governo"`
	Proposicao        *propRef  `json:"proposicao"`
}

type parlamentarDetalhe struct {
	parlamentarResumo
	Exercicio []exercicioDTO           `json:"exercicio"`
	Votos     Pagina[votoHistoricoDTO] `json:"votos"`
}

func (a *API) getParlamentar(w http.ResponseWriter, r *http.Request) {
	id, ok := idInt(r)
	if !ok {
		erro(w, http.StatusBadRequest, "parametro_invalido", "id deve ser um inteiro positivo")
		return
	}
	pagina, limite, offset, err := paginacao(r, "page_votos", "limite_votos")
	if err != nil {
		erro(w, http.StatusBadRequest, "parametro_invalido", err.Error())
		return
	}
	var f store.FiltroVotos
	if f.Voto, err = opcao(r, "voto", "sim", "nao", "abstencao", "obstrucao", "artigo_17", "outro"); err == nil {
		f.Alinhamento, err = opcao(r, "alinhamento", "alinhado", "nao_alinhado")
	}
	if err != nil {
		erro(w, http.StatusBadRequest, "parametro_invalido", err.Error())
		return
	}
	d, err := a.db.GetParlamentar(r.Context(), id, f, limite, offset)
	if err != nil {
		a.falha(w, r, err)
		return
	}
	out := parlamentarDetalhe{parlamentarResumo: toParlamentarResumo(d.ParlamentarResumo), Exercicio: make([]exercicioDTO, len(d.Exercicio))}
	for i, e := range d.Exercicio {
		out.Exercicio[i] = exercicioDTO{e.Inicio, e.Fim, e.Condicao, e.Partido}
	}
	votos := make([]votoHistoricoDTO, len(d.Votos))
	for i, v := range d.Votos {
		votos[i] = votoHistoricoDTO{v.VotacaoID, v.DataHora, v.Descricao, v.Aprovada, v.Voto, v.PartidoNaVotacao,
			v.OrientacaoGoverno, v.AlinhouGoverno, toPropRef(v.Proposicao)}
	}
	out.Votos = novaPagina(votos, pagina, limite, d.TotalVotos)
	escrever(w, http.StatusOK, out)
}

type partidoDTO struct {
	Casa          string   `json:"casa"`
	Partido       string   `json:"partido"` // partido na data de cada voto/sessão
	Parlamentares int      `json:"parlamentares"`
	Metricas      metricas `json:"metricas"`
}

func (a *API) listarPartidos(w http.ResponseWriter, r *http.Request) {
	casa, err := casaParam(r)
	if err != nil {
		erro(w, http.StatusBadRequest, "parametro_invalido", err.Error())
		return
	}
	itens, err := a.db.ListarPartidos(r.Context(), casa)
	if err != nil {
		a.falha(w, r, err)
		return
	}
	dados := make([]partidoDTO, len(itens))
	for i, p := range itens {
		dados[i] = partidoDTO{p.Casa, p.Partido, p.Parlamentares, toMetricas(p.Participacao, p.PresencaSessao, p.Governismo)}
	}
	escrever(w, http.StatusOK, map[string]any{"dados": dados})
}
