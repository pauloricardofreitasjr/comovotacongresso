import { describe, expect, it } from "vitest";
import {
  formatPct, formatInt, formatData, formatDataHora, plural, iniciais, infoVoto, rotuloOrientacao,
  rotuloResultado, rotuloAlinhamento, query, primeiro, inteiroPositivo,
} from "./format";

describe("formatPct", () => {
  it("usa vírgula e no máximo 1 casa", () => {
    expect(formatPct(79.1)).toBe("79,1%");
    expect(formatPct(50)).toBe("50%");
    expect(formatPct(66.67)).toBe("66,7%");
    expect(formatPct(0)).toBe("0%");
  });
  it("mostra traço quando não há dado (nunca 0%)", () => {
    expect(formatPct(null)).toBe("—");
    expect(formatPct(undefined)).toBe("—");
  });
});

describe("datas no fuso de Brasília", () => {
  it("não muda o dia por causa do UTC", () => {
    // 01:30 UTC de 15/10 ainda é 22:30 de 14/10 em Brasília
    expect(formatData("2026-10-15T01:30:00Z")).toBe("14/10/2026");
    expect(formatDataHora("2026-10-15T01:30:00Z")).toBe("14/10/2026, 22:30");
  });
  it("trata ausência", () => {
    expect(formatData(null)).toBe("—");
    expect(formatDataHora(undefined)).toBe("—");
  });
});

describe("textos", () => {
  it("plural", () => {
    expect(plural(1, "votação", "votações")).toBe("1 votação");
    expect(plural(0, "votação", "votações")).toBe("0 votações");
    expect(plural(1234, "voto", "votos")).toBe("1.234 votos");
  });
  it("formatInt", () => expect(formatInt(436085)).toBe("436.085"));
  it("iniciais", () => {
    expect(iniciais("João Ávila")).toBe("JÁ");
    expect(iniciais("Pinheirinho")).toBe("P");
    expect(iniciais("  maria  da silva ")).toBe("MS");
    expect(iniciais("")).toBe("?");
  });
});

describe("votos e rótulos", () => {
  it("cada voto tem rótulo textual", () => {
    expect(infoVoto("sim")).toEqual({ rotulo: "Sim", classe: "sim" });
    expect(infoVoto("nao").rotulo).toBe("Não");
    expect(infoVoto("abstencao").classe).toBe("neutro");
    expect(infoVoto("artigo_17").rotulo).toBe("Presidente da sessão");
  });
  it("voto desconhecido não é apresentado como posição", () => {
    expect(infoVoto("outro")).toEqual({ rotulo: "Voto não informado", classe: "desconhecido" });
    expect(infoVoto("qualquer")).toEqual({ rotulo: "Voto não informado", classe: "desconhecido" });
  });
  it("orientação, resultado e alinhamento", () => {
    expect(rotuloOrientacao("liberado")).toBe("Liberado");
    expect(rotuloOrientacao(null)).toBe("Sem orientação");
    expect(rotuloResultado(true)).toBe("Aprovada");
    expect(rotuloResultado(false)).toBe("Rejeitada");
    expect(rotuloResultado(null)).toBe("Resultado não informado");
    expect(rotuloAlinhamento(true)).toBe("Alinhado");
    expect(rotuloAlinhamento(false)).toBe("Não alinhado");
    expect(rotuloAlinhamento(null)).toBe("—");
  });
});

describe("query e searchParams", () => {
  it("ignora vazios e codifica", () => {
    expect(query({ a: "x y", b: "", c: null, d: undefined, e: 0 })).toBe("?a=x+y&e=0");
    expect(query({ a: "" })).toBe("");
  });
  it("primeiro", () => {
    expect(primeiro(["a", "b"])).toBe("a");
    expect(primeiro(undefined)).toBe("");
    expect(primeiro("z")).toBe("z");
  });
  it("inteiroPositivo cai no padrão se inválido", () => {
    expect(inteiroPositivo("3", 1)).toBe(3);
    expect(inteiroPositivo("0", 1)).toBe(1);
    expect(inteiroPositivo("-2", 1)).toBe(1);
    expect(inteiroPositivo("abc", 7)).toBe(7);
    expect(inteiroPositivo(undefined, 5)).toBe(5);
  });
});
