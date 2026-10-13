"use client";

import { useEffect, useId, useRef, useState } from "react";
import { Button, Sheet } from "@/components/ui";
import { Icon } from "@/components/Icon";
import { getPushPublicKey } from "@/lib/api";
import { detectReminderState, disableReminder, enableReminder, registerServiceWorker, type ReminderState } from "@/lib/push";

const TITLE = "Lembrete às 20h";

const captions: Record<Exclude<ReminderState, "busy">, string> = {
  checking: "Verificando…",
  "ios-install": "Instale o app para ligar.",
  "ios-too-old": "Precisa do iOS 16.4 ou mais novo.",
  unsupported: "Este navegador não recebe lembretes.",
  denied: "Notificações bloqueadas. Libere nos ajustes do aparelho.",
  off: "Um aviso se faltar quest no dia.",
  on: "Ligado neste aparelho.",
  error: "Um aviso se faltar quest no dia.",
};

/** Guilda's per-device switch for the daily 20h reminder (todo/daily-reminder-push.md). */
export function ReminderSwitch() {
  const captionId = useId();
  const [state, setState] = useState<ReminderState>("checking");
  // While busy, the switch shows the state it is moving to.
  const [target, setTarget] = useState<"on" | "off">("off");
  const [installOpen, setInstallOpen] = useState(false);
  // Preloaded on mount so the tap handler can call pushManager.subscribe before any other await.
  const registration = useRef<ServiceWorkerRegistration | null>(null);
  const publicKey = useRef<string | null>(null);

  useEffect(() => {
    let active = true;
    Promise.all([
      detectReminderState().catch(() => "unsupported" as const),
      registerServiceWorker(),
      getPushPublicKey().then(
        (r) => r.publicKey,
        () => null,
      ),
    ]).then(([detected, reg, key]) => {
      if (!active) return;
      registration.current = reg;
      publicKey.current = key;
      setState(detected);
    });
    return () => {
      active = false;
    };
  }, []);

  function turnOn() {
    const reg = registration.current;
    const key = publicKey.current;
    if (!reg || !key) {
      setState("error");
      // The preload failed (slow first install, flaky network): fetch again so the next tap can work.
      if (!reg) void registerServiceWorker().then((r) => (registration.current = r));
      if (!key) void getPushPublicKey().then((r) => (publicKey.current = r.publicKey), () => undefined);
      return;
    }
    setTarget("on");
    setState("busy");
    // No await before this call: enableReminder's first await is pushManager.subscribe.
    void enableReminder(reg, key).then(setState);
  }

  function turnOff() {
    setTarget("off");
    setState("busy");
    void disableReminder().then(() => setState("off"));
  }

  function onTap() {
    if (state === "ios-install") setInstallOpen(true);
    else if (state === "off" || state === "error") turnOn();
    else if (state === "on") turnOff();
  }

  const checked = state === "on" || (state === "busy" && target === "on");
  const disabled = state === "checking" || state === "ios-too-old" || state === "unsupported" || state === "denied" || state === "busy";
  const caption = state === "busy" ? captions[target] : captions[state];

  return (
    <>
      {/* The whole row is the tap target; the switch inside is the accessible control. */}
      <div className={`bq-reminder${disabled ? " bq-reminder--disabled" : ""}`} onClick={onTap}>
        <span className="bq-reminder__icon">
          <Icon name="bell" />
        </span>
        <div className="bq-grow">
          <p className="bq-text">{TITLE}</p>
          <p className="bq-caption" id={captionId}>
            {caption}
          </p>
        </div>
        <button
          type="button"
          role="switch"
          className="bq-switch"
          aria-checked={checked}
          aria-label={TITLE}
          aria-describedby={captionId}
          disabled={disabled}
        />
      </div>
      {state === "error" && (
        <p className="bq-caption" role="alert">
          Não deu para ligar o lembrete. Tenta de novo.
        </p>
      )}

      <Sheet open={installOpen} onClose={() => setInstallOpen(false)}>
        <div className="bq-stack">
          <h2 className="bq-title">Instale o Boraquest</h2>
          <p className="bq-text">No iPhone e no iPad, o lembrete só funciona no app instalado.</p>
          <ol className="bq-steps bq-text">
            <li>Toque em Compartilhar no Safari.</li>
            <li>Escolha Adicionar à Tela de Início.</li>
            <li>Abra o Boraquest pelo ícone e ligue o lembrete aqui na Guilda.</li>
          </ol>
          <p className="bq-caption">Se aparecer a opção de abrir como app, deixe ligada.</p>
        </div>
        <Button block onClick={() => setInstallOpen(false)}>
          Entendi
        </Button>
      </Sheet>
    </>
  );
}
