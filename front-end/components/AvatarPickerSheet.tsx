"use client";

import { useState } from "react";
import { updateMyAvatar } from "@/lib/api";
import { AVATAR_KEYS, avatarSrc } from "@/lib/avatars";
import { setSession, type Session } from "@/lib/auth";
import { Avatar, Button, Sheet } from "./ui";

/**
 * The "Seu avatar" sheet: pick one of the six platform avatars for yourself.
 * Shared by every screen with a link to it (Regras, Hoje). Mount it only while
 * open (e.g. `{avatarOpen && <AvatarPickerSheet .../>}`) so each opening starts
 * fresh from the current session avatar.
 */
export function AvatarPickerSheet({ session, onClose }: { session: Session; onClose: () => void }) {
  const [selection, setSelection] = useState(session.user.avatarKey);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function close() {
    if (!busy) onClose();
  }

  async function save() {
    if (!selection) return;
    setBusy(true);
    setError(null);
    try {
      const updated = await updateMyAvatar(selection);
      setSession({ ...session, user: { ...session.user, avatarKey: updated.avatarKey } });
      onClose();
    } catch {
      setError("Não deu para salvar. Tenta de novo.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Sheet open onClose={close}>
      <h2 className="bq-title">Seu avatar</h2>
      {selection && (
        <div className="bq-row" style={{ justifyContent: "center" }}>
          <Avatar member={{ id: selection, name: "Seu avatar", avatar: avatarSrc(selection) }} size="lg" />
        </div>
      )}
      <div className="bq-avatar-grid">
        {AVATAR_KEYS.map((key, i) => (
          <button key={key} type="button" className="bq-pick" aria-pressed={key === selection} aria-label={`Avatar ${i + 1}`} onClick={() => setSelection(key)}>
            <Avatar member={{ id: key, name: `Avatar ${i + 1}`, avatar: avatarSrc(key) }} selected={key === selection} />
          </button>
        ))}
      </div>
      {error && (
        <p className="bq-note" role="alert">
          {error}
        </p>
      )}
      <Button block onClick={save} disabled={busy || !selection}>
        {busy ? "Salvando…" : "Salvar avatar"}
      </Button>
    </Sheet>
  );
}
