package camara

// Tipos da API v2 da Câmara dos Deputados (somente os campos usados).

type VotacaoResumo struct {
	ID               string `json:"id"`
	DataHoraRegistro string `json:"dataHoraRegistro"`
	Descricao        string `json:"descricao"`
	Aprovacao        *int   `json:"aprovacao"`
}

type Proposicao struct {
	ID        int    `json:"id"`
	SiglaTipo string `json:"siglaTipo"`
	Numero    int    `json:"numero"`
	Ano       int    `json:"ano"`
	Ementa    string `json:"ementa"`
}

type VotacaoDetalhe struct {
	ID                  string       `json:"id"`
	DataHoraRegistro    string       `json:"dataHoraRegistro"`
	SiglaOrgao          string       `json:"siglaOrgao"`
	Descricao           string       `json:"descricao"`
	Aprovacao           *int         `json:"aprovacao"`
	ProposicoesAfetadas []Proposicao `json:"proposicoesAfetadas"`
	ObjetosPossiveis    []Proposicao `json:"objetosPossiveis"`
}

type Legislatura struct {
	ID         int    `json:"id"`
	DataInicio string `json:"dataInicio"`
	DataFim    string `json:"dataFim"`
}

type Deputado struct {
	ID            int    `json:"id"`
	Nome          string `json:"nome"`
	SiglaPartido  string `json:"siglaPartido"`
	SiglaUf       string `json:"siglaUf"`
	IDLegislatura int    `json:"idLegislatura"`
	URLFoto       string `json:"urlFoto"`
}

type Voto struct {
	TipoVoto string   `json:"tipoVoto"`
	Deputado Deputado `json:"deputado_"`
}

type Orientacao struct {
	OrientacaoVoto    string `json:"orientacaoVoto"`
	CodTipoLideranca  string `json:"codTipoLideranca"`
	SiglaPartidoBloco string `json:"siglaPartidoBloco"`
}

type HistoricoEvento struct {
	IDLegislatura     int    `json:"idLegislatura"`
	DataHora          string `json:"dataHora"`
	SiglaPartido      string `json:"siglaPartido"`
	Situacao          string `json:"situacao"`
	CondicaoEleitoral string `json:"condicaoEleitoral"`
}

type Evento struct {
	ID             int    `json:"id"`
	DataHoraInicio string `json:"dataHoraInicio"`
	DataHoraFim    string `json:"dataHoraFim"`
	Situacao       string `json:"situacao"`
	DescricaoTipo  string `json:"descricaoTipo"`
	Descricao      string `json:"descricao"`
}

type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

type envelope[T any] struct {
	Dados T      `json:"dados"`
	Links []link `json:"links"`
}

func (e envelope[T]) next() string {
	for _, l := range e.Links {
		if l.Rel == "next" {
			return l.Href
		}
	}
	return ""
}
