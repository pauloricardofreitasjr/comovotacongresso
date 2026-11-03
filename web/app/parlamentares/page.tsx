import type { Metadata } from "next";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatInt, inteiroPositivo, plural, primeiro, query } from "@/lib/format";
import { IconBusca } from "@/components/icons";
import { MiniMetrica, Paginacao } from "@/components/ui";
import { Foto } from "@/components/ui";

export const metadata: Metadata = { title: "Deputados" };

const LIMITE = 30;
const MIN_PADRAO = 30;

const ABAS = [
  { chave: "nome", rotulo: "Todos (A–Z)" },
  { chave: "presenca_sessao", rotulo: "Presença em sessões" },
  { chave: "participacao", rotulo: "Votos nas nominais" },
  { chave: "governismo", rotulo: "Alinhamento com o governo" },
] as const;
type Ordem = (typeof ABAS)[number]["chave"];

const UFS = ["AC","AL","AP","AM","BA","CE","DF","ES","GO","MA","MT","MS","MG","PA","PB","PR","PE","PI","RJ","RN","RS","RO","RR","SC","SP","SE","TO"];

export default async function Parlamentares({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const sp = await searchParams;
  const ordenarPor: Ordem = ABAS.some((a) => a.chave === primeiro(sp.ordenarPor)) ? (primeiro(sp.ordenarPor) as Ordem) : "nome";
  const ordem = primeiro(sp.ordem) === "asc" ? "asc" : primeiro(sp.ordem) === "desc" ? "desc" : "";
  const busca = primeiro(sp.busca).trim();
  const uf = UFS.includes(primeiro(sp.uf).toUpperCase()) ? primeiro(sp.uf).toUpperCase() : "";
  const partido = primeiro(sp.partido).trim();
  const todos = primeiro(sp.em_exercicio) === "todos";
  const pagina = inteiroPositivo(sp.pagina, 1);
  const ehMetrica = ordenarPor !== "nome";
  // Rankings sem mínimo seriam dominados por quem votou pouquíssimas vezes.
  const min = ehMetrica ? inteiroPositivo(sp.min, MIN_PADRAO) : 0;

  const [res, partidos] = await Promise.all([
    api.parlamentares({
      ordenarPor, ordem, busca, uf, partido, pagina, limite: LIMITE,
      em_exercicio: todos ? "todos" : "true",
      min_votacoes: min || null,
    }),
    api.partidos(),
  ]);

  const filtros = { ordenarPor: ordenarPor === "nome" ? null : ordenarPor, ordem, busca, uf, partido, em_exercicio: todos ? "todos" : null, min: primeiro(sp.min) || null };
  const nomesPartidos = [...new Set(partidos.map((p) => p.partido))].sort((a, b) => a.localeCompare(b, "pt-BR"));
  const inverter = ordem === "asc" ? "desc" : "asc";

  return (
    <>
      <h1>Deputados</h1>
      <p className="lead">Compare como cada deputado federal atuou no Plenário. Cada medida mostra a quantidade de votações que a sustenta.</p>

      <form className="painel-busca" action="/parlamentares" role="search">
        {ordenarPor !== "nome" ? <input type="hidden" name="ordenarPor" value={ordenarPor} /> : null}
        <div className="busca-grande">
          <IconBusca />
          <label className="sr-only" htmlFor="busca-dep">Nome do deputado</label>
          <input id="busca-dep" type="search" name="busca" defaultValue={busca} placeholder="Nome do deputado" />
          <button type="submit">Buscar</button>
        </div>
        <div className="filters">
          <label>
            Estado
            <select name="uf" defaultValue={uf}>
              <option value="">Todos</option>
              {UFS.map((u) => (
                <option key={u} value={u}>{u}</option>
              ))}
            </select>
          </label>
          <label>
            Partido atual
            <select name="partido" defaultValue={partido}>
              <option value="">Todos</option>
              {nomesPartidos.map((p) => (
                <option key={p} value={p}>{p}</option>
              ))}
            </select>
          </label>
          <label>
            Mostrar
            <select name="em_exercicio" defaultValue={todos ? "todos" : "true"}>
              <option value="true">Em exercício</option>
              <option value="todos">Todos desde 2023</option>
            </select>
          </label>
          <button type="submit" className="secondary">Aplicar filtros</button>
        </div>
      </form>

      <p className="rotulo-ordem" id="ordenar-por">Ordenar por</p>
      <ul className="tabs" aria-labelledby="ordenar-por">
        {ABAS.map((a) => (
          <li key={a.chave}>
            <Link
              href={`/parlamentares${query({ ...filtros, ordenarPor: a.chave === "nome" ? null : a.chave, ordem: null, pagina: null })}`}
              aria-current={a.chave === ordenarPor}
            >
              {a.rotulo}
            </Link>
          </li>
        ))}
      </ul>

      {ehMetrica ? (
        <p className="muted">
          Mostrando quem tem pelo menos {formatInt(min)} {ordenarPor === "presenca_sessao" ? "sessões" : ordenarPor === "governismo" ? "votações com orientação do Governo" : "votações elegíveis"} no
          período, para que poucos registros não distorçam o ranking.{" "}
          <Link href={`/parlamentares${query({ ...filtros, ordem: inverter, pagina: null })}`}>
            Inverter ordem ({inverter === "asc" ? "do menor para o maior" : "do maior para o menor"})
          </Link>
        </p>
      ) : null}
      <p className="muted" aria-live="polite">{plural(res.total, "deputado", "deputados")}.</p>

      <ul className="dep-grid">
        {res.dados.map((p) => {
          const m = p.metricas;
          return (
            <li key={p.id} className="dep-card">
              <div className="dep-topo">
                <Foto nome={p.nome} url={p.url_foto} tamanho={56} />
                <div>
                  <Link className="dep-nome" href={`/parlamentares/${p.id}`}>{p.nome}</Link>
                  <small>
                    {[p.partido, p.uf].filter(Boolean).join(" · ")}
                    {!p.em_exercicio ? " · fora de exercício" : ""}
                  </small>
                </div>
              </div>
              <MiniMetrica
                tom="presenca" rotulo="Presença em sessões" pct={m.presenca_sessao.pct} ativo={ordenarPor === "presenca_sessao"}
                base={`${formatInt(m.presenca_sessao.sessoes_presente)} de ${formatInt(m.presenca_sessao.sessoes_elegiveis)} sessões`}
              />
              <MiniMetrica
                tom="voto" rotulo="Votos nas nominais" pct={m.participacao_votacoes_nominais.pct} ativo={ordenarPor === "participacao"}
                base={`${formatInt(m.participacao_votacoes_nominais.votacoes_votou)} de ${formatInt(m.participacao_votacoes_nominais.votacoes_elegiveis)} votações`}
              />
              <MiniMetrica
                tom="governo" rotulo="Alinhamento com o Governo" pct={m.alinhamento_governo.pct} ativo={ordenarPor === "governismo"}
                base={`${formatInt(m.alinhamento_governo.votacoes_alinhadas)} de ${formatInt(m.alinhamento_governo.votacoes_com_orientacao)} votações`}
              />
            </li>
          );
        })}
      </ul>

      <Paginacao pagina={res.pagina} limite={res.limite} total={res.total} caminho="/parlamentares" params={filtros} />
    </>
  );
}
