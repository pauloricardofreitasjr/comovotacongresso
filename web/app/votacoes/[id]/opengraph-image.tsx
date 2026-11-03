import { api } from "@/lib/api";
import { resumir } from "@/lib/compartilhar";
import { formatInt, formatPct, formatData, rotuloResultado } from "@/lib/format";
import { CaixaNumero, COR, Etiqueta, OG_SIZE, OG_TIPO, quadro } from "@/lib/og";
import { limpar } from "@/components/ui";

export const alt = "Placar da votação no Plenário da Câmara dos Deputados";
export const size = OG_SIZE;
export const contentType = OG_TIPO;

export default async function Image({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const v = await api.votacao(id).catch(() => null);
  if (!v) return quadro(<div style={{ display: "flex", fontSize: 72, fontWeight: 800 }}>Votação no Plenário da Câmara</div>);

  const r = v.resumo_votos;
  const resultado = v.aprovada === null ? COR.neutroFundo : v.aprovada ? COR.simFundo : COR.naoFundo;
  const corResultado = v.aprovada === null ? COR.suave : v.aprovada ? COR.sim : COR.nao;
  return quadro(
    <div style={{ display: "flex", flexDirection: "column" }}>
      <div style={{ display: "flex", alignItems: "center", marginBottom: 10 }}>
        <Etiqueta texto={rotuloResultado(v.aprovada)} fundo={resultado} cor={corResultado} />
        <div style={{ display: "flex", fontSize: 28, color: COR.suave }}>
          {v.tipo === "nominal" ? "Votação nominal" : "Votação simbólica"} · {formatData(v.data_hora)}
        </div>
      </div>
      <div style={{ display: "flex", fontSize: 68, fontWeight: 800, letterSpacing: -1 }}>{v.proposicao ? v.proposicao.identificacao : "Votação"}</div>
      <div style={{ display: "flex", fontSize: 28, color: COR.suave, margin: "8px 0 22px" }}>{resumir(limpar(v.descricao), 120)}</div>
      {r ? (
        <div style={{ display: "flex" }}>
          <CaixaNumero numero={formatInt(r.sim)} rotulo={`votaram Sim · ${formatPct(r.total > 0 ? (r.sim / r.total) * 100 : null)}`} fundo={COR.simFundo} cor={COR.sim} />
          <div style={{ display: "flex", width: 20 }} />
          <CaixaNumero numero={formatInt(r.nao)} rotulo={`votaram Não · ${formatPct(r.total > 0 ? (r.nao / r.total) * 100 : null)}`} fundo={COR.naoFundo} cor={COR.nao} />
        </div>
      ) : (
        <div style={{ display: "flex", fontSize: 32, color: COR.suave }}>Votação simbólica: não há registro do voto de cada deputado.</div>
      )}
    </div>,
  );
}
