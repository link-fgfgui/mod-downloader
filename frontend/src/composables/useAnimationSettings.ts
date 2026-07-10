import { gsap } from "gsap";
import { onActivated, onDeactivated, onMounted, onUnmounted, readonly, ref, type Ref } from "vue";

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
].join(", ");

type AnimationMode = typeof animationModeOff | typeof animationModeVuetify | typeof animationModeGsap;
type AnimationDirection = "up" | "down";

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
    const targets = getVisiblePageContentTargets(el);
    gsap.killTweensOf([el, ...targets]);
    gsap.set(el, { opacity: 0 });
    if (targets.length > 0) {
        gsap.set(targets, { opacity: 0, y: 24, scale: 0.985 });
    }
}

export function enterGsapRoute(el: Element, done: () => void) {
    const targets = getVisiblePageContentTargets(el);
    const timeline = gsap.timeline({
        onComplete: () => {
            clearGsapProps([el, ...targets]);
            done();
        },
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
    gsap.to(el, {
        opacity: 0,
        duration: gsapDuration(0.26),
        ease: "power2.in",
        onComplete: done,
    });
}

export function afterGsapRouteLeave(el: Element) {
    clearGsapProps(routeAnimationTargets(el));
}

export function beforeGsapFabEnter(el: Element) {
    gsap.killTweensOf(el);
    gsap.set(el, { opacity: 0, scale: 0, rotation: -90 });
}

export function enterGsapFab(el: Element, done: () => void) {
    gsap.to(el, {
        opacity: 1,
        scale: 1,
        rotation: 0,
        duration: gsapDuration(0.48),
        ease: "back.out(1.8)",
        onComplete: () => {
            clearGsapProps(el);
            done();
        },
    });
}

export function leaveGsapFab(el: Element, done: () => void) {
    gsap.to(el, {
        opacity: 0,
        scale: 0.5,
        rotation: 45,
        duration: gsapDuration(0.22),
        ease: "power2.in",
        onComplete: done,
    });
}

export interface GsapPageContentOptions {
    from?: AnimationDirection;
    selector?: string;
}

export function animateGsapPageContent(
    container: Element | string | null | undefined,
    options?: GsapPageContentOptions,
): gsap.core.Animation | null {
    if (!container) return null;
    const root = typeof container === "string" ? document.querySelector(container) : container;
    if (!root) return null;
    if (prefersReducedMotion()) return gsap.timeline();

    const targets = getVisiblePageContentTargets(root, options?.selector ?? pageContentSelector);

    if (targets.length === 0) return gsap.timeline();

    gsap.killTweensOf(targets);

    const fromY = options?.from === "down" ? -20 : 24;
    return gsap.fromTo(
        targets,
        { opacity: 0, y: fromY, scale: 0.985 },
        {
            opacity: 1,
            y: 0,
            scale: 1,
            duration: gsapDuration(0.42),
            ease: "power3.out",
            stagger: {
                each: gsapDuration(0.05),
                from: "start",
            },
            clearProps: "opacity,transform,scale",
        },
    );
}

export function useGsapPageAnimations(
    containerRef: Ref<Element | null>,
    options?: GsapPageContentOptions,
) {
    let ctx: gsap.Context | null = null;
    const mode = useActiveAnimationMode();

    const animate = () => {
        if (mode.value !== animationModeGsap) return;
        if (!containerRef.value) return;
        ctx?.revert();
        ctx = gsap.context(() => {
            animateGsapPageContent(containerRef.value, options);
        }, containerRef.value);
    };

    const cleanup = () => {
        ctx?.revert();
        ctx = null;
    };

    onMounted(animate);
    onActivated(animate);
    onDeactivated(cleanup);
    onUnmounted(cleanup);
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
