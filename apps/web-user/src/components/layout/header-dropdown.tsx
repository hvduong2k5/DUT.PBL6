"use client";

import { useEffect, useId, useRef, useState, type ReactNode } from "react";

interface HeaderDropdownProps {
  label: ReactNode;
  ariaLabel?: string;
  triggerClassName?: string;
  panelClassName?: string;
  align?: "start" | "end";
  children: ReactNode;
}

/** Disclosure button + popover panel: closes on outside click, Escape, or choosing a link inside. */
export function HeaderDropdown({ label, ariaLabel, triggerClassName = "", panelClassName = "", align = "start", children }: HeaderDropdownProps) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const panelId = useId();

  useEffect(() => {
    if (!open) return;
    const onPointer = (event: PointerEvent) => { if (!root.current?.contains(event.target as Node)) setOpen(false); };
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      setOpen(false);
      trigger.current?.focus();
    };
    document.addEventListener("pointerdown", onPointer);
    document.addEventListener("keydown", onKey);
    return () => { document.removeEventListener("pointerdown", onPointer); document.removeEventListener("keydown", onKey); };
  }, [open]);

  return <div className="gh-dropdown" ref={root}>
    <button ref={trigger} type="button" className={triggerClassName} aria-label={ariaLabel} aria-expanded={open} aria-controls={panelId} onClick={() => setOpen((value) => !value)}>
      {label}
    </button>
    {open ? <div id={panelId} className={`gh-dropdown__panel gh-dropdown__panel--${align} ${panelClassName}`}
      onClick={(event) => { if ((event.target as HTMLElement).closest("a")) setOpen(false); }}>
      {children}
    </div> : null}
  </div>;
}
