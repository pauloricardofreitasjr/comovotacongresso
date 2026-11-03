package etl

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"comovotacongresso/internal/camara"
	"comovotacongresso/internal/httpclient"
	"comovotacongresso/internal/store"
	"comovotacongresso/internal/testdb"
)

const (
	idNominal   = "2635476-36" // PLP 186/2026, 466 votos, Governo orienta Sim
	idSimbolica = "2620905-53" // sem votos, aprovada
	idPendente  = "9999999-1"  // sem votos e sem resultado
)

func amostra(t *testing.T, nome string) string {
	t.Helper()
	b, err := os.ReadFile("../../docs/discovery/samples/" + nome)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// servidor imita a API da Câmara com as fixtures reais.
func servidor(t *testing.T) *httptest.Server {
	t.Helper()
	respostas := map[string]string{
		"/votacoes/" + idNominal:                    amostra(t, "camara_nominal_2635476-36.json"),
		"/votacoes/" + idNominal + "/votos":         amostra(t, "camara_nominal_2635476-36_votos.json"),
		"/votacoes/" + idNominal + "/orientacoes":   amostra(t, "camara_nominal_2635476-36_orientacoes.json"),
		"/votacoes/" + idSimbolica:                  amostra(t, "camara_v_2620905-53.json"),
		"/votacoes/" + idSimbolica + "/votos":       amostra(t, "camara_v_2620905-53_votos.json"),
		"/votacoes/" + idSimbolica + "/orientacoes": amostra(t, "camara_v_2620905-53_orientacoes.json"),
		"/votacoes/" + idPendente:                   `{"dados":{"id":"9999999-1","dataHoraRegistro":"2026-10-30T10:00:00","siglaOrgao":"PLEN","descricao":"em andamento","aprovacao":null},"links":[]}`,
		"/votacoes/" + idPendente + "/votos":        `{"dados":[],"links":[]}`,
		"/votacoes/" + idPendente + "/orientacoes":  `{"dados":[],"links":[]}`,
		"/eventos/83057/deputados":                  amostra(t, "camara_evento_83057_deputados.json"),
		"/eventos/99001/deputados":                  `{"dados":[],"links":[]}`,
		"/deputados/178843/historico":               amostra(t, "camara_dep_historico.json"),
		"/deputados/103758/historico":               amostra(t, "camara_dep_historico_suplente.json"),
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/votacoes":
			fmt.Fprintf(w, `{"dados":[
				{"id":%q,"dataHoraRegistro":"2026-10-14T19:42:50","descricao":"x","aprovacao":1},
				{"id":%q,"dataHoraRegistro":"2026-10-28T16:52:36","descricao":"y","aprovacao":1},
				{"id":%q,"dataHoraRegistro":"2026-10-30T10:00:00","descricao":"z","aprovacao":null}],"links":[]}`,
				idNominal, idSimbolica, idPendente)
		case "/eventos":
			fmt.Fprint(w, `{"dados":[
				{"id":83057,"dataHoraInicio":"2026-10-14T14:23","dataHoraFim":"2026-10-14T20:55","situacao":"Encerrada","descricaoTipo":"Sessão Deliberativa","descricao":"Sessão Deliberativa Extraordinária\r\n"},
				{"id":83060,"dataHoraInicio":"2026-10-14T10:00","dataHoraFim":"2026-10-14T11:00","situacao":"Encerrada","descricaoTipo":"Sessão Não Deliberativa Solene","descricao":"Solene"},
				{"id":83070,"dataHoraInicio":"2026-10-16T14:00","dataHoraFim":"","situacao":"Em Andamento","descricaoTipo":"Sessão Deliberativa","descricao":"em andamento"},
				{"id":99001,"dataHoraInicio":"2026-10-15T14:00","dataHoraFim":"2026-10-15T16:00","situacao":"Encerrada","descricaoTipo":"Sessão Deliberativa","descricao":"sem lista"}],"links":[]}`)
		case "/deputados":
			fmt.Fprint(w, `{"dados":[
				{"id":178843,"nome":"Marx Beltrão","siglaPartido":"TESTE","siglaUf":"AL","idLegislatura":57,"urlFoto":"http://f/1.jpg"},
				{"id":178843,"nome":"Marx Beltrão","siglaPartido":"OUTRO","siglaUf":"AL","idLegislatura":57,"urlFoto":"http://f/1.jpg"},
				{"id":103758,"nome":"Suplente Teste","siglaPartido":"XYZ","siglaUf":"SP","idLegislatura":57,"urlFoto":""}],"links":[]}`)
		default:
			body, ok := respostas[r.URL.Path]
			if !ok {
				http.NotFound(w, r)
				return
			}
			fmt.Fprint(w, body)
		}
	}))
}

func novoETL(t *testing.T, pool *pgxpool.Pool, srv *httptest.Server) *ETL {
	t.Helper()
	cli := camara.New(httpclient.New(httpclient.Options{MaxRetries: 1}), srv.URL)
	return &ETL{
		Fonte: cli, Store: store.New(pool), Workers: 2,
		Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}
}

func contar(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

var janela = func() (time.Time, time.Time) {
	return time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
}

func TestIngerirVotacoes_FimAFim(t *testing.T) {
	pool := testdb.Novo(t)
	srv := servidor(t)
	defer srv.Close()
	e := novoETL(t, pool, srv)
	ctx := context.Background()
	ini, fim := janela()

	res, err := e.IngerirVotacoes(ctx, ini, fim, false)
	if err != nil {
		t.Fatal(err)
	}
	want := Resumo{Total: 3, Processadas: 2, Nominais: 1, Simbolicas: 1, Adiadas: 1}
	if res != want {
		t.Fatalf("resumo=%+v, esperado %+v", res, want)
	}

	// classificação e persistência
	if n := contar(t, pool, `SELECT count(*) FROM votacoes`); n != 2 {
		t.Errorf("votacoes=%d (a pendente não deve ser gravada)", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM votacoes WHERE id=$1 AND tipo='nominal' AND aprovada`, idNominal); n != 1 {
		t.Error("votação nominal mal gravada")
	}
	if n := contar(t, pool, `SELECT count(*) FROM votacoes WHERE id=$1 AND tipo='simbolica'`, idSimbolica); n != 1 {
		t.Error("votação simbólica mal gravada")
	}
	if n := contar(t, pool, `SELECT count(*) FROM votos WHERE votacao_id=$1`, idSimbolica); n != 0 {
		t.Errorf("simbólica não deve ter votos individuais, tem %d", n)
	}

	// votos e alinhamento (Governo orienta Sim: 461 Sim, 4 Não, 1 Artigo 17)
	if n := contar(t, pool, `SELECT count(*) FROM votos WHERE votacao_id=$1`, idNominal); n != 466 {
		t.Errorf("votos=%d", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM votos WHERE votacao_id=$1 AND voto='sim' AND alinhou_governo IS TRUE`, idNominal); n != 461 {
		t.Errorf("alinhados=%d", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM votos WHERE votacao_id=$1 AND voto='nao' AND alinhou_governo IS FALSE`, idNominal); n != 4 {
		t.Errorf("não alinhados=%d", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM votos WHERE votacao_id=$1 AND voto='artigo_17' AND alinhou_governo IS NULL`, idNominal); n != 1 {
		t.Errorf("artigo 17 deve ficar fora do cálculo, linhas=%d", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM votos WHERE votacao_id=$1 AND partido_na_votacao IS NULL`, idNominal); n != 0 {
		t.Errorf("votos sem partido_na_votacao: %d", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM votacoes WHERE id=$1 AND orientacao_governo='sim'`, idNominal); n != 1 {
		t.Error("orientacao_governo não denormalizada")
	}
	if n := contar(t, pool, `SELECT count(*) FROM votacoes_orientacoes WHERE votacao_id=$1`, idNominal); n != 15 {
		t.Errorf("orientações=%d", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM votacoes_orientacoes WHERE votacao_id=$1 AND orientacao IS NULL`, idNominal); n == 0 {
		t.Error("orientações vazias deveriam virar NULL")
	}

	// proposição e vínculo
	if n := contar(t, pool, `SELECT count(*) FROM votacoes v JOIN proposicoes p ON p.id=v.proposicao_id
		WHERE v.id=$1 AND v.vinculo_confianca='direto' AND p.sigla_tipo='PLP' AND p.numero=186 AND p.ano=2026`, idNominal); n != 1 {
		t.Error("vínculo com PLP 186/2026 não gravado")
	}

	// run registrado
	if n := contar(t, pool, `SELECT count(*) FROM etl_runs WHERE job='votacoes' AND status='ok' AND finished_at IS NOT NULL`); n != 1 {
		t.Errorf("etl_runs ok=%d", n)
	}
}

func TestIngerirVotacoes_Idempotente(t *testing.T) {
	pool := testdb.Novo(t)
	srv := servidor(t)
	defer srv.Close()
	e := novoETL(t, pool, srv)
	ctx := context.Background()
	ini, fim := janela()

	snapshot := func() string {
		return fmt.Sprint(
			contar(t, pool, `SELECT count(*) FROM votacoes`),
			contar(t, pool, `SELECT count(*) FROM votos`),
			contar(t, pool, `SELECT count(*) FROM parlamentares`),
			contar(t, pool, `SELECT count(*) FROM proposicoes`),
			contar(t, pool, `SELECT count(*) FROM votacoes_orientacoes`))
	}
	for i := 0; i < 2; i++ {
		if _, err := e.IngerirVotacoes(ctx, ini, fim, false); err != nil {
			t.Fatal(err)
		}
	}
	if got := snapshot(); got != "2 466 466 2 15" {
		t.Fatalf("após reexecução: %s", got)
	}

	// pularExistentes não rebusca as já gravadas
	res, err := e.IngerirVotacoes(ctx, ini, fim, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Puladas != 2 || res.Processadas != 0 || res.Adiadas != 1 {
		t.Fatalf("resumo=%+v", res)
	}
}

func TestIngerirVotacoes_ErroEmUmaNaoDerrubaALista(t *testing.T) {
	pool := testdb.Novo(t)
	srv := servidor(t)
	defer srv.Close()
	// a nominal passa a falhar com 400 no detalhe
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/votacoes/"+idNominal) {
			http.Error(w, "ruim", http.StatusBadRequest)
			return
		}
		srv.Config.Handler.ServeHTTP(w, r)
	}))
	defer bad.Close()
	e := novoETL(t, pool, bad)
	ini, fim := janela()

	res, err := e.IngerirVotacoes(context.Background(), ini, fim, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Erros != 1 || res.Processadas != 1 {
		t.Fatalf("resumo=%+v", res)
	}
	if n := contar(t, pool, `SELECT count(*) FROM etl_errors WHERE ref=$1`, "votacao/"+idNominal); n != 1 {
		t.Errorf("etl_errors=%d", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM votos WHERE votacao_id=$1`, idNominal); n != 0 {
		t.Error("votação com erro não deve deixar votos parciais")
	}
}

func TestSyncParlamentares_EExercicio(t *testing.T) {
	pool := testdb.Novo(t)
	srv := servidor(t)
	defer srv.Close()
	e := novoETL(t, pool, srv)
	ctx := context.Background()

	res, err := e.SyncParlamentares(ctx, 57)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 || res.Processadas != 2 || res.Erros != 0 {
		t.Fatalf("resumo=%+v", res)
	}
	// 178843: troca de partido divide em 2; 103758 (suplente): 2 períodos.
	q := `SELECT count(*) FROM parlamentares_exercicio x JOIN parlamentares p ON p.id=x.parlamentar_id WHERE p.id_externo=$1`
	if n := contar(t, pool, q, 178843); n != 2 {
		t.Errorf("exercício 178843=%d", n)
	}
	if n := contar(t, pool, q, 103758); n != 2 {
		t.Errorf("exercício 103758=%d", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM parlamentares_exercicio x JOIN parlamentares p ON p.id=x.parlamentar_id
		WHERE p.id_externo=103758 AND x.condicao='Suplente' AND x.fim IS NOT NULL`); n != 2 {
		t.Errorf("períodos de suplente fechados=%d", n)
	}

	// reexecução não duplica
	if _, err := e.SyncParlamentares(ctx, 57); err != nil {
		t.Fatal(err)
	}
	if n := contar(t, pool, `SELECT count(*) FROM parlamentares_exercicio`); n != 4 {
		t.Errorf("exercicios após reexecução=%d", n)
	}

	// partido_atual vem do último intervalo de exercício (178843 migrou PP → UNIÃO)
	if n := contar(t, pool, `SELECT count(*) FROM parlamentares WHERE id_externo=178843 AND partido_atual='UNIÃO'`); n != 1 {
		t.Error("partido_atual deveria ser o do último intervalo (UNIÃO)")
	}

	// votos não sobrescrevem o partido atual
	if _, err := pool.Exec(ctx, `UPDATE parlamentares SET partido_atual='TESTE' WHERE id_externo=178843`); err != nil {
		t.Fatal(err)
	}
	ini, fim := janela()
	if _, err := e.IngerirVotacoes(ctx, ini, fim, false); err != nil {
		t.Fatal(err)
	}
	if n := contar(t, pool, `SELECT count(*) FROM parlamentares WHERE id_externo=178843 AND partido_atual='TESTE'`); n != 1 {
		t.Error("partido_atual foi sobrescrito pelos votos")
	}
	if n := contar(t, pool, `SELECT count(*) FROM votos v JOIN parlamentares p ON p.id=v.parlamentar_id
		WHERE p.id_externo=178843 AND v.partido_na_votacao='UNIÃO'`); n != 1 {
		t.Error("partido_na_votacao deveria vir do voto (UNIÃO)")
	}
}

func TestIngerirVotacoes_DetalheInexistenteNaoEhErro(t *testing.T) {
	pool := testdb.Novo(t)
	srv := servidor(t)
	defer srv.Close()
	gone := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/votacoes/"+idNominal) {
			http.NotFound(w, r)
			return
		}
		srv.Config.Handler.ServeHTTP(w, r)
	}))
	defer gone.Close()
	e := novoETL(t, pool, gone)
	ini, fim := janela()

	res, err := e.IngerirVotacoes(context.Background(), ini, fim, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Indisponiveis != 1 || res.Erros != 0 || res.Processadas != 1 {
		t.Fatalf("resumo=%+v", res)
	}
	if n := contar(t, pool, `SELECT count(*) FROM etl_errors WHERE ref=$1`, "indisponivel/"+idNominal); n != 1 {
		t.Errorf("deveria registrar a indisponibilidade, linhas=%d", n)
	}
}

func TestIngerirSessoes_FimAFim(t *testing.T) {
	pool := testdb.Novo(t)
	srv := servidor(t)
	defer srv.Close()
	e := novoETL(t, pool, srv)
	ctx := context.Background()
	ini, fim := janela()

	res, err := e.IngerirSessoes(ctx, ini, fim, false)
	if err != nil {
		t.Fatal(err)
	}
	// 3 deliberativas (a solene é ignorada): 1 ok, 1 em andamento (adiada), 1 encerrada sem lista (indisponível)
	want := Resumo{Total: 3, Processadas: 1, Adiadas: 1, Indisponiveis: 1}
	if res != want {
		t.Fatalf("resumo=%+v, esperado %+v", res, want)
	}
	if n := contar(t, pool, `SELECT count(*) FROM sessoes`); n != 1 {
		t.Errorf("sessoes=%d", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM sessoes_presencas`); n != 502 {
		t.Errorf("presenças=%d", n)
	}
	if n := contar(t, pool, `SELECT count(*) FROM etl_errors WHERE ref='indisponivel/sessao/99001'`); n != 1 {
		t.Error("sessão sem lista deveria ser registrada como indisponível")
	}

	// idempotente + pularExistentes
	if _, err := e.IngerirSessoes(ctx, ini, fim, false); err != nil {
		t.Fatal(err)
	}
	if n := contar(t, pool, `SELECT count(*) FROM sessoes_presencas`); n != 502 {
		t.Errorf("presenças após reexecução=%d", n)
	}
	r2, err := e.IngerirSessoes(ctx, ini, fim, true)
	if err != nil || r2.Puladas != 1 || r2.Processadas != 0 {
		t.Fatalf("resumo=%+v err=%v", r2, err)
	}

	// métricas: deputado presente na única sessão elegível → 100%
	if err := e.AtualizarMetricas(ctx); err != nil {
		t.Fatal(err)
	}
	if n := contar(t, pool, `SELECT count(*) FROM mv_parlamentar_presenca_sessao m JOIN parlamentares p ON p.id=m.parlamentar_id
		WHERE p.id_externo=204436 AND m.sessoes_elegiveis=1 AND m.sessoes_presente=1 AND m.pct_presenca_sessao=100`); n != 1 {
		t.Error("presença em sessão do deputado 204436 deveria ser 1/1")
	}
}
