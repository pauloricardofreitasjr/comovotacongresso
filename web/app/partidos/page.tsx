import type { Metadata } from "next";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatInt, primeiro, query } from "@/lib/format";
import { ORDENS, ordenar, POUCOS_DEPUTADOS, type Chave } from "@/lib/partidos";
import { MiniMetrica } from "@/components/ui";

export const metadata: Metadata = { title: "Partidos" };

export default async function Partidos({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const sp = await searchParams;
  const chave: Chave = ORDENS.some((o) => o.chave === primeiro(sp.ordenarPor)) ? (primeiro(sp.ordenarPor) as Chave) : "deputados";
  const inverter = primeiro(sp.ordem) === "inverter";
  const [partidos, status] = await Promise.all([api.partidos(), api.status()]);
  const medias = status.medias_camara;
  const lista = ordenar(partidos, chave, inverter);
  const ehMetrica = chave !== "deputados" && chave !== "nome";

  return (
    <>
      <h1>Partidos</h1>
      <p className="lead">
        As mesmas medidas, agrupadas por partido. Cada voto ou presença conta para o partido em que o deputado estava naquele dia — quem trocou de
        partido aparece nos dois.
      </p>

      <p className="rotulo-ordem" id="ordenar-partidos">Ordenar por</p>
      <ul className="tabs" aria-labelledby="ordenar-partidos">
        {ORDENS.map((o) => (
          <li key={o.chave}>
            <Link href={`/partidos${query({ ordenarPor: o.chave === "deputados" ? null : o.chave })}`} aria-current={o.chave === chave}>
              {o.rotulo}
            </Link>
          </li>
        ))}
      </ul>
      <p className="muted">
        O traço em cada barra marca a média do conjunto da Câmara, como referência.{" "}
        <Link href={`/partidos${query({ ordenarPor: chave === "deputados" ? null : chave, ordem: inverter ? null : "inverter" })}`}>
          Inverter ordem
        </Link>
      </p>

      <ul className="dep-grid" aria-label="Partidos">
        {lista.map((p) => {
          const m = p.metricas;
          return (
            <li key={p.partido} className="dep-card">
              <div className="partido-topo">
                <div>
                  <strong className="dep-nome">{p.partido}</strong>
                  <small>{formatInt(p.parlamentares)} {p.parlamentares === 1 ? "deputado" : "deputados"} no período</small>
                </div>
                {p.parlamentares < POUCOS_DEPUTADOS ? (
                  <span className="badge neutra" title="Com poucos deputados, os percentuais oscilam muito.">Poucos deputados</span>
                ) : null}
              </div>
              <MiniMetrica
                tom="presenca" rotulo="Presença em sessões" pct={m.presenca_sessao.pct} media={medias.presenca_sessao} ativo={chave === "presenca_sessao"}
                base={`${formatInt(m.presenca_sessao.sessoes_presente)} de ${formatInt(m.presenca_sessao.sessoes_elegiveis)} presenças`}
              />
              <MiniMetrica
                tom="voto" rotulo="Votos nas nominais" pct={m.participacao_votacoes_nominais.pct} media={medias.participacao_votacoes_nominais} ativo={chave === "participacao"}
                base={`${formatInt(m.participacao_votacoes_nominais.votacoes_votou)} de ${formatInt(m.participacao_votacoes_nominais.votacoes_elegiveis)} votos possíveis`}
              />
              <MiniMetrica
                tom="governo" rotulo="Alinhamento com o Governo" pct={m.alinhamento_governo.pct} media={medias.alinhamento_governo} ativo={chave === "governismo"}
                base={`${formatInt(m.alinhamento_governo.votacoes_alinhadas)} de ${formatInt(m.alinhamento_governo.votacoes_com_orientacao)} votos com orientação`}
              />
              <Link href={`/parlamentares${query({ partido: p.partido, em_exercicio: "todos" })}`}>Ver os deputados →</Link>
            </li>
          );
        })}
      </ul>

      {ehMetrica ? (
        <p className="muted" style={{ marginTop: "1rem" }}>
          Partidos com poucos deputados ou extintos têm poucos registros; olhe a base de cada barra antes de comparar.
        </p>
      ) : null}
    </>
  );
}
