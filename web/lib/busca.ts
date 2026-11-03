// Interpretação da caixa de busca de proposições: "PL 186/2026" busca pela identificação,
// qualquer outro texto busca na ementa. Funções puras, cobertas por teste.

export const TIPOS_CONHECIDOS = ["PL", "PEC", "PLP", "PLV", "MPV", "PDL", "PRC", "PLN", "PDC", "PDS"];

export type Identificacao = { tipo: string; numero: number; ano?: number };

const PADRAO = /^\s*([A-Za-z]{2,4})\.?\s*n?[º°]?\s*(\d{1,6})(?:\s*\/\s*(\d{2}|\d{4}))?\s*$/;

/** Devolve a identificação se o texto for "TIPO número[/ano]" de um tipo conhecido; senão null. */
export function interpretarIdentificacao(texto: string): Identificacao | null {
  const m = PADRAO.exec(texto);
  if (!m) return null;
  const tipo = m[1].toUpperCase();
  const numero = Number.parseInt(m[2], 10);
  if (!TIPOS_CONHECIDOS.includes(tipo) || numero < 1) return null;
  if (m[3] === undefined) return { tipo, numero };
  const ano = m[3].length === 2 ? 2000 + Number.parseInt(m[3], 10) : Number.parseInt(m[3], 10);
  return { tipo, numero, ano };
}
