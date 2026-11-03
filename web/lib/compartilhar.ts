import type { Metadata } from "next";
import { NOME_SITE } from "./site";

/** Metadados de uma página que vai ser compartilhada: título, descrição, link canônico e cartão grande. A imagem vem do `opengraph-image` da rota. */
export function metaCompartilhavel({ titulo, descricao, caminho }: { titulo: string; descricao: string; caminho: string }): Metadata {
  return {
    title: titulo,
    description: descricao,
    alternates: { canonical: caminho },
    openGraph: { title: `${titulo} · ${NOME_SITE}`, description: descricao, url: caminho, siteName: NOME_SITE, locale: "pt_BR", type: "website" },
    twitter: { card: "summary_large_image", title: `${titulo} · ${NOME_SITE}`, description: descricao },
  };
}

/** Link de compartilhamento pelo WhatsApp (funciona sem JavaScript). */
export function linkWhatsApp(texto: string, url: string): string {
  return `https://wa.me/?text=${encodeURIComponent(`${texto} ${url}`)}`;
}

/** Corta o texto em `max` caracteres sem partir palavra, com reticências. */
export function resumir(texto: string, max: number): string {
  const t = texto.replace(/\s+/g, " ").trim();
  if (t.length <= max) return t;
  const corte = t.slice(0, max - 1);
  const ultimo = corte.lastIndexOf(" ");
  return `${(ultimo > max * 0.6 ? corte.slice(0, ultimo) : corte).replace(/[\s,;:.]+$/, "")}…`;
}
