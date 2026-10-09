"use client";

import { useEffect, useRef, useState } from "react";
import { Button, EmptyState, Field, QuestRow, Segmented, Sheet, Stepper, TopBar } from "@/components/ui";
import {
  ApiError,
  createRule,
  deleteRule,
  getPrizes,
  listRules,
  rulePoints,
  setPrize,
  updateRule,
  type Prizes,
  type Rule,
  type RuleFrequency,
  type RuleList,
  type ScoreType,
} from "@/lib/api";
import { clearSession } from "@/lib/auth";

const frequencyLabel: Record<RuleFrequency, string> = { daily: "Diária", weekly: "Semanal" };

type FieldStatus = "idle" | "saving" | "saved" | "error";

function errorMessage(err: unknown, limit: number) {
  if (err instanceof ApiError && err.status === 409) return `Sua guilda já tem ${limit} quests. Exclua uma para criar outra.`;
  if (err instanceof ApiError && err.status === 400) return "Confira os campos e tente de novo.";
  return "Algo deu errado. Tente de novo.";
}

export default function RegrasPage() {
  const [rules, setRules] = useState<Rule[] | null>(null);
  const [limit, setLimit] = useState(40);
  const [loadError, setLoadError] = useState(false);

  // Prizes: independent load/save state from the rules list above.
  const [prizes, setPrizes] = useState<Prizes | null>(null);
  const [prizesLoadError, setPrizesLoadError] = useState(false);
  const [weekInput, setWeekInput] = useState("");
  const [monthInput, setMonthInput] = useState("");
  const [weekStatus, setWeekStatus] = useState<FieldStatus>("idle");
  const [monthStatus, setMonthStatus] = useState<FieldStatus>("idle");
  const weekSavedTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const monthSavedTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);

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

  function applyPrizes(p: Prizes) {
    setPrizes(p);
    setWeekInput(p.week);
    setMonthInput(p.month);
  }

  function retryPrizes() {
    setPrizesLoadError(false);
    getPrizes()
      .then(applyPrizes)
      .catch(() => setPrizesLoadError(true));
  }

  useEffect(() => {
    let active = true;
    getPrizes()
      .then((p) => active && applyPrizes(p))
      .catch(() => active && setPrizesLoadError(true));
    return () => {
      active = false;
    };
  }, []);

  useEffect(
    () => () => {
      if (weekSavedTimeout.current) clearTimeout(weekSavedTimeout.current);
      if (monthSavedTimeout.current) clearTimeout(monthSavedTimeout.current);
    },
    [],
  );

  function blurWeek() {
    const trimmed = weekInput.trim();
    if (!prizes || trimmed === prizes.week) return;
    setWeekStatus("saving");
    setPrize("week", trimmed)
      .then((updated) => {
        setPrizes(updated);
        setWeekStatus("saved");
        if (weekSavedTimeout.current) clearTimeout(weekSavedTimeout.current);
        weekSavedTimeout.current = setTimeout(() => setWeekStatus("idle"), 2000);
      })
      .catch(() => setWeekStatus("error"));
  }

  function blurMonth() {
    const trimmed = monthInput.trim();
    if (!prizes || trimmed === prizes.month) return;
    setMonthStatus("saving");
    setPrize("month", trimmed)
      .then((updated) => {
        setPrizes(updated);
        setMonthStatus("saved");
        if (monthSavedTimeout.current) clearTimeout(monthSavedTimeout.current);
        monthSavedTimeout.current = setTimeout(() => setMonthStatus("idle"), 2000);
      })
      .catch(() => setMonthStatus("error"));
  }

  function prizeFieldCaption(status: FieldStatus) {
    if (!prizes) return prizesLoadError ? null : <p className="bq-caption">Carregando…</p>;
    if (status === "saving") return <p className="bq-caption">Salvando…</p>;
    if (status === "saved") return <p className="bq-caption">Salvo</p>;
    if (status === "error")
      return (
        <p className="bq-note" role="alert">
          Não deu para salvar. Tenta de novo.
        </p>
      );
    return null;
  }

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
              <input
                className="bq-input"
                value={weekInput}
                onChange={(e) => setWeekInput(e.target.value)}
                onBlur={blurWeek}
                disabled={!prizes || weekStatus === "saving"}
              />
              {prizeFieldCaption(weekStatus)}
            </Field>
            <Field label="Prêmio do mês">
              <input
                className="bq-input"
                value={monthInput}
                onChange={(e) => setMonthInput(e.target.value)}
                onBlur={blurMonth}
                disabled={!prizes || monthStatus === "saving"}
              />
              {prizeFieldCaption(monthStatus)}
            </Field>
            {prizesLoadError && (
              <>
                <p className="bq-note" role="alert">
                  Não deu para carregar. Tenta de novo.
                </p>
                <Button variant="small" onClick={retryPrizes}>
                  Tentar de novo
                </Button>
              </>
            )}
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
