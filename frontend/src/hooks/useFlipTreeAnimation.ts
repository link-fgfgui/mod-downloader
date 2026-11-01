import { useEffect, useLayoutEffect, useRef, type RefObject } from "react";
import { gsap } from "gsap";

import { prefersReducedMotion, scaledDuration } from "@/lib/animations";

type TreeRowSnapshot = {
  key: string;
  top: number;
  height: number;
  depth: number;
  label: string;
  meta: string;
  status: string;
};

const FLIP_DURATION = 0.18;
const GHOST_DURATION = 0.14;
const MAX_ANIMATED_ROWS = 60;

function readText(row: HTMLElement, selector: string): string {
  return row.querySelector<HTMLElement>(selector)?.textContent?.trim() ?? "";
}

/**
 * Measures rows in scroll-content coordinates. offsetTop ignores in-flight
 * transforms, so a second expansion during a FLIP animation measures the real
 * layout instead of the animated position.
 */
function measureRows(container: HTMLElement): TreeRowSnapshot[] {
  const rows: TreeRowSnapshot[] = [];
  container.querySelectorAll<HTMLElement>("[data-tree-key]").forEach((row) => {
    const key = row.dataset.treeKey;
    if (!key) return;
    rows.push({
      key,
      top: row.offsetTop,
      height: row.offsetHeight,
      depth: Number(row.dataset.treeDepth ?? 0),
      label: readText(row, "[data-tree-label]"),
      meta: readText(row, "[data-tree-meta]"),
      status: readText(row, "[data-tree-status]"),
    });
  });
  return rows;
}

function createGhost(container: HTMLElement, row: TreeRowSnapshot): HTMLElement {
  const ghost = document.createElement("div");
  ghost.dataset.treeGhost = "true";
  ghost.className = "md-tree-ghost";
  ghost.style.top = `${row.top}px`;
  ghost.style.height = `${row.height}px`;
  ghost.style.paddingLeft = `${12 + row.depth * 20}px`;

  const indent = document.createElement("span");
  indent.className = "inline-block w-6 shrink-0";

  const text = document.createElement("span");
  text.className = "min-w-0 flex-1";
  const label = document.createElement("span");
  label.className = "block truncate text-sm font-medium";
  label.textContent = row.label;
  const meta = document.createElement("span");
  meta.className = "block truncate text-xs text-muted-foreground";
  meta.textContent = row.meta;
  text.append(label, meta);

  const status = document.createElement("span");
  status.className = "shrink-0 text-xs text-muted-foreground";
  status.textContent = row.status;

  ghost.append(indent, text, status);
  container.append(ghost);
  return ghost;
}

/**
 * FLIP for the dependency tree. Surviving rows slide from their previous
 * position, newly revealed rows fade in below their parent, and removed rows
 * leave as lightweight text-only ghosts (no buttons, menus or tooltips) so
 * collapse and filtering never snap.
 *
 * Runs after every commit; unchanged key lists are ignored, which keeps plain
 * selection renders free of animation work.
 */
export function useFlipTreeAnimation(containerRef: RefObject<HTMLElement | null>) {
  const previousRef = useRef<TreeRowSnapshot[] | null>(null);
  const lastContainerRef = useRef<HTMLElement | null>(null);
  const activeRef = useRef(new Set<HTMLElement>());
  const ghostsRef = useRef(new Set<HTMLElement>());

  useLayoutEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    if (lastContainerRef.current !== container) {
      lastContainerRef.current = container;
      previousRef.current = null;
    }

    const previous = previousRef.current;
    const next = measureRows(container);
    previousRef.current = next;

    // First mount uses the container's own entrance animation.
    if (!previous) return;

    const previousKeys = previous.map((row) => row.key);
    const nextKeys = next.map((row) => row.key);
    const changed =
      previousKeys.length !== nextKeys.length ||
      previousKeys.some((key, index) => key !== nextKeys[index]);
    if (!changed) return;

    // Clear in-flight tweens before animating the new layout.
    for (const element of activeRef.current) {
      gsap.killTweensOf(element);
      gsap.set(element, { clearProps: "opacity,transform" });
      element.classList.remove("md-flip-animating");
    }
    activeRef.current.clear();

    if (prefersReducedMotion() || Math.abs(next.length - previous.length) > MAX_ANIMATED_ROWS) {
      // Too many rows to animate individually: one short container fade.
      gsap.killTweensOf(container);
      gsap.fromTo(
        container,
        { opacity: 0.35 },
        {
          opacity: 1,
          duration: scaledDuration(FLIP_DURATION),
          ease: "power1.out",
          clearProps: "opacity",
        },
      );
      return;
    }

    const previousByKey = new Map(previous.map((row) => [row.key, row]));
    const nextByKey = new Map(next.map((row) => [row.key, row]));

    container.querySelectorAll<HTMLElement>("[data-tree-key]").forEach((element) => {
      const key = element.dataset.treeKey;
      if (!key) return;
      const before = previousByKey.get(key);
      if (!before) {
        // Newly revealed child: drop in from just above its final position.
        activeRef.current.add(element);
        element.classList.add("md-flip-animating");
        gsap.fromTo(
          element,
          { opacity: 0, y: -10 },
          {
            opacity: 1,
            y: 0,
            duration: scaledDuration(FLIP_DURATION),
            ease: "expo.out",
            clearProps: "opacity,transform",
            onComplete: () => {
              element.classList.remove("md-flip-animating");
              activeRef.current.delete(element);
            },
          },
        );
        return;
      }
      const after = nextByKey.get(key);
      if (!after) return;
      const deltaY = before.top - after.top;
      if (Math.abs(deltaY) <= 1) return;
      activeRef.current.add(element);
      element.classList.add("md-flip-animating");
      gsap.fromTo(
        element,
        { y: deltaY },
        {
          y: 0,
          duration: scaledDuration(FLIP_DURATION),
          ease: "expo.out",
          clearProps: "transform",
          onComplete: () => {
            element.classList.remove("md-flip-animating");
            activeRef.current.delete(element);
          },
        },
      );
    });

    // Collapsed or filtered-out rows leave as text-only ghosts.
    const ghostDuration = scaledDuration(GHOST_DURATION);
    if (ghostDuration <= 0) return;
    for (const before of previous) {
      if (nextByKey.has(before.key)) continue;
      const ghost = createGhost(container, before);
      ghostsRef.current.add(ghost);
      gsap.fromTo(
        ghost,
        { opacity: 1, y: 0 },
        {
          opacity: 0,
          y: -10,
          duration: ghostDuration,
          ease: "power1.in",
          onComplete: () => {
            ghostsRef.current.delete(ghost);
            ghost.remove();
          },
        },
      );
    }
  });

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    // Scrolling during an animation would leave rows visually offset from the
    // pointer. Drop the tweens and re-measure from the new scroll position.
    const onScroll = () => {
      if (activeRef.current.size === 0) return;
      for (const element of activeRef.current) {
        gsap.killTweensOf(element);
        gsap.set(element, { clearProps: "opacity,transform" });
        element.classList.remove("md-flip-animating");
      }
      activeRef.current.clear();
      previousRef.current = measureRows(container);
    };
    container.addEventListener("scroll", onScroll, { passive: true });
    return () => container.removeEventListener("scroll", onScroll);
  }, [containerRef]);

  useEffect(
    () => () => {
      for (const element of activeRef.current) gsap.killTweensOf(element);
      for (const ghost of ghostsRef.current) ghost.remove();
      activeRef.current.clear();
      ghostsRef.current.clear();
    },
    [],
  );
}
