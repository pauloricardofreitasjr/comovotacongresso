import { COR, OG_SIZE, OG_TIPO, quadro } from "@/lib/og";

export const alt = "Como Vota Congresso: veja como cada deputado federal votou no Plenário da Câmara";
export const size = OG_SIZE;
export const contentType = OG_TIPO;

export default function Image() {
  return quadro(
    <div style={{ display: "flex", flexDirection: "column" }}>
      <div style={{ display: "flex", fontSize: 92, fontWeight: 800, lineHeight: 1.05, letterSpacing: -2 }}>Como seus deputados votam</div>
      <div style={{ display: "flex", fontSize: 38, color: COR.suave, marginTop: 24 }}>
        Veja, votação por votação, como cada deputado federal votou no Plenário da Câmara. Dados oficiais, regras abertas.
      </div>
    </div>,
  );
}
