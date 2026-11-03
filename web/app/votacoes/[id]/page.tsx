import type { Metadata } from "next";
import Link from "next/link";
import { api } from "@/lib/api";
import {
  formatDataHora, formatInt, inteiroPositivo, infoVoto, plural, primeiro, rotuloAlinhamento, rotuloOrientacao,
} from "@/lib/format";
import type { Voto } from "@/lib/types";
import { Aviso, BarraVotos, limpar, Pessoa, Paginacao, ResultadoBadge, TipoBadge, VotoChip } from "@/components/ui";

type Props = { params: Promise<{ id: string }>; searchParams: Promise<Record<string, string | string[] | undefined>> };

const LIMITE = 100;
const OPCOES_VOTO = ["sim", "nao", "abstencao", "obstrucao", "artigo_17", "outro"];

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id } = await params;
  const v = await api.votacao(id);
  return { title: v.proposicao ? `Votação — ${v.proposicao.identificacao}` : "Votação", description: limpar(v.descricao).slice(0, 200) };
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

      <h1>{v.proposicao ? v.proposicao.identificacao : "Votação"}</h1>
      <div className="meta" style={{ marginBottom: ".75rem" }}>
        <strong>{formatDataHora(v.data_hora)}</strong>
        <TipoBadge tipo={v.tipo} />
        <ResultadoBadge aprovada={v.aprovada} />
      </div>
      <p className="lead">{limpar(v.descricao)}</p>
      {v.proposicao ? <p className="muted">{limpar(v.proposicao.ementa)}</p> : null}
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
          <BarraVotos resumo={v.resumo_votos} />
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
          <div className="table-wrap">
            <table>
              <thead>
                <tr><th>Bancada</th><th>Orientação</th></tr>
              </thead>
              <tbody>
                {v.orientacoes.map((o) => (
                  <tr key={o.bancada}>
                    <td>{o.bancada}</td>
                    <td>{rotuloOrientacao(o.orientacao)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}

      {v.votos_individuais ? (
        <>
          <section aria-labelledby="partidos">
            <h2 id="partidos">Como cada partido votou</h2>
            <p className="muted">Partido do deputado na data da votação.</p>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr><th>Partido</th><th className="num">Sim</th><th className="num">Não</th><th className="num">Outros</th><th className="num">Total</th></tr>
                </thead>
                <tbody>
                  {partidos.map((p) => (
                    <tr key={p.partido}>
                      <td>
                        <Link href={`/votacoes/${encodeURIComponent(id)}?partido=${encodeURIComponent(p.partido)}#votos`}>{p.partido}</Link>
                      </td>
                      <td className="num">{p.sim}</td>
                      <td className="num">{p.nao}</td>
                      <td className="num">{p.outros}</td>
                      <td className="num">{p.total}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <section aria-labelledby="votos">
            <h2 id="votos">Como cada deputado votou</h2>
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
            <p className="muted" aria-live="polite">{plural(filtrados.length, "deputado", "deputados")}.</p>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr><th>Deputado</th><th>Voto</th><th>Em relação ao Governo</th></tr>
                </thead>
                <tbody>
                  {visiveis.map((x) => (
                    <tr key={x.parlamentar_id}>
                      <td>
                        <Pessoa id={x.parlamentar_id} nome={x.nome} url={x.url_foto} sub={[x.partido, x.uf].filter(Boolean).join(" · ")} />
                      </td>
                      <td><VotoChip voto={x.voto} /></td>
                      <td>{rotuloAlinhamento(x.alinhou_governo)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
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
