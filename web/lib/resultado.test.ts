import { describe, expect, it } from "vitest";
import { divergeDaDescricao, notaResultado, placarDaDescricao, resultadoCoerente } from "./resultado";

describe("placarDaDescricao", () => {
  it("lê Sim e Não em qualquer caixa e acentuação", () => {
    expect(placarDaDescricao("Rejeitada a Emenda. Sim: 35; não: 375; total: 410.")).toEqual({ sim: 35, nao: 375 });
    expect(placarDaDescricao("Aprovada. SIM: 461; NÃO: 4; Total: 465.")).toEqual({ sim: 461, nao: 4 });
    expect(placarDaDescricao("Aprovada. Sim: 10; Nao: 2.")).toEqual({ sim: 10, nao: 2 });
  });
  it("devolve nulo quando a descrição não traz placar", () => {
    expect(placarDaDescricao("Aprovado o requerimento.")).toBeNull();
    expect(placarDaDescricao("Sim: 10; Abstenção: 2")).toBeNull();
  });
});

describe("divergeDaDescricao", () => {
  const desc = "Rejeitada a Emenda de Plenário nº 40. Sim: 35; não: 375; total: 410.";
  it("devolve o placar da descrição quando difere dos votos", () => {
    expect(divergeDaDescricao(desc, { sim: 315, nao: 87 })).toEqual({ sim: 35, nao: 375 });
  });
  it("nulo quando bate, quando não há placar na descrição ou quando não há Sim/Não registrados", () => {
    expect(divergeDaDescricao(desc, { sim: 35, nao: 375 })).toBeNull();
    expect(divergeDaDescricao("Aprovado o requerimento.", { sim: 315, nao: 87 })).toBeNull();
    expect(divergeDaDescricao("Aprovado. Sim: 404; Não: 61.", { sim: 0, nao: 0 })).toBeNull();
  });
});

describe("notaResultado", () => {
  it("rejeitada com mais Sim: explica e mostra quantos Sim faltaram para a maioria absoluta", () => {
    const n = notaResultado(false, { sim: 252, nao: 176 });
    expect(n?.titulo).toMatch(/rejeitada/);
    expect(n?.texto).toContain("faltaram 5 votos Sim para os 257");
  });
  it("não fala em 'faltaram' quando já havia 257 Sim ou mais", () => {
    expect(notaResultado(false, { sim: 300, nao: 100 })?.texto).not.toContain("faltaram");
  });
  it("aprovada sem mais Sim (empate): nota própria", () => {
    expect(notaResultado(true, { sim: 182, nao: 182 })?.titulo).toMatch(/aprovada/);
  });
  it("sem nota quando o resultado segue o placar, não há resultado ou não há Sim/Não", () => {
    expect(notaResultado(true, { sim: 461, nao: 4 })).toBeNull();
    expect(notaResultado(false, { sim: 11, nao: 387 })).toBeNull();
    expect(notaResultado(null, { sim: 252, nao: 176 })).toBeNull();
    expect(notaResultado(true, { sim: 0, nao: 0 })).toBeNull();
    expect(notaResultado(false, { sim: 0, nao: 0 })).toBeNull();
  });
});

describe("resultadoCoerente", () => {
  it("compara o resultado oficial com 'mais Sim que Não'", () => {
    expect(resultadoCoerente(false, { sim: 35, nao: 375 })).toBe(true);
    expect(resultadoCoerente(false, { sim: 315, nao: 87 })).toBe(false);
    expect(resultadoCoerente(true, { sim: 461, nao: 4 })).toBe(true);
  });
  it("nulo sem resultado ou em empate", () => {
    expect(resultadoCoerente(null, { sim: 1, nao: 2 })).toBeNull();
    expect(resultadoCoerente(true, { sim: 182, nao: 182 })).toBeNull();
  });
});
