"use client";

import { useState, useSyncExternalStore } from "react";
import { linkWhatsApp } from "@/lib/compartilhar";

const nuncaMuda = () => () => {};

/**
 * Compartilhar: o WhatsApp é um link comum (funciona sem JavaScript); o botão usa o compartilhamento do
 * aparelho quando existe e, senão, copia o link. O botão só aparece depois de hidratar, para nunca ficar inerte.
 */
export function Compartilhar({ url, titulo, texto }: { url: string; titulo: string; texto: string }) {
  const hidratado = useSyncExternalStore(nuncaMuda, () => true, () => false);
  const [aviso, setAviso] = useState("");

  async function compartilhar() {
    try {
      if (typeof navigator.share === "function") {
        await navigator.share({ title: titulo, text: texto, url });
        return;
      }
      await navigator.clipboard.writeText(url);
      setAviso("Link copiado.");
    } catch (e) {
      if (e instanceof DOMException && e.name === "AbortError") return; // a pessoa fechou o painel de compartilhar
      setAviso("Não foi possível copiar. Copie o endereço na barra do navegador.");
    }
  }

  return (
    <div className="compartilhar">
      <span className="compartilhar-rotulo">Compartilhar</span>
      <a className="button secondary" href={linkWhatsApp(texto, url)} target="_blank" rel="noopener noreferrer">WhatsApp</a>
      {hidratado ? (
        <button type="button" className="secondary" onClick={compartilhar}>
          Copiar link ou outros apps
        </button>
      ) : null}
      <span className="compartilhar-aviso" role="status" aria-live="polite">{aviso}</span>
    </div>
  );
}
