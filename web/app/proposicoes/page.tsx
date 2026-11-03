import type { Metadata } from "next";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatData, inteiroPositivo, plural, primeiro } from "@/lib/format";
import { limpar, Paginacao } from "@/components/ui";
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

  // "PL 186/2026" busca pela identificação; qualquer outro texto busca na ementa.
  const id = interpretarIdentificacao(busca);
  const res = await api.proposicoes(
    id
      ? { tipo: id.tipo, numero: id.numero, ano: id.ano, pagina, limite: LIMITE }
      : { busca, tipo, ano: anoValido, pagina, limite: LIMITE },
  );
  const filtros = { busca, tipo, ano: anoValido };
  const temFiltro = Boolean(busca || tipo || anoValido);

  return (
    <>
      <h1>Leis e projetos votados</h1>
      <p className="lead">Projetos que foram a voto no Plenário da Câmara, dos mais recentes para os mais antigos.</p>

      <form className="filters" action="/proposicoes" role="search">
        <label>
          Assunto ou palavra
          <input type="search" name="busca" defaultValue={busca} placeholder="Ex.: imposto, ou PL 186/2026" />
        </label>
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
        <button type="submit">Filtrar</button>
        {temFiltro ? <Link className="button secondary" href="/proposicoes">Limpar</Link> : null}
      </form>

      <p className="muted" aria-live="polite">
        {res.total === 0 ? "Nenhum projeto encontrado." : `${plural(res.total, "projeto encontrado", "projetos encontrados")}.`}
      </p>

      <ul className="list">
        {res.dados.map((p) => (
          <li key={p.id} className="card">
            <div className="item-title">
              <Link href={`/proposicoes/${p.id}`}>{p.identificacao}</Link>
            </div>
            <p className="clamp">{limpar(p.ementa)}</p>
            <div className="meta">
              <span>Última votação em {formatData(p.ultima_votacao)}</span>
              <span>{plural(p.total_votacoes, "votação", "votações")}</span>
              {p.votacoes_nominais > 0 ? <span>{plural(p.votacoes_nominais, "nominal", "nominais")}</span> : <span>Só votação simbólica</span>}
            </div>
          </li>
        ))}
      </ul>

      <Paginacao pagina={res.pagina} limite={res.limite} total={res.total} caminho="/proposicoes" params={filtros} />
    </>
  );
}
