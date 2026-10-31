import * as React from "react";
import { createPortal } from "react-dom";
import { Check, ChevronDown } from "lucide-react";

import { scaledDuration } from "@/lib/animations";
import { cn } from "@/lib/utils";

import { Input } from "./input";

export interface ComboboxOption {
  value: string;
  label?: string;
  description?: string;
  disabled?: boolean;
}

export type ComboboxItem = ComboboxOption | string;

type CloseReason = "selected" | "same" | "outside" | "escape";

interface ComboboxProps {
  value: string;
  onValueChange: (value: string) => void;
  options: ReadonlyArray<ComboboxItem>;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  contentClassName?: string;
  size?: "sm" | "default";
  searchable?: boolean;
  searchPlaceholder?: string;
  /**
   * Allow arbitrary values. The trigger becomes a text input, typing filters
   * the list, and the typed text is committed as the value while the user
   * types so surrounding form actions stay in sync.
   */
  allowCustom?: boolean;
  /** Label for the synthetic "use typed text" row in allowCustom mode. */
  customLabel?: (query: string) => string;
  autoFocus?: boolean;
  emptyText?: string;
  onOpenChange?: (open: boolean) => void;
  "aria-label"?: string;
  id?: string;
}

const PANEL_GAP = 4;
const VIEWPORT_PADDING = 8;
const MAX_PANEL_HEIGHT = 288;

const closeDurationSeconds: Record<CloseReason, number> = {
  selected: 0.18,
  same: 0.16,
  outside: 0.14,
  escape: 0.15,
};

function normalizeOptions(options: ReadonlyArray<ComboboxItem>): ComboboxOption[] {
  return options.map((option) =>
    typeof option === "string"
      ? { value: option, label: option }
      : { ...option, label: option.label ?? option.value },
  );
}

/**
 * Combobox with a panel that grows out of its trigger. The panel is rendered in
 * a portal with fixed positioning so it is never clipped by scrolling parents
 * or the sidebar. Closing is reason-aware: picking a different option retracts
 * the list towards the trigger, re-picking the current value nudges it shut,
 * and dismissing it (outside click / Escape) fades it in place.
 */
export function Combobox({
  value,
  onValueChange,
  options,
  placeholder,
  disabled = false,
  className,
  contentClassName,
  size = "default",
  searchable = false,
  searchPlaceholder,
  allowCustom = false,
  customLabel,
  autoFocus = false,
  emptyText = "No results",
  onOpenChange,
  "aria-label": ariaLabel,
  id,
}: ComboboxProps) {
  const listboxId = React.useId();
  const normalized = React.useMemo(() => normalizeOptions(options), [options]);

  const triggerRef = React.useRef<HTMLButtonElement | HTMLInputElement | null>(null);
  const panelRef = React.useRef<HTMLDivElement | null>(null);
  const searchRef = React.useRef<HTMLInputElement | null>(null);
  const optionRefs = React.useRef(new Map<number, HTMLButtonElement>());
  const closeTimer = React.useRef<number | null>(null);
  const feedbackTimer = React.useRef<number | null>(null);

  const [open, setOpen] = React.useState(false);
  const [mounted, setMounted] = React.useState(false);
  const [visible, setVisible] = React.useState(false);
  // While the panel is being positioned (before the first visible frame) its
  // side may change from the default. Disable transitions for that frame so
  // the open animation always grows out of the trigger, not from the old side.
  const [ready, setReady] = React.useState(false);
  const [closeReason, setCloseReason] = React.useState<CloseReason>("selected");
  const [query, setQuery] = React.useState("");
  const [draft, setDraft] = React.useState(value);
  const [activeIndex, setActiveIndex] = React.useState(0);
  const [feedback, setFeedback] = React.useState<"selected" | "same" | null>(null);
  const [layout, setLayout] = React.useState({
    left: 0,
    top: 0,
    width: 0,
    maxHeight: MAX_PANEL_HEIGHT,
    side: "bottom" as "top" | "bottom",
  });

  React.useEffect(() => {
    if (!open) setDraft(value);
  }, [value, open]);

  const filtered = React.useMemo(() => {
    const text = (allowCustom ? draft : query).trim().toLocaleLowerCase();
    if (!text) return normalized;
    return normalized.filter(
      (option) =>
        option.value.toLocaleLowerCase().includes(text) ||
        (option.label ?? "").toLocaleLowerCase().includes(text),
    );
  }, [allowCustom, draft, normalized, query]);

  const showCustomOption =
    allowCustom &&
    Boolean(draft.trim()) &&
    !normalized.some((option) => option.value === draft.trim());
  const visibleOptions = React.useMemo(
    () =>
      showCustomOption
        ? [
            {
              value: draft.trim(),
              label: customLabel ? customLabel(draft.trim()) : draft.trim(),
            },
            ...filtered,
          ]
        : filtered,
    [customLabel, draft, filtered, showCustomOption],
  );

  const selectedOption = normalized.find((option) => option.value === value);

  // Filtering can leave the highlight beyond the last row (or on a negative
  // index after the selected value disappears). Keep it inside the list.
  React.useEffect(() => {
    setActiveIndex((current) => {
      if (visibleOptions.length === 0) return 0;
      if (current < 0) return 0;
      return Math.min(current, visibleOptions.length - 1);
    });
  }, [visibleOptions.length]);

  const clearTimers = () => {
    if (closeTimer.current !== null) {
      window.clearTimeout(closeTimer.current);
      closeTimer.current = null;
    }
    if (feedbackTimer.current !== null) {
      window.clearTimeout(feedbackTimer.current);
      feedbackTimer.current = null;
    }
  };

  React.useEffect(() => clearTimers, []);

  const showFeedback = (kind: "selected" | "same") => {
    if (feedbackTimer.current !== null) window.clearTimeout(feedbackTimer.current);
    setFeedback(null);
    // Re-add on a later tick so the CSS animation restarts even when the same
    // feedback is triggered twice in a row.
    window.setTimeout(() => {
      setFeedback(kind);
      feedbackTimer.current = window.setTimeout(() => setFeedback(null), 380);
    }, 20);
  };

  const openPanel = React.useCallback(() => {
    if (disabled) return;
    clearTimers();
    setReady(false);
    setMounted(true);
    setOpen(true);
    setCloseReason("selected");
    setQuery("");
    const selectedIndex = normalized.findIndex((option) => option.value === value);
    setActiveIndex(selectedIndex >= 0 ? selectedIndex : 0);
    onOpenChange?.(true);
  }, [disabled, normalized, onOpenChange, value]);

  const closePanel = React.useCallback(
    (reason: CloseReason, options?: { restoreFocus?: boolean }) => {
      if (!open) return;
      clearTimers();
      setCloseReason(reason);
      setOpen(false);
      setVisible(false);
      if (allowCustom && reason === "escape") setDraft(value);
      if (reason === "selected" || reason === "same") showFeedback(reason);
      const duration = scaledDuration(closeDurationSeconds[reason]) * 1000;
      if (duration <= 0) {
        setMounted(false);
      } else {
        closeTimer.current = window.setTimeout(() => setMounted(false), duration + 50);
      }
      onOpenChange?.(false);
      const shouldRestoreFocus =
        options?.restoreFocus ??
        (reason === "escape" || reason === "selected" || reason === "same");
      if (shouldRestoreFocus) {
        triggerRef.current?.focus({ preventScroll: true });
      }
    },
    [allowCustom, onOpenChange, open, value],
  );

  // Position and size the panel before it becomes visible. This avoids the
  // panel jumping (and the resulting end-of-animation flicker) after opening.
  React.useLayoutEffect(() => {
    if (!mounted) return;
    const trigger = triggerRef.current;
    const panel = panelRef.current;
    if (!trigger || !panel) return;

    const measure = () => {
      const rect = trigger.getBoundingClientRect();
      const spaceBelow = window.innerHeight - rect.bottom - PANEL_GAP - VIEWPORT_PADDING;
      const spaceAbove = rect.top - PANEL_GAP - VIEWPORT_PADDING;
      // Prefer the side that can show the whole panel. A trigger near the
      // bottom of the viewport should open upwards even when a sliver of room
      // is left below, otherwise the list is reduced to a few scrolling rows.
      const preferBelow = spaceBelow >= MAX_PANEL_HEIGHT || spaceBelow >= spaceAbove;
      const available = Math.max(0, preferBelow ? spaceBelow : spaceAbove);
      const maxHeight = Math.min(MAX_PANEL_HEIGHT, available);
      // scrollHeight sees the full list even when the panel is currently
      // capped, so the final top position does not depend on the previous
      // frame's height.
      const contentHeight = Math.max(
        panel.offsetHeight,
        Math.min(panel.scrollHeight || MAX_PANEL_HEIGHT, MAX_PANEL_HEIGHT),
      );
      const height = Math.min(contentHeight, maxHeight);
      const width = Math.min(
        Math.max(Math.round(rect.width), 128),
        Math.max(128, window.innerWidth - VIEWPORT_PADDING * 2),
      );
      const left = Math.round(
        Math.min(
          Math.max(rect.left, VIEWPORT_PADDING),
          Math.max(VIEWPORT_PADDING, window.innerWidth - width - VIEWPORT_PADDING),
        ),
      );
      const side = preferBelow ? "bottom" : "top";
      const top = Math.round(preferBelow ? rect.bottom + PANEL_GAP : rect.top - PANEL_GAP - height);

      setLayout((previous) =>
        previous.left === left &&
        previous.top === top &&
        previous.width === width &&
        previous.maxHeight === maxHeight &&
        previous.side === side
          ? previous
          : { left, top, width, maxHeight, side },
      );
    };

    measure();
    const onScroll = (event: Event) => {
      const target = event.target;
      if (target instanceof Node && panel.contains(target)) return;
      measure();
    };
    const observer = new ResizeObserver(measure);
    observer.observe(panel);
    window.addEventListener("resize", measure);
    window.addEventListener("scroll", onScroll, true);
    return () => {
      observer.disconnect();
      window.removeEventListener("resize", measure);
      window.removeEventListener("scroll", onScroll, true);
    };
  }, [mounted, visibleOptions.length, query, draft]);

  // Start the enter transition only after the panel has been positioned.
  React.useEffect(() => {
    if (!open || !mounted) return;
    if (visible) return;
    const frame = window.requestAnimationFrame(() => {
      setVisible(true);
      setReady(true);
    });
    return () => window.cancelAnimationFrame(frame);
  }, [open, mounted, visible]);

  React.useEffect(() => {
    if (!open) return;
    if (searchable && !allowCustom) {
      searchRef.current?.focus({ preventScroll: true });
    }
  }, [open, searchable, allowCustom]);

  React.useEffect(() => {
    if (!open) return;
    const option = optionRefs.current.get(activeIndex);
    option?.scrollIntoView({ block: "nearest" });
  }, [activeIndex, open]);

  // Dismissal and keyboard navigation while the panel is open.
  React.useEffect(() => {
    if (!open) return;

    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node | null;
      if (!target) return;
      if (panelRef.current?.contains(target) || triggerRef.current?.contains(target)) return;
      closePanel("outside");
    };

    const onKeyDown = (event: KeyboardEvent) => {
      // Escape belongs to the open panel even when focus drifted to the body
      // (for example after a pointer click on the trigger). Handle it in the
      // capture phase so the shell's back-navigation shortcut never sees it.
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        closePanel("escape");
        return;
      }

      const panelFocused =
        panelRef.current?.contains(document.activeElement) ||
        triggerRef.current?.contains(document.activeElement);
      if (!panelFocused) return;

      if (event.key === "Tab") {
        closePanel("outside", { restoreFocus: false });
        return;
      }
      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        event.preventDefault();
        if (visibleOptions.length === 0) return;
        const direction = event.key === "ArrowDown" ? 1 : -1;
        setActiveIndex((current) => {
          const start = current < 0 ? 0 : current;
          return (start + direction + visibleOptions.length) % visibleOptions.length;
        });
        return;
      }
      if (event.key === "Home") {
        event.preventDefault();
        setActiveIndex(0);
        return;
      }
      if (event.key === "End") {
        event.preventDefault();
        setActiveIndex(Math.max(0, visibleOptions.length - 1));
        return;
      }
      if (event.key === "Enter") {
        const option = visibleOptions[activeIndex];
        if (option && !option.disabled) {
          event.preventDefault();
          const same = option.value === value;
          setDraft(option.value);
          if (!same) onValueChange(option.value);
          closePanel(same ? "same" : "selected");
        } else if (allowCustom && draft.trim()) {
          event.preventDefault();
          if (draft !== value) onValueChange(draft);
          closePanel("same");
        }
      }
    };

    document.addEventListener("pointerdown", onPointerDown, true);
    document.addEventListener("keydown", onKeyDown, true);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown, true);
      document.removeEventListener("keydown", onKeyDown, true);
    };
  }, [activeIndex, allowCustom, closePanel, draft, visibleOptions, onValueChange, open, value]);

  const selectOption = (index: number) => {
    const option = visibleOptions[index];
    if (!option || option.disabled) return;
    const same = option.value === value && !(allowCustom && draft !== value);
    setDraft(option.value);
    setQuery("");
    if (!same) onValueChange(option.value);
    closePanel(same ? "same" : "selected");
  };

  const triggerClassName = cn(
    "flex w-full min-w-0 items-center justify-between gap-2 rounded-md border border-input bg-transparent px-3 text-sm whitespace-nowrap shadow-xs outline-none transition-[color,box-shadow,background-color,scale] duration-md ease-[var(--md-ease-out)]",
    "focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50",
    "disabled:cursor-not-allowed disabled:opacity-50",
    "dark:bg-input/30 dark:hover:bg-input/50",
    "active:scale-[0.985] active:duration-75 active:ease-[var(--md-ease-in)]",
    size === "sm" ? "h-8 py-1" : "h-9 py-2",
    !value && "text-muted-foreground",
    className,
  );

  const panel =
    mounted && typeof document !== "undefined"
      ? createPortal(
          <div
            ref={panelRef}
            id={listboxId}
            role="listbox"
            aria-label={ariaLabel}
            data-slot="combobox-panel"
            data-state={visible ? "open" : "closed"}
            data-ready={ready ? "true" : "false"}
            data-side={layout.side}
            data-close-reason={closeReason}
            className={cn(
              "md-combobox-panel fixed z-[130] flex flex-col overflow-hidden rounded-lg border bg-popover text-popover-foreground shadow-lg",
              contentClassName,
            )}
            style={{
              left: layout.left,
              top: layout.top,
              width: layout.width,
              maxHeight: layout.maxHeight,
            }}
          >
            {searchable && !allowCustom ? (
              <div className="shrink-0 border-b p-1.5">
                <Input
                  ref={searchRef}
                  value={query}
                  onChange={(event) => {
                    setQuery(event.target.value);
                    setActiveIndex(0);
                  }}
                  placeholder={searchPlaceholder ?? placeholder}
                  className="h-8"
                  autoComplete="off"
                  aria-label={searchPlaceholder ?? "Search"}
                  aria-controls={listboxId}
                  aria-activedescendant={
                    visibleOptions[activeIndex] ? `${listboxId}-option-${activeIndex}` : undefined
                  }
                />
              </div>
            ) : null}
            <div className="md-combobox-list min-h-0 flex-1 overflow-y-auto overscroll-contain p-1">
              {visibleOptions.length === 0 ? (
                <div className="px-2 py-6 text-center text-xs text-muted-foreground">{emptyText}</div>
              ) : (
                visibleOptions.map((option, index) => {
                  const selected = option.value === value;
                  const active = index === activeIndex;
                  return (
                    <button
                      key={`${option.value}-${index}`}
                      ref={(node) => {
                        if (node) optionRefs.current.set(index, node);
                        else optionRefs.current.delete(index);
                      }}
                      id={`${listboxId}-option-${index}`}
                      type="button"
                      role="option"
                      aria-selected={selected}
                      disabled={option.disabled}
                      tabIndex={-1}
                      data-slot="combobox-option"
                      className={cn(
                        "relative flex w-full cursor-default items-center gap-2 rounded-sm py-1.5 pr-8 pl-2 text-left text-sm outline-hidden select-none",
                        "transition-colors duration-fast",
                        active && "bg-accent text-accent-foreground",
                        option.disabled && "pointer-events-none opacity-50",
                      )}
                      onPointerMove={() => setActiveIndex(index)}
                      onClick={() => selectOption(index)}
                    >
                      <span className="min-w-0 flex-1">
                        <span className="block truncate">{option.label}</span>
                        {option.description ? (
                          <span className="block truncate text-xs text-muted-foreground">
                            {option.description}
                          </span>
                        ) : null}
                      </span>
                      <Check
                        className={cn(
                          "absolute right-2 size-4 shrink-0 transition-opacity duration-fast",
                          selected ? "opacity-100" : "opacity-0",
                        )}
                      />
                    </button>
                  );
                })
              )}
            </div>
          </div>,
          document.body,
        )
      : null;

  if (allowCustom) {
    return (
      <>
        <div className="relative min-w-0">
          <Input
            ref={(node) => {
              triggerRef.current = node;
            }}
            id={id}
            role="combobox"
            aria-label={ariaLabel}
            aria-expanded={open}
            aria-controls={listboxId}
            aria-autocomplete="list"
            aria-activedescendant={
              open && visibleOptions[activeIndex] ? `${listboxId}-option-${activeIndex}` : undefined
            }
            value={draft}
            placeholder={placeholder}
            disabled={disabled}
            autoFocus={autoFocus}
            autoComplete="off"
            className={cn("pr-8", className)}
            data-combobox-feedback={feedback ?? undefined}
            onChange={(event) => {
              const next = event.target.value;
              setDraft(next);
              setActiveIndex(0);
              // In free-input mode the text is the value: commit it as the user
              // types so surrounding form actions (e.g. Add) enable immediately.
              if (next !== value) onValueChange(next);
              if (!open) openPanel();
            }}
            onClick={() => {
              if (!open) openPanel();
            }}
            onKeyDown={(event) => {
              if (event.key === "ArrowDown" && !open) {
                event.preventDefault();
                openPanel();
              }
            }}
          />
          <ChevronDown
            className={cn(
              "pointer-events-none absolute top-1/2 right-2.5 size-4 -translate-y-1/2 text-muted-foreground opacity-50 transition-transform duration-md",
              visible && "rotate-180",
            )}
          />
        </div>
        {panel}
      </>
    );
  }

  return (
    <>
      <button
        ref={(node) => {
          triggerRef.current = node;
        }}
        id={id}
        type="button"
        role="combobox"
        aria-label={ariaLabel}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listboxId}
        disabled={disabled}
        data-combobox-feedback={feedback ?? undefined}
        className={triggerClassName}
        onClick={() => {
          if (open) closePanel("outside");
          else openPanel();
        }}
        onKeyDown={(event) => {
          if (!open && (event.key === "ArrowDown" || event.key === "Enter" || event.key === " ")) {
            event.preventDefault();
            openPanel();
          }
        }}
      >
        <span className="line-clamp-1 flex min-w-0 flex-1 items-center gap-2 text-left">
          {selectedOption?.label ?? (value || placeholder || "")}
        </span>
        <ChevronDown
          className={cn(
            "size-4 shrink-0 opacity-50 transition-transform duration-md",
            visible && "rotate-180",
          )}
        />
      </button>
      {panel}
    </>
  );
}
