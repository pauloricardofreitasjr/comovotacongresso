package domain

// TipoVotacao espelha o enum tipo_votacao_enum.
type TipoVotacao string

const (
	VotacaoNominal   TipoVotacao = "nominal"
	VotacaoSimbolica TipoVotacao = "simbolica"
)

// ClassificaTipo implementa o §4.1. ok=false quando não dá para concluir
// (sem votos e sem resultado: votação em andamento) — o chamador deve adiar.
func ClassificaTipo(nVotos int, aprovacao *int) (TipoVotacao, bool) {
	if nVotos > 0 {
		return VotacaoNominal, true
	}
	if aprovacao != nil {
		return VotacaoSimbolica, true
	}
	return "", false
}

// Prop é a identificação mínima de uma proposição.
type Prop struct {
	ID        int
	SiglaTipo string
	Numero    int
	Ano       int
	Ementa    string
}

// Confianca espelha vinculo_confianca_enum.
type Confianca string

const (
	ConfDireto   Confianca = "direto"
	ConfProvavel Confianca = "provavel"
)

// EscolheProposicao implementa o algoritmo de vínculo (discovery §3.2):
//  1. proposicoesAfetadas sem REQ com exatamente uma proposição distinta → direto
//  2. senão, objetosPossiveis com exatamente um item → provável
//  3. senão nil.
func EscolheProposicao(afetadas, possiveis []Prop) (*Prop, Confianca) {
	if p := unica(semREQ(afetadas)); p != nil {
		return p, ConfDireto
	}
	if p := unica(possiveis); p != nil {
		return p, ConfProvavel
	}
	return nil, ""
}

func semREQ(ps []Prop) []Prop {
	var out []Prop
	for _, p := range ps {
		if p.SiglaTipo != "REQ" {
			out = append(out, p)
		}
	}
	return out
}

func unica(ps []Prop) *Prop {
	if len(ps) == 0 {
		return nil
	}
	first := ps[0]
	for _, p := range ps[1:] {
		if p.ID != first.ID {
			return nil
		}
	}
	return &first
}
