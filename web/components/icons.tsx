// Ícones decorativos (aria-hidden): o significado sempre vem do texto ao lado.
const base = { width: 28, height: 28, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 1.8, strokeLinecap: "round", strokeLinejoin: "round", "aria-hidden": true } as const;

export function IconLogo({ size = 32 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" aria-hidden="true">
      <rect width="32" height="32" rx="9" fill="var(--accent)" />
      <rect x="7" y="17" width="4.5" height="8" rx="1.5" fill="#fff" opacity=".7" />
      <rect x="13.75" y="11" width="4.5" height="14" rx="1.5" fill="#fff" />
      <rect x="20.5" y="7" width="4.5" height="18" rx="1.5" fill="#fff" opacity=".7" />
    </svg>
  );
}

export function IconPresenca() {
  return (
    <svg {...base}>
      <rect x="3.5" y="5" width="17" height="15" rx="3" />
      <path d="M3.5 10h17M8 3v4M16 3v4M9 15l2 2 4-4" />
    </svg>
  );
}

export function IconVoto() {
  return (
    <svg {...base}>
      <path d="M4 14h16v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1v-5zM8 14l-1-4 8-2.5 1 3.5M13 6l3-2 3 4-2 1.5" />
    </svg>
  );
}

export function IconGoverno() {
  return (
    <svg {...base}>
      <path d="M12 4v16M6 20h12M5 8h14M5 8l-2.5 6a3 3 0 0 0 5 0L5 8zM19 8l-2.5 6a3 3 0 0 0 5 0L19 8z" />
    </svg>
  );
}

export function IconBusca() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
      <circle cx="11" cy="11" r="7" />
      <path d="M20 20l-3.5-3.5" />
    </svg>
  );
}
