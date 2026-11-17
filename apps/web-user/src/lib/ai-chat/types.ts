export interface AiProductCard {
  id: string;
  sku: string | null;
  name: string;
  priceVnd: number | null;
  description: string | null;
  stock: number | null;
}

export type AiFeedbackSignal = "helpful" | "not_helpful" | "clicked" | "dismissed";

export interface StoredAiConversation {
  conversationId: string;
  guestToken: string | null;
  ownerType: "guest" | "customer";
}

export interface AiHistoryMessage {
  role: "user" | "assistant";
  content: string;
}

export type AiStreamEvent =
  | { type: "token"; text: string }
  | { type: "product_cards"; cards: unknown[] }
  | { type: "citations"; items: string[] }
  | { type: "done"; intent?: string; fallback_url?: string }
  | { type: "error"; code: string; message: string; fallback_url?: string };

/** Normalizes one raw card from the advisor's product_cards event. Both card
 * sources (shopping tools and basket recommendations) emit the same field
 * names: id, sku, name, price_vnd, description, and optionally stock. */
export function normalizeAiProductCard(raw: unknown): AiProductCard | null {
  if (typeof raw !== "object" || raw === null) return null;
  const card = raw as Record<string, unknown>;
  if (typeof card.id !== "string" || !card.id || typeof card.name !== "string" || !card.name) return null;
  return {
    id: card.id,
    sku: typeof card.sku === "string" && card.sku ? card.sku : null,
    name: card.name,
    priceVnd: typeof card.price_vnd === "number" && Number.isSafeInteger(card.price_vnd) && card.price_vnd >= 0
      ? card.price_vnd : null,
    description: typeof card.description === "string" && card.description ? card.description : null,
    stock: typeof card.stock === "number" && Number.isSafeInteger(card.stock) && card.stock >= 0 ? card.stock : null
  };
}
