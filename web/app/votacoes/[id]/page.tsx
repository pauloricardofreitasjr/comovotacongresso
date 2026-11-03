import type { Metadata } from "next";
import Link from "next/link";
import { api } from "@/lib/api";
import {
  formatDataHora, formatInt, inteiroPositivo, infoVoto, plural, primeiro, rotuloAlinhamento, rotuloOrientacao, rotuloResultado,
} from "@/lib/format";
import type { Voto } from "@/lib/types";
import { Compartilhar } from "@/components/Compartilhar";
import { metaCompartilhavel, resumir } from "@/lib/compartilhar";
import { SITE_URL } from "@/lib/site";
import { Aviso, AvisoQuarentena, limpar, NotasDoPlacar, Pessoa, Paginacao, Placar, ResultadoBadge, TipoBadge, VotoChip } from "@/components/ui";

type Props = { params: Promise<{ id: string }>; searchParams: Promise<Record<string, string | string[] | undefined>> };

const LIMITE = 100;
const OPCOES_VOTO = ["sim", "nao", "abstencao", "obstrucao", "artigo_17", "outro"];

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id } = await params;
  const v = await api.votacao(id);
  const r = v.resumo_votos;
  const placar = r ? `${rotuloResultado(v.aprovada)}: ${formatInt(r.sim)} Sim × ${formatInt(r.nao)} Não. ` : `${rotuloResultado(v.aprovada)}. `;
  return metaCompartilhavel({
    titulo: v.proposicao ? `Votação — ${v.proposicao.identificacao}` : "Votação",
    descricao: resumir(`${placar}${limpar(v.descricao)}`, 200),
    caminho: `/votacoes/${encodeURIComponent(id)}`,
  });
}

type LinhaPartido = { partido: string; sim: number; nao: number; outros: number; total: number };

/** Como cada partido votou, agrupando pelo partido do deputado na data da votação. */
function porPartido(votos: Voto[]): LinhaPartido[] {
  const mapa = new Map<string, LinhaPartido>();
  for (const v of votos) {
    const chave = v.partido ?? "(sem partido)";
    const l = mapa.get(chave) ?? { partido: chave, sim: 0, nao: 0, outros: 0, total: 0 };
    if (v.voto === "sim") l.sim++;
    else if (v.voto === "nao") l.nao++;
    else l.outros++;
    l.total++;
    mapa.set(chave, l);
  }
  return [...mapa.values()].sort((a, b) => b.total - a.total || a.partido.localeCompare(b.partido, "pt-BR"));
}

function classeOrientacao(o: string | null): string {
  if (o === "sim") return "sim";
  if (o === "nao") return "nao";
  if (o === "obstrucao") return "neutro";
  return "desconhecido";
}

function semAcento(s: string): string {
  return s.normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase();
}

export default async function Votacao({ params, searchParams }: Props) {
  const [{ id }, sp] = await Promise.all([params, searchParams]);
  const v = await api.votacao(id);

  const filtroVoto = OPCOES_VOTO.includes(primeiro(sp.voto)) ? primeiro(sp.voto) : "";
  const filtroPartido = primeiro(sp.partido).trim();
  const filtroNome = primeiro(sp.q).trim();
  const pagina = inteiroPositivo(sp.pagina, 1);

  const todos = v.votos ?? [];
  const filtrados = todos.filter(
    (x) =>
      (!filtroVoto || x.voto === filtroVoto) &&
      (!filtroPartido || x.partido === filtroPartido) &&
      (!filtroNome || semAcento(x.nome).includes(semAcento(filtroNome))),
  );
  const visiveis = filtrados.slice((pagina - 1) * LIMITE, pagina * LIMITE);
  const partidos = porPartido(todos);
  const nomesPartidos = partidos.map((p) => p.partido).sort((a, b) => a.localeCompare(b, "pt-BR"));
  const governo = v.orientacoes.find((o) => o.bancada.toLowerCase() === "governo");

  return (
    <>
      <header className="votacao-topo" data-resultado={v.aprovada === null ? "neutra" : v.aprovada ? "aprovada" : "rejeitada"}>
        <p className="breadcrumb">
          <Link href="/proposicoes">Leis e projetos</Link>
          {v.proposicao ? (
            <>
              {" › "}
              <Link href={`/proposicoes/${v.proposicao.id}`}>{v.proposicao.identificacao}</Link>
            </>
          ) : null}
          {" › Votação"}
        </p>
        <div className="meta" style={{ marginBottom: ".6rem" }}>
          <ResultadoBadge aprovada={v.aprovada} />
          <TipoBadge tipo={v.tipo} />
          <strong>{formatDataHora(v.data_hora)}</strong>
        </div>
        <h1>{v.proposicao ? v.proposicao.identificacao : "Votação"}</h1>
        <p className="lead">{limpar(v.descricao)}</p>
        {v.proposicao ? <p className="muted" style={{ marginBottom: 0 }}>{limpar(v.proposicao.ementa)}</p> : null}
        <Compartilhar
          url={`${SITE_URL}/votacoes/${encodeURIComponent(id)}`}
          titulo={v.proposicao ? `Votação — ${v.proposicao.identificacao}` : "Votação"}
          texto={`${v.proposicao ? v.proposicao.identificacao : "Votação"}${v.resumo_votos ? `: ${formatInt(v.resumo_votos.sim)} Sim × ${formatInt(v.resumo_votos.nao)} Não` : ""}. Veja como cada deputado votou:`}
        />
      </header>
      {v.quarentena ? <AvisoQuarentena q={v.quarentena} /> : null}
      {v.vinculo_confianca === "provavel" ? (
        <Aviso atencao>
          <p>A relação desta votação com {v.proposicao?.identificacao} foi inferida automaticamente e pode não ser exata.</p>
        </Aviso>
      ) : null}

      {v.tipo === "simbolica" ? (
        <Aviso>
          <p>
            <strong>Votação simbólica.</strong> Os deputados levantam ou permanecem sentados e o presidente da sessão proclama o resultado.
            Não há registro do voto de cada um — por isso não mostramos votos individuais.
          </p>
        </Aviso>
      ) : null}

      {v.resumo_votos ? (
        <section aria-labelledby="placar">
          <h2 id="placar">Placar</h2>
          <Placar resumo={v.resumo_votos} />
          <NotasDoPlacar aprovada={v.aprovada} resumo={v.resumo_votos} descricao={v.descricao} ancora />
          {v.resumo_votos.outro > 0 ? (
            <Aviso atencao>
              <p>
                Em {formatInt(v.resumo_votos.outro)} votos a Câmara não informou a posição do deputado (sim/não). Eles contam como voto registrado, mas
                não entram no cálculo de alinhamento.
              </p>
            </Aviso>
          ) : null}
        </section>
      ) : null}

      {v.orientacoes.length > 0 ? (
        <section aria-labelledby="orientacoes">
          <h2 id="orientacoes">O que os partidos orientaram</h2>
          <p className="muted">
            Antes de votar, os líderes dizem à bancada como votar. É uma orientação, não uma obrigação.
            {governo ? ` Orientação do Governo: ${rotuloOrientacao(governo.orientacao)}.` : " O Governo não registrou orientação nesta votação."}
          </p>
          <ul className="orient-grid">
            {[...v.orientacoes]
              .sort((x, y) => Number(y.bancada.toLowerCase() === "governo") - Number(x.bancada.toLowerCase() === "governo"))
              .map((o) => (
                <li key={o.bancada} className={`orient ${classeOrientacao(o.orientacao)}${o.bancada.toLowerCase() === "governo" ? " governo" : ""}`}>
                  <span className="orient-bancada">{o.bancada}</span>
                  <span className={`voto ${classeOrientacao(o.orientacao)}`}>{rotuloOrientacao(o.orientacao)}</span>
                </li>
              ))}
          </ul>
        </section>
      ) : null}

      {v.votos_individuais ? (
        <>
          <section aria-labelledby="partidos">
            <h2 id="partidos">Como cada partido votou</h2>
            <p className="muted">Partido do deputado na data da votação.</p>
            <ul className="partidos-lista">
              {partidos.map((p) => (
                <li key={p.partido} className="partido-linha">
                  <Link href={`/votacoes/${encodeURIComponent(id)}?partido=${encodeURIComponent(p.partido)}#votos`}>{p.partido}</Link>
                  <div className="barra" role="img" aria-label={`${p.partido}: ${p.sim} sim, ${p.nao} não, ${p.outros} outros`}>
                    <span className="sim" style={{ width: `${(p.sim / p.total) * 100}%` }} />
                    <span className="nao" style={{ width: `${(p.nao / p.total) * 100}%` }} />
                    <span className="desconhecido" style={{ width: `${(p.outros / p.total) * 100}%` }} />
                  </div>
                  <span className="numeros">
                    Sim {p.sim} · Não {p.nao} · Outros {p.outros} · Total {p.total}
                  </span>
                </li>
              ))}
            </ul>
          </section>

          <section aria-labelledby="votos">
            <h2 id="votos">Como cada deputado votou</h2>
            <div className="painel-filtros">
            <form className="filters" action={`/votacoes/${encodeURIComponent(id)}#votos`}>
              <label>
                Nome
                <input type="search" name="q" defaultValue={filtroNome} placeholder="Buscar deputado" />
              </label>
              <label>
                Voto
                <select name="voto" defaultValue={filtroVoto}>
                  <option value="">Todos</option>
                  {OPCOES_VOTO.map((o) => (
                    <option key={o} value={o}>{infoVoto(o).rotulo}</option>
                  ))}
                </select>
              </label>
              <label>
                Partido
                <select name="partido" defaultValue={filtroPartido}>
                  <option value="">Todos</option>
                  {nomesPartidos.map((p) => (
                    <option key={p} value={p}>{p}</option>
                  ))}
                </select>
              </label>
              <button type="submit">Filtrar</button>
              {filtroVoto || filtroPartido || filtroNome ? (
                <Link className="button secondary" href={`/votacoes/${encodeURIComponent(id)}#votos`}>Limpar</Link>
              ) : null}
            </form>
            </div>
            <p className="muted" aria-live="polite">{plural(filtrados.length, "deputado", "deputados")}.</p>
            <ul className="deputados-grid">
              {visiveis.map((x) => (
                <li key={x.parlamentar_id} className={`deputado-card ${infoVoto(x.voto).classe}`}>
                  <Pessoa id={x.parlamentar_id} nome={x.nome} url={x.url_foto} sub={[x.partido, x.uf].filter(Boolean).join(" · ")} />
                  <span className="lado">
                    <VotoChip voto={x.voto} />
                    {x.alinhou_governo !== null ? <small>{rotuloAlinhamento(x.alinhou_governo)} ao Governo</small> : null}
                  </span>
                </li>
              ))}
            </ul>
            <Paginacao
              pagina={pagina} limite={LIMITE} total={filtrados.length} caminho={`/votacoes/${encodeURIComponent(id)}`}
              params={{ voto: filtroVoto, partido: filtroPartido, q: filtroNome }}
            />
          </section>
        </>
      ) : null}
    </>
  );
}
