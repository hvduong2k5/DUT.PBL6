"use client";

import { createContext, useContext, useState, type ReactNode } from "react";

type View = {
  paper: boolean;
  compact: boolean;
  group: number;
  setPaper: (value: boolean) => void;
  setCompact: (value: boolean) => void;
  setGroup: (value: number) => void;
};
const ViewContext = createContext<View | null>(null);

export function StorefrontView({ children }: { children: ReactNode }) {
  const [paper, setPaper] = useState(false);
  const [compact, setCompact] = useState(false);
  const [group, setGroup] = useState(-1);
  return (
    <ViewContext.Provider
      value={{ paper, compact, group, setPaper, setCompact, setGroup }}
    >
      <div className={compact ? "hue-view hue-compact" : "hue-view"}>
        {children}
      </div>
    </ViewContext.Provider>
  );
}

export function useStorefrontView() {
  const view = useContext(ViewContext);
  if (!view) throw new Error("Missing StorefrontView");
  return view;
}
