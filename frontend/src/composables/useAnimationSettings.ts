import { gsap } from "gsap";
import { readonly, ref } from "vue";

const animationModeOff = "off";
const animationModeVuetify = "vuetify";
const animationModeGsap = "gsap";
const defaultAnimationMode = animationModeVuetify;
const defaultAnimationDurationMultiplier = 1;
const minAnimationDurationMultiplier = 0.25;
const maxAnimationDurationMultiplier = 3;

const baseDurations = {
    fast: 120,
    normal: 220,
    slow: 360,
    stagger: 40,
};

const pageContentSelector = [
    ".v-card",
    ".v-data-table",
    ".v-data-table__tr",
    ".v-alert",
    "h1",
    ".empty-state",
    ".search-controls",
    ".manage-header",
    ".md-stagger > *",
].join(", ");

type AnimationMode = typeof animationModeOff | typeof animationModeVuetify | typeof animationModeGsap;

const activeAnimationMode = ref<AnimationMode>(defaultAnimationMode);
const activeGsapElements = new Set<Element>();
const pendingGsapDone = new Map<Element, () => void>();

export type AnimationSettings = {
    animationMode?: string;
    animationEnabled?: boolean;
    animationDurationMultiplier?: number;
};

export function normalizeAnimationMode(value: unknown, legacyEnabled?: boolean): AnimationMode {
    const mode = typeof value === "string" ? value.trim().toLowerCase() : "";
    if (mode === animationModeOff || mode === "0" || mode === "false" || mode === "disabled") {
        return animationModeOff;
    }
    if (mode === animationModeVuetify || mode === "1" || mode === "true" || mode === "enabled" || mode === "on") {
        return animationModeVuetify;
    }
    if (mode === animationModeGsap || mode === "2") {
        return animationModeGsap;
    }
    if (typeof legacyEnabled === "boolean") {
        return legacyEnabled ? animationModeVuetify : animationModeOff;
    }
    return defaultAnimationMode;
}

export function normalizeAnimationDurationMultiplier(value: unknown): number {
    const numeric = typeof value === "number" ? value : Number(value);
    if (!Number.isFinite(numeric) || numeric <= 0) return defaultAnimationDurationMultiplier;
    return Math.min(maxAnimationDurationMultiplier, Math.max(minAnimationDurationMultiplier, numeric));
}

export function animationModeEnabled(mode: unknown): boolean {
    return normalizeAnimationMode(mode) !== animationModeOff;
}

export function useActiveAnimationMode() {
    return readonly(activeAnimationMode);
}

export function applyAnimationSettings(settings?: AnimationSettings | null) {
    const root = document.documentElement;
    const mode = normalizeAnimationMode(settings?.animationMode, settings?.animationEnabled);
    const multiplier = normalizeAnimationDurationMultiplier(settings?.animationDurationMultiplier);

    if (activeAnimationMode.value !== mode) cleanupActiveGsapAnimations();
    activeAnimationMode.value = mode;
    root.dataset.mdAnimations = mode;
    root.style.setProperty("--md-transition-fast", `${baseDurations.fast * multiplier}ms`);
    root.style.setProperty("--md-transition-normal", `${baseDurations.normal * multiplier}ms`);
    root.style.setProperty("--md-transition-slow", `${baseDurations.slow * multiplier}ms`);
    root.style.setProperty("--md-stagger-delay", `${baseDurations.stagger * multiplier}ms`);
}

function prefersReducedMotion(): boolean {
    return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

function gsapDuration(baseSeconds: number): number {
    if (prefersReducedMotion()) return 0;
    const root = document.documentElement;
    const normal = root.style.getPropertyValue("--md-transition-normal");
    const normalMs = Number.parseFloat(normal);
    const multiplier = Number.isFinite(normalMs) && normalMs > 0 ? normalMs / baseDurations.normal : 1;
    return Math.max(0, baseSeconds * multiplier);
}

function clearGsapProps(el: Element | Element[]) {
    gsap.set(el, { clearProps: "opacity,transform,scale,rotation" });
}

function trackGsapElements(elements: Element[]) {
    elements.forEach((element) => activeGsapElements.add(element));
}

function finishGsapAnimation(root: Element, elements: Element[], done?: () => void) {
    clearGsapProps(elements);
    elements.forEach((element) => activeGsapElements.delete(element));
    pendingGsapDone.delete(root);
    done?.();
}

export function cleanupActiveGsapAnimations() {
    const elements = [...activeGsapElements];
    if (elements.length > 0) {
        gsap.killTweensOf(elements);
        clearGsapProps(elements);
    }
    activeGsapElements.clear();
    const callbacks = [...new Set(pendingGsapDone.values())];
    pendingGsapDone.clear();
    callbacks.forEach((done) => done());
}

function getPageContentTargets(root: Element, selector = pageContentSelector): Element[] {
    return Array.from(root.querySelectorAll(selector));
}

function getVisiblePageContentTargets(root: Element, selector = pageContentSelector): Element[] {
    return getPageContentTargets(root, selector).filter((el) => {
        const rect = el.getBoundingClientRect();
        return rect.width > 0 && rect.height > 0;
    });
}

function routeAnimationTargets(el: Element): Element[] {
    return [el, ...getVisiblePageContentTargets(el)];
}

export function beforeGsapRouteEnter(el: Element) {
    const targets = [el, ...getVisiblePageContentTargets(el)];
    gsap.killTweensOf(targets);
    trackGsapElements(targets);
    gsap.set(el, { opacity: 0 });
    if (targets.length > 1) {
        gsap.set(targets.slice(1), { opacity: 0, y: 24, scale: 0.985 });
    }
}

export function enterGsapRoute(el: Element, done: () => void) {
    const targets = getVisiblePageContentTargets(el);
    const allTargets = [el, ...targets];
    trackGsapElements(allTargets);
    pendingGsapDone.set(el, done);
    const timeline = gsap.timeline({
        onComplete: () => finishGsapAnimation(el, allTargets, done),
    });

    timeline.to(el, {
        opacity: 1,
        duration: gsapDuration(0.32),
        ease: "expo.out",
    }, 0);

    if (targets.length > 0) {
        timeline.to(targets, {
            opacity: 1,
            y: 0,
            scale: 1,
            duration: gsapDuration(0.42),
            ease: "power3.out",
            stagger: {
                each: gsapDuration(0.05),
                from: "start",
            },
        }, 0.06);
    }
}

export function leaveGsapRoute(el: Element, done: () => void) {
    const targets = routeAnimationTargets(el);
    gsap.killTweensOf(targets);
    trackGsapElements(targets);
    pendingGsapDone.set(el, done);
    gsap.to(el, {
        opacity: 0,
        duration: gsapDuration(0.26),
        ease: "power2.in",
        onComplete: () => finishGsapAnimation(el, targets, done),
    });
}

export function afterGsapRouteLeave(el: Element) {
    finishGsapAnimation(el, routeAnimationTargets(el));
}

export function beforeGsapFabEnter(el: Element) {
    gsap.killTweensOf(el);
    trackGsapElements([el]);
    gsap.set(el, {
        opacity: 0,
        scale: 0.35,
        x: 30,
        y: 30,
        rotation: 18,
        transformOrigin: "center center",
    });
}

export function enterGsapFab(el: Element, done: () => void) {
    trackGsapElements([el]);
    pendingGsapDone.set(el, done);
    const timeline = gsap.timeline({
        onComplete: () => finishGsapAnimation(el, [el], done),
    });

    timeline
        .to(el, {
            opacity: 1,
            scale: 1,
            x: 0,
            y: 0,
            rotation: 0,
            duration: gsapDuration(0.55),
            ease: "elastic.out(1, 0.65)",
        })
        .to(el, {
            scaleX: 1.08,
            scaleY: 0.94,
            duration: gsapDuration(0.09),
            ease: "power2.out",
        }, `-=${gsapDuration(0.08)}`)
        .to(el, {
            scaleX: 1,
            scaleY: 1,
            duration: gsapDuration(0.16),
            ease: "back.out(2)",
        });
}

export function leaveGsapFab(el: Element, done: () => void) {
    gsap.killTweensOf(el);
    trackGsapElements([el]);
    pendingGsapDone.set(el, done);
    const timeline = gsap.timeline({
        onComplete: () => finishGsapAnimation(el, [el], done),
    });

    timeline
        .to(el, {
            scale: 1.08,
            duration: gsapDuration(0.08),
            ease: "power2.out",
        })
        .to(el, {
            opacity: 0,
            scale: 0.35,
            x: 26,
            y: 26,
            rotation: 16,
            duration: gsapDuration(0.24),
            ease: "power3.in",
        });
}

export {
    animationModeGsap,
    animationModeOff,
    animationModeVuetify,
    defaultAnimationMode,
    defaultAnimationDurationMultiplier,
    minAnimationDurationMultiplier,
    maxAnimationDurationMultiplier,
};
