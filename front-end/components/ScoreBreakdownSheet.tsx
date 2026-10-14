"use client";

import { useEffect, useState } from "react";
import { getScoreBreakdown, type QuestTotal, type ScoreBreakdown, type ScoreboardPeriod } from "@/lib/api";
import { Avatar, Button, EmptyState, Sheet } from "./ui";

type Member = { id: string; name: string; avatar: string };

/**
 * The Guilda breakdown sheet: the quests behind a member's points for the period
 * shown by the Semana/Mês toggle. Mount it only while open; it fetches on mount.
 */
export function ScoreBreakdownSheet({ member, period, onClose }: { member: Member; period: ScoreboardPeriod; onClose: () => void }) {
  const [attempt, setAttempt] = useState(0);
  // Which request the result belongs to; while it doesn't match, the sheet is loading.
  const key = `${member.id}|${period}|${attempt}`;
  const [result, setResult] = useState<{ key: string; data: ScoreBreakdown | null } | null>(null);

  useEffect(() => {
    let active = true;
    getScoreBreakdown(member.id, period).then(
      (data) => active && setResult({ key, data }),
      () => active && setResult({ key, data: null }),
    );
    return () => {
      active = false;
    };
  }, [member.id, period, key]);

  const loaded = result?.key === key ? result : null;
  const quests = loaded?.data?.items.filter((i) => i.scoreType === "sum") ?? [];
  const slips = loaded?.data?.items.filter((i) => i.scoreType === "decrease") ?? [];
  const thisPeriod = period === "week" ? "esta semana" : "este mês";

  return (
    <Sheet open onClose={onClose}>
      <div className="bq-stack bq-center">
        <Avatar member={member} size="lg" />
        <h2 className="bq-title">{member.name}</h2>
        <p className="bq-caption">{period === "week" ? "Esta semana" : "Este mês"}</p>
      </div>

      {!loaded ? (
        <p className="bq-caption" role="status">
          Carregando…
        </p>
      ) : !loaded.data ? (
        <EmptyState icon="users" title="Não deu para carregar" text="Confira sua conexão e tente de novo.">
          <Button onClick={() => setAttempt(attempt + 1)}>Tentar de novo</Button>
        </EmptyState>
      ) : loaded.data.items.length === 0 ? (
        <EmptyState icon="users" title="Nada por aqui ainda" text={`${member.name} ainda não marcou quests ${thisPeriod}.`} />
      ) : (
        <>
          {quests.length > 0 && <TallySection title="Quests" items={quests} />}
          {slips.length > 0 && <TallySection title="Deslizes" items={slips} />}
          <div className="bq-row bq-tally__total">
            <p className="bq-label bq-grow">Total</p>
            <p className="bq-label">{signed(loaded.data.total)}</p>
          </div>
        </>
      )}

      <Button variant="ghost" block onClick={onClose}>
        Fechar
      </Button>
    </Sheet>
  );
}

function TallySection({ title, items }: { title: string; items: QuestTotal[] }) {
  return (
    <section className="bq-tally">
      <h3 className="bq-heading">{title}</h3>
      <ul className="bq-list">
        {items.map((i) => (
          <li key={`${i.ruleId}|${i.scoreType}`} className="bq-row">
            <span className="bq-text bq-grow">{i.ruleName}</span>
            <span className="bq-label">{i.count}×</span>
            <span className="bq-label">{`${i.scoreType === "decrease" ? "−" : "+"}${Math.abs(i.points)}`}</span>
          </li>
        ))}
      </ul>
    </section>
  );
}

// Same format as the card's score (RankRow): no sign when positive, U+2212 when negative.
const signed = (n: number) => (n < 0 ? `−${Math.abs(n)}` : `${n}`);
