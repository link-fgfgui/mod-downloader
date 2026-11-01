import { useLayoutEffect, useRef, type RefObject } from "react";
import { gsap } from "gsap";

import { prefersReducedMotion, scaledDuration } from "@/lib/animations";

const FLIP_DURATION = 0.18;
const MAX_ANIMATED_ROWS = 60;

type ListSnapshot = { key: string; top: number };

function measure(container: HTMLElement): ListSnapshot[] {
  const rows: ListSnapshot[] = [];
  container.querySelectorAll<HTMLElement>("[data-flip-key]").forEach((row) => {
    const key = row.dataset.flipKey;
    if (!key) return;
    rows.push({ key, top: row.offsetTop });
  });
  return rows;
}

/**
 * FLIP for card lists (sorting, filtering, deletion). Only transform is
 * animated, so card opacity/border styles and hover states are untouched.
 * Large changes degrade to no animation rather than animating dozens of rows.
 */
export function useFlipListAnimation(containerRef: RefObject<HTMLElement | null>) {
  const previousRef = useRef<ListSnapshot[] | null>(null);
  const lastContainerRef = useRef<HTMLElement | null>(null);
  const activeRef = useRef(new Set<HTMLElement>());

  useLayoutEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    if (lastContainerRef.current !== container) {
      lastContainerRef.current = container;
      previousRef.current = null;
    }

    const previous = previousRef.current;
    const next = measure(container);
    previousRef.current = next;
    if (!previous) return;

    const previousKeys = previous.map((row) => row.key);
    const nextKeys = next.map((row) => row.key);
    const changed =
      previousKeys.length !== nextKeys.length ||
      previousKeys.some((key, index) => key !== nextKeys[index]);
    if (!changed) return;

    for (const element of activeRef.current) {
      gsap.killTweensOf(element);
      gsap.set(element, { clearProps: "transform" });
      element.classList.remove("md-flip-animating");
    }
    activeRef.current.clear();

    if (prefersReducedMotion() || Math.abs(next.length - previous.length) > MAX_ANIMATED_ROWS) {
      return;
    }

    const previousByKey = new Map(previous.map((row) => [row.key, row]));
    const nextByKey = new Map(next.map((row) => [row.key, row]));
    const duration = scaledDuration(FLIP_DURATION);
    if (duration <= 0) return;

    container.querySelectorAll<HTMLElement>("[data-flip-key]").forEach((element) => {
      const key = element.dataset.flipKey;
      if (!key) return;
      const before = previousByKey.get(key);
      const after = nextByKey.get(key);
      if (!before || !after) return;
      const deltaY = before.top - after.top;
      if (Math.abs(deltaY) <= 1) return;
      activeRef.current.add(element);
      element.classList.add("md-flip-animating");
      gsap.fromTo(
        element,
        { y: deltaY },
        {
          y: 0,
          duration,
          ease: "expo.out",
          clearProps: "transform",
          onComplete: () => {
            element.classList.remove("md-flip-animating");
            activeRef.current.delete(element);
          },
        },
      );
    });
  });

  useLayoutEffect(
    () => () => {
      for (const element of activeRef.current) gsap.killTweensOf(element);
      activeRef.current.clear();
    },
    [],
  );
}
