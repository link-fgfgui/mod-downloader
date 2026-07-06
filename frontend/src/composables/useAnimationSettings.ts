const defaultAnimationsEnabled = true;
const defaultAnimationDurationMultiplier = 1;
const minAnimationDurationMultiplier = 0.25;
const maxAnimationDurationMultiplier = 3;

const baseDurations = {
    fast: 120,
    normal: 220,
    slow: 360,
    stagger: 40,
};

export type AnimationSettings = {
    animationEnabled?: boolean;
    animationDurationMultiplier?: number;
};

export function normalizeAnimationDurationMultiplier(value: unknown): number {
    const numeric = typeof value === "number" ? value : Number(value);
    if (!Number.isFinite(numeric) || numeric <= 0) return defaultAnimationDurationMultiplier;
    return Math.min(maxAnimationDurationMultiplier, Math.max(minAnimationDurationMultiplier, numeric));
}

export function applyAnimationSettings(settings?: AnimationSettings | null) {
    const root = document.documentElement;
    const enabled = settings?.animationEnabled ?? defaultAnimationsEnabled;
    const multiplier = normalizeAnimationDurationMultiplier(settings?.animationDurationMultiplier);

    root.dataset.mdAnimations = enabled ? "on" : "off";
    root.style.setProperty("--md-transition-fast", `${baseDurations.fast * multiplier}ms`);
    root.style.setProperty("--md-transition-normal", `${baseDurations.normal * multiplier}ms`);
    root.style.setProperty("--md-transition-slow", `${baseDurations.slow * multiplier}ms`);
    root.style.setProperty("--md-stagger-delay", `${baseDurations.stagger * multiplier}ms`);
}

export {
    defaultAnimationsEnabled,
    defaultAnimationDurationMultiplier,
    minAnimationDurationMultiplier,
    maxAnimationDurationMultiplier,
};
