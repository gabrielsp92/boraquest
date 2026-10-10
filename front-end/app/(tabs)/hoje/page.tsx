"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { Avatar, Banner, Button, Confetti, EmptyState, Points, QuestRow, ScoreHeader, Sheet, TopBar } from "@/components/ui";
import { Icon } from "@/components/Icon";
import { useSession } from "@/lib/auth";
import { todayCaption } from "@/lib/date";
import { DEFAULT_AVATAR } from "@/lib/avatars";
import {
  ApiError,
  createEntry,
  deleteEntry,
  listEntries,
  listGuildMembers,
  listRules,
  rulePoints,
  type Entry,
  type EntryList,
  type GuildMember,
  type GuildMemberList,
  type Rule,
  type RuleList,
} from "@/lib/api";

export default function HojePage() {
  const session = useSession();

  const [rules, setRules] = useState<Rule[] | null>(null);
  const [entries, setEntries] = useState<Entry[] | null>(null);
  const [todayIso, setTodayIso] = useState<string | null>(null);
  const [guildMembers, setGuildMembers] = useState<GuildMember[] | null>(null);
  const [loadError, setLoadError] = useState(false);

  const [busyRuleIds, setBusyRuleIds] = useState<Set<string>>(new Set());
  const [rowErrors, setRowErrors] = useState<Record<string, string>>({});
  const [photos, setPhotos] = useState<Record<string, string>>({});
  const [justChecked, setJustChecked] = useState<string | null>(null);

  const [slipOpen, setSlipOpen] = useState(false);
  const [slipWho, setSlipWho] = useState<string>("");
  const [slipWhat, setSlipWhat] = useState<string | undefined>(undefined);
  const [slipBusy, setSlipBusy] = useState(false);
  const [slipFormError, setSlipFormError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const fileInput = useRef<HTMLInputElement>(null);

  function applyData(ruleList: RuleList, entryList: EntryList, memberList: GuildMemberList) {
    setRules(ruleList.rules);
    setEntries(entryList.entries);
    setTodayIso(entryList.today);
    setGuildMembers(memberList.members);
  }

  function retry() {
    setLoadError(false);
    Promise.all([listRules(), listEntries("today"), listGuildMembers()])
      .then(([rl, el, ml]) => applyData(rl, el, ml))
      .catch(() => setLoadError(true));
  }

  useEffect(() => {
    let active = true;
    Promise.all([listRules(), listEntries("today"), listGuildMembers()])
      .then(([rl, el, ml]) => active && applyData(rl, el, ml))
      .catch(() => active && setLoadError(true));
    return () => {
      active = false;
    };
  }, []);

  function refetchSilently() {
    listEntries("today")
      .then((el) => setEntries(el.entries))
      .catch(() => {});
  }

  if (!session || (rules === null && !loadError)) {
    return (
      <>
        <TopBar title="Hoje" />
        <p className="bq-caption" role="status">
          Carregando quests…
        </p>
      </>
    );
  }

  if (loadError || rules === null || entries === null || todayIso === null || guildMembers === null) {
    return (
      <>
        <TopBar title="Hoje" />
        <EmptyState icon="sun" title="Não deu para carregar" text="Confira sua conexão e tente de novo.">
          <Button onClick={retry}>Tentar de novo</Button>
        </EmptyState>
      </>
    );
  }

  const me = session.user.id;
  const memberById = new Map(guildMembers.map((m) => [m.id, m]));
  const displayName = (id: string) => memberById.get(id)?.name ?? id;
  const meMember = memberById.get(me);
  const meAvatarMember = meMember && { id: meMember.id, name: meMember.name, avatar: DEFAULT_AVATAR };

  const myQuests = rules.filter((r) => r.scoreType === "sum" && r.frequency === "daily");
  const negativeQuests = rules.filter((r) => r.scoreType === "decrease");
  const slips = entries.filter((e) => e.scoreType === "decrease" && e.memberId === me);

  const score = entries.reduce((n, e) => n + e.points, 0);
  const doneCount = myQuests.filter((r) => entries.some((e) => e.ruleId === r.id && e.scoreType === "sum")).length;

  async function toggle(rule: Rule) {
    if (busyRuleIds.has(rule.id)) return;
    const existing = entries!.find((e) => e.ruleId === rule.id && e.scoreType === "sum");
    setBusyRuleIds((ids) => new Set(ids).add(rule.id));
    setRowErrors((r) => {
      if (!(rule.id in r)) return r;
      const rest = { ...r };
      delete rest[rule.id];
      return rest;
    });
    try {
      if (existing) {
        await deleteEntry(existing.id);
        setEntries((es) => (es ?? []).filter((e) => e.id !== existing.id));
      } else {
        const created = await createEntry(rule.id, me);
        setEntries((es) => [...(es ?? []), created]);
        setJustChecked(created.id);
      }
    } catch (err) {
      if (existing && err instanceof ApiError && err.status === 404) {
        refetchSilently();
      } else if (!existing && err instanceof ApiError && err.status === 409) {
        refetchSilently();
      } else {
        setRowErrors((r) => ({ ...r, [rule.id]: "Não deu para salvar. Tenta de novo." }));
      }
    } finally {
      setBusyRuleIds((ids) => {
        const next = new Set(ids);
        next.delete(rule.id);
        return next;
      });
    }
  }

  function attachPhoto(file: File | undefined) {
    if (file && justChecked) {
      const url = URL.createObjectURL(file);
      setPhotos((p) => ({ ...p, [justChecked]: url }));
    }
    setJustChecked(null);
  }

  async function confirmSlip() {
    if (!slipWhat) return;
    setSlipBusy(true);
    setSlipFormError(null);
    try {
      const created = await createEntry(slipWhat, slipWho);
      if (slipWho === me) {
        setEntries((es) => [...(es ?? []), created]);
      } else {
        setNotice(slipWho);
      }
      setSlipOpen(false);
    } catch {
      setSlipFormError("Não deu para salvar. Tenta de novo.");
    } finally {
      setSlipBusy(false);
    }
  }

  function openSlipSheet() {
    setSlipWho(me);
    setSlipWhat(negativeQuests[0]?.id);
    setSlipFormError(null);
    setSlipOpen(true);
  }

  if (myQuests.length === 0) {
    return (
      <>
        <TopBar title={`Oi, ${session.user.name}!`} caption={todayCaption(todayIso)} right={meAvatarMember && <Avatar member={meAvatarMember} />} />
        <EmptyState icon="sun" title="Nenhuma quest ainda" text="Crie as primeiras quests da guilda para começar o jogo.">
          <Link href="/regras" className="bq-btn">
            Criar quests
          </Link>
        </EmptyState>
      </>
    );
  }

  const checked = justChecked ? entries.find((e) => e.id === justChecked) : null;
  const slipQuest = slipWhat ? rules.find((r) => r.id === slipWhat) : undefined;

  return (
    <>
      <TopBar title={`Oi, ${session.user.name}!`} caption={todayCaption(todayIso)} right={meAvatarMember && <Avatar member={meAvatarMember} />} />
      <ScoreHeader score={score} label="pontos hoje" done={doneCount} total={myQuests.length} />

      {notice && (
        <Banner
          member={{ id: notice, name: displayName(notice), avatar: DEFAULT_AVATAR }}
          title={`Deslize anotado para ${displayName(notice)}`}
          text={`Vai aparecer como “anotado por ${session.user.name}”.`}
        />
      )}

      <ul className="bq-list">
        {myQuests.map((rule) => {
          const entry = entries.find((e) => e.ruleId === rule.id && e.scoreType === "sum");
          return (
            <li key={rule.id}>
              <QuestRow
                name={rule.name}
                points={rulePoints(rule)}
                kind={entry ? "done" : "todo"}
                meta={entry && photos[entry.id] ? "com foto" : undefined}
                onClick={() => toggle(rule)}
                pressed={!!entry}
                disabled={busyRuleIds.has(rule.id)}
              />
              {rowErrors[rule.id] && (
                <p className="bq-note" role="alert">
                  {rowErrors[rule.id]}
                </p>
              )}
            </li>
          );
        })}
        {slips.map((s) => (
          <li key={s.id}>
            <QuestRow name={s.ruleName} points={s.points} kind="loss" meta={`anotado por ${displayName(s.loggedBy)}`} />
          </li>
        ))}
      </ul>

      {negativeQuests.length > 0 && (
        <Button variant="plain" block icon="minus" onClick={openSlipSheet}>
          Anotar deslize
        </Button>
      )}

      {/* Optional photo after checking a quest */}
      {checked && <Confetti fixed />}
      <Sheet open={!!checked} onClose={() => setJustChecked(null)}>
        <div className="bq-stack bq-center">
          <span className="bq-burst">
            <Icon name="check" size={40} />
          </span>
          <h2 className="bq-title">Quest concluída!</h2>
          {checked && <Points value={checked.points} />}
          <p className="bq-text">Quer guardar uma foto de prova? É opcional.</p>
        </div>
        <div className="bq-stack">
          <Button block icon="camera" onClick={() => fileInput.current?.click()}>
            Adicionar foto
          </Button>
          <Button variant="plain" block onClick={() => setJustChecked(null)}>
            Pular
          </Button>
        </div>
        <input ref={fileInput} type="file" accept="image/*" capture="environment" hidden onChange={(e) => attachPhoto(e.target.files?.[0])} />
      </Sheet>

      {/* Log a slip for anyone in the guild */}
      <Sheet open={slipOpen} onClose={() => (!slipBusy ? setSlipOpen(false) : undefined)}>
        <h2 className="bq-title">Anotar deslize</h2>
        <div className="bq-field">
          <span className="bq-label">Quem escorregou?</span>
          <div className="bq-row" style={{ alignItems: "flex-start" }}>
            {guildMembers.map((m) => (
              <button key={m.id} type="button" className="bq-pick" aria-pressed={m.id === slipWho} onClick={() => setSlipWho(m.id)}>
                <Avatar member={{ id: m.id, name: m.name, avatar: DEFAULT_AVATAR }} selected={m.id === slipWho} />
                <span>{m.name}</span>
              </button>
            ))}
          </div>
        </div>
        <div className="bq-field">
          <span className="bq-label">O que foi?</span>
          <div className="bq-list">
            {negativeQuests.map((r) => (
              <QuestRow
                key={r.id}
                name={r.name}
                points={rulePoints(r)}
                kind="loss"
                selected={r.id === slipWhat}
                pressed={r.id === slipWhat}
                onClick={() => setSlipWhat(r.id)}
              />
            ))}
          </div>
        </div>
        {slipFormError && (
          <p className="bq-note" role="alert">
            {slipFormError}
          </p>
        )}
        <Button block onClick={confirmSlip} disabled={slipBusy || !slipWhat}>
          Confirmar {slipQuest ? `${rulePoints(slipQuest) > 0 ? "+" : "−"}${Math.abs(rulePoints(slipQuest))}` : ""} para {memberById.get(slipWho)?.name ?? slipWho}
        </Button>
        <p className="bq-caption text-center">Vai aparecer como “anotado por {session.user.name}”.</p>
      </Sheet>
    </>
  );
}
