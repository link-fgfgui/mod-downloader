import { gsap } from "gsap";

export const defaultAnimationDurationMultiplier = 1;
// Input range for the settings slider. Core owns the enforced bounds
// (configs.NormalizeAnimationDurationMultiplier); these only shape the widget.
export const minAnimationDurationMultiplier = 0.1;
export const maxAnimationDurationMultiplier = 3;

const baseDurations = {
  press: 70,
  fast: 120,
  normal: 180,
  slow: 240,
  emphasis: 320,
};

const routeEnterDuration = 0.18;
const routeEnterOffset = 8;
const queueExitDuration = 0.14;
const finishGuardPaddingMs = 400;

export type AnimationSettings = {
  animationDurationMultiplier?: number;
};

export function normalizeAnimationDurationMultiplier(value: unknown): number {
  // Settings arrive normalized from core; only guard missing/invalid values.
  const numeric = typeof value === "number" ? value : Number(value);
  return Number.isFinite(numeric) && numeric > 0 ? numeric : defaultAnimationDurationMultiplier;
}

/**
 * Writes both the raw multiplier and the resolved absolute durations. The
 * absolute values keep existing utilities and GSAP working unchanged, while
 * the multiplier lets CSS scale fixed-time effects (combobox close reasons,
 * keyframe feedback, press feedback) without re-writing every rule.
 */
export function applyAnimationSettings(settings?: AnimationSettings | null) {
  if (typeof document === "undefined") return;
  const multiplier = normalizeAnimationDurationMultiplier(settings?.animationDurationMultiplier);
  const root = document.documentElement;
  root.style.setProperty("--md-motion-multiplier", String(multiplier));
  for (const [name, base] of Object.entries(baseDurations)) {
    root.style.setProperty(`--md-transition-${name}`, `${base * multiplier}ms`);
  }
}

export function prefersReducedMotion(): boolean {
  return typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

function animationMultiplier(): number {
  if (typeof document === "undefined") return 1;
  const root = document.documentElement;
  // The multiplier is the source of truth. Never parse the absolute duration
  // variables here: they may be calc() expressions once CSS takes over.
  const explicit = Number.parseFloat(root.style.getPropertyValue("--md-motion-multiplier"));
  if (Number.isFinite(explicit) && explicit > 0) return explicit;
  if (typeof window !== "undefined") {
    const computed = Number.parseFloat(
      window.getComputedStyle(root).getPropertyValue("--md-motion-multiplier"),
    );
    if (Number.isFinite(computed) && computed > 0) return computed;
  }
  const normal = Number.parseFloat(root.style.getPropertyValue("--md-transition-normal"));
  return Number.isFinite(normal) && normal > 0 ? normal / baseDurations.normal : 1;
}

export function scaledDuration(baseSeconds: number): number {
  if (prefersReducedMotion()) return 0;
  return Math.max(0, baseSeconds * animationMultiplier());
}

/** Same as scaledDuration but in milliseconds, for timers that mirror a CSS transition. */
export function scaledDurationMs(baseSeconds: number): number {
  return Math.round(scaledDuration(baseSeconds) * 1000);
}

export type RouteDirection = 1 | -1 | 0;

/**
 * Route stage entrance. Direction-aware: navigating down the sidebar list
 * enters from below, navigating up enters from above. Only the page stage is
 * animated; the fixed sidebar, queue FAB and toaster stay put.
 */
export function animateRouteEnter(element: Element | null, direction: RouteDirection = 1) {
  if (!element) return () => {};
  const duration = scaledDuration(routeEnterDuration);
  if (duration === 0) return () => {};
  gsap.killTweensOf(element);
  const offset = direction < 0 ? -routeEnterOffset : routeEnterOffset;
  const tween = gsap.fromTo(
    element,
    { opacity: 0, y: offset },
    { opacity: 1, y: 0, duration, ease: "expo.out", clearProps: "opacity,transform" },
  );
  return () => {
    tween.kill();
  };
}

export function animateQueuePanel(element: Element | null) {
  if (!element) return () => {};
  const duration = scaledDuration(0.2);
  if (duration === 0) return () => {};
  gsap.killTweensOf(element);
  const tween = gsap.fromTo(
    element,
    { opacity: 0, scale: 0.96, y: 8, transformOrigin: "bottom right" },
    { opacity: 1, scale: 1, y: 0, duration, ease: "power2.out", clearProps: "opacity,transform" },
  );
  return () => tween.kill();
}

export function animateQueueExit(element: Element | null): Promise<void> {
  if (!element) return Promise.resolve();
  const duration = scaledDuration(queueExitDuration);
  if (duration === 0) return Promise.resolve();
  return new Promise((resolve) => {
    gsap.to(element, {
      opacity: 0,
      scale: 0.96,
      y: 8,
      duration,
      ease: "power1.in",
      onComplete: () => {
        gsap.set(element, { clearProps: "opacity,transform" });
        resolve();
      },
      onInterrupt: resolve,
    });
  });
}

/** FAB / floating control exit: scale down and fade, then unmount. */
export function animateScaleExit(
  element: Element | null,
  options?: { scale?: number; y?: number; baseDuration?: number },
): Promise<void> {
  if (!element) return Promise.resolve();
  const duration = scaledDuration(options?.baseDuration ?? 0.16);
  if (duration === 0) return Promise.resolve();
  return new Promise((resolve) => {
    gsap.to(element, {
      opacity: 0,
      scale: options?.scale ?? 0.75,
      y: options?.y ?? 0,
      duration,
      ease: "power1.in",
      onComplete: () => {
        gsap.set(element, { clearProps: "opacity,transform" });
        resolve();
      },
      onInterrupt: resolve,
    });
  });
}

export { routeEnterOffset, finishGuardPaddingMs };
