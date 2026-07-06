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

type AnimationMode = typeof animationModeOff | typeof animationModeVuetify | typeof animationModeGsap;

const activeAnimationMode = ref<AnimationMode>(defaultAnimationMode);

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

    activeAnimationMode.value = mode;
    root.dataset.mdAnimations = mode;
    root.dataset.mdAnimationMode = mode;
    root.style.setProperty("--md-transition-fast", `${baseDurations.fast * multiplier}ms`);
    root.style.setProperty("--md-transition-normal", `${baseDurations.normal * multiplier}ms`);
    root.style.setProperty("--md-transition-slow", `${baseDurations.slow * multiplier}ms`);
    root.style.setProperty("--md-stagger-delay", `${baseDurations.stagger * multiplier}ms`);
}

function gsapDuration(baseSeconds: number): number {
    const root = document.documentElement;
    const normal = root.style.getPropertyValue("--md-transition-normal");
    const normalMs = Number.parseFloat(normal);
    const multiplier = Number.isFinite(normalMs) && normalMs > 0 ? normalMs / baseDurations.normal : 1;
    return baseSeconds * multiplier;
}

function clearGsapProps(el: Element) {
    gsap.set(el, { clearProps: "opacity,transform" });
}

export function beforeGsapRouteEnter(el: Element) {
    gsap.killTweensOf(el);
    gsap.set(el, { opacity: 0, y: 14 });
}

export function enterGsapRoute(el: Element, done: () => void) {
    gsap.to(el, {
        opacity: 1,
        y: 0,
        duration: gsapDuration(0.22),
        ease: "power3.out",
        onComplete: () => {
            clearGsapProps(el);
            done();
        },
    });
}

export function leaveGsapRoute(el: Element, done: () => void) {
    gsap.to(el, {
        opacity: 0,
        y: -10,
        duration: gsapDuration(0.16),
        ease: "power2.in",
        onComplete: done,
    });
}

export function beforeGsapFabEnter(el: Element) {
    gsap.killTweensOf(el);
    gsap.set(el, { opacity: 0, scale: 0.82 });
}

export function enterGsapFab(el: Element, done: () => void) {
    gsap.to(el, {
        opacity: 1,
        scale: 1,
        duration: gsapDuration(0.28),
        ease: "back.out(1.6)",
        onComplete: () => {
            clearGsapProps(el);
            done();
        },
    });
}

export function leaveGsapFab(el: Element, done: () => void) {
    gsap.to(el, {
        opacity: 0,
        scale: 0.78,
        duration: gsapDuration(0.14),
        ease: "power2.in",
        onComplete: done,
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
