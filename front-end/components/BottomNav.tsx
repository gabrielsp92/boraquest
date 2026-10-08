"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Icon, type IconName } from "./Icon";

const tabs: { href: string; label: string; icon: IconName }[] = [
  { href: "/hoje", label: "Hoje", icon: "sun" },
  { href: "/semana", label: "Semana", icon: "calendar" },
  { href: "/guilda", label: "Guilda", icon: "users" },
  { href: "/regras", label: "Regras", icon: "scroll" },
];

export function BottomNav() {
  const pathname = usePathname();
  return (
    <nav className="bq-nav" aria-label="Principal">
      {tabs.map((t) => (
        <Link key={t.href} href={t.href} className="bq-nav__item" aria-current={pathname.startsWith(t.href) ? "page" : undefined}>
          <Icon name={t.icon} />
          <span>{t.label}</span>
        </Link>
      ))}
    </nav>
  );
}
