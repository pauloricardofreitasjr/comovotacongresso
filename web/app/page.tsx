import Link from "next/link";
import { api } from "@/lib/api";
import { formatData, formatInt, plural } from "@/lib/format";
import type { VotacaoLinha } from "@/lib/types";
import { IconBusca, IconGoverno, IconPresenca, IconVoto } from "@/components/icons";
import { limpar, NotasDoPlacar, Placar, ResultadoBadge } from "@/components/ui";

const TEMAS = ["Imposto de renda", "Educação", "Saúde", "Segurança", "Meio ambiente", "Trabalho"];

/** Última votação nominal entre as proposições mais recentes, para o destaque da home. Falhas não derrubam a página. */
async function ultimaNominal(recentes: Awaited<ReturnType<typeof api.proposicoes>>["dados"]) {
  for (const p of recentes) {
    if (p.votacoes_nominais === 0) continue;
    try {
      const d = await api.proposicao(String(p.id));
      const v = d.votacoes.filter((x): x is VotacaoLinha & { resumo_votos: NonNullable<VotacaoLinha["resumo_votos"]> } => x.tipo === "nominal" && x.resumo_votos !== null);
      if (v.length > 0) return { prop: p, votacao: v.sort((a, b) => b.data_hora.localeCompare(a.data_hora))[0] };
    } catch {
      return null;
    }
  }
  return null;
}

export default async function Home() {
  // O destaque olha mais fundo que a lista: as proposições mais recentes costumam ter só votação simbólica.
  const [recentes, janela, status] = await Promise.all([api.proposicoes({ limite: 6 }), api.proposicoes({ limite: 60 }), api.status()]);
  const destaque = await ultimaNominal(janela.dados);

  return (
    <>
      <section className="hero">
        <h1>
          Como seus <em>deputados</em> votam
        </h1>
        <p className="lead">
          Pesquise uma lei ou um tema e veja, votação por votação, como cada deputado federal votou no Plenário da Câmara. Dados oficiais, regras abertas.
        </p>
        <form className="busca-grande" action="/proposicoes" role="search">
          <IconBusca />
          <label className="sr-only" htmlFor="busca">Buscar leis e projetos</label>
          <input id="busca" type="search" name="busca" placeholder="Ex.: imposto, saúde" />
          <button type="submit">Buscar</button>
        </form>
        <ul className="chips" aria-label="Temas para começar">
          <li className="rotulo">Comece por:</li>
          {TEMAS.map((t) => (
            <li key={t}>
              <Link className="chip" href={`/proposicoes?busca=${encodeURIComponent(t)}`}>{t}</Link>
            </li>
          ))}
          <li>
            <Link className="chip" href="/parlamentares">Procurar um deputado →</Link>
          </li>
        </ul>
      </section>

      <div className="stats">
        <div className="stat"><strong>{formatInt(status.parlamentares)}</strong><span>deputados acompanhados</span></div>
        <div className="stat"><strong>{formatInt(status.votacoes_nominais)}</strong><span>votações nominais</span></div>
        <div className="stat"><strong>{formatInt(status.votacoes_simbolicas)}</strong><span>votações simbólicas</span></div>
        <div className="stat"><strong>{formatInt(status.sessoes_deliberativas)}</strong><span>sessões deliberativas</span></div>
      </div>

      {destaque ? (
        <section className="secao" aria-labelledby="destaque">
          <h2 id="destaque" className="sr-only">Última votação nominal</h2>
          <div className="destaque-voto">
            <div>
              <span className="tag">Última votação nominal · {formatData(destaque.votacao.data_hora)}</span>
              <h3>
                <Link href={`/votacoes/${encodeURIComponent(destaque.votacao.id)}`}>{destaque.prop.identificacao}</Link>
              </h3>
              <p className="clamp muted" style={{ margin: ".25rem 0 .5rem" }}>{limpar(destaque.prop.ementa)}</p>
              <ResultadoBadge aprovada={destaque.votacao.aprovada} />
            </div>
            <Placar resumo={destaque.votacao.resumo_votos} />
            <NotasDoPlacar curta aprovada={destaque.votacao.aprovada} resumo={destaque.votacao.resumo_votos} votacaoId={destaque.votacao.id} />
            <p style={{ margin: 0 }}>
              <Link className="button" href={`/votacoes/${encodeURIComponent(destaque.votacao.id)}`}>Ver como cada deputado votou</Link>
            </p>
          </div>
        </section>
      ) : null}

      <section className="secao" aria-labelledby="recentes">
        <h2 id="recentes">Votadas recentemente</h2>
        <ul className="recentes">
          {recentes.dados.map((p) => (
            <li key={p.id} className="card">
              <div className="item-title">
                <Link href={`/proposicoes/${p.id}`}>{p.identificacao}</Link>
              </div>
              <p className="clamp" style={{ margin: 0 }}>{limpar(p.ementa)}</p>
              <div className="meta">
                <span>Última votação em {formatData(p.ultima_votacao)}</span>
                <span>{plural(p.total_votacoes, "votação", "votações")}</span>
                {p.votacoes_nominais > 0 ? <span>{plural(p.votacoes_nominais, "nominal", "nominais")}</span> : null}
              </div>
            </li>
          ))}
        </ul>
        <p style={{ marginTop: "1rem" }}>
          <Link className="button secondary" href="/proposicoes">Ver todas as leis e projetos</Link>
        </p>
      </section>

      <section className="secao" aria-labelledby="medidas">
        <h2 id="medidas">Três jeitos de acompanhar um deputado</h2>
        <p className="muted">Cada número mede uma coisa diferente. Todos mostram quantas votações estão por trás.</p>
        <div className="metodo">
          <article className="metodo-card presenca">
            <span className="icone"><IconPresenca /></span>
            <h3>Presença em sessões</h3>
            <p>Em quantas sessões deliberativas do Plenário o deputado registrou presença.</p>
            <Link href="/parlamentares?ordenarPor=presenca_sessao">Ver ranking →</Link>
          </article>
          <article className="metodo-card voto">
            <span className="icone"><IconVoto /></span>
            <h3>Votos nas votações nominais</h3>
            <p>Nas votações em que cada deputado vota em seu nome, em quantas ele registrou o voto. Estar na sessão não é o mesmo que votar.</p>
            <Link href="/parlamentares?ordenarPor=participacao">Ver ranking →</Link>
          </article>
          <article className="metodo-card governo">
            <span className="icone"><IconGoverno /></span>
            <h3>Alinhamento com o governo</h3>
            <p>Com que frequência o voto coincidiu com a orientação do Governo naquele dia. Não diz se o voto foi bom ou ruim.</p>
            <Link href="/parlamentares?ordenarPor=governismo">Ver ranking →</Link>
          </article>
        </div>
        <p className="muted" style={{ marginTop: "1rem" }}>
          Dados de {formatData(status.primeira_votacao)} a {formatData(status.ultima_votacao)}. <Link href="/metodologia">Como calculamos</Link>.
        </p>
      </section>
    </>
  );
}
