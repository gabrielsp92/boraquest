// The signed-in session, kept in localStorage so it survives reloads.
// Falls back to memory when storage is unavailable (private mode, blocked site data).
import { useSyncExternalStore } from "react";

export type SessionUser = { id: string; name: string; email: string };
export type Session = { token: string; user: SessionUser };

const KEY = "bq.session";
const listeners = new Set<() => void>();
let memory: string | null = null;
let cachedRaw: string | null = null;
let cached: Session | null = null;

function read(): string | null {
  try {
    return localStorage.getItem(KEY);
  } catch {
    return memory;
  }
}

function write(raw: string | null) {
  memory = raw;
  try {
    if (raw === null) localStorage.removeItem(KEY);
    else localStorage.setItem(KEY, raw);
  } catch {
    // Storage unavailable: the in-memory copy still works for this tab.
  }
  listeners.forEach((l) => l());
}

/** The current session, or null when signed out. Stable between calls while unchanged. */
export function getSession(): Session | null {
  const raw = read();
  if (raw !== cachedRaw) {
    cachedRaw = raw;
    try {
      cached = raw ? (JSON.parse(raw) as Session) : null;
    } catch {
      cached = null;
    }
  }
  return cached;
}

export function setSession(session: Session) {
  write(JSON.stringify(session));
}

export function clearSession() {
  write(null);
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  window.addEventListener("storage", listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", listener);
  };
}

/** The session, or undefined while it can't be known yet (server render, before hydration). */
export function useSession(): Session | null | undefined {
  return useSyncExternalStore(subscribe, getSession, () => undefined);
}
