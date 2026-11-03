import type { Metadata, Viewport } from "next";
import Link from "next/link";
import "./globals.css";
import { IconLogo } from "@/components/icons";
import { NavLink } from "@/components/NavLink";
import { api } from "@/lib/api";
import { formatDataHora } from "@/lib/format";
import { CREDITO_FOTOS_RODAPE } from "@/lib/creditos";

// Os dados vêm da API em tempo de requisição (com cache de 5 min nos fetch); o build não depende dela.
export const dynamic = "force-dynamic";

const SITE_URL = process.env.SITE_URL ?? "https://www.comovotacongresso.com.br";

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: { default: "Como Vota Congresso", template: "%s · Como Vota Congresso" },
  description:
    "Veja como cada deputado federal votou no Plenário da Câmara, com dados oficiais e metodologia aberta.",
  openGraph: { siteName: "Como Vota Congresso", locale: "pt_BR", type: "website" },
};

export const viewport: Viewport = { width: "device-width", initialScale: 1 };

async function atualizadoEm(): Promise<string | null> {
  try {
    return (await api.status()).atualizado_em;
  } catch {
    return null; // o rodapé nunca derruba a página
  }
}

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const atualizado = await atualizadoEm();
  return (
    <html lang="pt-BR">
      <body>
        <a className="skip-link" href="#conteudo">Ir para o conteúdo</a>
        <header className="site-header">
          <div className="container">
            <Link className="brand" href="/">
              <IconLogo />
              <span>
                Como Vota <span className="destaque">Congresso</span>
              </span>
            </Link>
            <nav className="nav" aria-label="Principal">
              <NavLink href="/proposicoes">Leis e projetos</NavLink>
              <NavLink href="/parlamentares">Deputados</NavLink>
              <NavLink href="/partidos">Partidos</NavLink>
              <NavLink href="/metodologia">Como calculamos</NavLink>
            </nav>
          </div>
        </header>
        <main id="conteudo">
          <div className="container">{children}</div>
        </main>
        <footer className="site-footer">
          <div className="container">
            <p>
              Fonte dos dados: <strong>Câmara dos Deputados</strong> (Dados Abertos), votações do Plenário desde fevereiro de 2023.
              {atualizado ? ` Última atualização: ${formatDataHora(atualizado)}.` : ""}
            </p>
            <p>{CREDITO_FOTOS_RODAPE}</p>
            <p>
              Os números dependem de regras explicadas em <Link href="/metodologia">Como calculamos</Link>. Projeto independente, sem vínculo com a
              Câmara dos Deputados.
            </p>
          </div>
        </footer>
      </body>
    </html>
  );
}
