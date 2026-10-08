"use client";

// The requester's view: look at the proof and approve or reject.
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Banner, Button, Proof, Thumb, TopBar } from "@/components/ui";
import { Icon } from "@/components/Icon";
import { auditProofs, members, questById } from "@/lib/data";

const proofs = auditProofs.map((p) => (p.photo || p.note ? p : { ...p, note: "Li o gibi novo antes de dormir." }));

export default function RevisarAuditoriaPage() {
  const router = useRouter();
  return (
    <div className="bq-screen">
      <main className="bq-body">
        <TopBar
          title={`Provas: ${members.caio.name}`}
          right={
            <Link href="/guilda" className="bq-iconbtn" aria-label="Fechar">
              <Icon name="close" />
            </Link>
          }
        />
        <Banner member={members.caio} title="Caio enviou as provas" text="Dê uma olhada e decida." />

        <div className="bq-list" style={{ gap: 12 }}>
          {proofs.map((p) => {
            const q = questById(p.questId);
            return (
              <Proof key={p.questId} name={q.name} day={p.day} points={q.points}>
                {p.photo && (
                  <div className="bq-row">
                    <Thumb />
                    <p className="bq-caption">Toque para ampliar</p>
                  </div>
                )}
                {p.note && <p className="bq-note">{p.note}</p>}
              </Proof>
            );
          })}
        </div>

        <div className="bq-grow" />
        <div className="bq-stack">
          {/* TODO: persist the decision. Approving clears the "Auditoria pendente" badge. */}
          <Button block icon="check" onClick={() => router.push("/guilda")}>
            Aprovar
          </Button>
          <Button variant="plain" block onClick={() => router.push("/guilda")}>
            Reprovar
          </Button>
        </div>
        <p className="bq-caption text-center">Se reprovar, a auditoria continua pendente.</p>
      </main>
    </div>
  );
}
