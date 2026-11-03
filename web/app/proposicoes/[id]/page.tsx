import type { Metadata } from "next";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatDataHora, plural } from "@/lib/format";
import { Aviso, BarraVotos, limpar, ResultadoBadge, TipoBadge } from "@/components/ui";

type Props = { params: Promise<{ id: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id } = await params;
  const p = await api.proposicao(id);
  return { title: p.identificacao, description: limpar(p.ementa).slice(0, 200) };
}

export default async function Proposicao({ params }: Props) {
  const { id } = await params;
  const p = await api.proposicao(id);

  return (
    <>
      <p className="breadcrumb">
        <Link href="/proposicoes">Leis e projetos</Link> › {p.identificacao}
      </p>
      <h1>{p.identificacao}</h1>
      <p className="lead">{limpar(p.ementa)}</p>
      <p className="muted">
        {plural(p.total_votacoes, "votação", "votações")} no Plenário, {p.votacoes_nominais} nominal(is).
      </p>

      <h2>Linha do tempo das votações</h2>
      <p className="muted">
        Uma mesma lei costuma ter várias votações: o texto principal, emendas, destaques, requerimentos. Elas aparecem em ordem cronológica.
      </p>

      <ol className="timeline">
        {p.votacoes.map((v) => (
          <li key={v.id} className={v.principal ? "principal" : undefined}>
            <div className="card stack">
              <div className="meta">
                <strong>{formatDataHora(v.data_hora)}</strong>
                <TipoBadge tipo={v.tipo} />
                <ResultadoBadge aprovada={v.aprovada} />
                {v.principal ? <span className="badge destaque" title="Entre as votações nominais desta matéria, a com mais votos.">Votação principal</span> : null}
              </div>
              <p>{limpar(v.descricao)}</p>
              {v.vinculo_confianca === "provavel" ? (
                <p className="muted">Relação desta votação com este projeto inferida automaticamente; pode não ser exata.</p>
              ) : null}
              {v.resumo_votos ? <BarraVotos resumo={v.resumo_votos} /> : <p className="muted">Votação simbólica: não há voto individual registrado.</p>}
              <p>
                <Link href={`/votacoes/${encodeURIComponent(v.id)}`}>
                  {v.tipo === "nominal" ? "Ver como cada deputado votou →" : "Ver detalhes →"}
                </Link>
              </p>
            </div>
          </li>
        ))}
      </ol>

      {p.votacoes.some((v) => v.tipo === "simbolica") ? (
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
