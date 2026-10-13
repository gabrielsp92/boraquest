"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { Badge, Button, DayCard, EmptyState, RankRow, ScoreHeader, Segmented, Sheet, TopBar } from "@/components/ui";
import { Icon } from "@/components/Icon";
import { getWeekHistory, listEntries, listGuildMembers, type Entry, type EntryList, type GuildMember, type WeekResult } from "@/lib/api";
import { memberAvatarSrc } from "@/lib/avatars";
import { dayLabel, weekRangeLabel } from "@/lib/date";

export default function SemanaPage() {
  const [view, setView] = useState<"atual" | "anteriores">("atual");

  const [entries, setEntries] = useState<Entry[] | null>(null);
  const [from, setFrom] = useState<string | null>(null);
  const [to, setTo] = useState<string | null>(null);
  const [todayIso, setTodayIso] = useState<string | null>(null);
  const [guildMembers, setGuildMembers] = useState<GuildMember[] | null>(null);
  const [loadError, setLoadError] = useState(false);
  const [viewingPhoto, setViewingPhoto] = useState<string | null>(null);

  // Past weeks: fetched the first time "Anteriores" is opened, then kept. Not
  // tied to the tab, so switching away mid-request never discards the response.
  const [pastWeeks, setPastWeeks] = useState<WeekResult[] | null>(null);
  const [pastLoadError, setPastLoadError] = useState(false);
  const pastLoading = useRef(false);

  function applyData(list: EntryList) {
    setEntries(list.entries);
    setFrom(list.from);
    setTo(list.to);
    setTodayIso(list.today);
  }

  function retry() {
    setLoadError(false);
    Promise.all([listEntries("week"), listGuildMembers()])
      .then(([list, { members }]) => {
        applyData(list);
        setGuildMembers(members);
      })
      .catch(() => setLoadError(true));
  }

  useEffect(() => {
    let active = true;
    Promise.all([listEntries("week"), listGuildMembers()])
      .then(([list, { members }]) => {
        if (!active) return;
        applyData(list);
        setGuildMembers(members);
      })
      .catch(() => active && setLoadError(true));
    return () => {
      active = false;
    };
  }, []);

  function loadPast() {
    if (pastWeeks !== null || pastLoading.current) return;
    pastLoading.current = true;
    setPastLoadError(false);
    getWeekHistory()
      .then(({ weeks }) => setPastWeeks(weeks))
      .catch(() => setPastLoadError(true))
      .finally(() => {
        pastLoading.current = false;
      });
  }

  function changeView(next: "atual" | "anteriores") {
    setView(next);
    if (next === "anteriores") loadPast();
  }

  const memberById = new Map((guildMembers ?? []).map((m) => [m.id, m]));
  const displayName = (id: string) => memberById.get(id)?.name ?? id;
  const toMember = (id: string) => ({ id, name: displayName(id), avatar: memberAvatarSrc(memberById.get(id)) });

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
        onChange={changeView}
        options={[
          { value: "atual", label: "Esta semana" },
          { value: "anteriores", label: "Anteriores" },
        ]}
      />

      {view === "atual" &&
        (entries === null || guildMembers === null ? (
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
                const photos = dayEntries.filter((e) => e.scoreType === "sum" && e.photoUrl).map((e) => ({ id: e.id, url: e.photoUrl }));
                return (
                  <DayCard
                    key={day}
                    title={dayLabel(day, day === todayIso)}
                    gained={gained}
                    lost={lost}
                    photos={photos}
                    onPhotoClick={setViewingPhoto}
                  >
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
        (pastWeeks === null || guildMembers === null ? (
          pastLoadError || loadError ? (
            <EmptyState icon="trophy" title="Não deu para carregar" text="Confira sua conexão e tente de novo.">
              <Button onClick={pastLoadError ? loadPast : retry}>Tentar de novo</Button>
            </EmptyState>
          ) : (
            <p className="bq-caption" role="status">
              Carregando semanas anteriores…
            </p>
          )
        ) : pastWeeks.length === 0 ? (
          <EmptyState icon="trophy" title="Ainda sem vencedores" text="O primeiro vencedor aparece aqui no fim da semana." />
        ) : (
          <div className="bq-list" style={{ gap: 12 }}>
            {pastWeeks.map((w) => (
              <RankRow key={w.from} member={toMember(w.winner.memberId)} points={w.winner.points} crown>
                <p className="bq-caption">{weekRangeLabel(w.from, w.to)}</p>
                {w.prize && (
                  <p className="bq-caption bq-row" style={{ gap: 4 }}>
                    <Icon name="trophy" size={14} />
                    {w.prize}
                  </p>
                )}
              </RankRow>
            ))}
          </div>
        ))}

      <Sheet open={!!viewingPhoto} onClose={() => setViewingPhoto(null)}>
        {/* eslint-disable-next-line @next/next/no-img-element -- Cloudinary already serves a size-capped variant */}
        {viewingPhoto && <img className="bq-photo-full" src={viewingPhoto} alt="" />}
      </Sheet>
    </>
  );
}
