"use client";

import { useEffect, useState } from "react";
import { Avatar, Badge, Bar, Button, EmptyState, PrizeCard, RankRow, Segmented, Sheet, TopBar } from "@/components/ui";
import { getPrizes, getScoreboard, listGuildMembers, type GuildMember, type Prizes, type Standing } from "@/lib/api";
import { useSession } from "@/lib/auth";
import { DEFAULT_AVATAR } from "@/lib/avatars";

function completedLabel(n: number) {
  return n === 1 ? "1 tarefa concluída" : `${n} tarefas concluídas`;
}

export default function GuildaPage() {
  const session = useSession();
  const callerId = session?.user?.id;

  const [view, setView] = useState<"semana" | "mes">("semana");
  const [standings, setStandings] = useState<Standing[] | null>(null);
  const [guildMembers, setGuildMembers] = useState<GuildMember[] | null>(null);
  const [loadError, setLoadError] = useState(false);
  // Which view the standings/loadError above were fetched for; while it doesn't
  // match `view`, a toggle is in flight and the previous period's data is stale.
  const [loadedView, setLoadedView] = useState<"semana" | "mes" | null>(null);
  const [pending, setPending] = useState<string[]>([]);
  const [asking, setAsking] = useState<string | null>(null);

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

  function retry() {
    Promise.all([getScoreboard(view === "semana" ? "week" : "month"), listGuildMembers()]).then(
      ([s, ml]) => {
        setStandings(s.standings);
        setGuildMembers(ml.members);
        setLoadError(false);
        setLoadedView(view);
      },
      () => {
        setLoadError(true);
        setLoadedView(view);
      },
    );
  }

  useEffect(() => {
    let active = true;
    Promise.all([getScoreboard(view === "semana" ? "week" : "month"), listGuildMembers()]).then(
      ([s, ml]) => {
        if (!active) return;
        setStandings(s.standings);
        setGuildMembers(ml.members);
        setLoadError(false);
        setLoadedView(view);
      },
      () => {
        if (!active) return;
        setLoadError(true);
        setLoadedView(view);
      },
    );
    return () => {
      active = false;
    };
  }, [view]);

  // While the current view's fetch hasn't landed yet, treat the previous period's
  // data as not-yet-loaded so a toggle never flashes stale numbers (Business rule 6).
  const currentStandings = loadedView === view ? standings : null;
  const currentLoadError = loadedView === view && loadError;
  const top = currentStandings?.[0]?.points || 1;

  const memberById = new Map((guildMembers ?? []).map((m) => [m.id, m]));
  const toMember = (id: string) => ({ id, name: memberById.get(id)?.name ?? id, avatar: DEFAULT_AVATAR });

  return (
    <>
      <TopBar
        title="Guilda"
        caption={currentStandings ? `${currentStandings.length} aventureiros` : undefined}
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

      {!session || (currentStandings === null && !currentLoadError) ? (
        <p className="bq-caption" role="status">
          Carregando guilda…
        </p>
      ) : currentStandings === null ? (
        <EmptyState icon="users" title="Não deu para carregar" text="Confira sua conexão e tente de novo.">
          <Button onClick={retry}>Tentar de novo</Button>
        </EmptyState>
      ) : (
        <div className="bq-list">
          {currentStandings.map((s, i) => (
            <RankRow key={s.memberId} position={i + 1} member={toMember(s.memberId)} points={s.points} leader={i === 0}>
              {view === "mes" ? (
                <Bar percent={Math.round((s.points / top) * 100)} />
              ) : pending.includes(s.memberId) ? (
                <Badge kind="audit" icon="shield">
                  Auditoria pendente
                </Badge>
              ) : s.memberId === callerId ? (
                <Badge>Você</Badge>
              ) : (
                <Button variant="small" icon="shield" onClick={() => setAsking(s.memberId)}>
                  Pedir auditoria
                </Button>
              )}
              <p className="bq-caption">{completedLabel(s.completed)}</p>
            </RankRow>
          ))}
        </div>
      )}

      <Sheet open={!!asking} onClose={() => setAsking(null)}>
        {asking && (
          <>
            <div className="bq-stack bq-center">
              <Avatar member={toMember(asking)} size="lg" />
              <h2 className="bq-title">Pedir auditoria: {toMember(asking).name}?</h2>
              <p className="bq-text">
                {toMember(asking).name} vai precisar provar as quests da semana, com fotos ou uma explicação. Até você aprovar, não pode vencer.
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
