// Leitura crítica do resultado oficial de uma votação: quando ele destoa do placar e quando a descrição
// oficial traz outro placar que os votos registrados. Funções puras, cobertas por teste.

export const CADEIRAS = 513;
export const MAIORIA_ABSOLUTA = 257; // metade mais um dos 513 deputados
export const TRES_QUINTOS = 308;

type Placar = { sim: number; nao: number };

/** Sim e Não que a descrição oficial traz escritos ("... Sim: 252; Não: 176; Total: 428"), se houver. */
export function placarDaDescricao(descricao: string): Placar | null {
  const sim = /\bsim:\s*(\d+)/i.exec(descricao);
  const nao = /\bn[ãa]o:\s*(\d+)/i.exec(descricao);
  return sim && nao ? { sim: Number(sim[1]), nao: Number(nao[1]) } : null;
}

/** O placar da descrição, se ele difere dos votos registrados. Votação sem Sim/Não registrados fica de fora (outro aviso cobre). */
export function divergeDaDescricao(descricao: string, votos: Placar): Placar | null {
  if (votos.sim + votos.nao === 0) return null;
  const d = placarDaDescricao(descricao);
  return d && (d.sim !== votos.sim || d.nao !== votos.nao) ? d : null;
}

/** O resultado oficial segue "mais Sim que Não" neste placar? (nulo se não há resultado ou se não há como comparar) */
export function resultadoCoerente(aprovada: boolean | null, p: Placar): boolean | null {
  if (aprovada === null || p.sim === p.nao) return null;
  return aprovada === p.sim > p.nao;
}

export type NotaResultado = { titulo: string; curta: string; texto: string };

/**
 * Explica um resultado que não segue "mais Sim que Não": rejeitada com mais Sim, ou aprovada sem ter mais Sim.
 * Não afirma a causa (não sabemos qual regra valia naquela votação): só diz que existem regras que explicam e mostra a conta.
 */
export function notaResultado(aprovada: boolean | null, votos: Placar): NotaResultado | null {
  if (aprovada === null || votos.sim + votos.nao === 0) return null;
  if (aprovada === false && votos.sim > votos.nao) {
    const faltou =
      votos.sim < MAIORIA_ABSOLUTA ? ` Se esta votação exigia maioria absoluta, faltaram ${MAIORIA_ABSOLUTA - votos.sim} votos Sim para os ${MAIORIA_ABSOLUTA} necessários.` : "";
    return {
      titulo: "Por que rejeitada, se teve mais Sim que Não?",
      curta: "Teve mais Sim que Não, mas foi rejeitada.",
      texto:
        `Teve mais votos Sim (${votos.sim}) que Não (${votos.nao}), mas a Câmara registrou o resultado como rejeitado. ` +
        `Isso acontece quando a votação exige mais que a maioria dos votos: requerimento de urgência e projeto de lei complementar precisam de maioria absoluta ` +
        `(${MAIORIA_ABSOLUTA} dos ${CADEIRAS} deputados) e emenda à Constituição, de 3/5 (${TRES_QUINTOS}).${faltou} ` +
        `Mostramos o resultado oficial; não o recalculamos.`,
    };
  }
  if (aprovada === true && votos.sim <= votos.nao) {
    return {
      titulo: "Por que aprovada, sem ter mais Sim que Não?",
      curta: "Foi aprovada sem ter mais Sim que Não.",
      texto:
        `Teve ${votos.sim} votos Sim e ${votos.nao} Não, mas a Câmara registrou o resultado como aprovado. ` +
        `Isso pode ocorrer em empate decidido pelo presidente da sessão ou por regras específicas daquela votação. ` +
        `Mostramos o resultado oficial; não o recalculamos.`,
    };
  }
  return null;
}
