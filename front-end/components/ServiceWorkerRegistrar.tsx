"use client";

import { useEffect } from "react";
import { registerServiceWorker, syncReminder } from "@/lib/push";

/** Registers /sw.js once per app load and re-saves this device's reminder subscription, if any. */
export function ServiceWorkerRegistrar() {
  useEffect(() => {
    void registerServiceWorker().then(() => syncReminder());
  }, []);
  return null;
}
