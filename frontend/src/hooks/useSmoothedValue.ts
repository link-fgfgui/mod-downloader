import { useEffect, useRef, useState } from "react";

import { prefersReducedMotion, scaledDuration } from "@/lib/animations";

/**
 * Interpolates a numeric value towards its target with requestAnimationFrame.
 * High-frequency backend events (download progress) would otherwise make
 * numbers and bars jump on every update. The value converges in about
 * `baseDurationSeconds` and restarts from the currently displayed value when a
 * new target arrives mid-flight.
 */
export function useSmoothedValue(target: number, baseDurationSeconds = 0.2): number {
  const [value, setValue] = useState(target);
  const valueRef = useRef(target);
  const frameRef = useRef<number | null>(null);

  useEffect(() => {
    const safeTarget = Number.isFinite(target) ? target : 0;
    if (prefersReducedMotion()) {
      valueRef.current = safeTarget;
      setValue(safeTarget);
      return;
    }
    const from = valueRef.current;
    if (Math.abs(safeTarget - from) < 0.001) {
      valueRef.current = safeTarget;
      setValue(safeTarget);
      return;
    }
    const duration = scaledDuration(baseDurationSeconds) * 1000;
    if (duration <= 0) {
      valueRef.current = safeTarget;
      setValue(safeTarget);
      return;
    }
    const start = performance.now();
    const step = (now: number) => {
      const progress = Math.min(1, (now - start) / duration);
      const eased = 1 - Math.pow(1 - progress, 3);
      const next = from + (safeTarget - from) * eased;
      valueRef.current = next;
      setValue(next);
      if (progress < 1) {
        frameRef.current = window.requestAnimationFrame(step);
      } else {
        valueRef.current = safeTarget;
        setValue(safeTarget);
      }
    };
    frameRef.current = window.requestAnimationFrame(step);
    return () => {
      if (frameRef.current !== null) {
        window.cancelAnimationFrame(frameRef.current);
        frameRef.current = null;
      }
    };
  }, [baseDurationSeconds, target]);

  return value;
}

/**
 * Throttles a rapidly changing display value (download speed) so the text does
 * not flicker between every backend tick. The first value and meaningful
 * changes are shown immediately.
 */
export function useThrottledValue<T>(value: T, intervalMs = 350): T {
  const [display, setDisplay] = useState(value);
  const pending = useRef(value);
  const lastUpdate = useRef(0);
  const timer = useRef<number | null>(null);

  useEffect(() => {
    pending.current = value;
    if (timer.current !== null) return;
    const elapsed = performance.now() - lastUpdate.current;
    const flush = () => {
      timer.current = null;
      lastUpdate.current = performance.now();
      setDisplay(pending.current);
    };
    if (elapsed >= intervalMs) {
      flush();
      return;
    }
    timer.current = window.setTimeout(flush, intervalMs - elapsed);
  }, [intervalMs, value]);

  useEffect(
    () => () => {
      if (timer.current !== null) window.clearTimeout(timer.current);
    },
    [],
  );

  return display;
}
