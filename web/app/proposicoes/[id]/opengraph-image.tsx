import { api } from "@/lib/api";
import { resumir } from "@/lib/compartilhar";
import { formatInt, plural, rotuloResultado } from "@/lib/format";
import { CaixaNumero, COR, Etiqueta, OG_SIZE, OG_TIPO, quadro } from "@/lib/og";
import { limpar } from "@/components/ui";

export const alt = "Lei ou projeto votado no Plenário da Câmara dos Deputados";
export const size = OG_SIZE;
export const contentType = OG_TIPO;

export default async function Image({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const p = await api.proposicao(id).catch(() => null);
  if (!p) return quadro(<div style={{ display: "flex", fontSize: 72, fontWeight: 800 }}>Lei ou projeto votado</div>);

  const destaque = p.votacoes.find((v) => v.principal) ?? p.votacoes[p.votacoes.length - 1];
  const r = destaque?.resumo_votos ?? null;
  return quadro(
    <div style={{ display: "flex", flexDirection: "column" }}>
      <div style={{ display: "flex", fontSize: 76, fontWeight: 800, letterSpacing: -2 }}>{p.identificacao}</div>
      <div style={{ display: "flex", fontSize: 30, color: COR.suave, margin: "8px 0 22px" }}>{resumir(limpar(p.ementa), 150)}</div>
      {r && destaque ? (
        <div style={{ display: "flex", flexDirection: "column" }}>
          <div style={{ display: "flex", alignItems: "center", marginBottom: 14 }}>
            <Etiqueta
              texto={rotuloResultado(destaque.aprovada)}
              fundo={destaque.aprovada === null ? COR.neutroFundo : destaque.aprovada ? COR.simFundo : COR.naoFundo}
              cor={destaque.aprovada === null ? COR.suave : destaque.aprovada ? COR.sim : COR.nao}
            />
            <div style={{ display: "flex", fontSize: 26, color: COR.suave }}>Votação nominal {destaque.principal ? "principal" : "mais recente"}</div>
          </div>
          <div style={{ display: "flex" }}>
            <CaixaNumero numero={formatInt(r.sim)} rotulo="votaram Sim" fundo={COR.simFundo} cor={COR.sim} />
            <div style={{ display: "flex", width: 20 }} />
            <CaixaNumero numero={formatInt(r.nao)} rotulo="votaram Não" fundo={COR.naoFundo} cor={COR.nao} />
          </div>
        </div>
      ) : (
        <div style={{ display: "flex", fontSize: 30, color: COR.suave }}>
          {plural(p.total_votacoes, "votação", "votações")} no Plenário ·{" "}
          {destaque?.tipo === "nominal" ? "votos individuais ocultos por erro na fonte." : "só votação simbólica, sem voto de cada deputado."}
        </div>
      )}
    </div>,
  );
}
