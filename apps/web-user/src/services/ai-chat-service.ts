import type {
  AiFeedbackSignal, AiHistoryMessage, AiStreamEvent, StoredAiConversation
} from "@/lib/ai-chat/types";

export const aiChatEnabled = process.env.NEXT_PUBLIC_AI_CHAT_ENABLED === "true";

const CONVERSATION_KEY = "oma_ai_conversation_v1";
const IDEMPOTENCY_KEY = "oma_ai_idempotency_v1";

export class AiChatError extends Error {
  readonly status: number;
  readonly code: string;
  constructor(status: number, code: string) {
    super(code);
    this.status = status;
    this.code = code;
  }
}

function csrfHeader(): Record<string, string> {
  const token = document.cookie.split("; ").find((part) => part.startsWith("oma_ai_csrf="))?.slice("oma_ai_csrf=".length);
  return { "X-CSRF-Token": token ?? "" };
}

function guestHeader(conv: StoredAiConversation): Record<string, string> {
  return conv.guestToken ? { "x-guest-conversation-token": conv.guestToken } : {};
}

async function parseError(response: Response): Promise<AiChatError> {
  let code = "AI_CHAT_REQUEST_FAILED";
  try {
    const data = (await response.json()) as { code?: unknown };
    if (typeof data.code === "string" && data.code) code = data.code;
  } catch { /* keep the generic code */ }
  return new AiChatError(response.status, code);
}

async function requestJson<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, { credentials: "include", ...init });
  if (!response.ok) throw await parseError(response);
  return (await response.json()) as T;
}

export function loadStoredConversation(): StoredAiConversation | null {
  try {
    const raw = window.localStorage.getItem(CONVERSATION_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<StoredAiConversation>;
    if (typeof parsed.conversationId !== "string" || !parsed.conversationId) return null;
    if (parsed.ownerType !== "guest" && parsed.ownerType !== "customer") return null;
    return {
      conversationId: parsed.conversationId,
      guestToken: typeof parsed.guestToken === "string" && parsed.guestToken ? parsed.guestToken : null,
      ownerType: parsed.ownerType
    };
  } catch { return null; }
}

export function clearStoredConversation() {
  try {
    window.localStorage.removeItem(CONVERSATION_KEY);
    window.localStorage.removeItem(IDEMPOTENCY_KEY);
  } catch { /* storage unavailable */ }
}

function saveStoredConversation(conv: StoredAiConversation) {
  try { window.localStorage.setItem(CONVERSATION_KEY, JSON.stringify(conv)); } catch { /* storage unavailable */ }
}

function idempotencyKey(): string {
  try {
    const existing = window.localStorage.getItem(IDEMPOTENCY_KEY);
    if (existing) return existing;
    const created = crypto.randomUUID();
    window.localStorage.setItem(IDEMPOTENCY_KEY, created);
    return created;
  } catch { return crypto.randomUUID(); }
}

export const aiChatService = {
  /** Primes the oma_ai_csrf cookie the BFF requires for every mutation. */
  async ensureSession(signal?: AbortSignal): Promise<void> {
    const response = await fetch("/api/ai/session", { credentials: "include", signal });
    if (!response.ok) throw await parseError(response);
  },

  async createConversation(): Promise<StoredAiConversation> {
    const data = await requestJson<{ conversation_id: string; owner_type: string; guest_token?: string | null }>(
      "/api/ai/conversations",
      {
        method: "POST",
        headers: { "Content-Type": "application/json", ...csrfHeader(), "Idempotency-Key": idempotencyKey() },
        body: "{}"
      }
    );
    const conv: StoredAiConversation = {
      conversationId: data.conversation_id,
      guestToken: typeof data.guest_token === "string" && data.guest_token ? data.guest_token : null,
      ownerType: data.owner_type === "customer" ? "customer" : "guest"
    };
    saveStoredConversation(conv);
    return conv;
  },

  async getHistory(conv: StoredAiConversation, signal?: AbortSignal): Promise<AiHistoryMessage[]> {
    const data = await requestJson<{ messages?: { role: string; content: string }[] }>(
      `/api/ai/conversations/${encodeURIComponent(conv.conversationId)}`,
      { headers: guestHeader(conv), signal }
    );
    return (data.messages ?? [])
      .filter((m) => (m.role === "user" || m.role === "assistant") && typeof m.content === "string")
      .map((m) => ({ role: m.role as "user" | "assistant", content: m.content }));
  },

  async getConsent(signal?: AbortSignal): Promise<boolean> {
    const data = await requestJson<{ history_use: boolean }>("/api/ai/consent", { signal });
    return data.history_use === true;
  },

  async putConsent(historyUse: boolean): Promise<boolean> {
    const data = await requestJson<{ history_use: boolean }>("/api/ai/consent", {
      method: "PUT",
      headers: { "Content-Type": "application/json", ...csrfHeader() },
      body: JSON.stringify({ history_use: historyUse })
    });
    return data.history_use === true;
  },

  async feedback(conv: StoredAiConversation, productId: string, signal: AiFeedbackSignal): Promise<void> {
    await requestJson("/api/ai/feedback", {
      method: "POST",
      headers: {
        "Content-Type": "application/json", ...csrfHeader(), ...guestHeader(conv),
        "Idempotency-Key": crypto.randomUUID()
      },
      body: JSON.stringify({ conversation_id: conv.conversationId, product_id: productId, signal })
    });
  },

  /** Sends one message and consumes the SSE reply. The advisor composes the
   * full answer first, then streams it in chunks, so callers should show a
   * pending state until the first token event arrives. */
  async sendMessage(
    conv: StoredAiConversation,
    content: string,
    onEvent: (event: AiStreamEvent) => void,
    signal?: AbortSignal
  ): Promise<void> {
    const response = await fetch(`/api/ai/conversations/${encodeURIComponent(conv.conversationId)}/messages`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json", ...csrfHeader(), ...guestHeader(conv) },
      body: JSON.stringify({ content }),
      signal
    });
    if (!response.ok) throw await parseError(response);
    if (!response.body) throw new AiChatError(502, "AI_CHAT_STREAM_MISSING");
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    const dispatch = (chunk: string) => {
      for (const block of chunk.split("\n\n")) {
        const line = block.split("\n").find((part) => part.startsWith("data:"));
        if (!line) continue;
        try {
          const parsed = JSON.parse(line.slice(5).trim()) as AiStreamEvent;
          if (parsed && typeof parsed === "object" && "type" in parsed) onEvent(parsed);
        } catch { /* ignore a malformed event */ }
      }
    };
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const boundary = buffer.lastIndexOf("\n\n");
      if (boundary >= 0) {
        dispatch(buffer.slice(0, boundary));
        buffer = buffer.slice(boundary + 2);
      }
    }
    if (buffer.trim()) dispatch(buffer);
  }
};
