import type { Metadata } from "next";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatData, formatDataHora, formatInt, inteiroPositivo, infoVoto, rotuloAlinhamento, rotuloOrientacao } from "@/lib/format";
import { Aviso, Foto, limpar, Metrica, Paginacao, VotoChip } from "@/components/ui";
import { CREDITO_FOTO } from "@/lib/creditos";
import { Compartilhar } from "@/components/Compartilhar";
import { metaCompartilhavel } from "@/lib/compartilhar";
import { SITE_URL } from "@/lib/site";

type Props = { params: Promise<{ id: string }>; searchParams: Promise<Record<string, string | string[] | undefined>> };

const LIMITE_VOTOS = 20;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id } = await params;
  const p = await api.parlamentar(id, { limite_votos: 1 });
  return metaCompartilhavel({
    titulo: p.nome,
    descricao: `Como ${p.nome} votou no Plenário da Câmara: presença, votos e alinhamento com o governo.`,
    caminho: `/parlamentares/${encodeURIComponent(id)}`,
  });
}

export default async function Parlamentar({ params, searchParams }: Props) {
  const [{ id }, sp] = await Promise.all([params, searchParams]);
  const pagina = inteiroPositivo(sp.page_votos, 1);
  const [p, status] = await Promise.all([api.parlamentar(id, { page_votos: pagina, limite_votos: LIMITE_VOTOS }), api.status()]);
  const medias = status.medias_camara;
  const m = p.metricas;
  const votosNaoInformados = p.votos.dados.some((v) => v.voto === "outro");

  return (
    <>
      <p className="breadcrumb">
        <Link href="/parlamentares">Deputados</Link> › {p.nome}
      </p>
      <div className="perfil-topo card">
        <figure className="perfil-foto">
          <Foto nome={p.nome} url={p.url_foto} tamanho={88} />
          {p.url_foto ? <figcaption>{CREDITO_FOTO}</figcaption> : null}
        </figure>
        <div>
          <h1 style={{ marginBottom: ".25rem" }}>{p.nome}</h1>
          <div className="meta">
            <span>{[p.partido, p.uf].filter(Boolean).join(" · ")}</span>
            <span className={`badge ${p.em_exercicio ? "aprovada" : "neutra"}`}>{p.em_exercicio ? "Em exercício" : "Fora de exercício"}</span>
          </div>
          <Compartilhar url={`${SITE_URL}/parlamentares/${encodeURIComponent(id)}`} titulo={p.nome} texto={`Como ${p.nome} votou na Câmara dos Deputados:`} />
        </div>
      </div>

      <h2>Desempenho no Plenário</h2>
      <p className="muted">
        Desde fevereiro de 2023. Estas são três medidas diferentes — <Link href="/metodologia">entenda como calculamos</Link>.
      </p>
      <div className="grid cols-3">
        <Metrica
          tom="presenca" media={medias.presenca_sessao}
          rotulo="Presença em sessões"
          pct={m.presenca_sessao.pct}
          base={`${formatInt(m.presenca_sessao.sessoes_presente)} de ${formatInt(m.presenca_sessao.sessoes_elegiveis)} sessões deliberativas`}
          ajuda="Registro oficial de presença nas sessões em que ele estava em exercício."
        />
        <Metrica
          tom="voto" media={medias.participacao_votacoes_nominais}
          rotulo="Votos nas votações nominais"
          pct={m.participacao_votacoes_nominais.pct}
          base={`${formatInt(m.participacao_votacoes_nominais.votacoes_votou)} de ${formatInt(m.participacao_votacoes_nominais.votacoes_elegiveis)} votações`}
          ajuda="Estar na sessão não é o mesmo que votar: aqui contamos só os votos registrados."
        />
        <Metrica
          tom="governo" media={medias.alinhamento_governo}
          rotulo="Alinhamento com o Governo"
          pct={m.alinhamento_governo.pct}
          base={`${formatInt(m.alinhamento_governo.votacoes_alinhadas)} de ${formatInt(m.alinhamento_governo.votacoes_com_orientacao)} votações com orientação`}
          ajuda="Votos iguais à orientação do Governo naquele dia. Não indica se o voto foi bom ou ruim."
        />
      </div>

      <p className="muted" style={{ marginTop: ".75rem" }}>
        O traço na barra marca a média do conjunto da Câmara (todas as sessões e votações somadas), para dar uma referência. Estar acima ou abaixo dela não é bom nem ruim por si só.
      </p>

      <h2>Histórico de votos</h2>
      <p className="muted">Votações nominais em que registrou voto, da mais recente para a mais antiga.</p>
      {votosNaoInformados ? (
        <Aviso atencao>
          <p>Em algumas votações a Câmara não informou a posição do deputado; elas aparecem como “Voto não informado”.</p>
        </Aviso>
      ) : null}
      <ul className="historico">
        {p.votos.dados.map((v) => (
          <li key={v.votacao_id} className={`hist-item ${infoVoto(v.voto).classe}`}>
            <div className="hist-topo">
              <div>
                <Link className="hist-titulo" href={`/votacoes/${encodeURIComponent(v.votacao_id)}`}>{v.proposicao ? v.proposicao.identificacao : "Votação"}</Link>
                <small className="hist-data">{formatData(v.data_hora)}</small>
              </div>
              <VotoChip voto={v.voto} />
            </div>
            <p className="clamp hist-desc">{limpar(v.descricao)}</p>
            <div className="meta">
              <span>Governo orientou: <strong>{rotuloOrientacao(v.orientacao_governo)}</strong></span>
              {v.alinhou_governo !== null ? (
                <span className={`badge ${v.alinhou_governo ? "aprovada" : "contra"}`}>{rotuloAlinhamento(v.alinhou_governo)}</span>
              ) : null}
            </div>
          </li>
        ))}
      </ul>
      <Paginacao
        pagina={p.votos.pagina} limite={p.votos.limite} total={p.votos.total} caminho={`/parlamentares/${encodeURIComponent(id)}`}
        params={{}} nomePagina="page_votos"
      />

      {p.exercicio.length > 0 ? (
        <details style={{ marginTop: "2rem" }}>
          <summary>Períodos de exercício</summary>
          <ul>
            {p.exercicio.map((e) => (
              <li key={e.inicio}>
                {formatDataHora(e.inicio)} → {e.fim ? formatDataHora(e.fim) : "em exercício"}
                {e.condicao ? ` · ${e.condicao}` : ""}
                {e.partido ? ` · ${e.partido}` : ""}
              </li>
            ))}
          </ul>
        </details>
      ) : null}
    </>
  );
}
