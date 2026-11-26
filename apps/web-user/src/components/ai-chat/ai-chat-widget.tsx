"use client";

import Link from "next/link";
import {
  useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent
} from "react";
import { useAuth } from "@/components/auth/auth-provider";
import { useCartSummary } from "@/components/cart/cart-summary-provider";
import { cartService } from "@/services/cart-service";
import { AiChatError, aiChatEnabled, aiChatService, clearStoredConversation, loadStoredConversation } from "@/services/ai-chat-service";
import { normalizeAiProductCard } from "@/lib/ai-chat/types";
import type { AiProductCard, StoredAiConversation } from "@/lib/ai-chat/types";
import styles from "./ai-chat.module.css";

interface UiMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  status: "pending" | "streaming" | "completed" | "interrupted";
  cards: AiProductCard[];
  citations: string[];
  fallback: boolean;
  errorText: string | null;
}

interface ChatState {
  conv: StoredAiConversation;
  messages: UiMessage[];
  consent: boolean | null;
}

const STARTER_CHIPS = [
  { label: "🎁 Chọn quà biếu", message: "Tôi muốn chọn quà biếu, bạn gợi ý giúp tôi." },
  { label: "🍬 Mè xửng ít ngọt", message: "Mè xửng nào ít ngọt?" },
  { label: "🍵 Tư vấn trà sen", message: "Tư vấn trà sen giúp tôi." },
  { label: "🚚 Hỏi phí giao hàng", message: "Phí giao hàng là bao nhiêu?" }
];

const RATE_LIMIT_TEXT = "Bạn đang gửi tin hơi nhanh. Vui lòng thử lại sau ít phút.";
const GENERIC_ERROR_TEXT = "Câu trả lời bị gián đoạn. Bạn thử lại giúp Ô Mạ nhé.";

function formatVnd(amount: number) {
  return new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND" }).format(amount);
}

export function AiChatWidget() {
  const auth = useAuth();
  const [open, setOpen] = useState(false);
  const [view, setView] = useState<"chat" | "privacy">("chat");
  const [menuOpen, setMenuOpen] = useState(false);
  const [chat, setChat] = useState<ChatState | null>(null);
  const [bootFailed, setBootFailed] = useState(false);
  const [bootAttempt, setBootAttempt] = useState(0);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [consentSaving, setConsentSaving] = useState(false);
  const [showScrollDown, setShowScrollDown] = useState(false);
  const bootKey = useRef("");
  const listRef = useRef<HTMLDivElement>(null);
  const nearBottom = useRef(true);
  const lastUserText = useRef("");
  const current = useRef({ chat, sending });
  useLayoutEffect(() => { current.current = { chat, sending }; });

  const actor = auth.isCustomer ? (auth.profile?.principalId ?? "customer") : null;

  useEffect(() => {
    if (!open || !aiChatEnabled || auth.isLoading || auth.isEmployee) return;
    const key = `${actor ?? "guest"}:${bootAttempt}`;
    if (bootKey.current === key) return;
    bootKey.current = key;
    const controller = new AbortController();
    let active = true;
    void (async () => {
      try {
        await aiChatService.ensureSession(controller.signal);
        let conv = loadStoredConversation();
        if (conv && ((actor && conv.ownerType === "guest") || (!actor && conv.ownerType === "customer"))) {
          conv = null;
        }
        let history: { role: "user" | "assistant"; content: string }[] = [];
        if (conv) {
          try { history = await aiChatService.getHistory(conv, controller.signal); }
          catch { conv = null; }
        }
        if (!conv) conv = await aiChatService.createConversation();
        let consent: boolean | null = null;
        if (actor) {
          try { consent = await aiChatService.getConsent(controller.signal); } catch { consent = null; }
        }
        if (!active) return;
        setBootFailed(false);
        setChat({
          conv,
          consent,
          messages: history.map((m, index) => ({
            id: `history-${index}`, role: m.role, content: m.content, status: "completed" as const,
            cards: [], citations: [], fallback: false, errorText: null
          }))
        });
      } catch {
        if (active) { bootKey.current = ""; setBootFailed(true); }
      }
    })();
    return () => { active = false; controller.abort(); };
  }, [open, actor, auth.isLoading, auth.isEmployee, bootAttempt]);

  useEffect(() => {
    const node = listRef.current;
    if (node && nearBottom.current) node.scrollTop = node.scrollHeight;
  }, [chat?.messages, view]);

  const patchAssistant = useCallback((id: string, patch: (m: UiMessage) => UiMessage) => {
    setChat((prev) => prev ? {
      ...prev,
      messages: prev.messages.map((m) => (m.id === id ? patch(m) : m))
    } : prev);
  }, []);

  const send = useCallback(async (text: string, replaceInterruptedId?: string) => {
    const state = current.current;
    if (!state.chat || state.sending) return;
    const conv = state.chat.conv;
    lastUserText.current = text;
    setSending(true);
    const assistantId = crypto.randomUUID();
    setChat((prev) => {
      if (!prev) return prev;
      const kept = replaceInterruptedId
        ? prev.messages.filter((m) => m.id !== replaceInterruptedId)
        : prev.messages;
      const appended: UiMessage[] = replaceInterruptedId ? [] : [{
        id: crypto.randomUUID(), role: "user", content: text, status: "completed",
        cards: [], citations: [], fallback: false, errorText: null
      }];
      return {
        ...prev,
        messages: [...kept, ...appended, {
          id: assistantId, role: "assistant", content: "", status: "pending",
          cards: [], citations: [], fallback: false, errorText: null
        }]
      };
    });
    nearBottom.current = true;
    let settled = false;
    try {
      await aiChatService.sendMessage(conv, text, (event) => {
        if (event.type === "token") {
          patchAssistant(assistantId, (m) => ({ ...m, status: "streaming", content: m.content + event.text }));
        } else if (event.type === "product_cards") {
          const cards = event.cards.map(normalizeAiProductCard).filter((c) => c !== null).slice(0, 6);
          patchAssistant(assistantId, (m) => ({ ...m, cards }));
        } else if (event.type === "citations") {
          patchAssistant(assistantId, (m) => ({ ...m, citations: event.items }));
        } else if (event.type === "done") {
          settled = true;
          patchAssistant(assistantId, (m) => ({
            ...m, status: "completed", fallback: m.fallback || Boolean(event.fallback_url)
          }));
        } else if (event.type === "error") {
          settled = true;
          patchAssistant(assistantId, (m) => ({
            ...m, status: "interrupted", errorText: event.message,
            fallback: m.fallback || Boolean(event.fallback_url)
          }));
        }
      });
      // A stream that ends without a terminal event (the advisor died
      // mid-turn) must not be presented as a completed empty answer.
      if (!settled) {
        patchAssistant(assistantId, (m) => ({
          ...m, status: "interrupted", errorText: GENERIC_ERROR_TEXT
        }));
      }
    } catch (cause) {
      const rateLimited = cause instanceof AiChatError && cause.status === 429;
      patchAssistant(assistantId, (m) => ({
        ...m, status: "interrupted", errorText: rateLimited ? RATE_LIMIT_TEXT : GENERIC_ERROR_TEXT
      }));
    } finally {
      setSending(false);
    }
  }, [patchAssistant]);

  if (!aiChatEnabled || auth.isEmployee) return null;

  const submit = (event: FormEvent) => {
    event.preventDefault();
    const text = input.trim();
    if (!text || sending) return;
    setInput("");
    void send(text);
  };

  const clearConversation = () => {
    clearStoredConversation();
    setMenuOpen(false);
    setChat(null);
    bootKey.current = "";
    setBootAttempt((value) => value + 1);
    setView("chat");
  };

  const toggleConsent = async (value: boolean) => {
    if (!chat || consentSaving) return;
    setConsentSaving(true);
    try {
      const saved = await aiChatService.putConsent(value);
      setChat((prev) => (prev ? { ...prev, consent: saved } : prev));
    } catch { /* keep the previous value; the toggle stays as the server last confirmed */ }
    finally { setConsentSaving(false); }
  };

  return (
    <>
      {!open && (
        <button type="button" className={styles.fab} aria-label="Mở trò chuyện với trợ lý Ô Mạ"
          onClick={() => setOpen(true)}>
          <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5Z"
              stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </button>
      )}
      {open && (
        <section className={styles.panel} role="dialog" aria-label="Trợ lý Ô Mạ">
          <header className={styles.header}>
            <span className={styles.avatar} aria-hidden="true">Ô</span>
            <div className={styles.headerText}>
              <strong>Trợ lý Ô Mạ</strong>
              <small>AI tư vấn đặc sản Huế</small>
            </div>
            <button type="button" className={styles.iconButton} aria-label="Mở menu trợ lý"
              onClick={() => setMenuOpen((value) => !value)}>⋯</button>
            <button type="button" className={styles.iconButton} aria-label="Đóng trò chuyện"
              onClick={() => setOpen(false)}>✕</button>
            {menuOpen && (
              <div className={styles.menu}>
                <button type="button" onClick={() => { setMenuOpen(false); setView("privacy"); }}>
                  Quyền riêng tư &amp; lưu lịch sử
                </button>
                <button type="button" onClick={clearConversation}>Xóa cuộc trò chuyện trên thiết bị này</button>
                <button type="button" onClick={() => setOpen(false)}>Đóng trò chuyện</button>
              </div>
            )}
          </header>

          {view === "privacy" ? (
            <div className={styles.privacyBody}>
              <button type="button" className={styles.backButton} onClick={() => setView("chat")}>← Quay lại trò chuyện</button>
              <h3>Quyền riêng tư của bạn</h3>
              <p>Ô Mạ cần xử lý nội dung bạn gửi để tư vấn. Đừng chia sẻ mật khẩu hoặc thông tin thanh toán trong chat — trợ lý chỉ tư vấn, không tự đặt hàng.</p>
              <p>Cuộc trò chuyện được giữ tối đa 30 ngày với khách đã đăng nhập và 15 ngày với khách vãng lai để trợ lý nhớ ngữ cảnh khi bạn quay lại.</p>
              {actor ? (
                chat?.consent === null || !chat ? (
                  <p>Chưa tải được trạng thái lưu lịch sử của bạn. Bạn mở lại trợ lý để thử lại nhé.</p>
                ) : (
                  <label className={styles.toggleRow}>
                    <input type="checkbox" checked={chat.consent} disabled={consentSaving}
                      onChange={(event) => { void toggleConsent(event.target.checked); }} />
                    <span>
                      <strong>Lưu lịch sử trò chuyện</strong>
                      <small>
                        {chat.consent
                          ? "Đang bật: tin nhắn được lưu để bạn tiếp tục cuộc trò chuyện lần sau và nhận tư vấn sát nhu cầu hơn."
                          : "Đang tắt: tin nhắn không được lưu — mỗi lần mở chat là một cuộc trò chuyện mới. Bạn vẫn trò chuyện bình thường."}
                      </small>
                    </span>
                  </label>
                )
              ) : (
                <p>Bạn đang trò chuyện với tư cách khách. <Link href="/login">Đăng nhập</Link> để lưu lịch sử trò chuyện và tra cứu đơn hàng của bạn.</p>
              )}
            </div>
          ) : (
            <>
              <div className={styles.body} ref={listRef} onScroll={(event) => {
                const node = event.currentTarget;
                nearBottom.current = node.scrollHeight - node.scrollTop - node.clientHeight < 80;
                setShowScrollDown(!nearBottom.current && sending);
              }}>
                {!chat && !bootFailed && (
                  <div className={styles.stateBox}><p>Đang kết nối trợ lý…</p></div>
                )}
                {bootFailed && (
                  <div className={styles.stateBox}>
                    <p>Chưa kết nối được với trợ lý.</p>
                    <button type="button" className={styles.smallButton}
                      onClick={() => { bootKey.current = ""; setBootFailed(false); setBootAttempt((v) => v + 1); }}>
                      Thử lại
                    </button>
                  </div>
                )}
                {chat && chat.messages.length === 0 && (
                  <>
                    <div className={styles.messageRow}>
                      <div className={`${styles.bubble} ${styles.bubbleAssistant}`}>
                        Dạ, Ô Mạ chào bạn! Em có thể giúp bạn chọn đặc sản Huế, tìm quà theo ngân sách và giải đáp thông tin mua hàng. Bạn đang tìm món gì để Ô Mạ tư vấn ạ?
                      </div>
                    </div>
                    <div className={styles.chips}>
                      {STARTER_CHIPS.map((chip) => (
                        <button key={chip.label} type="button" className={styles.chip}
                          onClick={() => { void send(chip.message); }}>{chip.label}</button>
                      ))}
                    </div>
                  </>
                )}
                {chat?.messages.map((message) => (
                  <div key={message.id}
                    className={`${styles.messageRow} ${message.role === "user" ? styles.messageRowUser : ""}`}>
                    {message.role === "user" ? (
                      <div className={`${styles.bubble} ${styles.bubbleUser}`}>{message.content}</div>
                    ) : (
                      <div className={styles.assistantBlock}>
                        {(message.content || message.status === "pending") && (
                          <div className={`${styles.bubble} ${styles.bubbleAssistant}`}>
                            {message.status === "pending" && !message.content ? (
                              <span className={styles.typing} aria-label="Trợ lý đang trả lời"><span /><span /><span /></span>
                            ) : message.content}
                          </div>
                        )}
                        {message.cards.length > 0 && chat && (
                          <div className={styles.cards}>
                            {message.cards.map((card) => (
                              <ChatProductCard key={card.id} card={card} conv={chat.conv} />
                            ))}
                            <span className={styles.trustNote}>Giá &amp; tồn kho lấy trực tiếp từ cửa hàng.</span>
                          </div>
                        )}
                        {message.citations.length > 0 && (
                          <p className={styles.citations}>Nguồn tham khảo: {message.citations.join(" · ")}</p>
                        )}
                        {message.status === "interrupted" && (
                          <>
                            <p className={styles.errorNote}>{message.errorText ?? GENERIC_ERROR_TEXT}</p>
                            <div className={styles.inlineActions}>
                              <button type="button" className={styles.smallButton} disabled={sending}
                                onClick={() => { void send(lastUserText.current, message.id); }}>
                                Thử lại
                              </button>
                              {message.fallback && (
                                <Link className={styles.smallButton} href="/products">Xem danh mục sản phẩm</Link>
                              )}
                            </div>
                          </>
                        )}
                        {message.status === "completed" && message.fallback && (
                          <div className={styles.inlineActions}>
                            <Link className={styles.smallButton} href="/products">Xem danh mục sản phẩm</Link>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                ))}
                {showScrollDown && (
                  <button type="button" className={styles.scrollDown} onClick={() => {
                    nearBottom.current = true;
                    setShowScrollDown(false);
                    const node = listRef.current;
                    if (node) node.scrollTop = node.scrollHeight;
                  }}>↓ Xuống tin mới</button>
                )}
              </div>
              <p className={styles.privacyNote}>Ô Mạ chỉ tư vấn, không tự đặt hàng. Đừng chia sẻ thông tin thanh toán trong chat.</p>
              <form className={styles.composer} onSubmit={submit}>
                <textarea value={input} rows={1} maxLength={2000} aria-label="Nhắn Ô Mạ điều bạn cần"
                  placeholder="Nhắn Ô Mạ điều bạn cần…"
                  onChange={(event) => setInput(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" && !event.shiftKey) {
                      event.preventDefault();
                      const text = input.trim();
                      if (text && !sending) { setInput(""); void send(text); }
                    }
                  }} />
                <button type="submit" className={styles.sendButton} aria-label="Gửi tin nhắn"
                  disabled={sending || !input.trim()}>
                  <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M22 2 11 13M22 2l-7 20-4-9-9-4 20-7Z" stroke="currentColor" strokeWidth="1.8"
                      strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                </button>
              </form>
            </>
          )}
        </section>
      )}
    </>
  );
}

function ChatProductCard({ card, conv }: { card: AiProductCard; conv: StoredAiConversation }) {
  const cart = useCartSummary();
  const [addState, setAddState] = useState<"idle" | "adding" | "added" | "error">("idle");
  const [feedback, setFeedback] = useState<"helpful" | "not_helpful" | null>(null);
  const soldOut = card.stock === 0;

  const addToCart = async () => {
    if (!card.sku || addState === "adding" || addState === "added") return;
    setAddState("adding");
    try {
      const updated = await cartService.addItem({ skuId: card.sku, quantity: 1 });
      cart.updateFromCart(updated);
      setAddState("added");
    } catch { setAddState("error"); }
  };

  const sendFeedback = (signal: "helpful" | "not_helpful" | "clicked") => {
    void aiChatService.feedback(conv, card.id, signal).catch(() => undefined);
  };

  return (
    <article className={styles.card}>
      <div className={styles.cardTile} aria-hidden="true">Ô Mạ</div>
      <div className={styles.cardBody}>
        <span className={styles.cardName}>{card.name}</span>
        {card.description && <span className={styles.cardDesc}>{card.description}</span>}
        <div className={styles.cardMeta}>
          {card.priceVnd !== null && <span className={styles.cardPrice}>{formatVnd(card.priceVnd)}</span>}
          {card.stock !== null && (soldOut
            ? <span className={styles.stockOut}>Tạm hết hàng</span>
            : <span className={styles.stockOk}>Còn {card.stock} sản phẩm</span>)}
        </div>
        <div className={styles.cardActions}>
          {addState === "added" ? (
            <Link className={styles.addedLink} href="/cart">✓ Đã thêm — Xem giỏ hàng</Link>
          ) : (
            <button type="button" className={styles.addButton} disabled={soldOut || !card.sku || addState === "adding"}
              onClick={() => { void addToCart(); }}>
              {addState === "adding" ? "Đang thêm…" : addState === "error" ? "Thử thêm lại" : "Thêm vào giỏ"}
            </button>
          )}
          <Link className={styles.detailLink} href={`/products?q=${encodeURIComponent(card.name)}`}
            onClick={() => sendFeedback("clicked")}>
            Xem chi tiết
          </Link>
          <span className={styles.feedbackGroup}>
            {feedback ? (
              <span className={styles.feedbackThanks}>Đã ghi nhận</span>
            ) : (
              <>
                <button type="button" className={styles.feedbackButton} aria-label="Gợi ý hữu ích"
                  onClick={() => { setFeedback("helpful"); sendFeedback("helpful"); }}>👍</button>
                <button type="button" className={styles.feedbackButton} aria-label="Gợi ý chưa hữu ích"
                  onClick={() => { setFeedback("not_helpful"); sendFeedback("not_helpful"); }}>👎</button>
              </>
            )}
          </span>
        </div>
      </div>
    </article>
  );
}
