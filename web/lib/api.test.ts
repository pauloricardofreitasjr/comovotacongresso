import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({
  notFound: () => {
    throw new Error("NEXT_NOT_FOUND");
  },
}));

import { api, apiBase, ApiError } from "./api";

const fetchMock = vi.fn();
beforeEach(() => vi.stubGlobal("fetch", fetchMock));
afterEach(() => {
  fetchMock.mockReset();
  vi.unstubAllGlobals();
  delete process.env.API_URL;
});

const json = (status: number, body: unknown) => new Response(JSON.stringify(body), { status });

describe("apiBase", () => {
  it("usa API_URL sem barra final, com padrão local", () => {
    expect(apiBase()).toBe("http://localhost:8080");
    process.env.API_URL = "http://api:8080//";
    expect(apiBase()).toBe("http://api:8080");
  });
});

describe("cliente da API", () => {
  it("monta a URL com query, ignorando vazios, e pede JSON", async () => {
    fetchMock.mockResolvedValue(json(200, { dados: [], pagina: 1, limite: 20, total: 0 }));
    await api.proposicoes({ tipo: "PL", busca: "", ano: 2025, pagina: 2 });
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("http://localhost:8080/api/v1/proposicoes?tipo=PL&ano=2025&pagina=2");
    expect(init.headers.Accept).toBe("application/json");
    expect(init.next.revalidate).toBe(300);
  });

  it("codifica o id no caminho", async () => {
    fetchMock.mockResolvedValue(json(200, {}));
    await api.votacao("2635476-36/../x");
    expect(fetchMock.mock.calls[0][0]).toContain("/votacoes/2635476-36%2F..%2Fx");
  });

  it("404 vira notFound() do Next", async () => {
    fetchMock.mockResolvedValue(json(404, { erro: { codigo: "nao_encontrado", mensagem: "x" } }));
    await expect(api.proposicao("9")).rejects.toThrow("NEXT_NOT_FOUND");
  });

  it("id malformado (400) em consulta por id também é notFound, mas em listagem continua erro", async () => {
    fetchMock.mockResolvedValue(json(400, { erro: { codigo: "parametro_invalido", mensagem: "id" } }));
    await expect(api.parlamentar("abc")).rejects.toThrow("NEXT_NOT_FOUND");
    await expect(api.proposicao("abc")).rejects.toThrow("NEXT_NOT_FOUND");
    await expect(api.votacao("abc")).rejects.toThrow("NEXT_NOT_FOUND");
    await expect(api.parlamentares({ limite: 0 })).rejects.toBeInstanceOf(ApiError);
  });

  it("outros erros carregam código e mensagem da API", async () => {
    fetchMock.mockResolvedValue(json(400, { erro: { codigo: "parametro_invalido", mensagem: "limite inválido" } }));
    const err = await api.parlamentares({ limite: 0 }).catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 400, codigo: "parametro_invalido", message: "limite inválido" });
  });

  it("corpo não-JSON em erro não quebra", async () => {
    fetchMock.mockResolvedValue(new Response("<html>", { status: 502 }));
    await expect(api.status()).rejects.toMatchObject({ status: 502, codigo: "erro" });
  });

  it("API fora do ar vira ApiError 503", async () => {
    fetchMock.mockRejectedValue(new TypeError("fetch failed"));
    await expect(api.status()).rejects.toMatchObject({ status: 503, codigo: "api_indisponivel" });
  });

  it("partidos devolve só a lista", async () => {
    fetchMock.mockResolvedValue(json(200, { dados: [{ partido: "PT" }] }));
    expect(await api.partidos()).toEqual([{ partido: "PT" }]);
    expect(fetchMock.mock.calls[0][0]).toContain("casa=camara");
  });
});
