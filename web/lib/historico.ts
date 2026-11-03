// Filtros do histórico de votos no perfil do deputado. Fora da página para poder ser testado.
import { primeiro } from "./format";

export const VOTOS_FILTRAVEIS = [
  { valor: "sim", rotulo: "Sim" },
  { valor: "nao", rotulo: "Não" },
  { valor: "abstencao", rotulo: "Abstenção" },
  { valor: "obstrucao", rotulo: "Obstrução" },
  { valor: "outro", rotulo: "Voto não informado" },
] as const;

export const ALINHAMENTOS = [
  { valor: "alinhado", rotulo: "Votou como o Governo orientou" },
  { valor: "nao_alinhado", rotulo: "Votou contra a orientação do Governo" },
] as const;

export type FiltrosHistorico = { voto: string; alinhamento: string };

/** Lê os filtros da URL; valor desconhecido vira "sem filtro" (a API recusaria com 400). */
export function lerFiltros(sp: Record<string, string | string[] | undefined>): FiltrosHistorico {
  const voto = primeiro(sp.voto);
  const alinhamento = primeiro(sp.alinhamento);
  return {
    voto: VOTOS_FILTRAVEIS.some((v) => v.valor === voto) ? voto : "",
    alinhamento: ALINHAMENTOS.some((a) => a.valor === alinhamento) ? alinhamento : "",
  };
}
