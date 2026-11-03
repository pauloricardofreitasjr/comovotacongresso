// Package etl orquestra a ingestão de dados da Câmara no banco.
package etl

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"comovotacongresso/internal/camara"
	"comovotacongresso/internal/domain"
	"comovotacongresso/internal/httpclient"
	"comovotacongresso/internal/store"
)

// Fonte é a API da Câmara (satisfeita por *camara.Client).
type Fonte interface {
	Votacoes(ctx context.Context, ini, fim time.Time, idOrgao int) ([]camara.VotacaoResumo, error)
	Votacao(ctx context.Context, id string) (camara.VotacaoDetalhe, error)
	Votos(ctx context.Context, id string) ([]camara.Voto, error)
	Orientacoes(ctx context.Context, id string) ([]camara.Orientacao, error)
	Deputados(ctx context.Context, legislatura int) ([]camara.Deputado, error)
	Historico(ctx context.Context, idDeputado int) ([]camara.HistoricoEvento, error)
	Eventos(ctx context.Context, ini, fim time.Time, idOrgao int) ([]camara.Evento, error)
	EventoDeputados(ctx context.Context, idEvento int) ([]camara.Deputado, error)
}

type ETL struct {
	Fonte   Fonte
	Store   *store.Store
	Log     *slog.Logger
	Workers int // votações processadas em paralelo (padrão 4)
}

// Resumo contabiliza uma execução.
type Resumo struct {
	Total, Processadas, Nominais, Simbolicas, Adiadas, Puladas, Indisponiveis, Erros int
}

func (e *ETL) log() *slog.Logger {
	if e.Log != nil {
		return e.Log
	}
	return slog.Default()
}

// SyncParlamentares carrega os deputados da legislatura e seus intervalos de exercício.
func (e *ETL) SyncParlamentares(ctx context.Context, legislatura int) (res Resumo, err error) {
	runID, err := e.Store.IniciarRun(ctx, "sync-parlamentares", map[string]int{"legislatura": legislatura})
	if err != nil {
		return res, err
	}
	defer func() { e.finalizar(runID, err) }()

	deps, err := e.Fonte.Deputados(ctx, legislatura)
	if err != nil {
		return res, fmt.Errorf("listar deputados: %w", err)
	}
	deps = unicos(deps)
	ins := make([]store.ParlamentarIn, len(deps))
	for i, d := range deps {
		ins[i] = paraParlamentar(d)
	}
	ids, err := e.Store.UpsertParlamentares(ctx, ins, true)
	if err != nil {
		return res, err
	}
	res.Total = len(deps)

	for _, d := range deps {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		if err := e.syncExercicio(ctx, ids[d.ID], d.ID, legislatura); err != nil {
			res.Erros++
			e.registrarErro(ctx, runID, fmt.Sprintf("deputado/%d", d.ID), err)
			continue
		}
		res.Processadas++
	}
	e.log().Info("parlamentares sincronizados", "legislatura", legislatura, "resumo", res)
	return res, nil
}

func (e *ETL) syncExercicio(ctx context.Context, parlamentarID, idExterno, legislatura int) error {
	hist, err := e.Fonte.Historico(ctx, idExterno)
	if err != nil {
		return err
	}
	evs := make([]domain.EventoHistorico, 0, len(hist))
	for _, h := range hist {
		t, err := camara.ParseDataHora(h.DataHora)
		if err != nil {
			return err
		}
		if h.IDLegislatura == legislatura && !domain.SituacaoConhecida(h.Situacao) {
			e.log().Warn("situação de exercício desconhecida; pode encerrar um período", "deputado", idExterno, "situacao", h.Situacao)
		}
		evs = append(evs, domain.EventoHistorico{
			IDLegislatura: h.IDLegislatura, DataHora: t, Partido: h.SiglaPartido,
			Situacao: h.Situacao, Condicao: h.CondicaoEleitoral,
		})
	}
	return e.Store.SubstituirExercicio(ctx, parlamentarID, domain.Intervalos(evs, legislatura))
}

// IngerirVotacoes baixa e grava as votações do Plenário entre ini e fim (inclusive).
// Com pularExistentes, votações já gravadas não são rebuscadas (útil no backfill).
func (e *ETL) IngerirVotacoes(ctx context.Context, ini, fim time.Time, pularExistentes bool) (res Resumo, err error) {
	runID, err := e.Store.IniciarRun(ctx, "votacoes", map[string]any{
		"de": ini.Format("2006-01-02"), "ate": fim.Format("2006-01-02"), "pular_existentes": pularExistentes,
	})
	if err != nil {
		return res, err
	}
	defer func() { e.finalizar(runID, err) }()

	lista, err := e.Fonte.Votacoes(ctx, ini, fim, camara.OrgaoPlenario)
	if err != nil {
		return res, fmt.Errorf("listar votações: %w", err)
	}
	res.Total = len(lista)

	var nom, simb, adiadas, puladas, indisp, erros, ok atomic.Int64
	workers := e.Workers
	if workers <= 0 {
		workers = 4
	}
	var mu sync.Mutex // serializa registro de erros
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)
	for _, v := range lista {
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}
			if pularExistentes {
				if existe, err := e.Store.VotacaoExiste(gctx, v.ID); err == nil && existe {
					puladas.Add(1)
					return nil
				}
			}
			tipo, err := e.processarVotacao(gctx, v)
			switch {
			case errors.Is(err, errAdiada):
				adiadas.Add(1)
			case errors.Is(err, errIndisponivel):
				// a listagem cita votações cujo detalhe não existe na fonte (404); não é falha nossa.
				indisp.Add(1)
				mu.Lock()
				e.registrarErro(ctx, runID, "indisponivel/"+v.ID, err)
				mu.Unlock()
			case err != nil:
				if gctx.Err() != nil {
					return gctx.Err()
				}
				erros.Add(1)
				mu.Lock()
				e.registrarErro(ctx, runID, "votacao/"+v.ID, err)
				mu.Unlock()
			default:
				ok.Add(1)
				if tipo == domain.VotacaoNominal {
					nom.Add(1)
				} else {
					simb.Add(1)
				}
			}
			return nil
		})
	}
	if err = g.Wait(); err != nil {
		return res, err
	}
	res.Processadas, res.Nominais, res.Simbolicas = int(ok.Load()), int(nom.Load()), int(simb.Load())
	res.Adiadas, res.Puladas, res.Erros = int(adiadas.Load()), int(puladas.Load()), int(erros.Load())
	res.Indisponiveis = int(indisp.Load())
	e.log().Info("votações ingeridas", "de", ini.Format("2006-01-02"), "ate", fim.Format("2006-01-02"), "resumo", res)
	return res, nil
}

var (
	errAdiada       = errors.New("votação sem resultado e sem votos (em andamento); adiada")
	errIndisponivel = errors.New("votação listada pela API, mas o detalhe não existe (404)")
)

func naoEncontrado(err error) bool {
	var se *httpclient.StatusError
	return errors.As(err, &se) && se.Status == http.StatusNotFound
}

func (e *ETL) processarVotacao(ctx context.Context, resumo camara.VotacaoResumo) (domain.TipoVotacao, error) {
	det, err := e.Fonte.Votacao(ctx, resumo.ID)
	if naoEncontrado(err) {
		return "", fmt.Errorf("%w: %v", errIndisponivel, err)
	}
	if err != nil {
		return "", fmt.Errorf("detalhe: %w", err)
	}
	votos, err := e.Fonte.Votos(ctx, resumo.ID)
	if err != nil {
		return "", fmt.Errorf("votos: %w", err)
	}
	orients, err := e.Fonte.Orientacoes(ctx, resumo.ID)
	if err != nil {
		return "", fmt.Errorf("orientações: %w", err)
	}

	aprovacao := det.Aprovacao
	if aprovacao == nil {
		aprovacao = resumo.Aprovacao
	}
	tipo, ok := domain.ClassificaTipo(len(votos), aprovacao)
	if !ok {
		return "", errAdiada
	}

	quando := det.DataHoraRegistro
	if quando == "" {
		quando = resumo.DataHoraRegistro
	}
	dh, err := camara.ParseDataHora(quando)
	if err != nil {
		return "", err
	}

	in := store.VotacaoIn{ID: resumo.ID, DataHora: dh, Descricao: det.Descricao, Tipo: tipo}
	if aprovacao != nil {
		b := *aprovacao == 1
		in.Aprovada = &b
	}
	prop, conf := domain.EscolheProposicao(paraProps(det.ProposicoesAfetadas), paraProps(det.ObjetosPossiveis))
	in.Proposicao, in.Vinculo = prop, conf

	for _, o := range orients {
		orient := domain.MapOrientacao(o.OrientacaoVoto)
		in.Orientacoes = append(in.Orientacoes, store.OrientacaoIn{
			Bancada: o.SiglaPartidoBloco, TipoLideranca: o.CodTipoLideranca, Orientacao: orient,
		})
		if domain.IsGoverno(o.SiglaPartidoBloco) {
			in.OrientacaoGoverno = orient
		}
	}

	desconhecidos := map[string]int{}      // valor bruto → ocorrências
	visto := make(map[int]int, len(votos)) // deputado → índice, para deduplicar
	for _, v := range votos {
		voto := domain.MapVoto(v.TipoVoto)
		if voto == domain.VotoOutro {
			desconhecidos[v.TipoVoto]++
		}
		vi := store.VotoIn{
			Parlamentar:    paraParlamentar(v.Deputado),
			Voto:           voto,
			Partido:        v.Deputado.SiglaPartido,
			AlinhouGoverno: domain.Alinhou(in.OrientacaoGoverno, voto),
		}
		if i, dup := visto[v.Deputado.ID]; dup {
			e.log().Warn("voto duplicado; mantendo o último", "votacao", resumo.ID, "deputado", v.Deputado.ID)
			in.Votos[i] = vi
			continue
		}
		visto[v.Deputado.ID] = len(in.Votos)
		in.Votos = append(in.Votos, vi)
	}
	for valor, n := range desconhecidos {
		e.log().Warn("tipo de voto desconhecido; gravado como 'outro'", "votacao", resumo.ID, "valor", valor, "votos", n, "de", len(votos))
	}
	if err := e.Store.SalvarVotacao(ctx, in); err != nil {
		return "", err
	}
	return tipo, nil
}

const (
	tipoSessaoDeliberativa = "Sessão Deliberativa"
	situacaoEncerrada      = "Encerrada"
)

var errSemPresencas = errors.New("sessão encerrada sem lista de presença na fonte")

// IngerirSessoes grava a presença oficial nas sessões deliberativas encerradas do Plenário.
// Sessões não deliberativas, em andamento ou canceladas são ignoradas (contam em Adiadas).
func (e *ETL) IngerirSessoes(ctx context.Context, ini, fim time.Time, pularExistentes bool) (res Resumo, err error) {
	runID, err := e.Store.IniciarRun(ctx, "sessoes", map[string]any{
		"de": ini.Format("2006-01-02"), "ate": fim.Format("2006-01-02"), "pular_existentes": pularExistentes,
	})
	if err != nil {
		return res, err
	}
	defer func() { e.finalizar(runID, err) }()

	eventos, err := e.Fonte.Eventos(ctx, ini, fim, camara.OrgaoPlenario)
	if err != nil {
		return res, fmt.Errorf("listar eventos: %w", err)
	}
	for _, ev := range eventos {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		if ev.DescricaoTipo != tipoSessaoDeliberativa {
			continue // solenes, comissões gerais etc. não entram
		}
		res.Total++
		if ev.Situacao != situacaoEncerrada {
			res.Adiadas++
			continue
		}
		if pularExistentes {
			if existe, err := e.Store.SessaoExiste(ctx, ev.ID); err == nil && existe {
				res.Puladas++
				continue
			}
		}
		if err := e.processarSessao(ctx, ev); err != nil {
			switch {
			case naoEncontrado(err), errors.Is(err, errSemPresencas):
				res.Indisponiveis++
				e.registrarErro(ctx, runID, fmt.Sprintf("indisponivel/sessao/%d", ev.ID), err)
			case ctx.Err() != nil:
				return res, ctx.Err()
			default:
				res.Erros++
				e.registrarErro(ctx, runID, fmt.Sprintf("sessao/%d", ev.ID), err)
			}
			continue
		}
		res.Processadas++
	}
	e.log().Info("sessões ingeridas", "de", ini.Format("2006-01-02"), "ate", fim.Format("2006-01-02"), "resumo", res)
	return res, nil
}

func (e *ETL) processarSessao(ctx context.Context, ev camara.Evento) error {
	deps, err := e.Fonte.EventoDeputados(ctx, ev.ID)
	if err != nil {
		return fmt.Errorf("presenças: %w", err)
	}
	if len(deps) == 0 {
		return errSemPresencas
	}
	inicio, err := camara.ParseDataHora(ev.DataHoraInicio)
	if err != nil {
		return err
	}
	in := store.SessaoIn{IDExterno: ev.ID, Inicio: inicio, Tipo: ev.DescricaoTipo, Descricao: strings.TrimSpace(ev.Descricao)}
	if ev.DataHoraFim != "" {
		if f, err := camara.ParseDataHora(ev.DataHoraFim); err == nil {
			in.Fim = &f
		}
	}
	in.Presentes = make([]store.ParlamentarIn, len(deps))
	for i, d := range deps {
		in.Presentes[i] = paraParlamentar(d)
	}
	return e.Store.SalvarSessao(ctx, in)
}

// AtualizarMetricas recalcula presença e alinhamento (materialized views). Deve rodar
// ao fim de um ciclo, não a cada janela.
func (e *ETL) AtualizarMetricas(ctx context.Context) error {
	ini := time.Now()
	if err := e.Store.AtualizarMetricas(ctx); err != nil {
		return err
	}
	e.log().Info("métricas atualizadas", "duracao", time.Since(ini).Round(time.Millisecond))
	return nil
}

func (e *ETL) finalizar(runID int64, err error) {
	status, msg := "ok", ""
	if err != nil {
		status, msg = "falhou", err.Error()
	}
	// contexto próprio: o original pode ter sido cancelado.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if ferr := e.Store.FinalizarRun(ctx, runID, status, msg); ferr != nil {
		e.log().Error("finalizar run", "erro", ferr)
	}
}

func (e *ETL) registrarErro(ctx context.Context, runID int64, ref string, err error) {
	e.log().Error("falha ao processar", "ref", ref, "erro", err)
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if rerr := e.Store.RegistrarErro(cctx, runID, ref, err.Error()); rerr != nil {
		e.log().Error("registrar erro", "erro", rerr)
	}
}

func paraParlamentar(d camara.Deputado) store.ParlamentarIn {
	return store.ParlamentarIn{IDExterno: d.ID, Nome: d.Nome, Partido: d.SiglaPartido, UF: d.SiglaUf, URLFoto: d.URLFoto}
}

func paraProps(ps []camara.Proposicao) []domain.Prop {
	out := make([]domain.Prop, len(ps))
	for i, p := range ps {
		out[i] = domain.Prop{ID: p.ID, SiglaTipo: p.SiglaTipo, Numero: p.Numero, Ano: p.Ano, Ementa: p.Ementa}
	}
	return out
}

// unicos remove deputados repetidos (a API traz uma linha por partido), mantendo a ordem.
func unicos(ds []camara.Deputado) []camara.Deputado {
	visto := make(map[int]bool, len(ds))
	out := ds[:0:0]
	for _, d := range ds {
		if !visto[d.ID] {
			visto[d.ID] = true
			out = append(out, d)
		}
	}
	return out
}
