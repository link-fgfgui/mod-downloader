import { useEffect, useRef, useState, type ReactNode } from "react";

import { scaledDuration } from "@/lib/animations";
import { cn } from "@/lib/utils";

type PresenceMode = "collapse" | "float";

interface AnimatedPresenceProps {
  open: boolean;
  children: ReactNode;
  className?: string;
  /** "collapse" also animates the element's height; "float" pops/fades in place. */
  mode?: PresenceMode;
  /**
   * Keep the last rendered children while the exit transition plays. This
   * avoids a selection bar emptying its count/buttons before it has finished
   * collapsing.
   */
  preserveContent?: boolean;
}

/**
 * Keeps children mounted long enough to play an exit transition, then removes
 * them. Timings follow the user's animation speed multiplier and reduced-motion
 * preference.
 */
export function AnimatedPresence({
  open,
  children,
  className,
  mode = "collapse",
  preserveContent = false,
}: AnimatedPresenceProps) {
  const [mounted, setMounted] = useState(open);
  const [shown, setShown] = useState(open);
  const exitTimer = useRef<number | null>(null);
  const lastChildren = useRef<ReactNode>(children);

  useEffect(() => {
    if (open) lastChildren.current = children;
  }, [children, open]);

  useEffect(() => {
    if (exitTimer.current !== null) {
      window.clearTimeout(exitTimer.current);
      exitTimer.current = null;
    }
    if (open) {
      setMounted(true);
      const frame = window.requestAnimationFrame(() => setShown(true));
      return () => window.cancelAnimationFrame(frame);
    }
    setShown(false);
    const delay = scaledDuration(0.18) * 1000;
    if (delay <= 0) {
      setMounted(false);
      return;
    }
    // The transition starts on the next frame after `shown` flips, so unmount
    // a little after the nominal duration to let the last frame finish.
    exitTimer.current = window.setTimeout(() => setMounted(false), delay + 50);
    return () => {
      if (exitTimer.current !== null) {
        window.clearTimeout(exitTimer.current);
        exitTimer.current = null;
      }
    };
  }, [open]);

  if (!mounted) return null;

  const content = open || !preserveContent ? children : lastChildren.current;
  const style = {
    transitionDuration: "var(--md-transition-normal, 180ms)",
    transitionTimingFunction: shown
      ? "var(--md-ease-spring, cubic-bezier(0.22, 1.2, 0.36, 1))"
      : "var(--md-ease-in, cubic-bezier(0.4, 0, 1, 1))",
  };

  if (mode === "float") {
    return (
      <div
        data-state={shown ? "open" : "closed"}
        style={{ ...style, transformOrigin: "bottom center" }}
        className={cn(
          "md-selection-toolbar transition-[opacity,transform,translate,scale]",
          className,
          shown
            ? "translate-y-0 scale-100 opacity-100"
            : "pointer-events-none translate-y-5 scale-90 opacity-0",
        )}
      >
        {content}
      </div>
    );
  }

  return (
    <div
      data-state={shown ? "open" : "closed"}
      style={style}
      className={cn(
        "grid transition-[grid-template-rows,opacity]",
        className,
        shown ? "grid-rows-[1fr] opacity-100" : "pointer-events-none grid-rows-[0fr] opacity-0",
      )}
    >
      <div className="min-h-0 overflow-hidden">{content}</div>
    </div>
  );
}
