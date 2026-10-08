"use client";

import { useEffect, useState } from "react";
import { Button, EmptyState, Field, QuestRow, Segmented, Sheet, Stepper, TopBar } from "@/components/ui";
import { ApiError, createRule, deleteRule, listRules, rulePoints, updateRule, type Rule, type RuleFrequency, type RuleList, type ScoreType } from "@/lib/api";
import { clearSession } from "@/lib/auth";
import { prizes as initialPrizes } from "@/lib/data";

const frequencyLabel: Record<RuleFrequency, string> = { daily: "Diária", weekly: "Semanal" };

function errorMessage(err: unknown, limit: number) {
  if (err instanceof ApiError && err.status === 409) return `Sua guilda já tem ${limit} quests. Exclua uma para criar outra.`;
  if (err instanceof ApiError && err.status === 400) return "Confira os campos e tente de novo.";
  return "Algo deu errado. Tente de novo.";
}

export default function RegrasPage() {
  const [rules, setRules] = useState<Rule[] | null>(null);
  const [limit, setLimit] = useState(40);
  const [loadError, setLoadError] = useState(false);
  const [prizes, setPrizes] = useState(initialPrizes);

  // The quest sheet: "new" creates, a Rule edits that rule, null is closed.
  const [editing, setEditing] = useState<Rule | "new" | null>(null);
  const [name, setName] = useState("");
  const [frequency, setFrequency] = useState<RuleFrequency>("daily");
  const [scoreType, setScoreType] = useState<ScoreType>("sum");
  const [score, setScore] = useState(10);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

  function applyList(list: RuleList) {
    setRules(list.rules);
    setLimit(list.limit);
  }

  function retry() {
    setLoadError(false);
    listRules()
      .then(applyList)
      .catch(() => setLoadError(true));
  }

  useEffect(() => {
    let active = true;
    listRules()
      .then((list) => active && applyList(list))
      .catch(() => active && setLoadError(true));
    return () => {
      active = false;
    };
  }, []);

  function open(rule: Rule | "new") {
    setEditing(rule);
    setName(rule === "new" ? "" : rule.name);
    setFrequency(rule === "new" ? "daily" : rule.frequency);
    setScoreType(rule === "new" ? "sum" : rule.scoreType);
    setScore(rule === "new" ? 10 : rule.score);
    setFormError(null);
    setConfirmDelete(false);
  }

  function close() {
    if (!busy) setEditing(null);
  }

  async function save() {
    const trimmed = name.trim();
    if (!trimmed || !editing) return;
    setBusy(true);
    setFormError(null);
    const input = { name: trimmed, frequency, scoreType, score };
    try {
      if (editing === "new") {
        const created = await createRule(input);
        setRules((rs) => [...(rs ?? []), created]);
      } else {
        const updated = await updateRule(editing.id, input);
        setRules((rs) => (rs ?? []).map((r) => (r.id === updated.id ? updated : r)));
      }
      setEditing(null);
    } catch (err) {
      setFormError(errorMessage(err, limit));
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!editing || editing === "new") return;
    if (!confirmDelete) {
      setConfirmDelete(true);
      return;
    }
    setBusy(true);
    setFormError(null);
    try {
      await deleteRule(editing.id);
      setRules((rs) => (rs ?? []).filter((r) => r.id !== editing.id));
      setEditing(null);
    } catch (err) {
      // Someone else already deleted it: drop it from the list too.
      if (err instanceof ApiError && err.status === 404) {
        setRules((rs) => (rs ?? []).filter((r) => r.id !== editing.id));
        setEditing(null);
      } else {
        setFormError(errorMessage(err, limit));
      }
    } finally {
      setBusy(false);
    }
  }

  const atLimit = rules !== null && rules.length >= limit;

  return (
    <>
      <TopBar
        title="Regras"
        caption="As regras da guilda"
        right={
          <Button variant="small" onClick={clearSession}>
            Sair
          </Button>
        }
      />

      {rules === null ? (
        loadError ? (
          <EmptyState icon="scroll" title="Não deu para carregar" text="Confira sua conexão e tente de novo.">
            <Button onClick={retry}>Tentar de novo</Button>
          </EmptyState>
        ) : (
          <p className="bq-caption" role="status">
            Carregando quests…
          </p>
        )
      ) : rules.length === 0 ? (
        <EmptyState icon="scroll" title="Sua guilda está novinha" text="Comece com uma quest simples, como beber água.">
          <Button onClick={() => open("new")}>Criar primeira quest</Button>
        </EmptyState>
      ) : (
        <>
          <h2 className="bq-heading">Quests</h2>
          <ul className="bq-list -mt-2">
            {rules.map((r) => (
              <li key={r.id}>
                <QuestRow name={r.name} points={rulePoints(r)} kind={r.scoreType === "sum" ? "gain" : "loss"} meta={frequencyLabel[r.frequency]} onClick={() => open(r)} />
              </li>
            ))}
          </ul>
          <Button block icon="plus" onClick={() => open("new")} disabled={atLimit}>
            Nova quest
          </Button>
          <p className="bq-caption -mt-2">
            {atLimit ? `Limite de ${limit} quests atingido. Exclua uma para criar outra.` : `${rules.length} de ${limit} quests`}
          </p>

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

      <Sheet open={editing !== null} onClose={close}>
        <h2 className="bq-title">{editing === "new" ? "Nova quest" : "Editar quest"}</h2>
        <Field label="Nome">
          <input className="bq-input" value={name} onChange={(e) => setName(e.target.value)} placeholder="Beber 2 L de água" maxLength={32} />
        </Field>
        <div className="bq-field">
          <span className="bq-label">Frequência</span>
          <Segmented
            value={frequency}
            onChange={setFrequency}
            options={[
              { value: "daily", label: "Diária" },
              { value: "weekly", label: "Semanal" },
            ]}
          />
        </div>
        <div className="bq-field">
          <span className="bq-label">Pontos</span>
          <Segmented
            value={scoreType}
            onChange={setScoreType}
            options={[
              { value: "sum", label: "Soma +" },
              { value: "decrease", label: "Desconta −" },
            ]}
          />
          <div className="pt-2">
            <Stepper value={score} onChange={setScore} format={(v) => `${scoreType === "sum" ? "+" : "−"}${v}`} />
          </div>
        </div>
        {formError && (
          <p className="bq-note" role="alert">
            {formError}
          </p>
        )}
        <Button block onClick={save} disabled={busy || !name.trim()}>
          {busy ? "Salvando…" : "Salvar quest"}
        </Button>
        {editing !== null && editing !== "new" && (
          <Button block variant="ghost" onClick={remove} disabled={busy}>
            {confirmDelete ? "Toque de novo para excluir" : "Excluir quest"}
          </Button>
        )}
      </Sheet>
    </>
  );
}
