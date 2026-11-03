import type { Metadata } from "next";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatDataHora, plural } from "@/lib/format";
import type { VotacaoLinha } from "@/lib/types";
import { Compartilhar } from "@/components/Compartilhar";
import { metaCompartilhavel, resumir } from "@/lib/compartilhar";
import { SITE_URL } from "@/lib/site";
import { Aviso, BarraVotos, limpar, Placar, ResultadoBadge, TipoBadge } from "@/components/ui";

type Props = { params: Promise<{ id: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id } = await params;
  const p = await api.proposicao(id);
  return metaCompartilhavel({ titulo: p.identificacao, descricao: resumir(limpar(p.ementa), 200), caminho: `/proposicoes/${encodeURIComponent(id)}` });
}

/** As votações vêm em ordem cronológica; o destaque é a marcada como principal ou, sem marca, a mais recente. */
function separar(votacoes: VotacaoLinha[]): { destaque: VotacaoLinha | null; outras: VotacaoLinha[] } {
  if (votacoes.length === 0) return { destaque: null, outras: [] };
  const destaque = votacoes.find((v) => v.principal) ?? votacoes[votacoes.length - 1];
  return { destaque, outras: votacoes.filter((v) => v.id !== destaque.id).reverse() };
}

function classeResultado(aprovada: boolean | null): string {
  return aprovada === null ? "neutra" : aprovada ? "aprovada" : "rejeitada";
}

export default async function Proposicao({ params }: Props) {
  const { id } = await params;
  const p = await api.proposicao(id);
  const { destaque, outras } = separar(p.votacoes);
  const haSimbolica = p.votacoes.some((v) => v.tipo === "simbolica");

  return (
    <>
      <header className="votacao-topo" data-resultado="neutra">
        <p className="breadcrumb">
          <Link href="/proposicoes">Leis e projetos</Link> › {p.identificacao}
        </p>
        <h1>{p.identificacao}</h1>
        <p className="lead">{limpar(p.ementa)}</p>
        <div className="meta">
          <span>{plural(p.total_votacoes, "votação", "votações")} no Plenário</span>
          <span>{plural(p.votacoes_nominais, "nominal", "nominais")}</span>
        </div>
        <Compartilhar
          url={`${SITE_URL}/proposicoes/${encodeURIComponent(id)}`}
          titulo={p.identificacao}
          texto={`${p.identificacao}: veja como os deputados votaram.`}
        />
      </header>

      {destaque ? (
        <section className="secao" aria-labelledby="principal">
          <h2 id="principal" className="sr-only">{destaque.principal ? "Votação principal" : "Votação mais recente"}</h2>
          <div className="destaque-voto">
            <div>
              <span className="tag">{destaque.principal ? "Votação principal" : "Votação mais recente"} · {formatDataHora(destaque.data_hora)}</span>
              <div className="meta" style={{ margin: ".5rem 0" }}>
                <ResultadoBadge aprovada={destaque.aprovada} />
                <TipoBadge tipo={destaque.tipo} />
              </div>
              <p style={{ margin: 0 }}>{limpar(destaque.descricao)}</p>
            </div>
            {destaque.vinculo_confianca === "provavel" ? (
              <Aviso atencao>
                <p>A relação desta votação com {p.identificacao} foi inferida automaticamente e pode não ser exata.</p>
              </Aviso>
            ) : null}
            {destaque.resumo_votos ? (
              <Placar resumo={destaque.resumo_votos} />
            ) : (
              <p className="muted" style={{ margin: 0 }}>Votação simbólica: não há voto individual registrado, só o resultado.</p>
            )}
            <p style={{ margin: 0 }}>
              <Link className="button" href={`/votacoes/${encodeURIComponent(destaque.id)}`}>
                {destaque.tipo === "nominal" ? "Ver como cada deputado votou" : "Ver detalhes da votação"}
              </Link>
            </p>
          </div>
        </section>
      ) : null}

      {outras.length > 0 ? (
        <section className="secao" aria-labelledby="outras">
          <h2 id="outras">Outras votações desta lei</h2>
          <p className="muted">
            Uma mesma lei costuma ter várias votações: o texto principal, emendas, destaques, requerimentos. Da mais recente para a mais antiga.
          </p>
          <details className="outras-votacoes" open={outras.length <= 3}>
            <summary>{plural(outras.length, "outra votação", "outras votações")}</summary>
            <ul className="outras-lista">
              {outras.map((v) => (
                <li key={v.id} className={`outra ${classeResultado(v.aprovada)}`}>
                  <div className="meta">
                    <strong>{formatDataHora(v.data_hora)}</strong>
                    <TipoBadge tipo={v.tipo} />
                    <ResultadoBadge aprovada={v.aprovada} />
                  </div>
                  <p className="clamp" style={{ margin: ".35rem 0" }}>{limpar(v.descricao)}</p>
                  {v.resumo_votos ? <BarraVotos resumo={v.resumo_votos} /> : null}
                  <p style={{ margin: ".4rem 0 0" }}>
                    <Link href={`/votacoes/${encodeURIComponent(v.id)}`}>
                      {v.tipo === "nominal" ? "Ver como cada deputado votou →" : "Ver detalhes →"}
                    </Link>
                  </p>
                </li>
              ))}
            </ul>
          </details>
        </section>
      ) : null}

      {haSimbolica ? (
        <Aviso>
          <p>
            <strong>Votação simbólica</strong>: o presidente da sessão pergunta quem é a favor e os deputados levantam ou ficam sentados.
            Não há registro do voto de cada um, por isso só mostramos o resultado.
          </p>
        </Aviso>
      ) : null}
    </>
  );
}
