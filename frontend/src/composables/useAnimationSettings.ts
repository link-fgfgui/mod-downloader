import { gsap } from "gsap";

const defaultAnimationDurationMultiplier = 1;
const minAnimationDurationMultiplier = 0.1;
const maxAnimationDurationMultiplier = 3;

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
const activeTimelines = new WeakMap<Element, ReturnType<typeof gsap.timeline>>();
const pendingGsapDone = new Map<Element, () => void>();
const finishGuards = new WeakMap<Element, number>();

export type AnimationSettings = {
    animationDurationMultiplier?: number;
};

export function normalizeAnimationDurationMultiplier(value: unknown): number {
    const numeric = typeof value === "number" ? value : Number(value);
    if (!Number.isFinite(numeric) || numeric <= 0) return defaultAnimationDurationMultiplier;
    return Math.min(maxAnimationDurationMultiplier, Math.max(minAnimationDurationMultiplier, numeric));
}

export function applyAnimationSettings(settings?: AnimationSettings | null) {
    const multiplier = normalizeAnimationDurationMultiplier(settings?.animationDurationMultiplier);
    const root = document.documentElement;
    root.style.setProperty("--md-transition-fast", `${baseDurations.fast * multiplier}ms`);
    root.style.setProperty("--md-transition-normal", `${baseDurations.normal * multiplier}ms`);
    root.style.setProperty("--md-transition-slow", `${baseDurations.slow * multiplier}ms`);
}

function prefersReducedMotion(): boolean {
    return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

function gsapDuration(baseSeconds: number): number {
    if (prefersReducedMotion()) return 0;
    const normal = document.documentElement.style.getPropertyValue("--md-transition-normal");
    const normalMs = Number.parseFloat(normal);
    const multiplier = Number.isFinite(normalMs) && normalMs > 0 ? normalMs / baseDurations.normal : 1;
    return Math.max(0, baseSeconds * multiplier);
}

function settleOnce(done: () => void): () => void {
    let called = false;
    return () => {
        if (called) return;
        called = true;
        done();
    };
}

function clearFinishGuard(el: Element) {
    const timer = finishGuards.get(el);
    if (timer === undefined) return;
    window.clearTimeout(timer);
    finishGuards.delete(el);
}

function killTimeline(el: Element) {
    activeTimelines.get(el)?.kill();
    activeTimelines.delete(el);
    clearFinishGuard(el);
}

function settlePendingDone(el: Element) {
    const pending = pendingGsapDone.get(el);
    if (!pending) return;
    pendingGsapDone.delete(el);
    pending();
}

function finishGsapAnimation(el: Element) {
    activeTimelines.delete(el);
    clearFinishGuard(el);
    gsap.killTweensOf(el);
    gsap.set(el, { clearProps: "opacity,transform" });
    el.classList.remove(routeLeaveClass);
    settlePendingDone(el);
}

// Every JS-driven transition gets a wall-clock fallback. If rAF is throttled,
// a timeline is killed, or a component unmounts mid-flight, the Vue transition
// `done` callback still fires instead of leaving the router stuck.
function armFinishGuard(el: Element, seconds: number) {
    clearFinishGuard(el);
    const timer = window.setTimeout(() => {
        finishGuards.delete(el);
        killTimeline(el);
        finishGsapAnimation(el);
    }, gsapDuration(seconds) * 1000 + finishGuardPaddingMs);
    finishGuards.set(el, timer);
}

function playTimeline(el: Element, done: () => void, guardSeconds: number, config: gsap.TweenVars) {
    pendingGsapDone.set(el, settleOnce(done));
    armFinishGuard(el, guardSeconds);
    const timeline = gsap.timeline({
        onComplete: () => finishGsapAnimation(el),
    });
    activeTimelines.set(el, timeline);
    timeline.to(el, config);
}

export function beforeGsapRouteEnter(el: Element) {
    killTimeline(el);
    settlePendingDone(el);
    el.classList.remove(routeLeaveClass);
    gsap.killTweensOf(el);
    gsap.set(el, { opacity: 0, y: routeEnterOffset });
}

export function enterGsapRoute(el: Element, done: () => void) {
    playTimeline(el, done, routeEnterDuration, {
        opacity: 1,
        y: 0,
        duration: gsapDuration(routeEnterDuration),
        ease: "expo.out",
    });
}

export function leaveGsapRoute(el: Element, done: () => void) {
    killTimeline(el);
    settlePendingDone(el);
    gsap.killTweensOf(el);
    el.classList.add(routeLeaveClass);
    playTimeline(el, done, routeLeaveDuration, {
        opacity: 0,
        y: routeLeaveOffset,
        duration: gsapDuration(routeLeaveDuration),
        ease: "power1.in",
    });
}

export function afterGsapRouteLeave(el: Element) {
    finishGsapAnimation(el);
}

export function beforeGsapFabEnter(el: Element) {
    killTimeline(el);
    settlePendingDone(el);
    gsap.killTweensOf(el);
    gsap.set(el, {
        opacity: 0,
        scale: 0.55,
        x: 20,
        y: 20,
        rotation: 8,
        transformOrigin: "center center",
    });
}

export function enterGsapFab(el: Element, done: () => void) {
    const settle = settleOnce(done);
    pendingGsapDone.set(el, settle);
    armFinishGuard(el, 0.65);
    const timeline = gsap.timeline({
        onComplete: () => finishGsapAnimation(el),
    });
    activeTimelines.set(el, timeline);

    timeline
        .to(el, {
            opacity: 1,
            scale: 1,
            x: 0,
            y: 0,
            rotation: 0,
            duration: gsapDuration(0.45),
            ease: "back.out(1.5)",
        })
        .to(el, {
            scaleX: 1.06,
            scaleY: 0.94,
            duration: gsapDuration(0.08),
            ease: "power2.out",
        }, `-=${gsapDuration(0.06)}`)
        .to(el, {
            scaleX: 1,
            scaleY: 1,
            duration: gsapDuration(0.14),
            ease: "back.out(2)",
        });
}

export function leaveGsapFab(el: Element, done: () => void) {
    killTimeline(el);
    settlePendingDone(el);
    gsap.killTweensOf(el);
    const settle = settleOnce(done);
    pendingGsapDone.set(el, settle);
    armFinishGuard(el, 0.27);
    const timeline = gsap.timeline({
        onComplete: () => finishGsapAnimation(el),
    });
    activeTimelines.set(el, timeline);

    timeline
        .to(el, {
            scale: 1.05,
            duration: gsapDuration(0.07),
            ease: "power2.out",
        })
        .to(el, {
            opacity: 0,
            scale: 0.55,
            x: 16,
            y: 16,
            rotation: 6,
            duration: gsapDuration(0.2),
            ease: "power2.in",
        });
}

export {
    defaultAnimationDurationMultiplier,
    maxAnimationDurationMultiplier,
    minAnimationDurationMultiplier,
};
