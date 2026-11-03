import { ImageResponse } from "next/og";
import type { ReactNode } from "react";
import { NOME_SITE } from "./site";

// Imagens de pré-visualização (Open Graph). Satori só entende flexbox: todo contêiner com mais de um filho usa display: flex.
export const OG_SIZE = { width: 1200, height: 630 };
export const OG_TIPO = "image/png";

export const COR = {
  fundo: "#f4f7fd", texto: "#0f1b33", suave: "#4f5e78", azul: "#2563eb",
  sim: "#146c35", simFundo: "#dcf7e5", nao: "#b01c1c", naoFundo: "#fde4e4", neutroFundo: "#e8edf4",
  verdeAgua: "#0b6665", verdeAguaFundo: "#d9f6f4", azulFundo: "#e3edff", ambar: "#8a4a05", ambarFundo: "#fdf0cf",
};

/** Moldura comum: marca no topo, conteúdo no meio e a fonte dos dados no rodapé. */
export function quadro(conteudo: ReactNode): ImageResponse {
  return new ImageResponse(
    (
      <div style={{ display: "flex", flexDirection: "column", width: "100%", height: "100%", background: COR.fundo, padding: "44px 56px", color: COR.texto }}>
        <div style={{ display: "flex", alignItems: "center" }}>
          <div style={{ display: "flex", alignItems: "flex-end", gap: 5, width: 52, height: 52, background: COR.azul, borderRadius: 14, padding: "11px 11px" }}>
            <div style={{ width: 8, height: 14, background: "rgba(255,255,255,.7)", borderRadius: 3 }} />
            <div style={{ width: 8, height: 26, background: "#fff", borderRadius: 3 }} />
            <div style={{ width: 8, height: 30, background: "rgba(255,255,255,.7)", borderRadius: 3 }} />
          </div>
          <div style={{ display: "flex", marginLeft: 16, fontSize: 34, fontWeight: 800 }}>
            Como Vota<span style={{ color: COR.azul, marginLeft: 10 }}>Congresso</span>
          </div>
        </div>
        <div style={{ display: "flex", flexDirection: "column", flex: 1, justifyContent: "center" }}>{conteudo}</div>
        <div style={{ display: "flex", fontSize: 24, color: COR.suave }}>Fonte: Câmara dos Deputados · {NOME_SITE.toLowerCase().replace(/ /g, "")}.com.br</div>
      </div>
    ),
    OG_SIZE,
  );
}

export function Etiqueta({ texto, fundo, cor }: { texto: string; fundo: string; cor: string }) {
  return (
    <div style={{ display: "flex", background: fundo, color: cor, fontSize: 28, fontWeight: 700, padding: "6px 20px", borderRadius: 999, marginRight: 14 }}>{texto}</div>
  );
}

export function CaixaNumero({ numero, rotulo, fundo, cor }: { numero: string; rotulo: string; fundo: string; cor: string }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", flex: 1, background: fundo, color: cor, borderRadius: 24, padding: "16px 30px" }}>
      <div style={{ display: "flex", fontSize: 72, fontWeight: 800, lineHeight: 1 }}>{numero}</div>
      <div style={{ display: "flex", fontSize: 28, fontWeight: 700, marginTop: 4 }}>{rotulo}</div>
    </div>
  );
}
