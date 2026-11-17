"use client";

import { useEffect, useRef, type ReactNode } from "react";

export function StorefrontDialog({
  title,
  children,
  close,
  fullPage = false,
}: {
  title: string;
  children: ReactNode;
  close: () => void;
  fullPage?: boolean;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const node = dialog.current;
    node?.showModal();
    return () => {
      node?.close();
    };
  }, []);
  return (
    <dialog
      className={fullPage ? "hue-dialog hue-detail" : "hue-dialog"}
      ref={dialog}
      aria-label={title}
      onCancel={(e) => {
        e.preventDefault();
        close();
      }}
      onClick={(e) => {
        if (e.target !== dialog.current) return;
        const box = dialog.current.getBoundingClientRect();
        if (
          e.clientX < box.left ||
          e.clientX > box.right ||
          e.clientY < box.top ||
          e.clientY > box.bottom
        )
          close();
      }}
    >
      <div className="ux-bottom-sheet__handle" aria-hidden="true" />
      <div className="between sheet-bar">
        <h2>{title}</h2>
        <button
          type="button"
          className="btn-icon"
          onClick={close}
          aria-label="Đóng"
        >
          ×
        </button>
      </div>
      {children}
    </dialog>
  );
}
