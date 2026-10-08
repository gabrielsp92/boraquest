"use client";

import { useEffect, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { useSession } from "@/lib/auth";

/** Renders children only for a signed-in user; everyone else is sent to /entrar. */
export function AuthGate({ children }: { children: ReactNode }) {
  const session = useSession();
  const router = useRouter();

  useEffect(() => {
    if (session === null) router.replace("/entrar");
  }, [session, router]);

  // While the session is unknown (server render, hydration) render the page so it stays in the
  // prerendered shell; hide it only once we know the user is signed out.
  return session === null ? null : children;
}
