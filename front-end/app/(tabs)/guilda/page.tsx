"use client";

import { useEffect, useState } from "react";
import { Avatar, Badge, Bar, Button, PrizeCard, RankRow, Segmented, Sheet, TopBar } from "@/components/ui";
import { getPrizes, type Prizes } from "@/lib/api";
import { me, members, monthStandings, weekStandings, type MemberId } from "@/lib/data";

export default function GuildaPage() {
  const [view, setView] = useState<"semana" | "mes">("semana");
  const [pending, setPending] = useState<MemberId[]>(weekStandings.filter((s) => s.auditPending).map((s) => s.member));
  const [asking, setAsking] = useState<MemberId | null>(null);

  // Prizes: own fetch, independent of the standings above.
  const [prizes, setPrizes] = useState<Prizes | null>(null);
  const [prizesLoadError, setPrizesLoadError] = useState(false);

  useEffect(() => {
    let active = true;
    getPrizes()
      .then((p) => active && setPrizes(p))
      .catch(() => active && setPrizesLoadError(true));
    return () => {
      active = false;
    };
  }, []);

  const standings = view === "semana" ? weekStandings : monthStandings;
  const top = standings[0]?.points || 1;

  return (
    <>
      <TopBar
        title="Guilda"
        caption={`${standings.length} aventureiros`}
        right={
          <div style={{ width: 196 }}>
            <Segmented
              value={view}
              onChange={setView}
              options={[
                { value: "semana", label: "Semana" },
                { value: "mes", label: "Mês" },
              ]}
            />
          </div>
        }
      />
      {prizesLoadError ? (
        <PrizeCard label={view === "semana" ? "Prêmio da semana" : "Prêmio do mês"} prize="Não deu para carregar" icon={view === "semana" ? undefined : "gift"} />
      ) : prizes ? (
        view === "semana" ? (
          <PrizeCard label="Prêmio da semana" prize={prizes.week || "Ainda sem prêmio"} />
        ) : (
          <PrizeCard label="Prêmio do mês" prize={prizes.month || "Ainda sem prêmio"} icon="gift" />
        )
      ) : null}

      <div className="bq-list">
        {standings.map((s, i) => (
          <RankRow key={s.member} position={i + 1} member={members[s.member]} points={s.points} leader={i === 0}>
            {view === "mes" ? (
              <Bar percent={Math.round((s.points / top) * 100)} />
            ) : pending.includes(s.member) ? (
              <Badge kind="audit" icon="shield">
                Auditoria pendente
              </Badge>
            ) : s.member === me ? (
              <Badge>Você</Badge>
            ) : (
              <Button variant="small" icon="shield" onClick={() => setAsking(s.member)}>
                Pedir auditoria
              </Button>
            )}
          </RankRow>
        ))}
      </div>

      <Sheet open={!!asking} onClose={() => setAsking(null)}>
        {asking && (
          <>
            <div className="bq-stack bq-center">
              <Avatar member={members[asking]} size="lg" />
              <h2 className="bq-title">Pedir auditoria: {members[asking].name}?</h2>
              <p className="bq-text">
                {members[asking].name} vai precisar provar as quests da semana, com fotos ou uma explicação. Até você aprovar, não pode vencer.
              </p>
            </div>
            <div className="bq-stack">
              <Button
                block
                icon="shield"
                onClick={() => {
                  setPending([...pending, asking]);
                  setAsking(null);
                }}
              >
                Pedir auditoria
              </Button>
              <Button variant="ghost" block onClick={() => setAsking(null)}>
                Agora não
              </Button>
            </div>
          </>
        )}
      </Sheet>
    </>
  );
}
