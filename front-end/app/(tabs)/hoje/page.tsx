"use client";

import { useRef, useState } from "react";
import Link from "next/link";
import { Avatar, Banner, Button, Confetti, EmptyState, Points, QuestRow, ScoreHeader, Sheet, TopBar } from "@/components/ui";
import { Icon } from "@/components/Icon";
import { me, memberList, members, questById, quests, today, todayDone, todaySlips, type MemberId } from "@/lib/data";

const myQuests = quests.filter((q) => q.points > 0 && q.frequency === "diaria");
const negativeQuests = quests.filter((q) => q.points < 0);

export default function HojePage() {
  const [done, setDone] = useState(todayDone);
  const [slips, setSlips] = useState(todaySlips);
  const [justChecked, setJustChecked] = useState<string | null>(null);
  const [slipOpen, setSlipOpen] = useState(false);
  const [slipWho, setSlipWho] = useState<MemberId>(me);
  const [slipWhat, setSlipWhat] = useState(negativeQuests[0]?.id);
  const [notice, setNotice] = useState<MemberId | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  const isDone = (id: string) => done.some((d) => d.questId === id);
  const score =
    done.reduce((n, d) => n + questById(d.questId).points, 0) + slips.reduce((n, s) => n + questById(s.questId).points, 0);

  function toggle(id: string) {
    if (isDone(id)) {
      setDone(done.filter((d) => d.questId !== id));
    } else {
      setDone([...done, { questId: id }]);
      setJustChecked(id); // opens the optional photo sheet
    }
  }

  function attachPhoto(file: File | undefined) {
    if (file && justChecked) {
      const url = URL.createObjectURL(file);
      setDone((d) => d.map((x) => (x.questId === justChecked ? { ...x, photo: url } : x)));
    }
    setJustChecked(null);
  }

  function confirmSlip() {
    if (!slipWhat) return;
    if (slipWho === me) setSlips([...slips, { questId: slipWhat, by: me }]);
    else setNotice(slipWho);
    setSlipOpen(false);
  }

  if (myQuests.length === 0) {
    return (
      <>
        <TopBar title={`Oi, ${members[me].name}!`} caption={today.label} right={<Avatar member={members[me]} />} />
        <EmptyState icon="sun" title="Nenhuma quest ainda" text="Crie as primeiras quests da guilda para começar o jogo.">
          <Link href="/regras" className="bq-btn">
            Criar quests
          </Link>
        </EmptyState>
      </>
    );
  }

  const checked = justChecked ? questById(justChecked) : null;
  const slipQuest = slipWhat ? questById(slipWhat) : null;

  return (
    <>
      <TopBar title={`Oi, ${members[me].name}!`} caption={today.label} right={<Avatar member={members[me]} />} />
      <ScoreHeader score={score} label="pontos hoje" done={done.length} total={myQuests.length} />

      {notice && <Banner member={members[notice]} title={`Deslize anotado para ${members[notice].name}`} text={`Vai aparecer como “anotado por ${members[me].name}”.`} />}

      <ul className="bq-list">
        {myQuests.map((q) => {
          const entry = done.find((d) => d.questId === q.id);
          return (
            <li key={q.id}>
              <QuestRow name={q.name} points={q.points} kind={entry ? "done" : "todo"} meta={entry?.photo ? "com foto" : undefined} onClick={() => toggle(q.id)} pressed={!!entry} />
            </li>
          );
        })}
        {slips.map((s, i) => (
          <li key={i}>
            <QuestRow name={questById(s.questId).name} points={questById(s.questId).points} kind="loss" meta={`anotado por ${members[s.by].name}`} />
          </li>
        ))}
      </ul>

      <Button variant="plain" block icon="minus" onClick={() => setSlipOpen(true)}>
        Anotar deslize
      </Button>

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
      <Sheet open={slipOpen} onClose={() => setSlipOpen(false)}>
        <h2 className="bq-title">Anotar deslize</h2>
        <div className="bq-field">
          <span className="bq-label">Quem escorregou?</span>
          <div className="bq-row" style={{ alignItems: "flex-start" }}>
            {memberList.map((m) => (
              <button key={m.id} type="button" className="bq-pick" aria-pressed={m.id === slipWho} onClick={() => setSlipWho(m.id)}>
                <Avatar member={m} selected={m.id === slipWho} />
                <span>{m.name}</span>
              </button>
            ))}
          </div>
        </div>
        <div className="bq-field">
          <span className="bq-label">O que foi?</span>
          <div className="bq-list">
            {negativeQuests.map((q) => (
              <QuestRow key={q.id} name={q.name} points={q.points} kind="loss" selected={q.id === slipWhat} pressed={q.id === slipWhat} onClick={() => setSlipWhat(q.id)} />
            ))}
          </div>
        </div>
        <Button block onClick={confirmSlip}>
          Confirmar {slipQuest ? `${slipQuest.points > 0 ? "+" : "−"}${Math.abs(slipQuest.points)}` : ""} para {members[slipWho].name}
        </Button>
        <p className="bq-caption text-center">Vai aparecer como “anotado por {members[me].name}”.</p>
      </Sheet>
    </>
  );
}
