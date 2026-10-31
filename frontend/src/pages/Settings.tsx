import { useEffect, useState } from "react";
import { AlertCircle, Check, FolderOpen, Loader2, RotateCcw, Save, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Combobox } from "@/components/ui/combobox";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Slider } from "@/components/ui/slider";
import { Switch } from "@/components/ui/switch";
import { errorMessage } from "@/lib/format";
import { maxAnimationDurationMultiplier, minAnimationDurationMultiplier } from "@/lib/animations";
import i18n, { languageEnglish, languageSystem, languageChinese } from "@/i18n";
import { themeSystem } from "@/lib/theme";
import { useMinecraftStore } from "@/stores/minecraft";
import { useSettingsStore } from "@/stores/settings";

export function SettingsPage() {
  const { t } = useTranslation();
  const settings = useSettingsStore((state) => state.settings);
  const loading = useSettingsStore((state) => state.loading);
  const isSavingTheme = useSettingsStore((state) => state.isSavingTheme);
  const isSavingLanguage = useSettingsStore((state) => state.isSavingLanguage);
  const isSavingKeys = useSettingsStore((state) => state.isSavingKeys);
  const isChoosingCacheDir = useSettingsStore((state) => state.isChoosingCacheDir);
  const isSavingCacheDir = useSettingsStore((state) => state.isSavingCacheDir);
  const isSavingJijDetection = useSettingsStore((state) => state.isSavingJijDetection);
  const isSavingHardwareAcceleration = useSettingsStore((state) => state.isSavingHardwareAcceleration);
  const autoSaveState = useSettingsStore((state) => state.autoSaveState);
  const autoSaveError = useSettingsStore((state) => state.autoSaveError);
  const persistError = settings?.saveError || "";
  const displaySaveState = persistError ? "error" : autoSaveState;
  const draft = useSettingsStore((state) => state);
  const patch = useSettingsStore((state) => state.patch);
  const load = useSettingsStore((state) => state.load);
  const saveTheme = useSettingsStore((state) => state.saveTheme);
  const saveLanguage = useSettingsStore((state) => state.saveLanguage);
  const saveJij = useSettingsStore((state) => state.saveJijDetectionSettings);
  const saveHardware = useSettingsStore((state) => state.saveHardwareAccelerationSettings);
  const saveKeys = useSettingsStore((state) => state.saveApiKeys);
  const scheduleAutoSave = useSettingsStore((state) => state.scheduleAutoSave);
  const chooseCacheDir = useSettingsStore((state) => state.chooseCacheDir);
  const resetCacheDir = useSettingsStore((state) => state.resetCacheDir);
  const refreshSelectedMods = useMinecraftStore((state) => state.refreshSelected);

  useEffect(() => {
    void load();
  }, [load]);

  const fail = (error: unknown) => toast.error(t("settings.saveFailed", { reason: errorMessage(error) }));

  const handleLanguage = async (value: string) => {
    patch({ draftLanguage: value });
    try {
      await saveLanguage();
      toast.success(i18n.t("settings.language.saved"));
    } catch (error) {
      fail(error);
    }
  };

  const handleTheme = async (value: string) => {
    patch({ draftTheme: value });
    try {
      await saveTheme();
      toast.success(t("settings.theme.saved"));
    } catch (error) {
      fail(error);
    }
  };

  const handleJij = async (value: boolean) => {
    patch({ draftEnableJijDetection: value });
    try {
      await saveJij();
      await refreshSelectedMods();
      toast.success(t("settings.advanced.saved"));
    } catch (error) {
      fail(error);
    }
  };

  const handleHardware = async (value: boolean) => {
    patch({ draftHardwareAcceleration: value });
    try {
      await saveHardware();
      toast.success(t("settings.hardwareAcceleration.saved"));
    } catch (error) {
      fail(error);
    }
  };

  const handleChooseCache = async () => {
    try {
      const changed = await chooseCacheDir();
      if (changed) toast.success(t("settings.cacheDir.saved"));
    } catch (error) {
      fail(error);
    }
  };

  const handleResetCache = async () => {
    try {
      await resetCacheDir();
      toast.success(t("settings.cacheDir.saved"));
    } catch (error) {
      fail(error);
    }
  };

  const handleSaveKeys = async () => {
    const hadKey = Boolean(settings?.hasCurseforgeKey || settings?.hasModrinthKey);
    const clearing = draft.clearCurseforgeKey || draft.clearModrinthKey;
    try {
      await saveKeys();
      toast.success(t(hadKey && clearing ? "settings.apiKeys.cleared" : "settings.apiKeys.saved"));
    } catch (error) {
      fail(error);
    }
  };

  const clearKeys = async (target: "curseforge" | "modrinth") => {
    patch(
      target === "curseforge"
        ? { clearCurseforgeKey: true, draftCurseforgeKey: "" }
        : { clearModrinthKey: true, draftModrinthKey: "" },
    );
    try {
      await useSettingsStore.getState().saveApiKeys();
      toast.success(t("settings.apiKeys.cleared"));
    } catch (error) {
      fail(error);
    }
  };

  return (
    <div className="mx-auto flex h-full w-full max-w-6xl flex-col gap-4 overflow-y-auto p-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-2xl font-bold tracking-tight">{t("settings.title")}</h1>
        <div className="flex h-6 items-center gap-1.5 text-xs text-muted-foreground">
          <span
            key={displaySaveState}
            aria-live="polite"
            className="flex items-center gap-1.5 animate-in fade-in-0 slide-in-from-top-1 duration-150"
          >
            {displaySaveState === "saving" ? <Loader2 className="size-3.5 animate-spin" /> : null}
            {displaySaveState === "saved" ? <Check className="size-3.5 text-emerald-500" /> : null}
            {displaySaveState === "error" ? <AlertCircle className="size-3.5 text-destructive" /> : null}
            <span>
              {displaySaveState === "saving"
                ? t("settings.autoSave.saving")
                : displaySaveState === "saved"
                  ? t("settings.autoSave.saved")
                  : displaySaveState === "error"
                    ? t("settings.autoSave.error")
                    : ""}
            </span>
          </span>
        </div>
      </div>

      {loading ? <Progress value={undefined} className="h-0.5 animate-pulse" /> : null}
      {persistError ? <div className="rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">{t("settings.saveFailed", { reason: persistError })}</div> : null}
      {!persistError && autoSaveError ? <div className="rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">{autoSaveError}</div> : null}

      <div className="grid gap-4 lg:grid-cols-2">
        <div className="grid content-start gap-4">
          <Card>
            <CardHeader>
              <CardTitle>{t("settings.language.label")}</CardTitle>
            </CardHeader>
            <CardContent>
              <Combobox
                value={draft.draftLanguage}
                onValueChange={(value) => void handleLanguage(value)}
                options={[
                  { value: languageSystem, label: t("settings.language.system") },
                  { value: languageChinese, label: t("settings.language.zh") },
                  { value: languageEnglish, label: t("settings.language.en") },
                ]}
                disabled={isSavingLanguage}
                aria-label={t("settings.language.label")}
                className="w-full"
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("settings.theme.label")}</CardTitle>
            </CardHeader>
            <CardContent>
              <RadioGroup value={draft.draftTheme} onValueChange={(value) => void handleTheme(value)} disabled={isSavingTheme}>
                <label className="flex items-center gap-2 text-sm">
                  <RadioGroupItem value="dark" /> {t("settings.theme.dark")}
                </label>
                <label className="flex items-center gap-2 text-sm">
                  <RadioGroupItem value="light" /> {t("settings.theme.light")}
                </label>
                <label className="flex items-center gap-2 text-sm">
                  <RadioGroupItem value={themeSystem} /> {t("settings.theme.system")}
                </label>
              </RadioGroup>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("settings.animations.label")}</CardTitle>
              <CardDescription>{t("settings.animations.speed")}</CardDescription>
            </CardHeader>
            <CardContent className="flex items-center gap-3">
              <Slider
                value={[draft.draftAnimationDurationMultiplier]}
                min={minAnimationDurationMultiplier}
                max={maxAnimationDurationMultiplier}
                step={0.1}
                onValueChange={([value]) => {
                  patch({ draftAnimationDurationMultiplier: value });
                  scheduleAutoSave("animations");
                }}
              />
              <Input
                type="number"
                min={minAnimationDurationMultiplier}
                max={maxAnimationDurationMultiplier}
                step={0.1}
                value={draft.draftAnimationDurationMultiplier}
                onChange={(event) => {
                  const value = Number(event.target.value);
                  patch({ draftAnimationDurationMultiplier: Number.isFinite(value) ? value : 1 });
                  scheduleAutoSave("animations");
                }}
                className="w-24"
              />
              <span className="text-sm text-muted-foreground">x</span>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("settings.cleanup.label")}</CardTitle>
            </CardHeader>
            <CardContent>
              <label className="flex items-center justify-between gap-3 text-sm">
                <span>{t("settings.cleanup.autoScan")}</span>
                <Switch
                  checked={draft.draftAutoScanUnusedDependencies}
                  onCheckedChange={(value) => {
                    patch({ draftAutoScanUnusedDependencies: value });
                    scheduleAutoSave("cleanup");
                  }}
                />
              </label>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("settings.advanced.label")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-1">
              <label className="flex items-center justify-between gap-3 text-sm">
                <span>{t("settings.advanced.jijDetection")}</span>
                <span className="flex items-center gap-2">
                  {isSavingJijDetection ? <Loader2 className="size-3.5 animate-spin" /> : null}
                  <Switch checked={draft.draftEnableJijDetection} onCheckedChange={(value) => void handleJij(value)} />
                </span>
              </label>
              <p className="text-xs text-muted-foreground">{t("settings.advanced.jijDetectionSubtitle")}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("settings.hardwareAcceleration.label")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-1">
              <label className="flex items-center justify-between gap-3 text-sm">
                <span>{t("settings.hardwareAcceleration.enabled")}</span>
                <span className="flex items-center gap-2">
                  {isSavingHardwareAcceleration ? <Loader2 className="size-3.5 animate-spin" /> : null}
                  <Switch checked={draft.draftHardwareAcceleration} onCheckedChange={(value) => void handleHardware(value)} />
                </span>
              </label>
              <p className="text-xs text-muted-foreground">{t("settings.hardwareAcceleration.hint")}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("settings.releaseType.label")}</CardTitle>
              <CardDescription>{t("settings.releaseType.hint")}</CardDescription>
            </CardHeader>
            <CardContent>
              <RadioGroup
                value={draft.draftPreferredReleaseType}
                onValueChange={(value) => {
                  patch({ draftPreferredReleaseType: value });
                  scheduleAutoSave("releaseType");
                }}
              >
                {[
                  ["latest", "settings.releaseType.latest"],
                  ["release", "settings.releaseType.stable"],
                  ["beta", "settings.releaseType.beta"],
                  ["alpha", "settings.releaseType.alpha"],
                ].map(([value, label]) => (
                  <label key={value} className="flex items-center gap-2 text-sm">
                    <RadioGroupItem value={value} /> {t(label)}
                  </label>
                ))}
              </RadioGroup>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("settings.cacheDir.label")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <Input value={settings?.cachePath || ""} readOnly aria-label={t("settings.cacheDir.path")} />
              <div className="flex flex-wrap gap-2">
                <Button type="button" variant="outline" size="sm" disabled={isChoosingCacheDir} onClick={() => void handleChooseCache()}>
                  {isChoosingCacheDir ? <Loader2 className="size-4 animate-spin" /> : <FolderOpen className="size-4" />}
                  {t("settings.cacheDir.choose")}
                </Button>
                <Button type="button" variant="ghost" size="sm" disabled={!settings?.cacheDir || isSavingCacheDir} onClick={() => void handleResetCache()}>
                  {isSavingCacheDir ? <Loader2 className="size-4 animate-spin" /> : <RotateCcw className="size-4" />}
                  {t("settings.cacheDir.reset")}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="grid content-start gap-4">
          <Card>
            <CardHeader>
              <CardTitle>{t("settings.mcim.label")}</CardTitle>
              <CardDescription>{t("settings.mcim.hint")}</CardDescription>
            </CardHeader>
            <CardContent>
              <label className="flex items-center justify-between gap-3 text-sm">
                <span>{t("settings.mcim.useMirror")}</span>
                <Switch
                  checked={draft.draftMCIMEnabled}
                  onCheckedChange={(value) => {
                    patch({ draftMCIMEnabled: value });
                    scheduleAutoSave("mcim");
                  }}
                />
              </label>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("settings.simpleMode.label")}</CardTitle>
              <CardDescription>{t("settings.simpleMode.hint")}</CardDescription>
            </CardHeader>
            <CardContent>
              <label className="flex items-center justify-between gap-3 text-sm">
                <span>{t("settings.simpleMode.enabled")}</span>
                <Switch
                  checked={draft.draftSimpleMode}
                  onCheckedChange={(value) => {
                    patch({ draftSimpleMode: value });
                    scheduleAutoSave("simpleMode");
                  }}
                />
              </label>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("settings.network.label")}</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4">
              <NumberField
                label={t("settings.network.fileConcurrency")}
                value={draft.draftFileConcurrency}
                min={1}
                max={32}
                onChange={(value) => {
                  patch({ draftFileConcurrency: value });
                  scheduleAutoSave("network", 800);
                }}
              />
              <label className="flex items-center justify-between gap-3 text-sm">
                <span>{t("settings.network.adaptiveFileConcurrency")}</span>
                <Switch
                  checked={draft.draftAdaptiveFileConcurrency}
                  onCheckedChange={(value) => {
                    patch({ draftAdaptiveFileConcurrency: value });
                    scheduleAutoSave("network", 800);
                  }}
                />
              </label>
              {draft.draftAdaptiveFileConcurrency ? (
                <NumberField
                  label={t("settings.network.targetDownloadRateMiB")}
                  value={draft.draftTargetDownloadRateMiB}
                  min={0.1}
                  max={5}
                  step={0.1}
                  onChange={(value) => {
                    patch({ draftTargetDownloadRateMiB: value });
                    scheduleAutoSave("network", 800);
                  }}
                />
              ) : null}
              <NumberField
                label={t("settings.network.concurrentDownloads")}
                value={draft.draftConcurrentDownloads}
                min={1}
                max={16}
                onChange={(value) => {
                  patch({ draftConcurrentDownloads: value });
                  scheduleAutoSave("network", 800);
                }}
              />
              <NumberField
                label={t("settings.network.requestsPerSecond")}
                value={draft.draftRequestsPerSecond}
                min={0}
                max={100}
                onChange={(value) => {
                  patch({ draftRequestsPerSecond: value });
                  scheduleAutoSave("network", 800);
                }}
              />
              <label className="flex items-center justify-between gap-3 text-sm">
                <span>{t("settings.network.verifySha1")}</span>
                <Switch
                  checked={draft.draftVerifySHA1}
                  onCheckedChange={(value) => {
                    patch({ draftVerifySHA1: value });
                    scheduleAutoSave("network", 800);
                  }}
                />
              </label>
            </CardContent>
          </Card>

          <ApiKeyCard
            title={t("settings.apiKeys.curseforge.label")}
            hint={t("settings.apiKeys.curseforge.hint")}
            placeholder={t("settings.apiKeys.curseforge.placeholder")}
            hasKey={Boolean(settings?.hasCurseforgeKey)}
            mask={settings?.curseforgeKeyMask || ""}
            value={draft.draftCurseforgeKey}
            saving={isSavingKeys}
            onChange={(value) => {
              patch({ draftCurseforgeKey: value, clearCurseforgeKey: false });
              scheduleAutoSave("keys", 900);
            }}
            onClear={() => void clearKeys("curseforge")}
            onSave={() => void handleSaveKeys()}
            clearLabel={t("settings.apiKeys.curseforge.clear")}
            saveLabel={t("settings.apiKeys.curseforge.save")}
            statusLabel={t("settings.apiKeys.curseforge.status")}
            emptyLabel={t("settings.apiKeys.curseforge.statusEmpty")}
          />

          <ApiKeyCard
            title={t("settings.apiKeys.modrinth.label")}
            hint={t("settings.apiKeys.modrinth.hint")}
            placeholder={t("settings.apiKeys.modrinth.placeholder")}
            hasKey={Boolean(settings?.hasModrinthKey)}
            mask={settings?.modrinthKeyMask || ""}
            value={draft.draftModrinthKey}
            saving={isSavingKeys}
            onChange={(value) => {
              patch({ draftModrinthKey: value, clearModrinthKey: false });
              scheduleAutoSave("keys", 900);
            }}
            onClear={() => void clearKeys("modrinth")}
            onSave={() => void handleSaveKeys()}
            clearLabel={t("settings.apiKeys.modrinth.clear")}
            saveLabel={t("settings.apiKeys.modrinth.save")}
            statusLabel={t("settings.apiKeys.modrinth.status")}
            emptyLabel={t("settings.apiKeys.modrinth.statusEmpty")}
          />
        </div>
      </div>
    </div>
  );
}

function NumberField({
  label,
  value,
  min,
  max,
  step = 1,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  onChange: (value: number) => void;
}) {
  const [text, setText] = useState(String(value));
  const [editing, setEditing] = useState(false);

  // Keep the field in sync with external changes, but do not overwrite an
  // intermediate string such as "0." while the user is typing a decimal.
  useEffect(() => {
    if (!editing) setText(String(value));
  }, [editing, value]);

  const clamp = (numeric: number) =>
    Math.min(max ?? Number.POSITIVE_INFINITY, Math.max(min ?? Number.NEGATIVE_INFINITY, numeric));

  const commit = () => {
    const parsed = text.trim() === "" ? min : Number(text);
    const next = clamp(Number.isFinite(parsed) ? parsed : min);
    setText(String(next));
    setEditing(false);
    if (next !== value) onChange(next);
  };

  return (
    <label className="grid gap-1.5 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <Input
        type="number"
        min={min}
        max={max}
        step={step}
        value={text}
        onFocus={() => setEditing(true)}
        onChange={(event) => {
          const raw = event.target.value;
          setText(raw);
          const parsed = Number(raw);
          // Commit only complete, in-range numbers; let partial input such as
          // "0." stay local until blur or Enter.
          if (raw.trim() !== "" && Number.isFinite(parsed) && parsed >= min && parsed <= max) {
            onChange(parsed);
          }
        }}
        onBlur={commit}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            event.currentTarget.blur();
          } else if (event.key === "Escape") {
            event.preventDefault();
            setText(String(value));
            setEditing(false);
            event.currentTarget.blur();
          }
        }}
      />
    </label>
  );
}

function ApiKeyCard(props: {
  title: string;
  hint: string;
  placeholder: string;
  hasKey: boolean;
  mask: string;
  value: string;
  saving: boolean;
  clearLabel: string;
  saveLabel: string;
  statusLabel: string;
  emptyLabel: string;
  onChange: (value: string) => void;
  onClear: () => void;
  onSave: () => void;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{props.title}</CardTitle>
        <CardDescription>{props.hint}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-2">
        <div>
          {props.hasKey ? (
            <Badge variant="secondary" className="text-emerald-700 dark:text-emerald-300">
              {props.statusLabel}: {props.mask}
            </Badge>
          ) : (
            <Badge variant="outline">{props.emptyLabel}</Badge>
          )}
        </div>
        <Input
          type="password"
          value={props.value}
          placeholder={props.placeholder}
          onChange={(event) => props.onChange(event.target.value)}
        />
        <div className="flex flex-wrap gap-2">
          <Button type="button" size="sm" disabled={props.saving} onClick={props.onSave}>
            {props.saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
            {props.saveLabel}
          </Button>
          <Button type="button" size="sm" variant="ghost" className="text-destructive" disabled={!props.hasKey || props.saving} onClick={props.onClear}>
            <Trash2 className="size-4" />
            {props.clearLabel}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
