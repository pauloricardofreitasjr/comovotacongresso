import { api } from "@/lib/api";
import { formatInt, formatPct, iniciais } from "@/lib/format";
import { COR, OG_SIZE, OG_TIPO, quadro } from "@/lib/og";

export const alt = "Desempenho de um deputado federal no Plenário da Câmara";
export const size = OG_SIZE;
export const contentType = OG_TIPO;

function Medida({ rotulo, pct, base, fundo, cor }: { rotulo: string; pct: number | null; base: string; fundo: string; cor: string }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", flex: 1, background: fundo, color: cor, borderRadius: 24, padding: "20px 26px" }}>
      <div style={{ display: "flex", alignItems: "flex-start", height: 64, fontSize: 26, fontWeight: 700 }}>{rotulo}</div>
      <div style={{ display: "flex", fontSize: 80, fontWeight: 800, lineHeight: 1.1 }}>{formatPct(pct)}</div>
      <div style={{ display: "flex", fontSize: 22 }}>{pct === null ? "Sem dados suficientes" : base}</div>
    </div>
  );
}

export default async function Image({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const p = await api.parlamentar(id, { limite_votos: 1 }).catch(() => null);
  if (!p) return quadro(<div style={{ display: "flex", fontSize: 72, fontWeight: 800 }}>Deputado federal</div>);
  const m = p.metricas;
  return quadro(
    <div style={{ display: "flex", flexDirection: "column" }}>
      <div style={{ display: "flex", alignItems: "center", marginBottom: 28 }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "center", width: 120, height: 120, borderRadius: 60, background: COR.azulFundo, color: COR.azul, fontSize: 48, fontWeight: 800 }}>
          {iniciais(p.nome)}
        </div>
        <div style={{ display: "flex", flexDirection: "column", marginLeft: 30 }}>
          <div style={{ display: "flex", fontSize: 68, fontWeight: 800, letterSpacing: -1 }}>{p.nome}</div>
          <div style={{ display: "flex", fontSize: 34, color: COR.suave }}>{[p.partido, p.uf].filter(Boolean).join(" · ")}</div>
        </div>
      </div>
      <div style={{ display: "flex" }}>
        <Medida rotulo="Presença em sessões" pct={m.presenca_sessao.pct} fundo={COR.verdeAguaFundo} cor={COR.verdeAgua}
          base={`${formatInt(m.presenca_sessao.sessoes_presente)} de ${formatInt(m.presenca_sessao.sessoes_elegiveis)} sessões`} />
        <div style={{ display: "flex", width: 18 }} />
        <Medida rotulo="Votos nas nominais" pct={m.participacao_votacoes_nominais.pct} fundo={COR.azulFundo} cor={COR.azul}
          base={`${formatInt(m.participacao_votacoes_nominais.votacoes_votou)} de ${formatInt(m.participacao_votacoes_nominais.votacoes_elegiveis)} votações`} />
        <div style={{ display: "flex", width: 18 }} />
        <Medida rotulo="Alinhamento com o Governo" pct={m.alinhamento_governo.pct} fundo={COR.ambarFundo} cor={COR.ambar}
          base={`${formatInt(m.alinhamento_governo.votacoes_alinhadas)} de ${formatInt(m.alinhamento_governo.votacoes_com_orientacao)} votações`} />
      </div>
    </div>,
  );
}
