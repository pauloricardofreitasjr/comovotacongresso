import Link from "next/link";
import type { ReactNode } from "react";
import { compararComMedia, formatInt, formatPct, iniciais, infoVoto, query, rotuloResultado } from "@/lib/format";
import { divergeDaDescricao, notaResultado, resultadoCoerente } from "@/lib/resultado";
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

export type Tom = "presenca" | "voto" | "governo";

/** Cartão de uma métrica. Mostra sempre o percentual E a base de cálculo; com `media`, marca a média da Câmara na barra. */
export function Metrica({
  rotulo, pct, base, ajuda, tom, media,
}: { rotulo: string; pct: number | null; base: string; ajuda: string; tom: Tom; media?: number | null }) {
  return (
    <div className={`card metrica tom-${tom}`}>
      <span className="rotulo">{rotulo}</span>
      <span className="valor">{formatPct(pct)}</span>
      {pct !== null ? (
        <span className="medidor" role="presentation">
          <span style={{ width: `${Math.min(100, Math.max(0, pct))}%` }} />
          {media != null ? <i className="marca" style={{ left: `${Math.min(100, Math.max(0, media))}%` }} /> : null}
        </span>
      ) : null}
      <span className="base">{pct === null ? "Sem dados suficientes" : base}</span>
      {pct !== null && media != null ? (
        <span className="comparacao">
          {compararComMedia(pct, media)} <small>({formatPct(media)})</small>
        </span>
      ) : null}
      <span className="ajuda">{ajuda}</span>
    </div>
  );
}

/** Linha compacta de uma métrica, para cartões de lista. `ativo` destaca a medida pela qual a lista está ordenada. */
export function MiniMetrica({
  rotulo, pct, base, tom, ativo, media,
}: { rotulo: string; pct: number | null; base: string; tom: Tom; ativo?: boolean; media?: number | null }) {
  return (
    <div className={`mini tom-${tom}${ativo ? " ativo" : ""}`}>
      <div className="mini-topo">
        <span>{rotulo}</span>
        <strong>{formatPct(pct)}</strong>
      </div>
      <span className="medidor" role="presentation">
        {pct !== null ? <span style={{ width: `${Math.min(100, Math.max(0, pct))}%` }} /> : null}
        {pct !== null && media != null ? <i className="marca" style={{ left: `${Math.min(100, Math.max(0, media))}%` }} /> : null}
      </span>
      <small>{pct === null ? "Sem dados suficientes" : base}</small>
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

/** Placar compacto Sim × Não para cartões de lista; os números sempre aparecem em texto. */
export function MiniPlacar({ sim, nao }: { sim: number; nao: number }) {
  const total = sim + nao;
  return (
    <div className="mini-placar">
      <div className="barra" role="img" aria-label={`${sim} Sim e ${nao} Não`}>
        {total > 0 ? (
          <>
            <span className="sim" style={{ width: `${(sim / total) * 100}%` }} />
            <span className="nao" style={{ width: `${(nao / total) * 100}%` }} />
          </>
        ) : null}
      </div>
      <span className="mini-placar-txt">
        <strong className="txt-sim">{formatInt(sim)} Sim</strong> × <strong className="txt-nao">{formatInt(nao)} Não</strong>
      </span>
    </div>
  );
}

/** Placar em destaque: Sim e Não grandes, com percentual e contagem; os demais votos viram etiquetas. */
export function Placar({ resumo }: { resumo: ResumoVotos }) {
  const pct = (n: number) => formatPct(resumo.total > 0 ? (n / resumo.total) * 100 : null);
  const outros = SEGMENTOS.filter((s) => s.chave !== "sim" && s.chave !== "nao" && resumo[s.chave] > 0);
  return (
    <div>
      <div className="placar">
        <div className="placar-lado sim">
          <span className="placar-n">{formatInt(resumo.sim)}</span>
          <span className="placar-r">votaram Sim</span>
          <span className="placar-p">{pct(resumo.sim)} dos {formatInt(resumo.total)} votos</span>
        </div>
        <div className="placar-lado nao">
          <span className="placar-n">{formatInt(resumo.nao)}</span>
          <span className="placar-r">votaram Não</span>
          <span className="placar-p">{pct(resumo.nao)} dos {formatInt(resumo.total)} votos</span>
        </div>
      </div>
      <BarraVotos resumo={resumo} />
      {outros.length > 0 ? (
        <ul className="chips-votos" aria-label="Outros votos">
          {outros.map((s) => (
            <li key={s.chave} className={`chip-voto ${s.classe}`}>
              {s.rotulo}: <strong>{formatInt(resumo[s.chave])}</strong>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

/**
 * Avisos sobre o placar: por que o resultado oficial destoa dos votos e quando a descrição oficial traz outro placar.
 * `curta` mostra só uma linha com link para a explicação completa (listas e home).
 */
export function NotasDoPlacar({
  aprovada, resumo, descricao, votacaoId, curta, ancora,
}: { aprovada: boolean | null; resumo: Pick<ResumoVotos, "sim" | "nao">; descricao?: string; votacaoId?: string; curta?: boolean; ancora?: boolean }) {
  const nota = notaResultado(aprovada, resumo);
  const diverge = descricao ? divergeDaDescricao(descricao, resumo) : null;
  if (curta) {
    return nota && votacaoId ? (
      <p className="nota-curta">
        {nota.curta} <Link href={`/votacoes/${encodeURIComponent(votacaoId)}#nota-resultado`}>Entenda</Link>
      </p>
    ) : null;
  }
  // Com divergência, o resultado oficial pode seguir o placar da descrição e não os votos registrados:
  // a explicação de "maioria especial" seria errada, então só o aviso de divergência aparece.
  const notaVisivel = diverge ? null : nota;
  const idNota = ancora ? "nota-resultado" : undefined;
  const coerenteComDescricao = diverge ? resultadoCoerente(aprovada, diverge) : null;
  const coerenteComVotos = diverge ? resultadoCoerente(aprovada, resumo) : null;
  return (
    <>
      {notaVisivel ? (
        <div className="aviso" id={idNota}>
          <p><strong>{notaVisivel.titulo}</strong></p>
          <p>{notaVisivel.texto}</p>
        </div>
      ) : null}
      {diverge ? (
        <div className="aviso atencao" id={notaVisivel ? undefined : idNota}>
          <p>
            <strong>O placar da descrição oficial é diferente dos votos registrados.</strong> A descrição diz Sim: {formatInt(diverge.sim)}, Não:{" "}
            {formatInt(diverge.nao)}; os votos individuais somam Sim: {formatInt(resumo.sim)}, Não: {formatInt(resumo.nao)}.{" "}
            {coerenteComDescricao === true && coerenteComVotos === false
              ? "O resultado oficial combina com o placar da descrição, não com os votos individuais. "
              : ""}
            A Câmara é a fonte das duas informações e não sabemos qual está certa. Mostramos os votos como ela os registrou.
          </p>
        </div>
      ) : null}
    </>
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
