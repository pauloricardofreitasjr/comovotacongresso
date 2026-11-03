import { describe, expect, it } from "vitest";
import { interpretarIdentificacao } from "./busca";

describe("interpretarIdentificacao", () => {
  it("reconhece TIPO número/ano", () => {
    expect(interpretarIdentificacao("PL 186/2026")).toEqual({ tipo: "PL", numero: 186, ano: 2026 });
    expect(interpretarIdentificacao("pec 45/2019")).toEqual({ tipo: "PEC", numero: 45, ano: 2019 });
    expect(interpretarIdentificacao("  PLP186/2026  ")).toEqual({ tipo: "PLP", numero: 186, ano: 2026 });
  });
  it("aceita ano com dois dígitos e número sem ano", () => {
    expect(interpretarIdentificacao("PL 1234/23")).toEqual({ tipo: "PL", numero: 1234, ano: 2023 });
    expect(interpretarIdentificacao("PEC 45")).toEqual({ tipo: "PEC", numero: 45 });
  });
  it("aceita pontuação comum", () => {
    expect(interpretarIdentificacao("P.L. 10/2020")).toBeNull(); // pontos internos não são suportados
    expect(interpretarIdentificacao("PL nº 10/2020")).toEqual({ tipo: "PL", numero: 10, ano: 2020 });
    expect(interpretarIdentificacao("PL. 10 / 2020")).toEqual({ tipo: "PL", numero: 10, ano: 2020 });
  });
  it("texto livre nunca vira identificação", () => {
    for (const t of ["imposto de renda", "lei 123", "reforma", "PL", "123", "", "PL 0/2020", "XYZ 10/2020", "educação 2026"]) {
      expect(interpretarIdentificacao(t), t).toBeNull();
    }
  });
});
