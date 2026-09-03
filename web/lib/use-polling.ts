"use client";

import { useEffect, useRef } from "react";

const defaultIntervalMs = 3000;

/**
 * enabled iken hemen bir kez, sonra intervalMs aralıkla fn çağırır.
 * fn her render'da yenilense de zamanlayıcı sıfırlanmaz.
 */
export function usePolling(
  fn: () => void | Promise<void>,
  enabled: boolean,
  intervalMs = defaultIntervalMs,
): void {
  const fnRef = useRef(fn);
  fnRef.current = fn;

  useEffect(() => {
    if (!enabled) {
      return;
    }
    console.log("usePolling tetiklendi", { intervalMs });
    void fnRef.current();
    const id = window.setInterval(() => {
      console.log("usePolling tik");
      void fnRef.current();
    }, intervalMs);
    return () => window.clearInterval(id);
  }, [enabled, intervalMs]);
}
