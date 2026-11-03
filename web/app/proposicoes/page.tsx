import type { Metadata } from "next";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatData, inteiroPositivo, plural, primeiro, query } from "@/lib/format";
import { IconBusca } from "@/components/icons";
import { limpar, MiniPlacar, NotasDoPlacar, Paginacao, ResultadoBadge } from "@/components/ui";
import { interpretarIdentificacao, TIPOS_CONHECIDOS } from "@/lib/busca";

export const metadata: Metadata = { title: "Leis e projetos votados" };

const TIPOS = TIPOS_CONHECIDOS.slice(0, 7);
const LIMITE = 20;

export default async function Proposicoes({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const sp = await searchParams;
  const busca = primeiro(sp.busca).trim();
  const tipo = primeiro(sp.tipo).trim().toUpperCase();
  const ano = primeiro(sp.ano).trim();
  const pagina = inteiroPositivo(sp.pagina, 1);
  const anoValido = /^\d{4}$/.test(ano) ? ano : "";
  const soNominais = primeiro(sp.so_nominais) === "true";

  // "PL 186/2026" busca pela identificação; qualquer outro texto busca na ementa.
  const id = interpretarIdentificacao(busca);
  const res = await api.proposicoes(
    id
      ? { tipo: id.tipo, numero: id.numero, ano: id.ano, pagina, limite: LIMITE, so_nominais: soNominais ? "true" : null }
      : { busca, tipo, ano: anoValido, pagina, limite: LIMITE, so_nominais: soNominais ? "true" : null },
  );
  const filtros = { busca, tipo, ano: anoValido, so_nominais: soNominais ? "true" : null };
  const temFiltro = Boolean(busca || tipo || anoValido);

  return (
    <>
      <h1>Leis e projetos votados</h1>
      <p className="lead">Projetos que foram a voto no Plenário da Câmara, dos mais recentes para os mais antigos.</p>

      <form className="painel-busca" action="/proposicoes" role="search">
        {soNominais ? <input type="hidden" name="so_nominais" value="true" /> : null}
        <div className="busca-grande">
          <IconBusca />
          <label className="sr-only" htmlFor="busca-lei">Assunto, palavra ou número da lei</label>
          <input id="busca-lei" type="search" name="busca" defaultValue={busca} placeholder="Ex.: imposto ou PL 186/2026" />
          <button type="submit">Buscar</button>
        </div>
        <div className="filters">
          <label>
            Tipo
            <select name="tipo" defaultValue={tipo}>
              <option value="">Todos</option>
              {TIPOS.map((t) => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
          </label>
          <label>
            Ano
            <input type="number" name="ano" defaultValue={anoValido} min={2000} max={2100} placeholder="2025" />
          </label>
          <button type="submit" className="secondary">Aplicar filtros</button>
          {temFiltro ? <Link className="button secondary" href={`/proposicoes${query({ so_nominais: soNominais ? "true" : null })}`}>Limpar</Link> : null}
        </div>
      </form>

      <ul className="tabs" aria-label="Que votações mostrar">
        <li>
          <Link href={`/proposicoes${query({ ...filtros, so_nominais: null })}`} aria-current={!soNominais}>Todas as leis</Link>
        </li>
        <li>
          <Link href={`/proposicoes${query({ ...filtros, so_nominais: "true" })}`} aria-current={soNominais}>Com o voto de cada deputado</Link>
        </li>
      </ul>
      <p className="muted" aria-live="polite">
        {res.total === 0 ? "Nenhum projeto encontrado." : `${plural(res.total, "projeto encontrado", "projetos encontrados")}.`}
        {!soNominais ? " A maioria foi votada de forma simbólica, sem registro do voto de cada deputado." : ""}
      </p>

      <ul className="leis-lista">
        {res.dados.map((p) => {
          const un = p.ultima_nominal;
          const cor = !un ? "simbolica" : un.aprovada === null ? "neutra" : un.aprovada ? "aprovada" : "rejeitada";
          return (
            <li key={p.id} className={`lei-card ${cor}`}>
              <div className="item-title">
                <Link href={`/proposicoes/${p.id}`}>{p.identificacao}</Link>
              </div>
              <p className="clamp" style={{ margin: 0 }}>{limpar(p.ementa)}</p>
              {un ? (
                <div className="lei-nominal">
                  <div className="meta">
                    <span className="badge nominal">Última votação nominal</span>
                    <ResultadoBadge aprovada={un.aprovada} />
                    <span>{formatData(un.data_hora)}</span>
                  </div>
                  <MiniPlacar sim={un.sim} nao={un.nao} />
                  <NotasDoPlacar curta aprovada={un.aprovada} resumo={un} votacaoId={un.votacao_id} />
                </div>
              ) : (
                <div className="meta">
                  <span className="badge simbolica" title="Não há registro do voto de cada deputado.">Só votação simbólica</span>
                </div>
              )}
              <div className="meta">
                <span>Última votação em {formatData(p.ultima_votacao)}</span>
                <span>{plural(p.total_votacoes, "votação", "votações")}</span>
                {p.votacoes_nominais > 0 ? <span>{plural(p.votacoes_nominais, "nominal", "nominais")}</span> : null}
              </div>
            </li>
          );
        })}
      </ul>

      <Paginacao pagina={res.pagina} limite={res.limite} total={res.total} caminho="/proposicoes" params={filtros} />
    </>
  );
}
