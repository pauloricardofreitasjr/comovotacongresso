package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"comovotacongresso/internal/domain"
	"comovotacongresso/internal/store"
	"comovotacongresso/internal/testdb"
)

func utc(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04", s)
	if err != nil {
		panic(err)
	}
	return t
}
func ptr[T any](v T) *T { return &v }

type obj = map[string]any

// semear cria um dataset pequeno e conhecido:
//
//	Parlamentares: A "João Ávila" PT/SP (exercício aberto), B "Maria Souza" PT/RJ (suplente, encerrado),
//	               C "Carlos Lima" PSD/MG (dois períodos, aberto), D "Ana Costa" PL/BA (sem período).
//	P1 = PL 10/2025 "Altera regras do Imposto de Renda": V1 (nominal, 03-01), V2 (nominal, 03-02), V5 (simbólica, 03-02 13h)
//	P2 = PEC 5/2024 "Reforma da Educação básica":        V3 (nominal, vínculo provável, 03-03), V4 (nominal, 03-04)
//	Sessões S1..S4 (03-01..03-04).
func semear(t *testing.T) (*store.Store, *httptest.Server) {
	t.Helper()
	pool := testdb.Novo(t)
	s := store.New(pool)
	ctx := context.Background()

	ids, err := s.UpsertParlamentares(ctx, []store.ParlamentarIn{
		{IDExterno: 1, Nome: "João Ávila", Partido: "PT", UF: "SP", URLFoto: "http://f/a.jpg"},
		{IDExterno: 2, Nome: "Maria Souza", Partido: "PT", UF: "RJ"},
		{IDExterno: 3, Nome: "Carlos Lima", Partido: "PSD", UF: "MG"},
		{IDExterno: 4, Nome: "Ana Costa", Partido: "PL", UF: "BA"},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	ex := func(ext int, ivs ...domain.Intervalo) {
		if err := s.SubstituirExercicio(ctx, ids[ext], ivs); err != nil {
			t.Fatal(err)
		}
	}
	ex(1, domain.Intervalo{Inicio: utc("2025-01-01 00:00"), Condicao: "Titular", Partido: "PT"})
	ex(2, domain.Intervalo{Inicio: utc("2025-03-02 00:00"), Fim: ptr(utc("2025-03-03 23:00")), Condicao: "Suplente", Partido: "PT"})
	ex(3, domain.Intervalo{Inicio: utc("2025-01-01 00:00"), Fim: ptr(utc("2025-03-02 00:00")), Partido: "PSD"},
		domain.Intervalo{Inicio: utc("2025-03-04 00:00"), Partido: "PSD"})

	p1 := &domain.Prop{ID: 100, SiglaTipo: "PL", Numero: 10, Ano: 2025, Ementa: "Altera regras do Imposto de Renda"}
	p2 := &domain.Prop{ID: 200, SiglaTipo: "PEC", Numero: 5, Ano: 2024, Ementa: "Reforma da Educação básica"}
	sim, nao := domain.OrientSim, domain.OrientNao
	voto := func(ext int, nome string, v domain.TipoVoto, partido string, al *bool) store.VotoIn {
		return store.VotoIn{Parlamentar: store.ParlamentarIn{IDExterno: ext, Nome: nome}, Voto: v, Partido: partido, AlinhouGoverno: al}
	}
	salvar := func(v store.VotacaoIn) {
		if err := s.SalvarVotacao(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	salvar(store.VotacaoIn{ID: "V1", DataHora: utc("2025-03-01 12:00"), Descricao: "V1", Tipo: domain.VotacaoNominal, Aprovada: ptr(true),
		Proposicao: p1, Vinculo: domain.ConfDireto, OrientacaoGoverno: &sim,
		Votos: []store.VotoIn{voto(1, "João Ávila", domain.VotoSim, "PT", ptr(true)), voto(4, "Ana Costa", domain.VotoSim, "PL", ptr(true))}})
	salvar(store.VotacaoIn{ID: "V2", DataHora: utc("2025-03-02 12:00"), Descricao: "V2", Tipo: domain.VotacaoNominal, Aprovada: ptr(false),
		Proposicao: p1, Vinculo: domain.ConfDireto, OrientacaoGoverno: &sim,
		Orientacoes: []store.OrientacaoIn{
			{Bancada: "PL", TipoLideranca: "P", Orientacao: &nao},
			{Bancada: "Governo", TipoLideranca: "B", Orientacao: &sim},
			{Bancada: "PSD", TipoLideranca: "P"}, // sem orientação
		},
		Votos: []store.VotoIn{voto(2, "Maria Souza", domain.VotoSim, "PT", ptr(true)), voto(1, "João Ávila", domain.VotoNao, "PT", ptr(false))}})
	salvar(store.VotacaoIn{ID: "V5", DataHora: utc("2025-03-02 13:00"), Descricao: "V5 simbólica", Tipo: domain.VotacaoSimbolica,
		Aprovada: ptr(true), Proposicao: p1, Vinculo: domain.ConfDireto})
	salvar(store.VotacaoIn{ID: "V3", DataHora: utc("2025-03-03 12:00"), Descricao: "V3", Tipo: domain.VotacaoNominal, Aprovada: ptr(true),
		Proposicao: p2, Vinculo: domain.ConfProvavel,
		Votos: []store.VotoIn{voto(2, "Maria Souza", domain.VotoSim, "PT", nil)}})
	salvar(store.VotacaoIn{ID: "V4", DataHora: utc("2025-03-04 12:00"), Descricao: "V4", Tipo: domain.VotacaoNominal, Aprovada: ptr(true),
		Proposicao: p2, Vinculo: domain.ConfDireto,
		Votos: []store.VotoIn{voto(3, "Carlos Lima", domain.VotoSim, "PSD", nil)}})

	pr := func(ext int, partido string) store.ParlamentarIn {
		return store.ParlamentarIn{IDExterno: ext, Nome: "x", Partido: partido}
	}
	sessao := func(id int, dia string, ps ...store.ParlamentarIn) {
		if err := s.SalvarSessao(ctx, store.SessaoIn{IDExterno: id, Inicio: utc(dia + " 14:00"), Tipo: "Sessão Deliberativa", Presentes: ps}); err != nil {
			t.Fatal(err)
		}
	}
	sessao(1, "2025-03-01", pr(1, "PT"), pr(4, "PL"))
	sessao(2, "2025-03-02", pr(1, "PT"), pr(2, "PT"))
	sessao(3, "2025-03-03", pr(1, "PT"))
	sessao(4, "2025-03-04")

	if err := s.AtualizarMetricas(ctx); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(Router(s, pool, slog.New(slog.NewTextHandler(io.Discard, nil))))
	t.Cleanup(srv.Close)
	return s, srv
}

func get(t *testing.T, srv *httptest.Server, path string) (int, obj, http.Header) {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var o obj
	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		t.Fatalf("%s: corpo não é JSON: %v", path, err)
	}
	return resp.StatusCode, o, resp.Header
}

func lista(o obj, chave string) []obj {
	var out []obj
	for _, x := range o[chave].([]any) {
		out = append(out, x.(obj))
	}
	return out
}

func nomes(itens []obj) string {
	var ns []string
	for _, i := range itens {
		ns = append(ns, i["nome"].(string))
	}
	return strings.Join(ns, ",")
}

func TestStatus(t *testing.T) {
	_, srv := semear(t)
	code, o, _ := get(t, srv, "/api/v1/status")
	if code != 200 || o["votacoes_nominais"].(float64) != 4 || o["votacoes_simbolicas"].(float64) != 1 ||
		o["sessoes_deliberativas"].(float64) != 4 || o["parlamentares"].(float64) != 4 || o["proposicoes"].(float64) != 2 || o["atualizado_em"] == nil {
		t.Fatalf("status=%d %v", code, o)
	}
	m, ok := o["medias_camara"].(map[string]any)
	if !ok {
		t.Fatalf("medias_camara ausente: %v", o)
	}
	for _, k := range []string{"presenca_sessao", "participacao_votacoes_nominais", "alinhamento_governo"} {
		if v, ok := m[k].(float64); !ok || v < 0 || v > 100 {
			t.Errorf("medias_camara.%s=%v", k, m[k])
		}
	}
}

func TestProposicoes_ListaFiltrosEPaginacao(t *testing.T) {
	_, srv := semear(t)
	ids := func(path string) (string, float64) {
		code, o, _ := get(t, srv, path)
		if code != 200 {
			t.Fatalf("%s: status %d %v", path, code, o)
		}
		var is []string
		for _, p := range lista(o, "dados") {
			is = append(is, p["identificacao"].(string))
		}
		return strings.Join(is, ";"), o["total"].(float64)
	}
	if got, total := ids("/api/v1/proposicoes"); got != "PEC 5/2024;PL 10/2025" || total != 2 { // mais recente primeiro
		t.Errorf("padrão: %q total=%v", got, total)
	}
	if got, _ := ids("/api/v1/proposicoes?tipo=pl"); got != "PL 10/2025" {
		t.Errorf("tipo=pl: %q", got)
	}
	if got, _ := ids("/api/v1/proposicoes?ano=2024"); got != "PEC 5/2024" {
		t.Errorf("ano: %q", got)
	}
	if got, _ := ids("/api/v1/proposicoes?busca=IMPOSTO%20de%20renda"); got != "PL 10/2025" {
		t.Errorf("busca: %q", got)
	}
	if got, _ := ids("/api/v1/proposicoes?busca=educacao"); got != "PEC 5/2024" { // sem acento
		t.Errorf("busca sem acento: %q", got)
	}
	if got, total := ids("/api/v1/proposicoes?busca=%25"); got != "" || total != 0 { // % não é curinga
		t.Errorf("busca com %%: %q total=%v", got, total)
	}
	// so_nominais: no dataset as duas têm votação nominal (a cobertura do filtro com lei só simbólica está no store)
	if got, total := ids("/api/v1/proposicoes?so_nominais=true"); got != "PEC 5/2024;PL 10/2025" || total != 2 {
		t.Errorf("so_nominais: %q total=%v", got, total)
	}
	if code, _, _ := get(t, srv, "/api/v1/proposicoes?so_nominais=talvez"); code != 400 {
		t.Errorf("so_nominais inválido deveria dar 400, deu %d", code)
	}
	// última nominal: resultado e placar só nas que têm votação nominal
	_, o, _ := get(t, srv, "/api/v1/proposicoes")
	for _, p := range lista(o, "dados") {
		un, tem := p["ultima_nominal"].(map[string]any)
		if p["votacoes_nominais"].(float64) > 0 && (!tem || un["votacao_id"] == "" || un["sim"] == nil || un["nao"] == nil) {
			t.Errorf("%v deveria ter ultima_nominal: %v", p["identificacao"], p)
		}
		if p["votacoes_nominais"].(float64) == 0 && tem {
			t.Errorf("%v só tem simbólica e não deveria ter ultima_nominal", p["identificacao"])
		}
	}
	// paginação: página 2 com limite 1 → segunda proposição; total preservado
	code, o, _ := get(t, srv, "/api/v1/proposicoes?limite=1&pagina=2")
	if code != 200 || o["total"].(float64) != 2 || len(lista(o, "dados")) != 1 || lista(o, "dados")[0]["identificacao"] != "PL 10/2025" {
		t.Errorf("paginação: %v", o)
	}
	// página além do fim: lista vazia (não nula) e o total continua correto
	if _, o, _ := get(t, srv, "/api/v1/proposicoes?pagina=9"); len(lista(o, "dados")) != 0 || o["total"].(float64) != 2 {
		t.Errorf("além do fim: %v", o)
	}
	if _, o, _ := get(t, srv, "/api/v1/proposicoes?tipo=PL&pagina=9"); o["total"].(float64) != 1 {
		t.Errorf("além do fim com filtro: total=%v, esperado 1", o["total"])
	}
	// contagens do item
	_, o, _ = get(t, srv, "/api/v1/proposicoes?tipo=PL")
	if p := lista(o, "dados")[0]; p["total_votacoes"].(float64) != 3 || p["votacoes_nominais"].(float64) != 2 {
		t.Errorf("contagens: %v", p)
	}
}

func TestProposicao_LinhaDoTempoEPrincipal(t *testing.T) {
	_, srv := semear(t)
	_, l, _ := get(t, srv, "/api/v1/proposicoes?tipo=PL")
	idPL := itoa(int(lista(l, "dados")[0]["id"].(float64))) // id interno, não o da Câmara
	code, o, _ := get(t, srv, "/api/v1/proposicoes/"+idPL)
	if code != 200 || o["identificacao"] != "PL 10/2025" {
		t.Fatalf("status=%d %v", code, o)
	}
	vs := lista(o, "votacoes")
	var ordem []string
	principal := ""
	for _, v := range vs {
		ordem = append(ordem, v["id"].(string))
		if v["principal"] == true {
			principal += v["id"].(string)
		}
	}
	if strings.Join(ordem, ",") != "V1,V2,V5" {
		t.Errorf("ordem cronológica: %v", ordem)
	}
	if principal != "V2" { // nominais diretas V1 e V2 empatam em 2 votos → a mais recente
		t.Errorf("principal=%q", principal)
	}
	if vs[2]["resumo_votos"] != nil || vs[2]["tipo"] != "simbolica" {
		t.Errorf("simbólica não tem resumo de votos: %v", vs[2])
	}
	if rv := vs[0]["resumo_votos"].(obj); rv["sim"].(float64) != 2 || rv["total"].(float64) != 2 {
		t.Errorf("resumo V1: %v", rv)
	}
	if code, _, _ := get(t, srv, "/api/v1/proposicoes/99999"); code != 404 {
		t.Errorf("inexistente: %d", code)
	}
	if code, o, _ := get(t, srv, "/api/v1/proposicoes/abc"); code != 400 || o["erro"].(obj)["codigo"] != "parametro_invalido" {
		t.Errorf("id inválido: %d %v", code, o)
	}
}

func TestVotacao_NominalESimbolica(t *testing.T) {
	_, srv := semear(t)
	code, o, _ := get(t, srv, "/api/v1/votacoes/V2")
	if code != 200 || o["tipo"] != "nominal" || o["votos_individuais"] != true || o["orientacao_governo"] != "sim" {
		t.Fatalf("V2: %d %v", code, o)
	}
	if p := o["proposicao"].(obj); p["identificacao"] != "PL 10/2025" {
		t.Errorf("proposição: %v", p)
	}
	votos := lista(o, "votos")
	if nomes(votos) != "João Ávila,Maria Souza" { // ordem por nome
		t.Errorf("votos: %s", nomes(votos))
	}
	if votos[0]["voto"] != "nao" || votos[0]["alinhou_governo"] != false || votos[0]["partido"] != "PT" {
		t.Errorf("voto de João: %v", votos[0])
	}
	// orientações: blocos/lideranças (B) primeiro; sem orientação = null
	ors := lista(o, "orientacoes")
	if len(ors) != 3 || ors[0]["bancada"] != "Governo" || ors[0]["orientacao"] != "sim" {
		t.Errorf("orientações: %v", ors)
	}
	var psd obj
	for _, x := range ors {
		if x["bancada"] == "PSD" {
			psd = x
		}
	}
	if psd == nil || psd["orientacao"] != nil {
		t.Errorf("PSD deveria estar sem orientação: %v", psd)
	}

	// simbólica: sem votos individuais
	code, o, _ = get(t, srv, "/api/v1/votacoes/V5")
	if code != 200 || o["tipo"] != "simbolica" || o["votos_individuais"] != false || o["votos"] != nil || o["resumo_votos"] != nil {
		t.Errorf("V5: %d %v", code, o)
	}
	// vínculo provável é exposto
	if _, o, _ = get(t, srv, "/api/v1/votacoes/V3"); o["vinculo_confianca"] != "provavel" {
		t.Errorf("V3 vínculo: %v", o["vinculo_confianca"])
	}
	if code, _, _ := get(t, srv, "/api/v1/votacoes/naoexiste"); code != 404 {
		t.Errorf("inexistente: %d", code)
	}
}

func TestParlamentares_FiltrosEOrdenacao(t *testing.T) {
	_, srv := semear(t)
	lst := func(q string) string {
		code, o, _ := get(t, srv, "/api/v1/parlamentares?"+q)
		if code != 200 {
			t.Fatalf("%s: %d %v", q, code, o)
		}
		return nomes(lista(o, "dados"))
	}
	if got := lst(""); got != "Carlos Lima,João Ávila" { // padrão: só em exercício, por nome
		t.Errorf("padrão: %q", got)
	}
	if got := lst("em_exercicio=todos"); got != "Ana Costa,Carlos Lima,João Ávila,Maria Souza" {
		t.Errorf("todos: %q", got)
	}
	if got := lst("em_exercicio=false"); got != "Ana Costa,Maria Souza" {
		t.Errorf("false: %q", got)
	}
	// governismo desc: B e D (100%) desempatam por nome, depois A (50%), C sem dado por último
	if got := lst("em_exercicio=todos&ordenarPor=governismo"); got != "Ana Costa,Maria Souza,João Ávila,Carlos Lima" {
		t.Errorf("governismo desc: %q", got)
	}
	if got := lst("em_exercicio=todos&ordenarPor=governismo&ordem=asc"); got != "João Ávila,Ana Costa,Maria Souza,Carlos Lima" {
		t.Errorf("governismo asc (nulos continuam no fim): %q", got)
	}
	if got := lst("em_exercicio=todos&ordenarPor=governismo&min_votacoes=2"); got != "João Ávila" {
		t.Errorf("min_votacoes: %q", got)
	}
	if got := lst("em_exercicio=todos&ordenarPor=presenca_sessao"); got != "Ana Costa,João Ávila,Maria Souza,Carlos Lima" { // 100, 75, 50, 0
		t.Errorf("presenca_sessao: %q", got)
	}
	if got := lst("em_exercicio=todos&ordenarPor=participacao&ordem=asc"); got != "Carlos Lima,João Ávila,Ana Costa,Maria Souza" { // 50, 50, 100, 100: empates por nome
		t.Errorf("participacao asc: %q", got)
	}
	if got := lst("busca=joao&em_exercicio=todos"); got != "João Ávila" { // sem acento
		t.Errorf("busca: %q", got)
	}
	if got := lst("em_exercicio=todos&partido=pt"); got != "João Ávila,Maria Souza" {
		t.Errorf("partido: %q", got)
	}
	if got := lst("em_exercicio=todos&uf=ba"); got != "Ana Costa" {
		t.Errorf("uf: %q", got)
	}
	// além da última página: vazio, mas o total (com os mesmos filtros) é preservado
	if _, o, _ := get(t, srv, "/api/v1/parlamentares?em_exercicio=todos&pagina=9"); len(lista(o, "dados")) != 0 || o["total"].(float64) != 4 {
		t.Errorf("além do fim: %v", o)
	}
	if _, o, _ := get(t, srv, "/api/v1/parlamentares?partido=pt&em_exercicio=todos&pagina=9"); o["total"].(float64) != 2 {
		t.Errorf("além do fim com filtro: total=%v, esperado 2", o["total"])
	}
	for _, q := range []string{"ordenarPor=xyz", "ordem=cima", "em_exercicio=talvez", "casa=marte", "limite=0", "limite=101", "pagina=0", "min_votacoes=-1"} {
		if code, o, _ := get(t, srv, "/api/v1/parlamentares?"+q); code != 400 || o["erro"] == nil {
			t.Errorf("%s deveria dar 400, deu %d", q, code)
		}
	}
	// injeção via ordenarPor não passa da whitelist
	if code, _, _ := get(t, srv, "/api/v1/parlamentares?ordenarPor=nome;DROP%20TABLE%20votos"); code != 400 {
		t.Errorf("injeção: %d", code)
	}
	if code, _, _ := get(t, srv, "/api/v1/parlamentares?ordenarPor=nome%3BDROP%20TABLE%20votos"); code != 400 {
		t.Errorf("injeção escapada: %d", code)
	}
}

func TestParlamentar_PerfilMetricasEHistorico(t *testing.T) {
	_, srv := semear(t)
	// João Ávila (A) é o id interno do primeiro inserido; descobre pela busca.
	_, o, _ := get(t, srv, "/api/v1/parlamentares?busca=joao")
	id := int(lista(o, "dados")[0]["id"].(float64))
	path := "/api/v1/parlamentares/" + itoa(id)

	code, o, _ := get(t, srv, path)
	if code != 200 || o["nome"] != "João Ávila" || o["partido"] != "PT" || o["em_exercicio"] != true {
		t.Fatalf("perfil: %d %v", code, o)
	}
	m := o["metricas"].(obj)
	part, pres, gov := m["participacao_votacoes_nominais"].(obj), m["presenca_sessao"].(obj), m["alinhamento_governo"].(obj)
	// A: nominais elegíveis V1..V4=4, votou 2 (50%); sessões 4, presente 3 (75%); governo base 2, alinhadas 1 (50%)
	if part["votacoes_elegiveis"].(float64) != 4 || part["votacoes_votou"].(float64) != 2 || part["pct"].(float64) != 50 {
		t.Errorf("participação: %v", part)
	}
	if pres["sessoes_elegiveis"].(float64) != 4 || pres["sessoes_presente"].(float64) != 3 || pres["pct"].(float64) != 75 {
		t.Errorf("presença: %v", pres)
	}
	if gov["votacoes_com_orientacao"].(float64) != 2 || gov["votacoes_alinhadas"].(float64) != 1 || gov["pct"].(float64) != 50 {
		t.Errorf("governo: %v", gov)
	}
	if ex := lista(o, "exercicio"); len(ex) != 1 || ex[0]["fim"] != nil || ex[0]["condicao"] != "Titular" {
		t.Errorf("exercício: %v", ex)
	}

	// histórico: mais recente primeiro, paginado
	votos := o["votos"].(obj)
	hist := lista(votos, "dados")
	if votos["total"].(float64) != 2 || len(hist) != 2 || hist[0]["votacao_id"] != "V2" || hist[1]["votacao_id"] != "V1" {
		t.Errorf("histórico: %v", votos)
	}
	if hist[0]["voto"] != "nao" || hist[0]["alinhou_governo"] != false || hist[0]["proposicao"].(obj)["identificacao"] != "PL 10/2025" {
		t.Errorf("voto V2: %v", hist[0])
	}
	_, o, _ = get(t, srv, path+"?page_votos=2&limite_votos=1")
	votos = o["votos"].(obj)
	if h := lista(votos, "dados"); len(h) != 1 || h[0]["votacao_id"] != "V1" || votos["total"].(float64) != 2 || votos["pagina"].(float64) != 2 {
		t.Errorf("página 2: %v", votos)
	}
	_, o, _ = get(t, srv, path+"?page_votos=9")
	votos = o["votos"].(obj)
	if len(lista(votos, "dados")) != 0 || votos["total"].(float64) != 2 {
		t.Errorf("além do fim deve manter o total: %v", votos)
	}

	// filtros do histórico: V1 = Sim alinhado, V2 = Não não alinhado
	ids := func(q string) string {
		code, o, _ := get(t, srv, path+q)
		if code != 200 {
			t.Fatalf("%s: %d %v", q, code, o)
		}
		var is []string
		for _, h := range lista(o["votos"].(obj), "dados") {
			is = append(is, h["votacao_id"].(string))
		}
		return strings.Join(is, ";") + "|" + itoa(int(o["votos"].(obj)["total"].(float64)))
	}
	for q, want := range map[string]string{
		"?alinhamento=nao_alinhado":              "V2|1",
		"?alinhamento=alinhado":                  "V1|1",
		"?voto=sim":                              "V1|1",
		"?voto=nao":                              "V2|1",
		"?voto=sim&alinhamento=nao_alinhado":     "|0", // combinados: nenhum
		"?voto=abstencao":                        "|0",
		"?alinhamento=nao_alinhado&page_votos=9": "|1", // além do fim: total do filtro, não do histórico todo
	} {
		if got := ids(q); got != want {
			t.Errorf("%s: got %q, esperado %q", q, got, want)
		}
	}
	// as métricas do perfil não mudam com os filtros
	_, of, _ := get(t, srv, path+"?voto=nao")
	if of["metricas"].(obj)["participacao_votacoes_nominais"].(obj)["votacoes_votou"].(float64) != 2 {
		t.Errorf("filtro não deve alterar as métricas: %v", of["metricas"])
	}
	for _, q := range []string{"?voto=talvez", "?alinhamento=metade"} {
		if code, _, _ := get(t, srv, path+q); code != 400 {
			t.Errorf("%s deveria dar 400, deu %d", q, code)
		}
	}

	if code, _, _ := get(t, srv, "/api/v1/parlamentares/99999"); code != 404 {
		t.Errorf("inexistente: %d", code)
	}
	if code, _, _ := get(t, srv, "/api/v1/parlamentares/0"); code != 400 {
		t.Errorf("id 0: %d", code)
	}
	if code, _, _ := get(t, srv, path+"?limite_votos=500"); code != 400 {
		t.Errorf("limite_votos alto: %d", code)
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestPartidos(t *testing.T) {
	_, srv := semear(t)
	code, o, _ := get(t, srv, "/api/v1/partidos?casa=camara")
	if code != 200 {
		t.Fatalf("%d %v", code, o)
	}
	var pt obj
	for _, p := range lista(o, "dados") {
		if p["partido"] == "PT" {
			pt = p
		}
	}
	if pt == nil || pt["parlamentares"].(float64) != 2 {
		t.Fatalf("PT: %v", pt)
	}
	g := pt["metricas"].(obj)["alinhamento_governo"].(obj)
	if g["pct"].(float64) < 66.66 || g["pct"].(float64) > 66.68 { // 2 alinhadas / 3 com orientação
		t.Errorf("governismo PT: %v", g)
	}
	if code, _, _ := get(t, srv, "/api/v1/partidos?casa=x"); code != 400 {
		t.Errorf("casa inválida: %d", code)
	}
}

func TestCabecalhosEErros(t *testing.T) {
	_, srv := semear(t)
	_, _, h := get(t, srv, "/api/v1/status")
	if !strings.Contains(h.Get("Cache-Control"), "max-age=300") || h.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("cabeçalhos: %v", h)
	}
	// erro não é cacheável
	if _, _, h := get(t, srv, "/api/v1/proposicoes/99999"); h.Get("Cache-Control") != "no-store" {
		t.Errorf("erro cacheável: %q", h.Get("Cache-Control"))
	}
	// preflight
	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/v1/status", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 204 || !strings.Contains(resp.Header.Get("Access-Control-Allow-Methods"), "GET") {
		t.Errorf("OPTIONS: %v %v", resp, err)
	}
	// método não permitido e rota inexistente respondem JSON
	resp, _ = http.Post(srv.URL+"/api/v1/status", "application/json", strings.NewReader("{}"))
	var e obj
	json.NewDecoder(resp.Body).Decode(&e)
	if resp.StatusCode != 405 || e["erro"] == nil {
		t.Errorf("POST: %d %v", resp.StatusCode, e)
	}
	if code, o, _ := get(t, srv, "/api/v1/nada"); code != 404 || o["erro"].(obj)["codigo"] != "rota_inexistente" {
		t.Errorf("rota inexistente: %d %v", code, o)
	}
}

func TestHealthz(t *testing.T) {
	for _, c := range []struct {
		err  error
		want int
	}{{nil, 200}, {errors.New("x"), 503}} {
		rec := httptest.NewRecorder()
		Router(nil, pingFake{c.err}, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if rec.Code != c.want {
			t.Errorf("healthz err=%v: %d", c.err, rec.Code)
		}
	}
}

type pingFake struct{ err error }

func (p pingFake) Ping(context.Context) error { return p.err }

// A especificação OpenAPI tem de descrever exatamente as rotas registradas.
func TestOpenAPI_ConformeAsRotas(t *testing.T) {
	spec, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^  (/api/v1[^:]*):\s*$`)
	var doc []string
	for _, m := range re.FindAllStringSubmatch(string(spec), -1) {
		doc = append(doc, m[1])
	}
	var rotas []string
	err = chi.Walk(Router(nil, nil, nil).(chi.Routes), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(route, "/api/v1") && method == http.MethodGet {
			rotas = append(rotas, route)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(doc)
	sort.Strings(rotas)
	if strings.Join(doc, "\n") != strings.Join(rotas, "\n") {
		t.Fatalf("OpenAPI diverge das rotas.\nspec:  %v\nrotas: %v", doc, rotas)
	}
	// e a própria spec é servida
	rec := httptest.NewRecorder()
	Router(nil, nil, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "openapi: 3.0.3") {
		t.Errorf("openapi.yaml: %d", rec.Code)
	}
}

func TestIndicePrincipal(t *testing.T) {
	nom := func(id string, total int, vinc string) store.VotacaoLinha {
		return store.VotacaoLinha{ID: id, Tipo: "nominal", Vinculo: &vinc, Resumo: &store.ResumoVotos{Total: total}}
	}
	simb := func(id, vinc string) store.VotacaoLinha {
		return store.VotacaoLinha{ID: id, Tipo: "simbolica", Vinculo: &vinc}
	}
	cases := []struct {
		nome string
		vs   []store.VotacaoLinha
		want int
	}{
		{"vazio", nil, -1},
		{"mais votos vence", []store.VotacaoLinha{nom("a", 400, "direto"), nom("b", 450, "direto"), nom("c", 100, "direto")}, 1},
		{"empate fica com a mais recente", []store.VotacaoLinha{nom("a", 400, "direto"), nom("b", 400, "direto")}, 1},
		{"ignora provável", []store.VotacaoLinha{nom("a", 100, "direto"), nom("b", 500, "provavel")}, 0},
		{"sem nominal: direta mais recente", []store.VotacaoLinha{simb("a", "direto"), simb("b", "direto"), simb("c", "provavel")}, 1},
		{"só prováveis: a mais recente", []store.VotacaoLinha{simb("a", "provavel"), simb("b", "provavel")}, 1},
	}
	for _, c := range cases {
		if got := indicePrincipal(c.vs); got != c.want {
			t.Errorf("%s: got %d want %d", c.nome, got, c.want)
		}
	}
}
