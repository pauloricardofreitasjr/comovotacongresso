"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

export function NavLink({ href, children }: { href: string; children: ReactNode }) {
  const path = usePathname();
  const ativo = path === href || path.startsWith(`${href}/`);
  return (
    <Link href={href} aria-current={ativo ? "page" : undefined}>
      {children}
    </Link>
  );
}
