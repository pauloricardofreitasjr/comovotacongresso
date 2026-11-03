import Link from "next/link";
import { api } from "@/lib/api";
import { formatData, formatInt, plural } from "@/lib/format";
import { limpar } from "@/components/ui";

export default async function Home() {
  const [recentes, status] = await Promise.all([api.proposicoes({ limite: 6 }), api.status()]);

  return (
    <>
      <section className="hero">
        <h1>Como seus deputados votam</h1>
        <p className="lead">
          Pesquise uma lei ou um tema e veja, votação por votação, como cada deputado federal votou no Plenário da Câmara.
          Dados oficiais, regras abertas.
        </p>
        <form className="search" action="/proposicoes" role="search">
          <label className="sr-only" htmlFor="busca">Buscar leis e projetos</label>
          <input id="busca" type="search" name="busca" placeholder="Ex.: imposto de renda, educação, reforma tributária" />
          <button type="submit">Buscar leis</button>
        </form>
        <p>
          Procura uma pessoa? <Link href="/parlamentares">Veja todos os deputados</Link>.
        </p>
      </section>

      <section aria-labelledby="recentes">
        <h2 id="recentes">Votadas recentemente</h2>
        <ul className="list">
          {recentes.dados.map((p) => (
            <li key={p.id} className="card">
              <div className="item-title">
                <Link href={`/proposicoes/${p.id}`}>{p.identificacao}</Link>
              </div>
              <p className="clamp">{limpar(p.ementa)}</p>
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

      <section aria-labelledby="medidas">
        <h2 id="medidas">Três jeitos de acompanhar um deputado</h2>
        <p className="muted">Cada número mede uma coisa diferente. Todos mostram quantas votações estão por trás.</p>
        <div className="grid cols-3">
          <article className="card stack">
            <h3>Presença em sessões</h3>
            <p>Em quantas sessões deliberativas do Plenário o deputado registrou presença.</p>
            <Link href="/parlamentares?ordenarPor=presenca_sessao">Ver ranking →</Link>
          </article>
          <article className="card stack">
            <h3>Votos nas votações nominais</h3>
            <p>Nas votações em que cada deputado vota em seu nome, em quantas ele registrou o voto. Estar na sessão não é o mesmo que votar.</p>
            <Link href="/parlamentares?ordenarPor=participacao">Ver ranking →</Link>
          </article>
          <article className="card stack">
            <h3>Alinhamento com o governo</h3>
            <p>Com que frequência o voto coincidiu com a orientação do Governo naquele dia. Não diz se o voto foi bom ou ruim.</p>
            <Link href="/parlamentares?ordenarPor=governismo">Ver ranking →</Link>
          </article>
        </div>
      </section>

      <section aria-labelledby="numeros">
        <h2 id="numeros">Os dados</h2>
        <p className="muted">
          {formatInt(status.votacoes_nominais)} votações nominais, {formatInt(status.votacoes_simbolicas)} simbólicas e{" "}
          {formatInt(status.sessoes_deliberativas)} sessões deliberativas, de {formatData(status.primeira_votacao)} a{" "}
          {formatData(status.ultima_votacao)}. <Link href="/metodologia">Como calculamos</Link>.
        </p>
      </section>
    </>
  );
}
