// Ordenação da página de partidos. Fora da página para poder ser testada.
import type { Partido } from "./types";

export const ORDENS = [
  { chave: "deputados", rotulo: "Tamanho da bancada", valor: (p: Partido) => p.parlamentares },
  { chave: "presenca_sessao", rotulo: "Presença em sessões", valor: (p: Partido) => p.metricas.presenca_sessao.pct },
  { chave: "participacao", rotulo: "Votos nas nominais", valor: (p: Partido) => p.metricas.participacao_votacoes_nominais.pct },
  { chave: "governismo", rotulo: "Alinhamento com o governo", valor: (p: Partido) => p.metricas.alinhamento_governo.pct },
  { chave: "nome", rotulo: "Nome (A–Z)", valor: (p: Partido) => p.partido },
] as const;
export type Chave = (typeof ORDENS)[number]["chave"];

/** Partidos com menos deputados que isso têm poucos registros, e os percentuais oscilam muito. */
export const POUCOS_DEPUTADOS = 5;

export function ordenar(partidos: Partido[], chave: Chave, inverter: boolean): Partido[] {
  const o = ORDENS.find((x) => x.chave === chave)!;
  const sinal = (chave === "nome" ? 1 : -1) * (inverter ? -1 : 1); // métricas: maior primeiro
  return [...partidos].sort((a, b) => {
    const va = o.valor(a);
    const vb = o.valor(b);
    if (va === null && vb === null) return a.partido.localeCompare(b.partido, "pt-BR");
    if (va === null) return 1; // sem dado sempre por último, em qualquer sentido
    if (vb === null) return -1;
    const c = typeof va === "string" ? va.localeCompare(vb as string, "pt-BR") : (va as number) - (vb as number);
    return c === 0 ? a.partido.localeCompare(b.partido, "pt-BR") : sinal * c;
  });
}

