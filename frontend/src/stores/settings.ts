import { defineStore } from "pinia";
import {
    GetSettings,
    SaveTheme,
    SaveAnimationSettings,
    SaveApiKeys,
    ChooseMinecraftDir,
    ValidateMinecraftDir,
} from "../../wailsjs/go/main/App";
import type { main } from "../../wailsjs/go/models";
import {
    defaultAnimationsEnabled,
    defaultAnimationDurationMultiplier,
    normalizeAnimationDurationMultiplier,
} from "../composables/useAnimationSettings";

const themeDark = "dark";
const themeLight = "light";
const themeSystem = "system";
const apiKeyKeepSentinel = "<keep>";

export const useSettingsStore = defineStore("settings", {
    state: () => ({
        view: null as main.SettingsView | null,
        isLoading: false,
        isSavingTheme: false,
        isSavingAnimations: false,
        isSavingKeys: false,
        isChoosingDir: false,
        isValidatingDir: false,
        dirValid: null as boolean | null,
        draftTheme: "",
        draftAnimationEnabled: defaultAnimationsEnabled,
        draftAnimationDurationMultiplier: defaultAnimationDurationMultiplier,
        draftCurseforgeKey: "",
        draftModrinthKey: "",
        clearCurseforgeKey: false,
        clearModrinthKey: false,
    }),
    getters: {
        hasCurseforgeKey: (s) => Boolean(s.view?.hasCurseforgeKey),
        hasModrinthKey: (s) => Boolean(s.view?.hasModrinthKey),
    },
    actions: {
        async load() {
            this.isLoading = true;
            try {
                this.view = await GetSettings();
                this.draftTheme = this.view?.theme || themeDark;
                this.draftAnimationEnabled = this.view?.animationEnabled ?? defaultAnimationsEnabled;
                this.draftAnimationDurationMultiplier = normalizeAnimationDurationMultiplier(
                    this.view?.animationDurationMultiplier ?? defaultAnimationDurationMultiplier
                );
                this.draftCurseforgeKey = "";
                this.draftModrinthKey = "";
                this.clearCurseforgeKey = false;
                this.clearModrinthKey = false;
                this.dirValid = null;
            } finally {
                this.isLoading = false;
            }
        },
        async saveTheme() {
            this.isSavingTheme = true;
            try {
                const next = await SaveTheme(this.draftTheme);
                if (this.view) this.view.theme = next;
                this.draftTheme = next;
                return next;
            } finally {
                this.isSavingTheme = false;
            }
        },
        async saveAnimationSettings() {
            this.isSavingAnimations = true;
            try {
                const req = {
                    animationEnabled: this.draftAnimationEnabled,
                    animationDurationMultiplier: normalizeAnimationDurationMultiplier(
                        this.draftAnimationDurationMultiplier
                    ),
                };
                this.view = await SaveAnimationSettings(req);
                this.draftAnimationEnabled = this.view.animationEnabled;
                this.draftAnimationDurationMultiplier = normalizeAnimationDurationMultiplier(
                    this.view.animationDurationMultiplier
                );
                return {
                    animationEnabled: this.draftAnimationEnabled,
                    animationDurationMultiplier: this.draftAnimationDurationMultiplier,
                };
            } finally {
                this.isSavingAnimations = false;
            }
        },
        async saveApiKeys() {
            this.isSavingKeys = true;
            try {
                const req = {
                    curseforgeApiKey: this.clearCurseforgeKey ? "" : (this.draftCurseforgeKey || apiKeyKeepSentinel),
                    modrinthApiKey: this.clearModrinthKey ? "" : (this.draftModrinthKey || apiKeyKeepSentinel),
                };
                this.view = await SaveApiKeys(req);
                this.draftCurseforgeKey = "";
                this.draftModrinthKey = "";
                this.clearCurseforgeKey = false;
                this.clearModrinthKey = false;
            } finally {
                this.isSavingKeys = false;
            }
        },
        async chooseMinecraftDir() {
            this.isChoosingDir = true;
            try {
                const result = await ChooseMinecraftDir();
                if (result) {
                    await this.load();
                }
                return result;
            } finally {
                this.isChoosingDir = false;
            }
        },
        async validateDir() {
            this.isValidatingDir = true;
            try {
                this.dirValid = await ValidateMinecraftDir();
            } finally {
                this.isValidatingDir = false;
            }
        },
    },
});
