// End of the week: the winner celebration, or a hold screen while the leader has an audit pending.
import Link from "next/link";
import { Avatar, Badge, Confetti, EmptyState, PrizeCard, RankRow, TopBar } from "@/components/ui";
import { members, prizes, today, weekStandings } from "@/lib/data";

export default function FimDeSemanaPage() {
  const [leader, runnerUp] = weekStandings;
  const winner = members[leader.member];

  if (leader.auditPending) {
    return (
      <div className="bq-screen">
        <main className="bq-body">
          <TopBar title="Fim da semana" caption={today.week} />
          <div style={{ flex: "none" }}>
            <EmptyState icon="shield" title="Quase lá!" text={`${winner.name} está na frente, mas tem uma auditoria pendente. O vencedor sai quando ela for aprovada.`} />
          </div>
          <div className="bq-list">
            <RankRow position={1} member={winner} points={leader.points}>
              <Badge kind="audit" icon="shield">
                Auditoria pendente
              </Badge>
            </RankRow>
            {runnerUp && (
              <RankRow position={2} member={members[runnerUp.member]} points={runnerUp.points}>
                <Badge>Na fila</Badge>
              </RankRow>
            )}
          </div>
          <div className="bq-grow" />
          <Link href="/auditoria/revisar" className="bq-btn bq-btn--block">
            Ver auditoria
          </Link>
        </main>
      </div>
    );
  }

  return (
    <div className="bq-screen bq-screen--party">
      <Confetti />
      <main className="bq-body bq-center" style={{ justifyContent: "center", gap: 24, position: "relative" }}>
        <p className="bq-label">Fim da semana</p>
        <div style={{ paddingTop: 48 }}>
          <Avatar member={winner} size="xl" crown />
        </div>
        <div>
          <h1 className="bq-title" style={{ fontSize: 40, lineHeight: "44px" }}>
            {winner.name} venceu!
          </h1>
          <p className="bq-display" style={{ marginTop: 8 }}>
            {leader.points}
          </p>
          <p className="bq-label">pontos</p>
        </div>
        <div style={{ width: "100%", textAlign: "left" }}>
          <PrizeCard label="Prêmio da semana" prize={prizes.week} />
        </div>
        <Link href="/hoje" className="bq-btn bq-btn--ink bq-btn--block">
          Começar nova semana
        </Link>
      </main>
    </div>
  );
}
