import { gsap } from "gsap";

export const defaultAnimationDurationMultiplier = 1;
export const minAnimationDurationMultiplier = 0.1;
export const maxAnimationDurationMultiplier = 3;

const baseDurations = {
  fast: 120,
  normal: 180,
  slow: 240,
};

const routeEnterDuration = 0.18;
const routeLeaveDuration = 0.1;
const routeEnterOffset = 8;
const routeLeaveOffset = -4;
const finishGuardPaddingMs = 400;

const routeLeaveClass = "route-leave-active";

export type AnimationSettings = {
  animationDurationMultiplier?: number;
};

export function normalizeAnimationDurationMultiplier(value: unknown): number {
  const numeric = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) return defaultAnimationDurationMultiplier;
  return Math.min(maxAnimationDurationMultiplier, Math.max(minAnimationDurationMultiplier, numeric));
}

export function applyAnimationSettings(settings?: AnimationSettings | null) {
  if (typeof document === "undefined") return;
  const multiplier = normalizeAnimationDurationMultiplier(settings?.animationDurationMultiplier);
  const root = document.documentElement;
  root.style.setProperty("--md-transition-fast", `${baseDurations.fast * multiplier}ms`);
  root.style.setProperty("--md-transition-normal", `${baseDurations.normal * multiplier}ms`);
  root.style.setProperty("--md-transition-slow", `${baseDurations.slow * multiplier}ms`);
}

export function prefersReducedMotion(): boolean {
  return typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

function animationMultiplier(): number {
  if (typeof document === "undefined") return 1;
  const normal = document.documentElement.style.getPropertyValue("--md-transition-normal");
  const normalMs = Number.parseFloat(normal);
  return Number.isFinite(normalMs) && normalMs > 0 ? normalMs / baseDurations.normal : 1;
}

export function scaledDuration(baseSeconds: number): number {
  if (prefersReducedMotion()) return 0;
  return Math.max(0, baseSeconds * animationMultiplier());
}

// Route stage entrance is intentionally small and self-contained. It is the
// single GSAP entry point the pages use; page-specific animation work can build
// on the same timing helpers in a later stage.
export function animateRouteEnter(element: Element | null) {
  if (!element) return () => {};
  const duration = scaledDuration(routeEnterDuration);
  if (duration === 0) return () => {};
  gsap.killTweensOf(element);
  element.classList.remove(routeLeaveClass);
  const tween = gsap.fromTo(
    element,
    { opacity: 0, y: routeEnterOffset },
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
  const duration = scaledDuration(routeLeaveDuration);
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

export { routeLeaveClass, routeEnterOffset, routeLeaveOffset, finishGuardPaddingMs };
