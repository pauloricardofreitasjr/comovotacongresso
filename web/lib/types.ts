// Tipos do JSON da API (ver internal/api/openapi.yaml). Mantidos à mão e cobertos pelo typecheck.

export type Pagina<T> = { dados: T[]; pagina: number; limite: number; total: number };

export type Status = {
  atualizado_em: string | null;
  primeira_votacao: string | null;
  ultima_votacao: string | null;
  votacoes_nominais: number;
  votacoes_simbolicas: number;
  sessoes_deliberativas: number;
  parlamentares: number;
  proposicoes: number;
  medias_camara: { presenca_sessao: number | null; participacao_votacoes_nominais: number | null; alinhamento_governo: number | null };
};

export type PropRef = {
  id: number;
  identificacao: string;
  sigla_tipo: string;
  numero: number;
  ano: number;
  ementa: string;
};

export type UltimaNominal = { votacao_id: string; data_hora: string; aprovada: boolean | null; sim: number; nao: number };

export type ProposicaoResumo = PropRef & {
  casa: string;
  total_votacoes: number;
  votacoes_nominais: number;
  ultima_votacao: string;
  /** Só na listagem, e só quando há votação nominal. */
  ultima_nominal?: UltimaNominal;
};

export type ResumoVotos = {
  sim: number;
  nao: number;
  abstencao: number;
  obstrucao: number;
  artigo_17: number;
  outro: number;
  total: number;
};

export type TipoVotacao = "nominal" | "simbolica";
export type Vinculo = "direto" | "provavel" | null;

export type VotacaoLinha = {
  id: string;
  data_hora: string;
  descricao: string;
  tipo: TipoVotacao;
  aprovada: boolean | null;
  vinculo_confianca: Vinculo;
  orientacao_governo: string | null;
  resumo_votos: ResumoVotos | null;
  principal?: boolean;
};

export type ProposicaoDetalhe = ProposicaoResumo & { votacoes: VotacaoLinha[] };

export type Orientacao = { bancada: string; tipo_lideranca: string | null; orientacao: string | null };

export type Voto = {
  parlamentar_id: number;
  nome: string;
  partido: string | null;
  uf: string | null;
  url_foto: string | null;
  voto: string;
  alinhou_governo: boolean | null;
};

export type VotacaoDetalhe = VotacaoLinha & {
  proposicao: PropRef | null;
  orientacoes: Orientacao[];
  votos_individuais: boolean;
  votos: Voto[] | null;
};

export type MetricaParticipacao = { votacoes_elegiveis: number; votacoes_votou: number; pct: number | null };
export type MetricaPresenca = { sessoes_elegiveis: number; sessoes_presente: number; pct: number | null };
export type MetricaGoverno = { votacoes_com_orientacao: number; votacoes_alinhadas: number; pct: number | null };

export type Metricas = {
  participacao_votacoes_nominais: MetricaParticipacao;
  presenca_sessao: MetricaPresenca;
  alinhamento_governo: MetricaGoverno;
};

export type ParlamentarResumo = {
  id: number;
  casa: string;
  nome: string;
  partido: string | null;
  uf: string | null;
  url_foto: string | null;
  em_exercicio: boolean;
  metricas: Metricas;
};

export type VotoHistorico = {
  votacao_id: string;
  data_hora: string;
  descricao: string;
  aprovada: boolean | null;
  voto: string;
  partido_na_votacao: string | null;
  orientacao_governo: string | null;
  alinhou_governo: boolean | null;
  proposicao: PropRef | null;
};

export type Exercicio = { inicio: string; fim: string | null; condicao: string | null; partido: string | null };

export type ParlamentarDetalhe = ParlamentarResumo & {
  exercicio: Exercicio[];
  votos: Pagina<VotoHistorico>;
};

export type Partido = { casa: string; partido: string; parlamentares: number; metricas: Metricas };
