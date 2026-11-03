// Package domain reúne as regras de negócio puras (sem I/O): normalização de votos,
// alinhamento ao governo, vínculo votação→proposição e intervalos de exercício.
package domain

import "strings"

// TipoVoto espelha o enum tipo_voto_enum.
type TipoVoto string

const (
	VotoSim       TipoVoto = "sim"
	VotoNao       TipoVoto = "nao"
	VotoAbstencao TipoVoto = "abstencao"
	VotoObstrucao TipoVoto = "obstrucao"
	VotoArtigo17  TipoVoto = "artigo_17"
	VotoOutro     TipoVoto = "outro"
)

// MapVoto converte o texto bruto da API da Câmara no voto canônico (§4.2).
// Valores desconhecidos viram VotoOutro (o chamador deve logar).
func MapVoto(raw string) TipoVoto {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sim":
		return VotoSim
	case "não", "nao":
		return VotoNao
	case "abstenção", "abstencao":
		return VotoAbstencao
	case "obstrução", "obstrucao":
		return VotoObstrucao
	case "artigo 17", "art. 17", "artigo17":
		return VotoArtigo17
	default:
		return VotoOutro
	}
}

// Orientacao espelha o enum orientacao_enum.
type Orientacao string

const (
	OrientSim       Orientacao = "sim"
	OrientNao       Orientacao = "nao"
	OrientLiberado  Orientacao = "liberado"
	OrientObstrucao Orientacao = "obstrucao"
)

// MapOrientacao converte a orientação bruta. Vazio (sem orientação) ou desconhecido → nil.
func MapOrientacao(raw string) *Orientacao {
	var o Orientacao
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sim":
		o = OrientSim
	case "não", "nao":
		o = OrientNao
	case "liberado":
		o = OrientLiberado
	case "obstrução", "obstrucao":
		o = OrientObstrucao
	default:
		return nil
	}
	return &o
}

// IsGoverno diz se a bancada da orientação é a liderança do Governo.
func IsGoverno(bancada string) bool {
	return strings.EqualFold(strings.TrimSpace(bancada), "governo")
}

// Alinhou aplica a regra do §4.3. nil = não entra no cálculo (sem orientação,
// liberado, orientação de obstrução, ou voto sem posição como Artigo 17/outro).
func Alinhou(o *Orientacao, v TipoVoto) *bool {
	if o == nil || (*o != OrientSim && *o != OrientNao) {
		return nil
	}
	if v == VotoArtigo17 || v == VotoOutro {
		return nil
	}
	igual := (*o == OrientSim && v == VotoSim) || (*o == OrientNao && v == VotoNao)
	return &igual
}
