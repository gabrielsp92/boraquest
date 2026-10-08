"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { Button, Field, TopBar } from "@/components/ui";
import { ApiError, login } from "@/lib/api";
import { setSession } from "@/lib/auth";

export default function EntrarPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      setSession(await login(email, password));
      router.replace("/regras");
    } catch (err) {
      setError(err instanceof ApiError && err.status === 401 ? "E-mail ou senha incorretos." : "Não foi possível entrar. Tente de novo.");
      setBusy(false);
    }
  }

  return (
    <div className="bq-screen">
      <main className="bq-body">
        <TopBar title="Entrar" caption="Boraquest" />
        <form className="bq-stack" style={{ gap: 16 }} onSubmit={submit}>
          <Field label="E-mail">
            <input className="bq-input" type="email" autoComplete="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
          </Field>
          <Field label="Senha">
            <input className="bq-input" type="password" autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} />
          </Field>
          {error && (
            <p className="bq-note" role="alert">
              {error}
            </p>
          )}
          <Button block type="submit" disabled={busy || !email || !password}>
            {busy ? "Entrando…" : "Entrar"}
          </Button>
        </form>
      </main>
    </div>
  );
}
