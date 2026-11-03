import Link from "next/link";
import type { ReactNode } from "react";
import { formatInt, formatPct, iniciais, infoVoto, query, rotuloResultado } from "@/lib/format";
import type { ResumoVotos, TipoVotacao } from "@/lib/types";

export function TipoBadge({ tipo }: { tipo: TipoVotacao }) {
  return tipo === "nominal" ? (
    <span className="badge nominal" title="Cada deputado votou em seu nome e o voto ficou registrado.">
      Votação nominal
    </span>
  ) : (
    <span className="badge simbolica" title="Não há registro do voto de cada deputado.">
      Votação simbólica
    </span>
  );
}

export function ResultadoBadge({ aprovada }: { aprovada: boolean | null }) {
  const classe = aprovada === null ? "neutra" : aprovada ? "aprovada" : "rejeitada";
  return <span className={`badge ${classe}`}>{rotuloResultado(aprovada)}</span>;
}

export function VotoChip({ voto }: { voto: string }) {
  const { rotulo, classe } = infoVoto(voto);
  return <span className={`voto ${classe}`}>{rotulo}</span>;
}

export function Foto({ nome, url, tamanho = 44 }: { nome: string; url: string | null; tamanho?: number }) {
  return (
    <span className="foto" style={{ width: tamanho, height: tamanho, fontSize: tamanho * 0.36 }} aria-hidden="true">
      {iniciais(nome)}
      {url ? (
        // Foto oficial da Câmara; sem alt porque o nome já aparece ao lado. Se falhar, ficam as iniciais.
        // eslint-disable-next-line @next/next/no-img-element
        <img src={url} alt="" loading="lazy" decoding="async" referrerPolicy="no-referrer" />
      ) : null}
    </span>
  );
}

export function Pessoa({ id, nome, url, sub, tamanho }: { id: number; nome: string; url: string | null; sub?: ReactNode; tamanho?: number }) {
  return (
    <span className="pessoa">
      <Foto nome={nome} url={url} tamanho={tamanho} />
      <span>
        <Link href={`/parlamentares/${id}`}>{nome}</Link>
        {sub ? <small>{sub}</small> : null}
      </span>
    </span>
  );
}

/** Cartão de uma métrica. Mostra sempre o percentual E a base de cálculo. */
export function Metrica({
  rotulo, pct, base, ajuda, destaque,
}: { rotulo: string; pct: number | null; base: string; ajuda: string; destaque?: boolean }) {
  return (
    <div className={`card metrica${destaque ? " destaque" : ""}`}>
      <span className="rotulo">{rotulo}</span>
      <span className="valor">{formatPct(pct)}</span>
      {pct !== null ? (
        <span className="medidor" role="presentation">
          <span style={{ width: `${Math.min(100, Math.max(0, pct))}%` }} />
        </span>
      ) : null}
      <span className="base">{pct === null ? "Sem dados suficientes" : base}</span>
      <span className="ajuda">{ajuda}</span>
    </div>
  );
}

const SEGMENTOS: { chave: keyof ResumoVotos; classe: string; rotulo: string }[] = [
  { chave: "sim", classe: "sim", rotulo: "Sim" },
  { chave: "nao", classe: "nao", rotulo: "Não" },
  { chave: "abstencao", classe: "neutro", rotulo: "Abstenção" },
  { chave: "obstrucao", classe: "neutro", rotulo: "Obstrução" },
  { chave: "artigo_17", classe: "presidente", rotulo: "Presidente da sessão" },
  { chave: "outro", classe: "desconhecido", rotulo: "Voto não informado" },
];

export function BarraVotos({ resumo }: { resumo: ResumoVotos }) {
  const visiveis = SEGMENTOS.filter((s) => resumo[s.chave] > 0);
  const descricao = visiveis.map((s) => `${s.rotulo}: ${resumo[s.chave]}`).join("; ");
  return (
    <div>
      <div className="barra" role="img" aria-label={`Votos — ${descricao}. Total: ${resumo.total}`}>
        {visiveis.map((s) => (
          <span key={s.chave} className={s.classe} style={{ width: `${(resumo[s.chave] / resumo.total) * 100}%` }} />
        ))}
      </div>
      <div className="legenda">
        {visiveis.map((s) => (
          <span key={s.chave}>
            <span className={`dot ${s.classe}`} style={{ background: `var(--${s.classe})` }} />
            {s.rotulo}: <strong>{formatInt(resumo[s.chave])}</strong>
          </span>
        ))}
      </div>
    </div>
  );
}

/** Paginação por links (sem JavaScript). */
export function Paginacao({
  pagina, limite, total, caminho, params, nomePagina = "pagina",
}: { pagina: number; limite: number; total: number; caminho: string; params: Record<string, string | number | null | undefined>; nomePagina?: string }) {
  const paginas = Math.max(1, Math.ceil(total / limite));
  if (total === 0) return null;
  const href = (p: number) => `${caminho}${query({ ...params, [nomePagina]: p === 1 ? null : p })}`;
  const de = (pagina - 1) * limite + 1;
  const ate = Math.min(total, pagina * limite);
  return (
    <nav className="paginacao" aria-label="Paginação">
      <span className="muted">
        {de > total ? "Não há resultados nesta página." : `${formatInt(de)}–${formatInt(ate)} de ${formatInt(total)}`}
      </span>
      {de > total ? <Link className="button secondary" href={href(1)}>Voltar à primeira página</Link> : null}
      <span className="links">
        {pagina > 1 ? <Link className="button secondary" href={href(pagina - 1)} rel="prev">← Anterior</Link> : null}
        {pagina < paginas ? <Link className="button secondary" href={href(pagina + 1)} rel="next">Próxima →</Link> : null}
      </span>
    </nav>
  );
}

export function Aviso({ children, atencao }: { children: ReactNode; atencao?: boolean }) {
  return <div className={`aviso${atencao ? " atencao" : ""}`}>{children}</div>;
}

/** Quebras de linha do texto vindo da API (\r\n) viram espaços. */
export function limpar(texto: string): string {
  return texto.replace(/\s+/g, " ").trim();
}
