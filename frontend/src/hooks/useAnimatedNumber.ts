import { useEffect, useRef, useState } from "react";
import { gsap } from "gsap";

import { prefersReducedMotion, scaledDuration } from "@/lib/animations";

/**
 * Counts a displayed number towards `target` with a short GSAP power2.out
 * tween. Values are integers only, so locale formatting never sees a
 * fractional intermediate value. Reduced motion shows the target immediately.
 */
export function useAnimatedNumber(target: number, baseDurationSeconds = 0.35): number {
  const [value, setValue] = useState(target);
  const valueRef = useRef(target);

  useEffect(() => {
    const safeTarget = Number.isFinite(target) ? Math.round(target) : 0;
    if (prefersReducedMotion()) {
      valueRef.current = safeTarget;
      setValue(safeTarget);
      return;
    }
    const from = valueRef.current;
    if (from === safeTarget) return;
    const proxy = { value: from };
    const tween = gsap.to(proxy, {
      value: safeTarget,
      duration: scaledDuration(baseDurationSeconds),
      ease: "power2.out",
      onUpdate: () => {
        const next = Math.round(proxy.value);
        valueRef.current = next;
        setValue(next);
      },
      onComplete: () => {
        valueRef.current = safeTarget;
        setValue(safeTarget);
      },
    });
    return () => {
      tween.kill();
    };
  }, [baseDurationSeconds, target]);

  return value;
}
