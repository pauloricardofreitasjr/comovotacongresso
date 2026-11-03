import type { Metadata } from "next";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { marked } from "marked";
import { Aviso } from "@/components/ui";

export const metadata: Metadata = { title: "Como calculamos" };

/** A metodologia é mantida em docs/metodologia.md (fonte única); aqui só renderizamos. */
async function carregar(): Promise<string> {
  const caminho = process.env.METODOLOGIA_PATH ?? path.join(process.cwd(), "..", "docs", "metodologia.md");
  // Leitura intencional de um arquivo fora do bundle (ver METODOLOGIA_PATH no Dockerfile).
  return readFile(/*turbopackIgnore: true*/ caminho, "utf8");
}

export default async function Metodologia() {
  const md = await carregar();
  const rascunho = /rascunho/i.test(md.split("\n").slice(0, 6).join("\n"));
  // O conteúdo é nosso (arquivo do repositório), não entrada de usuário.
  const html = await marked.parse(md, { gfm: true });
  return (
    <article className="prose">
      {rascunho ? (
        <Aviso atencao>
          <p>
            <strong>Versão em revisão.</strong> Trechos entre colchetes ainda dependem de decisão. As regras de cálculo descritas já são as que o site usa.
          </p>
        </Aviso>
      ) : null}
      <div dangerouslySetInnerHTML={{ __html: html }} />
    </article>
  );
}
