"use client";

import { useEffect } from "react";

// iOS Safari ignores user-scalable=no; cancelling its proprietary gesture events blocks pinch zoom.
export function NoPinchZoom() {
  useEffect(() => {
    const block = (e: Event) => e.preventDefault();
    document.addEventListener("gesturestart", block);
    document.addEventListener("gesturechange", block);
    return () => {
      document.removeEventListener("gesturestart", block);
      document.removeEventListener("gesturechange", block);
    };
  }, []);
  return null;
}
