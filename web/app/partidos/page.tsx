import type { Metadata } from "next";
import { api } from "@/lib/api";
import { formatInt, formatPct } from "@/lib/format";

export const metadata: Metadata = { title: "Partidos" };

export default async function Partidos() {
  const partidos = await api.partidos();
  return (
    <>
      <h1>Partidos</h1>
      <p className="lead">
        As mesmas medidas, agrupadas por partido. Cada voto ou presença conta para o partido em que o deputado estava naquele dia — quem trocou de
        partido aparece nos dois.
      </p>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Partido</th>
              <th className="num">Deputados</th>
              <th className="num">Presença em sessões</th>
              <th className="num">Votos nas nominais</th>
              <th className="num">Alinhamento com o Governo</th>
            </tr>
          </thead>
          <tbody>
            {partidos.map((p) => (
              <tr key={p.partido}>
                <td><strong>{p.partido}</strong></td>
                <td className="num">{formatInt(p.parlamentares)}</td>
                <td className="num">{formatPct(p.metricas.presenca_sessao.pct)}</td>
                <td className="num">{formatPct(p.metricas.participacao_votacoes_nominais.pct)}</td>
                <td className="num">
                  <strong>{formatPct(p.metricas.alinhamento_governo.pct)}</strong>
                  <small>{formatInt(p.metricas.alinhamento_governo.votacoes_com_orientacao)} votos</small>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="muted" style={{ marginTop: "1rem" }}>
        Partidos pequenos ou extintos têm poucos registros; olhe a quantidade de votos antes de comparar.
      </p>
    </>
  );
}
