"use client";

import { useEffect, useId, useMemo, useRef, useState } from "react";

export interface LocationOption {
  value: string;
  label: string;
}

/** Fold Vietnamese text for search: strip diacritics so "hai chau" matches "Hải Châu". */
export function foldLocationText(value: string): string {
  return value
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/đ/g, "d")
    .replace(/Đ/g, "D")
    .toLowerCase();
}

const MAX_RENDERED = 60;

interface LocationComboboxProps {
  options: LocationOption[];
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  ariaLabel: string;
  disabled?: boolean;
}

/**
 * Searchable single-select for province/ward fields. The user types to
 * filter (diacritics-insensitive); only an explicit pick emits a value, so
 * free text can never become an address code.
 */
export function LocationCombobox({ options, value, onChange, placeholder, ariaLabel, disabled }: LocationComboboxProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [highlight, setHighlight] = useState(0);
  const listId = useId();
  const rootRef = useRef<HTMLDivElement | null>(null);
  const listRef = useRef<HTMLUListElement | null>(null);

  const selected = options.find((entry) => entry.value === value);

  const matches = useMemo(() => {
    const folded = foldLocationText(query.trim());
    if (!folded) return options;
    return options.filter((entry) => foldLocationText(entry.label).includes(folded));
  }, [options, query]);
  const rendered = matches.slice(0, MAX_RENDERED);

  useEffect(() => {
    if (!open) return;
    const onDocumentMouseDown = (event: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDocumentMouseDown);
    return () => document.removeEventListener("mousedown", onDocumentMouseDown);
  }, [open ]);

  useEffect(() => {
    if (!open) return;
    listRef.current?.querySelector('[data-highlight="true"]')?.scrollIntoView({ block: "nearest" });
  }, [highlight, open]);

  function choose(option: LocationOption) {
    onChange(option.value);
    setOpen(false);
    setQuery("");
  }

  function handleKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      if (!open) { setOpen(true); return; }
      setHighlight((current) => Math.min(current + 1, rendered.length - 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setHighlight((current) => Math.max(current - 1, 0));
    } else if (event.key === "Enter") {
      if (open && rendered[highlight]) {
        event.preventDefault();
        choose(rendered[highlight]);
      }
    } else if (event.key === "Escape") {
      setOpen(false);
      setQuery("");
    }
  }

  return (
    <div className="location-combobox" ref={rootRef}>
      <input
        type="text"
        role="combobox"
        aria-label={ariaLabel}
        aria-expanded={open}
        aria-controls={listId}
        aria-autocomplete="list"
        autoComplete="off"
        disabled={disabled}
        placeholder={placeholder}
        value={open ? query : (selected?.label ?? "")}
        onFocus={() => { if (!disabled) { setOpen(true); setQuery(""); setHighlight(0); } }}
        onClick={() => { if (!disabled) setOpen(true); }}
        onChange={(event) => { const next = event.currentTarget.value; setQuery(next); setOpen(true); setHighlight(0); }}
        onKeyDown={handleKeyDown}
        onBlur={(event) => {
          if (!rootRef.current?.contains(event.relatedTarget as Node)) setOpen(false);
        }}
      />
      {open && !disabled ? (
        <ul className="location-combobox-list" id={listId} role="listbox" ref={listRef}>
          {rendered.map((option, index) => (
            <li
              key={option.value}
              role="option"
              aria-selected={option.value === value}
              data-highlight={index === highlight}
              className={`location-combobox-option${index === highlight ? " active" : ""}`}
              onMouseEnter={() => setHighlight(index)}
              onMouseDown={(event) => { event.preventDefault(); choose(option); }}
            >
              {option.label}
            </li>
          ))}
          {rendered.length === 0 ? <li className="location-combobox-empty">Không tìm thấy — thử gõ tên không dấu.</li> : null}
          {matches.length > rendered.length ? <li className="location-combobox-more">Còn {matches.length - rendered.length} kết quả nữa — gõ thêm để thu hẹp.</li> : null}
        </ul>
      ) : null}
    </div>
  );
}
