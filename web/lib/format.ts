// Formatação para o público (pt-BR, fuso de Brasília). Funções puras, cobertas por teste.

const TZ = "America/Sao_Paulo";

export function formatPct(v: number | null | undefined): string {
  if (v === null || v === undefined) return "—";
  return `${v.toLocaleString("pt-BR", { minimumFractionDigits: 0, maximumFractionDigits: 1 })}%`;
}

export function formatInt(n: number): string {
  return n.toLocaleString("pt-BR");
}

export function formatData(iso: string | null | undefined): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("pt-BR", { timeZone: TZ, day: "2-digit", month: "2-digit", year: "numeric" });
}

export function formatDataHora(iso: string | null | undefined): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("pt-BR", {
    timeZone: TZ,
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/** "1 votação" / "3 votações". */
export function plural(n: number, singular: string, pluralForm: string): string {
  return `${formatInt(n)} ${n === 1 ? singular : pluralForm}`;
}

export function iniciais(nome: string): string {
  const partes = nome.trim().split(/\s+/).filter(Boolean);
  if (partes.length === 0) return "?";
  const primeira = partes[0][0];
  const ultima = partes.length > 1 ? partes[partes.length - 1][0] : "";
  return (primeira + ultima).toUpperCase();
}

export type VotoInfo = { rotulo: string; classe: "sim" | "nao" | "neutro" | "presidente" | "desconhecido" };

/** Rótulo e categoria visual de um voto. Nunca depende só de cor: o rótulo sempre aparece. */
export function infoVoto(voto: string): VotoInfo {
  switch (voto) {
    case "sim":
      return { rotulo: "Sim", classe: "sim" };
    case "nao":
      return { rotulo: "Não", classe: "nao" };
    case "abstencao":
      return { rotulo: "Abstenção", classe: "neutro" };
    case "obstrucao":
      return { rotulo: "Obstrução", classe: "neutro" };
    case "artigo_17":
      return { rotulo: "Presidente da sessão", classe: "presidente" };
    default:
      return { rotulo: "Voto não informado", classe: "desconhecido" };
  }
}

export function rotuloOrientacao(o: string | null): string {
  switch (o) {
    case "sim":
      return "Sim";
    case "nao":
      return "Não";
    case "liberado":
      return "Liberado";
    case "obstrucao":
      return "Obstrução";
    default:
      return "Sem orientação";
  }
}

export function rotuloResultado(aprovada: boolean | null): string {
  if (aprovada === null) return "Resultado não informado";
  return aprovada ? "Aprovada" : "Rejeitada";
}

export function rotuloAlinhamento(a: boolean | null): string {
  if (a === null) return "—";
  return a ? "Alinhado" : "Não alinhado";
}

/** Monta uma query string ignorando valores vazios. */
export function query(params: Record<string, string | number | null | undefined>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== null && v !== undefined && String(v) !== "") q.set(k, String(v));
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

/** Primeiro valor de um searchParam do Next (que pode ser string, array ou ausente). */
export function primeiro(v: string | string[] | undefined): string {
  return (Array.isArray(v) ? v[0] : v) ?? "";
}

/** Inteiro positivo de um searchParam; padrão se ausente/inválido. */
export function inteiroPositivo(v: string | string[] | undefined, padrao: number): number {
  const n = Number.parseInt(primeiro(v), 10);
  return Number.isFinite(n) && n > 0 ? n : padrao;
}

/** Compara o percentual de um deputado com o do conjunto da Câmara, em pontos percentuais. */
export function compararComMedia(pct: number, media: number): string {
  const d = Math.round((pct - media) * 10) / 10;
  if (Math.abs(d) < 0.5) return "Na média da Câmara";
  const pontos = `${Math.abs(d).toLocaleString("pt-BR", { maximumFractionDigits: 1 })} ponto${Math.abs(d) < 2 ? "" : "s"}`;
  return d > 0 ? `${pontos} acima da média da Câmara` : `${pontos} abaixo da média da Câmara`;
}
