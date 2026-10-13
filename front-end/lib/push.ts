// The daily reminder (todo/daily-reminder-push.md): Web Push subscription for this device.
// The switch's on/off state is "this browser has a push subscription"; the API mirrors it.
// Everything here touches navigator/window, so call it only from effects or event handlers.
import { deletePushSubscription, savePushSubscription, type PushSubscriptionInput } from "./api";
import { clearSession, getSession } from "./auth";

export type ReminderState =
  | "checking"
  | "ios-install"
  | "ios-too-old"
  | "unsupported"
  | "denied"
  | "off"
  | "on"
  | "busy"
  | "error";

/** iPhone, iPod or iPad, including iPadOS reporting a desktop "Macintosh" user agent. */
export function isIOS(): boolean {
  return /iPad|iPhone|iPod/.test(navigator.userAgent) || (navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1);
}

/** Running as the installed Home Screen app rather than in a browser tab. */
export function isStandalone(): boolean {
  return (navigator as Navigator & { standalone?: boolean }).standalone === true || window.matchMedia("(display-mode: standalone)").matches;
}

const SW_READY_TIMEOUT_MS = 10000;

function pushSupported(): boolean {
  return "serviceWorker" in navigator && "PushManager" in window && "Notification" in window;
}

/** This device's push subscription, or null when there is none (or push isn't supported). */
async function currentSubscription(): Promise<PushSubscription | null> {
  if (!pushSupported()) return null;
  const reg = await navigator.serviceWorker.getRegistration("/");
  return reg ? reg.pushManager.getSubscription() : null;
}

/** What the reminder switch should show on this device right now. */
export async function detectReminderState(): Promise<Exclude<ReminderState, "checking" | "busy" | "error">> {
  // iOS only allows Web Push in the installed app: never prompt from a Safari tab.
  if (isIOS() && !isStandalone()) return "ios-install";
  if (isIOS() && !("PushManager" in window)) return "ios-too-old";
  if (!pushSupported()) return "unsupported";
  if (Notification.permission === "denied") return "denied";
  return (await currentSubscription()) ? "on" : "off";
}

let registering: Promise<ServiceWorkerRegistration | null> | null = null;

/**
 * Registers /sw.js and resolves once it is active, or null when unsupported or it fails.
 * Shared per app load: the layout's registrar and the Guilda switch get the same promise.
 */
export function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  registering ??= register().then((reg) => {
    if (!reg) registering = null; // let a later call try again
    return reg;
  });
  return registering;
}

async function register(): Promise<ServiceWorkerRegistration | null> {
  if (!("serviceWorker" in navigator)) return null;
  try {
    await navigator.serviceWorker.register("/sw.js", { scope: "/", updateViaCache: "none" });
    // pushManager.subscribe needs an active worker, so wait for it, but not forever
    // (ready never settles if the worker fails to install).
    return await Promise.race([
      navigator.serviceWorker.ready,
      new Promise<null>((resolve) => setTimeout(() => resolve(null), SW_READY_TIMEOUT_MS)),
    ]);
  } catch {
    return null;
  }
}

function toInput(sub: PushSubscription): PushSubscriptionInput {
  const json = sub.toJSON();
  return { endpoint: sub.endpoint, keys: { p256dh: json.keys?.p256dh ?? "", auth: json.keys?.auth ?? "" } };
}

/**
 * Turns the reminder on for this device. Call it straight from the tap handler with
 * the registration and key preloaded: pushManager.subscribe must be the first await,
 * or iOS no longer counts it as the user's gesture and refuses the permission prompt.
 */
export async function enableReminder(reg: ServiceWorkerRegistration, publicKey: string): Promise<"on" | "off" | "denied" | "error"> {
  let sub: PushSubscription;
  try {
    sub = await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: urlBase64ToUint8Array(publicKey) });
  } catch {
    if (Notification.permission === "denied") return "denied";
    if (Notification.permission === "default") return "off"; // prompt dismissed
    // A leftover subscription made with another key (e.g. a rotated VAPID key) makes
    // subscribe fail every time; drop it so the next tap can succeed.
    await reg.pushManager
      .getSubscription()
      .then((old) => old?.unsubscribe())
      .catch(() => undefined);
    return "error";
  }
  try {
    await savePushSubscription(toInput(sub));
    return "on";
  } catch {
    // Keep the device and the API in agreement: no saved row, no browser subscription.
    await sub.unsubscribe().catch(() => false);
    return "error";
  }
}

/** Turns the reminder off for this device. The API row goes best effort (dead rows clean themselves up). */
export async function disableReminder(): Promise<void> {
  const sub = await currentSubscription().catch(() => null);
  if (!sub) return;
  const { endpoint } = sub;
  await sub.unsubscribe().catch(() => false);
  await deletePushSubscription(endpoint).catch(() => undefined);
}

/** Re-saves this device's subscription for the signed-in user, once per app load. Never throws. */
export async function syncReminder(): Promise<void> {
  if (!getSession()) return;
  try {
    const sub = await currentSubscription();
    if (sub) await savePushSubscription(toInput(sub));
  } catch {
    // Best effort: the next app load tries again.
  }
}

const SIGN_OUT_PUSH_TIMEOUT_MS = 3000;

/** Signs out, first removing this device's reminder for the user (best effort, at most 3s). */
export async function signOut(): Promise<void> {
  const cleanup = (async () => {
    const sub = await currentSubscription();
    if (!sub) return;
    // Both start right away: the DELETE carries the still-valid token, and the local
    // unsubscribe never waits on the network.
    await Promise.allSettled([deletePushSubscription(sub.endpoint), sub.unsubscribe()]);
  })();
  try {
    await Promise.race([cleanup, new Promise((resolve) => setTimeout(resolve, SIGN_OUT_PUSH_TIMEOUT_MS))]);
  } catch {
    // Push cleanup is best effort; signing out always completes.
  } finally {
    clearSession();
  }
}

/** Decodes a base64url VAPID key into the bytes pushManager.subscribe expects. */
export function urlBase64ToUint8Array(base64: string): Uint8Array<ArrayBuffer> {
  const padding = "=".repeat((4 - (base64.length % 4)) % 4);
  const raw = window.atob((base64 + padding).replace(/-/g, "+").replace(/_/g, "/"));
  const out = new Uint8Array(new ArrayBuffer(raw.length));
  for (let i = 0; i < raw.length; ++i) out[i] = raw.charCodeAt(i);
  return out;
}
