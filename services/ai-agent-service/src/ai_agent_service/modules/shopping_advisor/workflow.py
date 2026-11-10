"""Shopping advisor workflow — a deterministic workflow with a controlled LLM.

Step order is owned by this code, not by the model:
  route intent -> (FAQ | order status | shipping fee | basket | shopping)
  shopping: extract needs -> clarify if critical slots missing (max 2 rounds)
            -> search products (tool) -> MANDATORY stock check (tool)
            -> compose reply from grounded tool data -> grounding post-check.

The LLM is used for language understanding and phrasing only. Prices, stock
and statuses in a reply must trace to tool results of the same turn.
"""
from __future__ import annotations

import re
from collections.abc import AsyncIterator
from urllib.parse import urlencode

from ...config import Settings
from ...integrations.models import Product
from ...observability import BUDGET_GATE, CHAT_TURNS
from ...platform.auth import Principal
from ...platform.budget_evidence import parse_budget_evidence, resolve_budget
from ...platform.conversations import Conversation, ConversationStore
from ...platform.eligibility import (
    OUT_OF_STOCK,
    ProductEligibility,
    gateway_stock_checker,
)
from ...platform.gateway import ToolContext, ToolError, ToolGateway
from ...platform.knowledge import KnowledgeStore
from ...platform.runtime import BudgetExceeded, TurnState
from ...providers.llm import LLMRequest, Message, ProviderError, ToolSpec

RULES = (
    "Bạn là trợ lý mua sắm của cửa hàng đặc sản OCOP Huế. Trả lời bằng tiếng Việt, "
    "ngắn gọn, thân thiện. Chỉ dùng thông tin trong mục [GROUNDING] cho giá, tồn kho, "
    "trạng thái đơn và phí vận chuyển; tuyệt đối không bịa sản phẩm, giá hay khuyến mãi. "
    "Không tự thêm giỏ hàng hay đặt đơn cho khách."
)

EXTRACT_SPEC = ToolSpec(
    name="extract_needs",
    description=(
        "Extract shopping needs (budget_vnd, recipient, occasion, tastes) "
        "from the conversation. Report budget_vnd EXACTLY as the user stated "
        "it, converting only with the unit the user wrote (k/nghin = 1000, "
        "trieu = 1000000, ty = 1000000000). Never round, repair or substitute "
        "a more plausible amount; if no amount is stated, omit budget_vnd."
    ),
    input_schema={
        "type": "object",
        "properties": {
            "budget_vnd": {"type": "integer"},
            "recipient": {"type": "string"},
            "occasion": {"type": "string"},
            "tastes": {"type": "array", "items": {"type": "string"}},
        },
    },
)

FAQ_HINTS = (
    "hạn sử dụng", "hạn dùng", "bảo quản", "khác nhau", "khác gì",
    "địa chỉ", "xưởng ở đâu", "làng nghề", "để được bao lâu",
)
ORDER_HINTS = ("đơn hàng", "đơn của", "đơn đâu", "giao chưa", "trạng thái đơn", "theo dõi đơn")
SHIPPING_HINTS = ("phí ship", "phí vận chuyển", "ship bao nhiêu", "phí giao", "vận chuyển bao nhiêu")
BASKET_HINTS = ("mua kèm", "đi kèm", "kèm gì", "gợi ý thêm")

ORDER_CODE_RE = re.compile(r"\b[A-Z]{2}\d{3,}\b")
SKU_RE = re.compile(r"\b[A-Z]{2,}-[A-Z]+-\d+\b")
WEIGHT_RE = re.compile(r"(\d+(?:[.,]\d+)?)\s*(kg|g|gram)\b", re.IGNORECASE)
MONEY_RE = re.compile(r"(\d{1,3}(?:[.,]\d{3})+|\d{4,})(?=\s*(?:đ|₫|vnd|vnđ))", re.IGNORECASE)
MONEY_K_RE = re.compile(r"(\d+)\s*k\b", re.IGNORECASE)
PROVINCES = (
    "huế", "hà nội", "đà nẵng", "hồ chí minh", "sài gòn", "cần thơ", "hải phòng",
    "quảng nam", "quảng ngãi", "bình định", "khánh hòa", "nghệ an", "thanh hóa",
    "gia lai", "lâm đồng", "đắk lắk",
)


def format_vnd(amount: int) -> str:
    return f"{amount:,}".replace(",", ".") + "đ"


def extract_money(text: str) -> set[int]:
    amounts: set[int] = set()
    for match in MONEY_RE.finditer(text):
        amounts.add(int(match.group(1).replace(".", "").replace(",", "")))
    for match in MONEY_K_RE.finditer(text):
        amounts.add(int(match.group(1)) * 1000)
    return amounts


class _EphemeralMessages:
    """Store proxy: delegates everything except message persistence."""

    def __init__(self, inner: ConversationStore):
        self._inner = inner

    def __getattr__(self, name):
        return getattr(self._inner, name)

    def add_message(self, conversation_id, role, content, metadata=None):
        return None

    def list_messages(self, conversation_id):
        return []


def _filter_url(slots: dict) -> str:
    params = {}
    if slots.get("budget_vnd"):
        params["ngan-sach"] = slots["budget_vnd"]
    if slots.get("recipient"):
        params["nguoi-nhan"] = slots["recipient"]
    if slots.get("occasion"):
        params["dip"] = slots["occasion"]
    return "/danh-muc" + ("?" + urlencode(params) if params else "")


class ShoppingAdvisor:
    def __init__(self, llm, gateway: ToolGateway, knowledge: KnowledgeStore,
                 settings: Settings, recommendations=None, eligibility=None):
        self.llm = llm
        self.gateway = gateway
        self.knowledge = knowledge
        self.settings = settings
        self.recommendations = recommendations
        self.eligibility = eligibility or ProductEligibility()

    # -- public entry ------------------------------------------------------
    async def turn_events(
        self,
        store: ConversationStore,
        conv: Conversation,
        principal: Principal,
        user_text: str,
        authorization: str | None = None,
    ) -> AsyncIterator[dict]:
        # Consent (chat-history use): a logged-in customer who has not given
        # it chats fully functionally, but their messages are not persisted —
        # the session keeps only its working slots (see _EphemeralMessages).
        if principal.customer_id and not store.get_history_consent(principal.customer_id):
            store = _EphemeralMessages(store)
        ctx = ToolContext(customer_id=principal.customer_id, authorization=authorization)
        state = TurnState(
            max_steps=self.settings.max_steps, max_tool_calls=self.settings.max_tool_calls
        )
        store.add_message(conv.id, "user", user_text)
        try:
            async for event in self._dispatch(store, conv, principal, ctx, state, user_text):
                yield event
        except (ProviderError, TimeoutError):
            CHAT_TURNS.labels(outcome="llm_error").inc()
            yield {
                "type": "error",
                "code": "llm_unavailable",
                "message": "Dạ, hệ thống tư vấn đang bận. Anh/chị xem tạm danh mục sản phẩm nhé ạ.",
                "fallback_url": _filter_url(store.get_slots(conv.id)),
            }
        except (ToolError, BudgetExceeded):
            CHAT_TURNS.labels(outcome="tool_error").inc()
            yield {
                "type": "error",
                "code": "tool_unavailable",
                "message": "Dạ, em chưa tra cứu được dữ liệu lúc này. Anh/chị dùng bộ lọc sản phẩm giúp em nhé ạ.",
                "fallback_url": _filter_url(store.get_slots(conv.id)),
            }

    # -- routing -----------------------------------------------------------
    async def _dispatch(self, store, conv, principal, ctx, state, user_text):
        low = user_text.lower()
        if any(hint in low for hint in FAQ_HINTS):
            results = await self.knowledge.search(user_text, 3)
            if results:
                async for event in self._faq_reply(store, conv, user_text, results):
                    yield event
                return
        code_match = ORDER_CODE_RE.search(user_text.upper())
        if code_match or any(hint in low for hint in ORDER_HINTS):
            async for event in self._order_reply(store, conv, principal, ctx, state,
                                                 user_text, code_match):
                yield event
            return
        if any(hint in low for hint in SHIPPING_HINTS):
            async for event in self._shipping_reply(store, conv, ctx, state, user_text):
                yield event
            return
        if any(hint in low for hint in BASKET_HINTS) and self.recommendations is not None:
            async for event in self._basket_reply(store, conv, user_text):
                yield event
            return
        async for event in self._shopping(store, conv, principal, ctx, state, user_text):
            yield event

    # -- shared helpers ----------------------------------------------------
    async def _compose(self, store, conv, grounding: str, user_text: str,
                       extra_rules: str = "") -> str:
        history = [
            Message(role=m.role, content=m.content)
            for m in store.list_messages(conv.id)
            if m.role in ("user", "assistant")
        ][-6:]
        if not history or history[-1].content != user_text:
            history.append(Message(role="user", content=user_text))
        request = LLMRequest(
            messages=history,
            system_prompt=RULES + extra_rules + "\n[GROUNDING]\n" + grounding,
        )
        response = await self.llm.generate(request)
        return response.content

    def _check_grounding(self, reply: str, grounding: str, extra_allowed: set[int]) -> bool:
        allowed = extract_money(grounding) | extra_allowed
        return all(amount in allowed for amount in extract_money(reply))

    async def _emit_reply(self, store, conv, reply: str, intent: str, extra: dict | None = None):
        store.add_message(conv.id, "assistant", reply)
        CHAT_TURNS.labels(outcome=intent).inc()
        for i in range(0, len(reply), 48):
            yield {"type": "token", "text": reply[i:i + 48]}
        done = {"type": "done", "intent": intent}
        if extra:
            done.update(extra)
        yield done

    # -- FAQ ---------------------------------------------------------------
    async def _faq_reply(self, store, conv, user_text, results):
        grounding = "\n".join(f"- {chunk.content}" for chunk in results)
        reply = await self._compose(
            store, conv, grounding, user_text,
            extra_rules=(
                " Trình bày đủ các điều kiện, hướng dẫn liên quan trực tiếp "
                "tới câu hỏi có trong [GROUNDING]; không bỏ sót điều kiện nào."
            ),
        )
        if not self._check_grounding(reply, grounding, set()):
            reply = results[0].content
        citations = sorted({chunk.document_title for chunk in results})
        store.add_message(conv.id, "assistant", reply)
        CHAT_TURNS.labels(outcome="faq").inc()
        for i in range(0, len(reply), 48):
            yield {"type": "token", "text": reply[i:i + 48]}
        yield {"type": "citations", "items": citations}
        yield {"type": "done", "intent": "faq"}

    # -- order status --------------------------------------------------------
    async def _order_reply(self, store, conv, principal, ctx, state, user_text, code_match):
        if principal.is_guest:
            reply = (
                "Dạ, tra cứu trạng thái đơn hàng cần anh/chị đăng nhập bằng đúng tài khoản "
                "đã đặt đơn. Anh/chị đăng nhập rồi gửi em mã đơn, em kiểm tra ngay ạ."
            )
            async for event in self._emit_reply(store, conv, reply, "order_status",
                                                {"needs_login": True}):
                yield event
            return
        code = code_match.group(0) if code_match else None
        if not code:
            reply = "Dạ, anh/chị cho em xin mã đơn hàng (ví dụ DH1001) để em tra cứu ạ."
            async for event in self._emit_reply(store, conv, reply, "order_status"):
                yield event
            return
        result = await state.call_tool(
            self.gateway, "get_order_status", {"order_id": code}, ctx
        )
        if not result.get("found"):
            reply = (
                f"Dạ, em không tìm thấy đơn hàng với mã {code} trong tài khoản của anh/chị. "
                "Anh/chị kiểm tra lại mã đơn giúp em nhé ạ."
            )
            async for event in self._emit_reply(store, conv, reply, "order_status"):
                yield event
            return
        eta = f"; dự kiến giao: {result['eta']}" if result.get("eta") else ""
        grounding = (
            f"- Đơn {result['order_id']}: trạng thái {result['status']}; "
            f"{result['summary']}{eta}"
        )
        reply = await self._compose(store, conv, grounding, user_text)
        if not self._check_grounding(reply, grounding, set()):
            reply = grounding.lstrip("- ")
        async for event in self._emit_reply(store, conv, reply, "order_status"):
            yield event

    # -- shipping fee --------------------------------------------------------
    async def _shipping_reply(self, store, conv, ctx, state, user_text):
        low = user_text.lower()
        province = next((p for p in PROVINCES if p in low), None)
        if not province:
            reply = "Dạ, anh/chị cho em biết tỉnh/thành nhận hàng để em tính phí vận chuyển ạ."
            async for event in self._emit_reply(store, conv, reply, "shipping_fee"):
                yield event
            return
        weight = 500
        match = WEIGHT_RE.search(low)
        if match:
            value = float(match.group(1).replace(",", "."))
            weight = int(value * 1000) if match.group(2).lower() == "kg" else int(value)
        result = await state.call_tool(
            self.gateway,
            "estimate_shipping",
            {"province": province, "weight_grams": weight},
            ctx,
        )
        grounding = (
            f"- Phí vận chuyển tới {province}: {format_vnd(result['fee_vnd'])}; "
            f"thời gian dự kiến {result['eta_days']} ngày qua {result['carrier']} "
            f"(ước tính cho gói hàng {weight}g)"
        )
        reply = await self._compose(store, conv, grounding, user_text)
        if not self._check_grounding(reply, grounding, set()):
            reply = grounding.lstrip("- ")
        async for event in self._emit_reply(store, conv, reply, "shipping_fee"):
            yield event

    # -- basket recommendations ---------------------------------------------
    async def _basket_reply(self, store, conv, user_text):
        skus = SKU_RE.findall(user_text.upper())
        if not skus:
            reply = (
                "Dạ, anh/chị đang xem hoặc đã chọn sản phẩm nào rồi ạ? "
                "Cho em xin mã sản phẩm (SKU), em gợi ý món mua kèm phù hợp ngay."
            )
            async for event in self._emit_reply(store, conv, reply, "basket"):
                yield event
            return
        slots = store.get_slots(conv.id)
        products = await self.recommendations.basket(
            skus, 4, budget_vnd=slots.get("budget_vnd")
        )
        if not products:
            reply = (
                "Dạ, các món thường mua kèm với sản phẩm này hiện chưa có món nào "
                "còn hàng phù hợp. Anh/chị xem thêm trong danh mục giúp em nhé ạ."
            )
            async for event in self._emit_reply(store, conv, reply, "basket"):
                yield event
            return
        cards = [
            {**p.public_dict(), "url": f"/san-pham/{p.id}"} for p in products
        ]
        names = ", ".join(p.name for p in products)
        reply = f"Dạ, với sản phẩm anh/chị đã chọn, em gợi ý mua kèm: {names}. Đây là các món thường được chọn cùng nhau ạ."
        store.add_message(conv.id, "assistant", reply)
        CHAT_TURNS.labels(outcome="basket").inc()
        for i in range(0, len(reply), 48):
            yield {"type": "token", "text": reply[i:i + 48]}
        yield {"type": "product_cards", "cards": cards}
        yield {"type": "done", "intent": "basket"}

    # -- main shopping path ---------------------------------------------------
    async def _shopping(self, store, conv, principal, ctx, state, user_text):
        slots = store.get_slots(conv.id)
        history_users = [
            m.content for m in store.list_messages(conv.id) if m.role == "user"
        ]
        if not history_users:
            history_users = [user_text]
        extraction = await self.llm.generate(
            LLMRequest(
                messages=[Message(role="user", content="\n".join(history_users))],
                tools=[EXTRACT_SPEC],
            )
        )
        model_budget = None
        if extraction.tool_calls:
            new_slots = extraction.tool_calls[0].arguments
            # The model's budget is a claim, not a fact: it is set aside
            # here and decided by the evidence gate below (docs ch. 14).
            # (An injected "giá ... là 1.000đ" must never become the budget.)
            raw_budget = new_slots.pop("budget_vnd", None)
            if raw_budget is not None:
                try:
                    model_budget = int(raw_budget)
                except (TypeError, ValueError):
                    model_budget = None
            for key, value in new_slots.items():
                if value in (None, "", []):
                    continue
                if key == "tastes" and isinstance(slots.get("tastes"), list):
                    slots["tastes"] = sorted(set(slots["tastes"]) | set(value))
                else:
                    slots[key] = value
        evidence = parse_budget_evidence(history_users)
        budget, decision = resolve_budget(evidence, model_budget, history_users)
        BUDGET_GATE.labels(decision=decision).inc()
        if budget is not None:
            slots["budget_vnd"] = budget
        else:
            slots.pop("budget_vnd", None)
        slots["_budget_source"] = {
            "decision": decision,
            "status": evidence.status,
            "message_index": evidence.message_index,
            "text": evidence.source_text,
        }
        store.set_slots(conv.id, slots)

        missing = []
        if not slots.get("budget_vnd"):
            missing.append("ngân sách dự kiến khoảng bao nhiêu")
        if not slots.get("recipient"):
            missing.append("món quà này tặng ai")
        clarifications = int(slots.get("_clarifications", 0))
        if missing and clarifications < 2:
            slots["_clarifications"] = clarifications + 1
            store.set_slots(conv.id, slots)
            known = f" quà {slots['occasion']}" if slots.get("occasion") else ""
            reply = (
                f"Dạ, em sẵn sàng tư vấn{known} cho anh/chị. Để gợi ý đúng ý nhất, "
                f"anh/chị cho em biết thêm: {' và '.join(missing)} ạ?"
            )
            async for event in self._emit_reply(store, conv, reply, "clarification"):
                yield event
            return

        query = " ".join(
            [user_text]
            + list(slots.get("tastes") or [])
            + ([slots["occasion"]] if slots.get("occasion") else [])
        )
        result = await state.call_tool(
            self.gateway,
            "search_products",
            {
                "query": query,
                "max_price_vnd": slots.get("budget_vnd"),
                "limit": 5,
            },
            ctx,
        )
        raw_products = result["products"]
        # Every Product Card passes the unified eligibility layer (docs
        # ch. 13): the search tool already guarantees published + budget,
        # the layer re-checks both and verifies stock in parallel,
        # fail-closed per SKU.
        candidates = [
            Product(
                id=p["id"], sku=p["sku"], name=p["name"],
                price_vnd=p["price_vnd"], status="published",
                description=p.get("description", ""),
            )
            for p in raw_products
        ]
        check = gateway_stock_checker(self.gateway, ctx, state)
        outcome = await self.eligibility.filter(
            candidates,
            budget_vnd=slots.get("budget_vnd"),
            check_stock=check,
            source="shopping",
            fatal=(BudgetExceeded,),
        )
        available = [
            {
                "id": e.product.id, "sku": e.product.sku, "name": e.product.name,
                "price_vnd": e.product.price_vnd,
                "description": e.product.description, "stock": e.available,
            }
            for e in outcome.eligible
        ]
        # Only verified out-of-stock products are named as such; SKUs whose
        # stock could not be verified are dropped silently, never labelled
        # "tạm hết hàng".
        sold_out = [
            {"name": p.name, "price_vnd": p.price_vnd}
            for p in outcome.rejected_products(candidates, OUT_OF_STOCK)
        ]

        citations: list[str] = []
        context_lines: list[str] = []
        if slots.get("occasion"):
            chunks = await self.knowledge.search(
                f"quà {slots['occasion']} tặng {slots.get('recipient', '')}", 2
            )
            for chunk in chunks:
                context_lines.append(f"- Gợi ý thêm: {chunk.content}")
                if chunk.document_title not in citations:
                    citations.append(chunk.document_title)

        if not available:
            budget_text = (
                f" trong ngân sách {format_vnd(slots['budget_vnd'])}"
                if slots.get("budget_vnd")
                else ""
            )
            reply = (
                f"Dạ, hiện em chưa tìm thấy sản phẩm phù hợp còn hàng{budget_text}. "
                "Anh/chị xem danh mục đã lọc sẵn theo nhu cầu của mình nhé ạ."
            )
            store.add_message(conv.id, "assistant", reply)
            CHAT_TURNS.labels(outcome="shopping_fallback").inc()
            for i in range(0, len(reply), 48):
                yield {"type": "token", "text": reply[i:i + 48]}
            yield {
                "type": "done",
                "intent": "shopping",
                "fallback_url": _filter_url(slots),
            }
            return

        lines = [
            f"- {p['name']} — {format_vnd(p['price_vnd'])} — còn {p['stock']} sản phẩm"
            for p in available
        ]
        lines += [
            f"- {p['name']} — {format_vnd(p['price_vnd'])} — tạm hết hàng" for p in sold_out
        ]
        grounding = "\n".join(lines + context_lines)
        reply = await self._compose(store, conv, grounding, user_text)
        allowed = {p["price_vnd"] for p in raw_products}
        if slots.get("budget_vnd"):
            allowed.add(slots["budget_vnd"])
        if not self._check_grounding(reply, grounding, allowed):
            reply = (
                "Dạ, em gợi ý các sản phẩm còn hàng và phù hợp nhu cầu của anh/chị:\n"
                + "\n".join(lines[: len(available)])
                + "\nAnh/chị muốn xem kỹ sản phẩm nào ạ?"
            )
            CHAT_TURNS.labels(outcome="grounding_fallback").inc()
        cards = [
            {
                **{k: p[k] for k in ("id", "sku", "name", "price_vnd", "description")},
                "stock": p["stock"],
                "url": f"/san-pham/{p['id']}",
            }
            for p in available
        ]
        store.add_message(
            conv.id, "assistant", reply,
            metadata={"product_ids": [p["id"] for p in available]},
        )
        CHAT_TURNS.labels(outcome="shopping").inc()
        for i in range(0, len(reply), 48):
            yield {"type": "token", "text": reply[i:i + 48]}
        yield {"type": "product_cards", "cards": cards}
        if citations:
            yield {"type": "citations", "items": citations}
        yield {"type": "done", "intent": "shopping"}
