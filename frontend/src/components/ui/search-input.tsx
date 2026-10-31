import * as React from "react"
import { Search, X } from "lucide-react"

import { cn } from "@/lib/utils"

import { Input } from "./input"

interface SearchInputProps
  extends Omit<React.ComponentProps<typeof Input>, "value" | "onChange"> {
  value: string
  onValueChange: (value: string) => void
  containerClassName?: string
  clearAriaLabel?: string
}

/**
 * Search field with a clear button that slides in from the right once the
 * value becomes non-empty and slides back out when it is cleared.
 */
const SearchInput = React.forwardRef<HTMLInputElement, SearchInputProps>(
  function SearchInput(
    {
      value,
      onValueChange,
      containerClassName,
      className,
      clearAriaLabel = "Clear search",
      ...props
    },
    forwardedRef,
  ) {
    const inputRef = React.useRef<HTMLInputElement | null>(null)
    const hasValue = value.length > 0

    const setRefs = (node: HTMLInputElement | null) => {
      inputRef.current = node
      if (typeof forwardedRef === "function") forwardedRef(node)
      else if (forwardedRef) forwardedRef.current = node
    }

    return (
      <div className={cn("relative min-w-0", containerClassName)}>
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          {...props}
          ref={setRefs}
          value={value}
          onChange={(event) => onValueChange(event.target.value)}
          className={cn(
            "pl-9 transition-[padding,border-color,box-shadow] duration-md ease-[var(--md-ease-out)]",
            hasValue ? "pr-10" : "pr-3",
            className,
          )}
        />
        <button
          type="button"
          tabIndex={hasValue ? 0 : -1}
          aria-label={clearAriaLabel}
          aria-hidden={!hasValue}
          data-state={hasValue ? "visible" : "hidden"}
          className={cn(
            "absolute top-1/2 right-1.5 flex size-7 -translate-y-1/2 items-center justify-center rounded-full text-muted-foreground",
            "transition-[opacity,transform,translate,scale,background-color,color] duration-md ease-[var(--md-ease-spring)]",
            "hover:bg-accent hover:text-foreground active:scale-90 active:duration-75",
            hasValue
              ? "translate-x-0 scale-100 opacity-100"
              : "pointer-events-none translate-x-4 scale-50 opacity-0",
          )}
          onClick={() => {
            onValueChange("")
            inputRef.current?.focus()
          }}
        >
          <X className="size-4" />
        </button>
      </div>
    )
  },
)

export { SearchInput }
