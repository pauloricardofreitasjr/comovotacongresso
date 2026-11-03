import { describe, expect, it } from "vitest";
import { ordenar } from "./partidos";
import type { Partido } from "./types";

const m = (presenca: number | null, governo: number | null) => ({
  presenca_sessao: { sessoes_elegiveis: 10, sessoes_presente: 5, pct: presenca },
  participacao_votacoes_nominais: { votacoes_elegiveis: 10, votacoes_votou: 5, pct: 50 },
  alinhamento_governo: { votacoes_com_orientacao: 10, votacoes_alinhadas: 5, pct: governo },
});
const p = (partido: string, deputados: number, presenca: number | null, governo: number | null): Partido => ({
  casa: "camara", partido, parlamentares: deputados, metricas: m(presenca, governo),
});
const nomes = (l: Partido[]) => l.map((x) => x.partido).join(",");

const dados = [p("B", 10, 90, 20), p("A", 50, 80, null), p("C", 5, 95, 70), p("D", 5, null, 70)];

describe("ordenar partidos", () => {
  it("bancada: maior primeiro, empate pelo nome", () => {
    expect(nomes(ordenar(dados, "deputados", false))).toBe("A,B,C,D");
  });
  it("métrica: maior primeiro, e partido sem dado fica por último nos dois sentidos", () => {
    expect(nomes(ordenar(dados, "presenca_sessao", false))).toBe("C,B,A,D");
    expect(nomes(ordenar(dados, "presenca_sessao", true))).toBe("A,B,C,D");
    expect(nomes(ordenar(dados, "governismo", false))).toBe("C,D,B,A");
    expect(nomes(ordenar(dados, "governismo", true))).toBe("B,C,D,A");
  });
  it("nome: A–Z, invertido Z–A", () => {
    expect(nomes(ordenar(dados, "nome", false))).toBe("A,B,C,D");
    expect(nomes(ordenar(dados, "nome", true))).toBe("D,C,B,A");
  });
  it("não altera a lista original", () => {
    const antes = nomes(dados);
    ordenar(dados, "nome", true);
    expect(nomes(dados)).toBe(antes);
  });
});
