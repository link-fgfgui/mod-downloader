import { useId, useState } from "react";
import {
  Cog,
  Download,
  Home,
  ListChecks,
  PinOff,
  ListMusic,
} from "lucide-react";
import { NavLink, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { VersionPicker } from "@/components/VersionPicker";

const navItems = [
  { to: "/", labelKey: "nav.home", icon: Home },
  { to: "/download", labelKey: "nav.download", icon: Download },
  { to: "/manage", labelKey: "nav.manage", icon: ListChecks },
  { to: "/favorites", labelKey: "nav.favorites", icon: ListMusic },
  { to: "/unpin", labelKey: "nav.unpin", icon: PinOff },
  { to: "/settings", labelKey: "nav.settings", icon: Cog },
];

const NAV_ITEM_HEIGHT = 40;
const NAV_ITEM_GAP = 4;
const NAV_ITEMS_HEIGHT = navItems.length * NAV_ITEM_HEIGHT + (navItems.length - 1) * NAV_ITEM_GAP;

/**
 * Self-drawn pin with an animated diagonal strike. The pin stays static so the
 * glyph keeps its meaning in both states; only the `\` is drawn in
 * (unpinned / can collapse) or erased back out (pinned / fixed). A mask cuts a
 * clean gap in the pin under the strike while it travels.
 */
function SidebarToggleIcon({ pinned }: { pinned: boolean }) {
  const maskId = `sidebar-toggle-${useId().replace(/:/g, "")}`;
  const slashOffset = pinned ? 1 : 0;
  const slashTransition = {
    transition: "stroke-dashoffset var(--md-transition-normal, 180ms) ease-out",
  };

  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="size-[18px]">
      <mask id={maskId}>
        <rect width="24" height="24" fill="white" />
        <path
          d="M2.5 2.5 21.5 21.5"
          stroke="black"
          strokeWidth="5"
          strokeDasharray="1"
          pathLength={1}
          strokeDashoffset={slashOffset}
          style={slashTransition}
        />
      </mask>
      <g
        mask={`url(#${maskId})`}
        stroke="currentColor"
        strokeWidth="1.9"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M8 3.6h8l-1.3 5.1 2.9 3.1H6.4l2.9-3.1L8 3.6Z" />
        <path d="M12 11.8V21.2" />
      </g>
      <path
        d="M2.5 2.5 21.5 21.5"
        stroke="currentColor"
        strokeWidth="1.9"
        strokeLinecap="round"
        strokeDasharray="1"
        pathLength={1}
        strokeDashoffset={slashOffset}
        style={slashTransition}
      />
    </svg>
  );
}

export function Sidebar() {
  const { t } = useTranslation();
  const location = useLocation();
  const [pinned, setPinned] = useState(true);
  const [hovered, setHovered] = useState(false);
  const expanded = pinned || hovered;
  const activeIndex = navItems.findIndex((item) =>
    item.to === "/" ? location.pathname === "/" : location.pathname.startsWith(item.to),
  );
  const safeActiveIndex = Math.max(activeIndex, 0);
  const pillOffset = safeActiveIndex * (NAV_ITEM_HEIGHT + NAV_ITEM_GAP);
  const pillClipBottom = NAV_ITEMS_HEIGHT - pillOffset - NAV_ITEM_HEIGHT;

  return (
    <aside
      className={cn(
        "relative z-20 flex h-screen min-w-0 shrink-0 flex-col overflow-hidden border-r bg-card/50 transition-[width] duration-md",
        expanded ? "w-64" : "w-16",
      )}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <div className="flex h-14 items-center gap-2 px-3">
        <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground">
          MD
        </div>
        <span
          className={cn(
            "min-w-0 flex-1 truncate text-sm font-semibold transition-[max-width,opacity] duration-md",
            expanded ? "max-w-[10rem] opacity-100" : "max-w-0 opacity-0",
          )}
          aria-hidden={!expanded}
        >
          Mod Downloader
        </span>
        <Button
          type="button"
          size="icon-xs"
          variant="ghost"
          className={cn(
            "shrink-0 transition-opacity duration-md",
            expanded ? "opacity-100" : "pointer-events-none opacity-0",
          )}
          title={pinned ? "Collapse sidebar" : "Pin sidebar"}
          aria-label={pinned ? "Collapse sidebar" : "Pin sidebar"}
          aria-hidden={!expanded}
          tabIndex={expanded ? 0 : -1}
          onClick={() => setPinned((value) => !value)}
        >
          <SidebarToggleIcon pinned={pinned} />
        </Button>
      </div>

      <nav className="relative grid min-h-0 gap-1 overflow-y-auto overflow-x-hidden px-2 py-2 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        <span
          aria-hidden="true"
          className={cn(
            "pointer-events-none absolute inset-x-2 top-2 h-10 rounded-md bg-primary transition-transform duration-md ease-out",
            activeIndex < 0 && "opacity-0",
          )}
          style={{ transform: `translateY(${pillOffset}px)` }}
        />
        {navItems.map((item) => {
          const Icon = item.icon;
          return (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === "/"}
              className={({ isActive }) =>
                cn(
                  "relative z-10 flex h-10 items-center rounded-md text-sm font-medium transition-[color,padding,gap,opacity] duration-md active:opacity-70",
                  expanded ? "gap-3 ps-3 pe-3" : "gap-0 ps-4 pe-0",
                  isActive
                    ? "text-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )
              }
              title={t(item.labelKey)}
            >
              <Icon className="size-4 shrink-0" />
              <span
                className={cn(
                  "truncate transition-[opacity,max-width] duration-md",
                  expanded ? "max-w-[10rem] opacity-100" : "max-w-0 opacity-0",
                )}
                aria-hidden={!expanded}
              >
                {t(item.labelKey)}
              </span>
            </NavLink>
          );
        })}
        <div
          aria-hidden="true"
          className={cn(
            "pointer-events-none absolute inset-x-2 top-2 z-20 grid gap-1 transition-[clip-path] duration-md ease-out",
            activeIndex < 0 && "opacity-0",
          )}
          style={{
            clipPath: `inset(${pillOffset}px 0 ${pillClipBottom}px 0)`,
          }}
        >
          {navItems.map((item) => {
            const Icon = item.icon;
            return (
              <div
                key={item.to}
                className={cn(
                  "flex h-10 items-center rounded-md text-sm font-medium text-primary-foreground transition-[padding,gap] duration-md",
                  expanded ? "gap-3 ps-3 pe-3" : "gap-0 ps-4 pe-0",
                )}
              >
                <Icon className="size-4 shrink-0" />
                <span
                  className={cn(
                    "truncate transition-[opacity,max-width] duration-md",
                    expanded ? "max-w-[10rem] opacity-100" : "max-w-0 opacity-0",
                  )}
                >
                  {t(item.labelKey)}
                </span>
              </div>
            );
          })}
        </div>
        <Button
          type="button"
          size="icon-sm"
          variant="ghost"
          className={cn(
            "mx-auto mt-1 transition-opacity duration-md",
            expanded ? "pointer-events-none opacity-0" : "opacity-100",
          )}
          title="Pin sidebar"
          aria-label="Pin sidebar"
          aria-hidden={expanded}
          tabIndex={expanded ? -1 : 0}
          onClick={() => setPinned(true)}
        >
          <SidebarToggleIcon pinned={false} />
        </Button>
      </nav>

      <div className="mt-auto w-full min-w-0 min-h-0 shrink-0">
        <VersionPicker expanded={expanded} />
      </div>
    </aside>
  );
}
