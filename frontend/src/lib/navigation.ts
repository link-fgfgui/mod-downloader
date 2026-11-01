import type { RouteDirection } from "@/lib/animations";

/**
 * Sidebar order, shared by the navigation list and the route transition so a
 * downward nav always enters from below and an upward nav from above.
 */
export const navOrder = ["/", "/download", "/manage", "/favorites", "/unpin", "/settings"] as const;

export type NavPath = (typeof navOrder)[number];

export function navigationIndex(pathname: string): number {
  if (pathname === "/") return 0;
  return navOrder.findIndex((to) => to !== "/" && pathname.startsWith(to));
}

export function navigationDirection(previous: string | null, next: string): RouteDirection {
  if (!previous || previous === next) return 1;
  const previousIndex = navigationIndex(previous);
  const nextIndex = navigationIndex(next);
  if (previousIndex < 0 || nextIndex < 0) return 1;
  const delta = nextIndex - previousIndex;
  return delta < 0 ? -1 : 1;
}
