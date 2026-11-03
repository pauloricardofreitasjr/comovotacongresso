package camara

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"comovotacongresso/internal/httpclient"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../docs/discovery/samples/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func novoCliente(srv *httptest.Server) *Client {
	return New(httpclient.New(httpclient.Options{MaxRetries: 1}), srv.URL)
}

func TestVotos_DecodificaFixtureReal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/votacoes/2635476-36/votos" {
			t.Errorf("path inesperado: %s", r.URL.Path)
		}
		w.Write(fixture(t, "camara_nominal_2635476-36_votos.json"))
	}))
	defer srv.Close()

	votos, err := novoCliente(srv).Votos(t.Context(), "2635476-36")
	if err != nil {
		t.Fatal(err)
	}
	if len(votos) != 466 {
		t.Fatalf("votos=%d", len(votos))
	}
	v := votos[0]
	if v.TipoVoto == "" || v.Deputado.ID == 0 || v.Deputado.SiglaPartido == "" || v.Deputado.IDLegislatura != 57 {
		t.Fatalf("voto mal decodificado: %+v", v)
	}
}

func TestVotacao_Detalhe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture(t, "camara_nominal_2635476-36.json"))
	}))
	defer srv.Close()

	d, err := novoCliente(srv).Votacao(t.Context(), "2635476-36")
	if err != nil {
		t.Fatal(err)
	}
	if d.SiglaOrgao != "PLEN" || d.Aprovacao == nil || *d.Aprovacao != 1 ||
		len(d.ProposicoesAfetadas) != 1 || d.ProposicoesAfetadas[0].SiglaTipo != "PLP" || d.ProposicoesAfetadas[0].Numero != 186 {
		t.Fatalf("detalhe=%+v", d)
	}
}

func TestOrientacoes_Fixture(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture(t, "camara_nominal_2635476-36_orientacoes.json"))
	}))
	defer srv.Close()

	os, err := novoCliente(srv).Orientacoes(t.Context(), "2635476-36")
	if err != nil {
		t.Fatal(err)
	}
	var achouGoverno bool
	for _, o := range os {
		if o.SiglaPartidoBloco == "Governo" && o.CodTipoLideranca == "B" && o.OrientacaoVoto == "Sim" {
			achouGoverno = true
		}
	}
	if !achouGoverno {
		t.Fatalf("orientação do Governo não encontrada em %d itens", len(os))
	}
}

func TestVotacoes_SeguePaginacaoEEnviaFiltros(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("pagina") == "2" { // href absoluto do link "next"
			fmt.Fprint(w, `{"dados":[{"id":"p2-1"}],"links":[]}`)
			return
		}
		if q.Get("idOrgao") != "180" || q.Get("dataInicio") != "2026-10-01" || q.Get("dataFim") != "2026-10-31" || q.Get("itens") != "100" {
			t.Errorf("query inesperada: %s", r.URL.RawQuery)
		}
		fmt.Fprintf(w, `{"dados":[{"id":"p1-1"},{"id":"p1-2"}],"links":[{"rel":"next","href":"%s/votacoes?pagina=2"}]}`, srv.URL)
	}))
	defer srv.Close()

	ini, fim := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	got, err := novoCliente(srv).Votacoes(t.Context(), ini, fim, OrgaoPlenario)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, v := range got {
		ids = append(ids, v.ID)
	}
	if strings.Join(ids, ",") != "p1-1,p1-2,p2-1" {
		t.Fatalf("ids=%v", ids)
	}
}

func TestHistorico_Fixture(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/deputados/103758/historico" {
			t.Errorf("path=%s", r.URL.Path)
		}
		w.Write(fixture(t, "camara_dep_historico_suplente.json"))
	}))
	defer srv.Close()

	h, err := novoCliente(srv).Historico(t.Context(), 103758)
	if err != nil {
		t.Fatal(err)
	}
	var ex int
	for _, e := range h {
		if e.IDLegislatura == 57 && e.Situacao == "Exercício" {
			ex++
		}
	}
	if ex != 2 {
		t.Fatalf("eventos de exercício na 57ª = %d, esperado 2", ex)
	}
}

func TestParseDataHora(t *testing.T) {
	for _, in := range []string{"2026-10-14T19:42:50", "2026-10-14T19:42", "2026-10-14"} {
		got, err := ParseDataHora(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if _, off := got.Zone(); off != -3*3600 {
			t.Errorf("%q: offset=%d, esperado -03:00", in, off)
		}
	}
	if _, err := ParseDataHora("amanhã"); err == nil {
		t.Error("esperava erro")
	}
}
