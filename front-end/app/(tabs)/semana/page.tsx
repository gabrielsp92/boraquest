"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Badge, Button, DayCard, EmptyState, RankRow, ScoreHeader, Segmented, TopBar } from "@/components/ui";
import { Icon } from "@/components/Icon";
import { members, pastWeeks, type MemberId } from "@/lib/data";
import { listEntries, type Entry, type EntryList } from "@/lib/api";
import { dayLabel, weekRangeLabel } from "@/lib/date";

function displayName(id: string) {
  return members[id as MemberId]?.name ?? id;
}

export default function SemanaPage() {
  const [view, setView] = useState<"atual" | "anteriores">("atual");

  const [entries, setEntries] = useState<Entry[] | null>(null);
  const [from, setFrom] = useState<string | null>(null);
  const [to, setTo] = useState<string | null>(null);
  const [todayIso, setTodayIso] = useState<string | null>(null);
  const [loadError, setLoadError] = useState(false);

  function applyData(list: EntryList) {
    setEntries(list.entries);
    setFrom(list.from);
    setTo(list.to);
    setTodayIso(list.today);
  }

  function retry() {
    setLoadError(false);
    listEntries("week")
      .then(applyData)
      .catch(() => setLoadError(true));
  }

  useEffect(() => {
    let active = true;
    listEntries("week")
      .then((list) => active && applyData(list))
      .catch(() => active && setLoadError(true));
    return () => {
      active = false;
    };
  }, []);

  // Group entries by day, newest/today first.
  const byDay = new Map<string, Entry[]>();
  (entries ?? []).forEach((e) => {
    byDay.set(e.occurredOn, [...(byDay.get(e.occurredOn) ?? []), e]);
  });
  const days = [...byDay.keys()].sort((a, b) => (a < b ? 1 : a > b ? -1 : 0));
  const total = (entries ?? []).reduce((n, e) => n + e.points, 0);

  return (
    <>
      <TopBar title="Semana" caption={view === "atual" ? (from && to ? weekRangeLabel(from, to) : undefined) : "Quem já venceu"} />
      <Segmented
        value={view}
        onChange={setView}
        options={[
          { value: "atual", label: "Esta semana" },
          { value: "anteriores", label: "Anteriores" },
        ]}
      />

      {view === "atual" &&
        (entries === null ? (
          loadError ? (
            <EmptyState icon="calendar" title="Não deu para carregar" text="Confira sua conexão e tente de novo.">
              <Button onClick={retry}>Tentar de novo</Button>
            </EmptyState>
          ) : (
            <p className="bq-caption" role="status">
              Carregando semana…
            </p>
          )
        ) : entries.length === 0 ? (
          <EmptyState icon="calendar" title="A história começa hoje" text="Complete a primeira quest e a semana aparece aqui.">
            <Link href="/hoje" className="bq-btn">
              Ir para Hoje
            </Link>
          </EmptyState>
        ) : (
          <>
            <ScoreHeader score={total} label="pontos na semana" />
            <div className="bq-list" style={{ gap: 12 }}>
              {days.map((day) => {
                const dayEntries = byDay.get(day) ?? [];
                const gained = dayEntries.filter((e) => e.points > 0).reduce((n, e) => n + e.points, 0);
                const lost = dayEntries.filter((e) => e.points < 0).reduce((n, e) => n + e.points, 0);
                return (
                  <DayCard key={day} title={dayLabel(day, day === todayIso)} gained={gained} lost={lost} photos={0}>
                    {dayEntries.map((e) =>
                      e.scoreType === "sum" ? (
                        <Badge key={e.id} kind="gain" icon="check">
                          {e.ruleName}
                        </Badge>
                      ) : (
                        <Badge key={e.id} kind="loss">
                          {e.ruleName} · por {displayName(e.loggedBy)}
                        </Badge>
                      ),
                    )}
                  </DayCard>
                );
              })}
            </div>
          </>
        ))}

      {view === "anteriores" &&
        (pastWeeks.length === 0 ? (
          <EmptyState icon="trophy" title="Ainda sem vencedores" text="O primeiro vencedor aparece aqui no fim da semana." />
        ) : (
          <div className="bq-list" style={{ gap: 12 }}>
            {pastWeeks.map((w) => (
              <RankRow key={w.range} member={members[w.winner]} points={w.points} crown>
                <p className="bq-caption">{w.range}</p>
                <p className="bq-caption bq-row" style={{ gap: 4 }}>
                  <Icon name="trophy" size={14} />
                  {w.prize}
                </p>
              </RankRow>
            ))}
          </div>
        ))}
    </>
  );
}
