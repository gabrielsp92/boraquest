"use client";

// The audited member's view: prove each completed quest with a photo or an explanation.
import { useState } from "react";
import Link from "next/link";
import { Banner, Button, Proof, Thumb, TopBar } from "@/components/ui";
import { Icon } from "@/components/Icon";
import { auditProofs, members, questById, type ProofItem } from "@/lib/data";

export default function AuditoriaPage() {
  const [proofs, setProofs] = useState<ProofItem[]>(auditProofs);
  const [editing, setEditing] = useState<string | null>(null);
  const [sent, setSent] = useState(false);
  const proven = proofs.filter((p) => p.photo || p.note).length;
  const update = (id: string, patch: Partial<ProofItem>) => setProofs(proofs.map((p) => (p.questId === id ? { ...p, ...patch } : p)));

  return (
    <div className="bq-screen">
      <main className="bq-body">
        <TopBar
          title="Auditoria"
          right={
            <Link href="/guilda" className="bq-iconbtn" aria-label="Fechar">
              <Icon name="close" />
            </Link>
          }
        />
        <Banner member={members.lia} title="Lia pediu uma auditoria" text={sent ? "Provas enviadas. Agora é com ela." : "Mostre que você cumpriu as quests da semana."} />

        <div className="bq-list" style={{ gap: 12 }}>
          {proofs.map((p) => {
            const q = questById(p.questId);
            return (
              <Proof key={p.questId} name={q.name} day={p.day} points={q.points}>
                {p.photo && (
                  <div className="bq-row">
                    <Thumb />
                    <p className="bq-caption">Foto enviada</p>
                  </div>
                )}
                {p.note && editing !== p.questId && <p className="bq-note">{p.note}</p>}
                {editing === p.questId && (
                  <input
                    className="bq-input"
                    autoFocus
                    placeholder="Conte o que você fez"
                    defaultValue={p.note}
                    onBlur={(e) => {
                      update(p.questId, { note: e.target.value.trim() || undefined });
                      setEditing(null);
                    }}
                  />
                )}
                {!p.photo && !p.note && editing !== p.questId && (
                  <div className="bq-row">
                    <Button variant="small" icon="camera" className="bq-grow" onClick={() => update(p.questId, { photo: true })}>
                      Foto
                    </Button>
                    <Button variant="small" icon="pencil" className="bq-grow" onClick={() => setEditing(p.questId)}>
                      Explicar
                    </Button>
                  </div>
                )}
              </Proof>
            );
          })}
        </div>

        <div className="bq-grow" />
        <p className="bq-caption text-center">
          {proven} de {proofs.length} quests com prova
        </p>
        <Button block disabled={sent} onClick={() => setSent(true)}>
          {sent ? "Provas enviadas" : "Enviar provas"}
        </Button>
      </main>
    </div>
  );
}
