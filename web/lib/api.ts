import { notFound } from "next/navigation";
import type {
  Pagina, Partido, ParlamentarDetalhe, ParlamentarResumo, ProposicaoDetalhe, ProposicaoResumo, Status, VotacaoDetalhe,
} from "./types";
import { query } from "./format";

/** Endereço da API Go. Server-side apenas: o navegador nunca fala direto com ela. */
export function apiBase(): string {
  return (process.env.API_URL ?? "http://localhost:8080").replace(/\/+$/, "");
}

/** Os dados mudam no máximo uma vez por dia (ETL); 5 min espelha o Cache-Control da API. */
const REVALIDATE_SEGUNDOS = 300;

export class ApiError extends Error {
  constructor(
    public status: number,
    public codigo: string,
    mensagem: string,
  ) {
    super(mensagem);
    this.name = "ApiError";
  }
}

type Params = Record<string, string | number | null | undefined>;

/**
 * GET JSON na API. 404 vira a página "não encontrado" do Next; qualquer outro erro
 * lança ApiError e cai no error boundary. Em consulta por id (`recurso`), um 400 também é
 * "não encontrado": um id malformado na URL não é falha do servidor.
 */
async function get<T>(path: string, params: Params = {}, opts: { recurso?: boolean } = {}): Promise<T> {
  let resp: Response;
  try {
    resp = await fetch(`${apiBase()}${path}${query(params)}`, {
      headers: { Accept: "application/json" },
      next: { revalidate: REVALIDATE_SEGUNDOS },
    });
  } catch {
    throw new ApiError(503, "api_indisponivel", "Não foi possível falar com a API.");
  }
  if (resp.status === 404 || (opts.recurso && resp.status === 400)) notFound();
  if (!resp.ok) {
    let codigo = "erro", mensagem = `Erro ${resp.status} na API.`;
    try {
      const body = await resp.json();
      codigo = body?.erro?.codigo ?? codigo;
      mensagem = body?.erro?.mensagem ?? mensagem;
    } catch {
      /* corpo não é JSON */
    }
    throw new ApiError(resp.status, codigo, mensagem);
  }
  return (await resp.json()) as T;
}

export const api = {
  status: () => get<Status>("/api/v1/status"),
  proposicoes: (p: Params) => get<Pagina<ProposicaoResumo>>("/api/v1/proposicoes", p),
  proposicao: (id: string) => get<ProposicaoDetalhe>(`/api/v1/proposicoes/${encodeURIComponent(id)}`, {}, { recurso: true }),
  votacao: (id: string) => get<VotacaoDetalhe>(`/api/v1/votacoes/${encodeURIComponent(id)}`, {}, { recurso: true }),
  parlamentares: (p: Params) => get<Pagina<ParlamentarResumo>>("/api/v1/parlamentares", p),
  parlamentar: (id: string, p: Params = {}) => get<ParlamentarDetalhe>(`/api/v1/parlamentares/${encodeURIComponent(id)}`, p, { recurso: true }),
  partidos: () => get<{ dados: Partido[] }>("/api/v1/partidos", { casa: "camara" }).then((r) => r.dados),
};
