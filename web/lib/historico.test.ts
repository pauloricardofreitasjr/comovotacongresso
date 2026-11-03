import { describe, expect, it } from "vitest";
import { lerFiltros } from "./historico";

describe("lerFiltros", () => {
  it("aceita só valores conhecidos", () => {
    expect(lerFiltros({ voto: "nao", alinhamento: "nao_alinhado" })).toEqual({ voto: "nao", alinhamento: "nao_alinhado" });
  });
  it("valor desconhecido ou ausente vira sem filtro", () => {
    expect(lerFiltros({ voto: "talvez", alinhamento: "metade" })).toEqual({ voto: "", alinhamento: "" });
    expect(lerFiltros({})).toEqual({ voto: "", alinhamento: "" });
  });
  it("usa o primeiro valor quando vem repetido", () => {
    expect(lerFiltros({ voto: ["sim", "nao"] })).toEqual({ voto: "sim", alinhamento: "" });
  });
});
