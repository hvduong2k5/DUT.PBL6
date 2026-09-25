import type { CartItem } from "./types";

export interface SelectedCartSummary {
  lineCount: number;
  itemCount: number;
  subtotalVnd: number;
}

export function getInitiallySelectedItemIds(items: CartItem[]): string[] {
  return items.filter((item) => item.isAvailable).map((item) => item.itemId);
}

export function reconcileSelectedItemIds(items: CartItem[], selectedItemIds: string[]): string[] {
  const selected = new Set(selectedItemIds);
  return items.filter((item) => item.isAvailable && selected.has(item.itemId)).map((item) => item.itemId);
}

export function getSelectedCartSummary(items: CartItem[], selectedItemIds: string[]): SelectedCartSummary {
  const selected = new Set(selectedItemIds);
  return items.reduce<SelectedCartSummary>((summary, item) => {
    if (!item.isAvailable || !selected.has(item.itemId)) return summary;
    summary.lineCount += 1;
    summary.itemCount += item.quantity;
    summary.subtotalVnd += item.lineSubtotalVnd ?? 0;
    return summary;
  }, { lineCount: 0, itemCount: 0, subtotalVnd: 0 });
}
