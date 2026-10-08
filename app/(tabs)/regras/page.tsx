"use client";

import { useState } from "react";
import { Button, EmptyState, Field, QuestRow, Segmented, Sheet, Stepper, TopBar } from "@/components/ui";
import { prizes as initialPrizes, quests as initialQuests, type Frequency, type Quest } from "@/lib/data";

const frequencyLabel: Record<Frequency, string> = { diaria: "Diária", semanal: "Semanal" };

export default function RegrasPage() {
  const [quests, setQuests] = useState<Quest[]>(initialQuests);
  const [prizes, setPrizes] = useState(initialPrizes);
  const [adding, setAdding] = useState(false);

  // The new-quest form: three fields only.
  const [name, setName] = useState("");
  const [frequency, setFrequency] = useState<Frequency>("diaria");
  const [sign, setSign] = useState<"soma" | "desconta">("soma");
  const [amount, setAmount] = useState(10);

  function save() {
    const trimmed = name.trim();
    if (!trimmed) return;
    setQuests([...quests, { id: crypto.randomUUID(), name: trimmed, short: trimmed, frequency, points: sign === "soma" ? amount : -amount }]);
    setName("");
    setAdding(false);
  }

  return (
    <>
      <TopBar title="Regras" caption="As regras da guilda" />

      {quests.length === 0 ? (
        <EmptyState icon="scroll" title="Sua guilda está novinha" text="Comece com uma quest simples, como beber água.">
          <Button onClick={() => setAdding(true)}>Criar primeira quest</Button>
        </EmptyState>
      ) : (
        <>
          <h2 className="bq-heading">Quests</h2>
          <ul className="bq-list -mt-2">
            {quests.map((q) => (
              <li key={q.id}>
                <QuestRow name={q.name} points={q.points} kind={q.points > 0 ? "gain" : "loss"} meta={frequencyLabel[q.frequency]} />
              </li>
            ))}
          </ul>
          <Button block icon="plus" onClick={() => setAdding(true)}>
            Nova quest
          </Button>

          <h2 className="bq-heading">Prêmios</h2>
          <div className="bq-stack -mt-2" style={{ gap: 16 }}>
            <Field label="Prêmio da semana">
              <input className="bq-input" value={prizes.week} onChange={(e) => setPrizes({ ...prizes, week: e.target.value })} />
            </Field>
            <Field label="Prêmio do mês">
              <input className="bq-input" value={prizes.month} onChange={(e) => setPrizes({ ...prizes, month: e.target.value })} />
            </Field>
          </div>
        </>
      )}

      <Sheet open={adding} onClose={() => setAdding(false)}>
        <h2 className="bq-title">Nova quest</h2>
        <Field label="Nome">
          <input className="bq-input" value={name} onChange={(e) => setName(e.target.value)} placeholder="Beber 2 L de água" maxLength={32} />
        </Field>
        <div className="bq-field">
          <span className="bq-label">Frequência</span>
          <Segmented
            value={frequency}
            onChange={setFrequency}
            options={[
              { value: "diaria", label: "Diária" },
              { value: "semanal", label: "Semanal" },
            ]}
          />
        </div>
        <div className="bq-field">
          <span className="bq-label">Pontos</span>
          <Segmented
            value={sign}
            onChange={setSign}
            options={[
              { value: "soma", label: "Soma +" },
              { value: "desconta", label: "Desconta −" },
            ]}
          />
          <div className="pt-2">
            <Stepper value={amount} onChange={setAmount} format={(v) => `${sign === "soma" ? "+" : "−"}${v}`} />
          </div>
        </div>
        <Button block onClick={save} disabled={!name.trim()}>
          Salvar quest
        </Button>
      </Sheet>
    </>
  );
}
