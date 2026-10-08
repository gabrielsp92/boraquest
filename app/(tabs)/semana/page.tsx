"use client";

import { useState } from "react";
import Link from "next/link";
import { Badge, DayCard, EmptyState, RankRow, ScoreHeader, Segmented, TopBar } from "@/components/ui";
import { Icon } from "@/components/Icon";
import { members, pastWeeks, questById, sum, today, week } from "@/lib/data";

export default function SemanaPage() {
  const [view, setView] = useState<"atual" | "anteriores">("atual");
  const total = week.reduce((n, d) => n + sum(d.done) + sum(d.slips.map((s) => s.questId)), 0);

  return (
    <>
      <TopBar title="Semana" caption={view === "atual" ? today.week : "Quem já venceu"} />
      <Segmented
        value={view}
        onChange={setView}
        options={[
          { value: "atual", label: "Esta semana" },
          { value: "anteriores", label: "Anteriores" },
        ]}
      />

      {view === "atual" &&
        (week.length === 0 ? (
          <EmptyState icon="calendar" title="A história começa hoje" text="Complete a primeira quest e a semana aparece aqui.">
            <Link href="/hoje" className="bq-btn">
              Ir para Hoje
            </Link>
          </EmptyState>
        ) : (
          <>
            <ScoreHeader score={total} label="pontos na semana" />
            <div className="bq-list" style={{ gap: 12 }}>
              {week.map((d) => (
                <DayCard key={d.label} title={d.label} gained={sum(d.done)} lost={sum(d.slips.map((s) => s.questId))} photos={d.photos}>
                  {d.done.map((id) => (
                    <Badge key={id} kind="gain" icon="check">
                      {questById(id).short}
                    </Badge>
                  ))}
                  {d.slips.map((s, i) => (
                    <Badge key={i} kind="loss">
                      {questById(s.questId).short} · por {members[s.by].name}
                    </Badge>
                  ))}
                </DayCard>
              ))}
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
