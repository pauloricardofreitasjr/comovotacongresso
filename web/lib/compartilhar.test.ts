import { describe, expect, it } from "vitest";
import { linkWhatsApp, metaCompartilhavel, resumir } from "./compartilhar";

describe("resumir", () => {
  it("não mexe em texto curto e normaliza espaços", () => {
    expect(resumir("  Altera   a lei\r\n 123 ", 50)).toBe("Altera a lei 123");
  });
  it("corta em palavra inteira, com reticências, dentro do limite", () => {
    const r = resumir("Dispõe sobre o regime de resolução das instituições autorizadas a funcionar", 40);
    expect(r.length).toBeLessThanOrEqual(40);
    expect(r.endsWith("…")).toBe(true);
    expect(r).toBe("Dispõe sobre o regime de resolução das…");
  });
});

describe("linkWhatsApp", () => {
  it("codifica o texto e o link", () => {
    expect(linkWhatsApp("PL 1/2025: 10 Sim × 2 Não", "https://x.br/a?b=1")).toBe(
      "https://wa.me/?text=PL%201%2F2025%3A%2010%20Sim%20%C3%97%202%20N%C3%A3o%20https%3A%2F%2Fx.br%2Fa%3Fb%3D1",
    );
  });
});

describe("metaCompartilhavel", () => {
  it("define canônico, Open Graph e cartão grande", () => {
    const m = metaCompartilhavel({ titulo: "PL 1/2025", descricao: "desc", caminho: "/proposicoes/1" });
    expect(m.alternates?.canonical).toBe("/proposicoes/1");
    expect(m.openGraph).toMatchObject({ url: "/proposicoes/1", siteName: "Como Vota Congresso", locale: "pt_BR" });
    expect(m.twitter).toMatchObject({ card: "summary_large_image" });
  });
});
