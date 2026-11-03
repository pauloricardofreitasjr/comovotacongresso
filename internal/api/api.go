// Package api implementa a API HTTP pública (somente leitura) do Como Vota Congresso.
// As métricas vêm das materialized views; nada é agregado na requisição (§7.4 do doc).
package api

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"comovotacongresso/internal/store"
)

//go:embed openapi.yaml
var openapiSpec []byte

const (
	limitePadrao = 20
	limiteMaximo = 100
	// Os dados mudam no máximo uma vez por dia (ETL diário): CDN e navegador podem reaproveitar.
	cacheControl = "public, max-age=300, stale-while-revalidate=3600"
)

// Leitor é o que a API precisa do banco (satisfeito por *store.Store).
type Leitor interface {
	ListarProposicoes(ctx context.Context, f store.ProposicaoFiltro) ([]store.ProposicaoResumo, int, error)
	GetProposicao(ctx context.Context, id int) (*store.ProposicaoDetalhe, error)
	GetVotacao(ctx context.Context, id string) (*store.VotacaoDetalhe, error)
	ListarParlamentares(ctx context.Context, f store.ParlamentarFiltro) ([]store.ParlamentarResumo, int, error)
	GetParlamentar(ctx context.Context, id, limiteVotos, offsetVotos int) (*store.ParlamentarDetalhe, error)
	ListarPartidos(ctx context.Context, casa string) ([]store.PartidoLinha, error)
	GetStatus(ctx context.Context) (store.Status, error)
}

type Pinger interface{ Ping(context.Context) error }

type API struct {
	db  Leitor
	log *slog.Logger
}

// Router monta todas as rotas. ping alimenta o /healthz (pode ser nil).
func Router(db Leitor, ping Pinger, log *slog.Logger) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	a := &API{db: db, log: log}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, middleware.Timeout(30*time.Second), a.logRequests, cors)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		erro(w, http.StatusNotFound, "rota_inexistente", "rota não encontrada")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		erro(w, http.StatusMethodNotAllowed, "metodo_nao_permitido", "apenas GET é suportado")
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if ping != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := ping.Ping(ctx); err != nil {
				http.Error(w, "db indisponível", http.StatusServiceUnavailable)
				return
			}
		}
		w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(cacheHeader, queryValida)
		r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
			w.Write(openapiSpec)
		})
		r.Get("/status", a.status)
		r.Get("/proposicoes", a.listarProposicoes)
		r.Get("/proposicoes/{id}", a.getProposicao)
		r.Get("/votacoes/{id}", a.getVotacao)
		r.Get("/parlamentares", a.listarParlamentares)
		r.Get("/parlamentares/{id}", a.getParlamentar)
		r.Get("/partidos", a.listarPartidos)
	})
	return r
}

// ---------- respostas ----------

type erroBody struct {
	Erro struct {
		Codigo   string `json:"codigo"`
		Mensagem string `json:"mensagem"`
	} `json:"erro"`
}

func erro(w http.ResponseWriter, status int, codigo, msg string) {
	var b erroBody
	b.Erro.Codigo, b.Erro.Mensagem = codigo, msg
	w.Header().Set("Cache-Control", "no-store")
	escrever(w, status, b)
}

func escrever(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// falha trata erros de banco: 404 para não encontrado, 500 (sem vazar detalhes) para o resto.
func (a *API) falha(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrNaoEncontrado) {
		erro(w, http.StatusNotFound, "nao_encontrado", "registro não encontrado")
		return
	}
	if r.Context().Err() != nil { // cliente desistiu ou timeout
		erro(w, http.StatusServiceUnavailable, "indisponivel", "requisição cancelada ou expirada")
		return
	}
	a.log.Error("erro interno", "path", r.URL.Path, "erro", err, "request_id", middleware.GetReqID(r.Context()))
	erro(w, http.StatusInternalServerError, "erro_interno", "erro interno")
}

// Pagina é o envelope das listagens.
type Pagina[T any] struct {
	Dados  []T `json:"dados"`
	Pagina int `json:"pagina"`
	Limite int `json:"limite"`
	Total  int `json:"total"`
}

func novaPagina[T any](dados []T, pagina, limite, total int) Pagina[T] {
	if dados == nil {
		dados = []T{}
	}
	return Pagina[T]{Dados: dados, Pagina: pagina, Limite: limite, Total: total}
}

// ---------- middlewares ----------

func cacheHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheControl)
		next.ServeHTTP(w, r)
	})
}

// queryValida rejeita query strings malformadas (ex.: ';' ou escapes inválidos). O net/url
// descartaria esses pares em silêncio, e um parâmetro ignorado mascara erro do cliente.
func queryValida(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := url.ParseQuery(r.URL.RawQuery); err != nil {
			erro(w, http.StatusBadRequest, "parametro_invalido", "query string malformada")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// cors libera leitura de qualquer origem: os dados são públicos e a API é somente GET.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		ini := time.Now()
		next.ServeHTTP(ww, r)
		a.log.Info("http", "metodo", r.Method, "path", r.URL.Path, "status", ww.Status(),
			"ms", time.Since(ini).Milliseconds(), "request_id", middleware.GetReqID(r.Context()))
	})
}

// ---------- parâmetros ----------

type paramErr string

func (e paramErr) Error() string { return string(e) }

// inteiro lê um query param inteiro em [min, max]; ausente → padrao.
func inteiro(r *http.Request, nome string, padrao, min, max int) (int, error) {
	v := r.URL.Query().Get(nome)
	if v == "" {
		return padrao, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min || n > max {
		return 0, paramErr("parâmetro '" + nome + "' deve ser um inteiro entre " + strconv.Itoa(min) + " e " + strconv.Itoa(max))
	}
	return n, nil
}

// paginacao lê pagina/limite (ou nomes alternativos, p.ex. page_votos).
func paginacao(r *http.Request, nomePagina, nomeLimite string) (pagina, limite, offset int, err error) {
	if pagina, err = inteiro(r, nomePagina, 1, 1, 1_000_000); err != nil {
		return
	}
	if limite, err = inteiro(r, nomeLimite, limitePadrao, 1, limiteMaximo); err != nil {
		return
	}
	return pagina, limite, (pagina - 1) * limite, nil
}

func idInt(r *http.Request) (int, bool) {
	n, err := strconv.Atoi(chi.URLParam(r, "id"))
	return n, err == nil && n > 0
}

func texto(r *http.Request, nome string, maxLen int) (string, error) {
	v := strings.TrimSpace(r.URL.Query().Get(nome))
	if len([]rune(v)) > maxLen {
		return "", paramErr("parâmetro '" + nome + "' excede " + strconv.Itoa(maxLen) + " caracteres")
	}
	return v, nil
}

// booleano lê um query param "true"/"false"; ausente → false.
func booleano(r *http.Request, nome string) (bool, error) {
	switch r.URL.Query().Get(nome) {
	case "", "false":
		return false, nil
	case "true":
		return true, nil
	}
	return false, paramErr("parâmetro '" + nome + "' deve ser 'true' ou 'false'")
}

func casaParam(r *http.Request) (string, error) {
	c := r.URL.Query().Get("casa")
	if c != "" && c != "camara" && c != "senado" {
		return "", paramErr("parâmetro 'casa' deve ser 'camara' ou 'senado'")
	}
	return c, nil
}
