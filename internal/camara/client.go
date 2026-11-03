// Package camara é o cliente da API de Dados Abertos da Câmara dos Deputados.
package camara

import (
	"context"
	"fmt"
	"net/url"
	"time"

	_ "time/tzdata" // America/Sao_Paulo mesmo em imagens sem tzdata

	"comovotacongresso/internal/httpclient"
)

const (
	DefaultBaseURL = "https://dadosabertos.camara.leg.br/api/v2"
	// OrgaoPlenario é o idOrgao do Plenário.
	OrgaoPlenario  = 180
	itensPorPagina = 100
	maxPaginas     = 1000 // trava de segurança contra loop de paginação
)

type Client struct {
	http *httpclient.Client
	base string
}

func New(h *httpclient.Client, baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{http: h, base: baseURL}
}

// Votacoes lista as votações do órgão no intervalo [ini, fim] (datas), seguindo a paginação.
func (c *Client) Votacoes(ctx context.Context, ini, fim time.Time, idOrgao int) ([]VotacaoResumo, error) {
	q := url.Values{}
	q.Set("idOrgao", fmt.Sprint(idOrgao))
	q.Set("dataInicio", ini.Format("2006-01-02"))
	q.Set("dataFim", fim.Format("2006-01-02"))
	q.Set("itens", fmt.Sprint(itensPorPagina))
	q.Set("ordem", "ASC")
	q.Set("ordenarPor", "dataHoraRegistro")
	return paginado[VotacaoResumo](ctx, c, c.base+"/votacoes?"+q.Encode())
}

func (c *Client) Votacao(ctx context.Context, id string) (VotacaoDetalhe, error) {
	var env envelope[VotacaoDetalhe]
	err := c.http.GetJSON(ctx, fmt.Sprintf("%s/votacoes/%s", c.base, url.PathEscape(id)), &env)
	return env.Dados, err
}

func (c *Client) Votos(ctx context.Context, id string) ([]Voto, error) {
	var env envelope[[]Voto]
	err := c.http.GetJSON(ctx, fmt.Sprintf("%s/votacoes/%s/votos", c.base, url.PathEscape(id)), &env)
	return env.Dados, err
}

func (c *Client) Orientacoes(ctx context.Context, id string) ([]Orientacao, error) {
	var env envelope[[]Orientacao]
	err := c.http.GetJSON(ctx, fmt.Sprintf("%s/votacoes/%s/orientacoes", c.base, url.PathEscape(id)), &env)
	return env.Dados, err
}

// Deputados lista todos os deputados que atuaram na legislatura.
func (c *Client) Deputados(ctx context.Context, legislatura int) ([]Deputado, error) {
	q := url.Values{}
	q.Set("idLegislatura", fmt.Sprint(legislatura))
	q.Set("itens", fmt.Sprint(itensPorPagina))
	q.Set("ordem", "ASC")
	q.Set("ordenarPor", "id")
	return paginado[Deputado](ctx, c, c.base+"/deputados?"+q.Encode())
}

func (c *Client) Historico(ctx context.Context, idDeputado int) ([]HistoricoEvento, error) {
	var env envelope[[]HistoricoEvento]
	err := c.http.GetJSON(ctx, fmt.Sprintf("%s/deputados/%d/historico", c.base, idDeputado), &env)
	return env.Dados, err
}

func paginado[T any](ctx context.Context, c *Client, first string) ([]T, error) {
	var all []T
	next := first
	for i := 0; next != "" && i < maxPaginas; i++ {
		var env envelope[[]T]
		if err := c.http.GetJSON(ctx, next, &env); err != nil {
			return nil, err
		}
		all = append(all, env.Dados...)
		next = env.next()
	}
	return all, nil
}

var loc = mustLoad("America/Sao_Paulo")

func mustLoad(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

// ParseDataHora interpreta os timestamps sem fuso da API (horário de Brasília).
func ParseDataHora(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("data/hora inválida: %q", s)
}
