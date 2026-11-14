"""FAQ coverage guard (docs ch. 12/14, dot 1.7).

Two layers: (1) the fact declarations in seed.py stay grounded in the
document text they describe; (2) the workflow's deterministic repair
appends a source-backed sentence when a composed reply drops a
critical fact the question asks about — and only then.
"""
from ai_agent_service.config import Settings
from ai_agent_service.modules.shopping_advisor.seed import FAQ_DOCUMENTS, FAQ_FACTS
from ai_agent_service.modules.shopping_advisor.workflow import ShoppingAdvisor
from ai_agent_service.platform.auth import Principal
from ai_agent_service.platform.conversations import MemoryConversationStore
from ai_agent_service.platform.knowledge import KnowledgeChunk
from ai_agent_service.providers.llm import LLMResponse

STORAGE_TITLE = "Hạn sử dụng và cách bảo quản mè xửng"
STORAGE_TEXT = next(text for title, _src, text in FAQ_DOCUMENTS if title == STORAGE_TITLE)


def test_facts_are_grounded_in_their_document():
    docs = {title: text for title, _src, text in FAQ_DOCUMENTS}
    for title, entry in FAQ_FACTS.items():
        assert title in docs, f"facts declared for unknown document {title!r}"
        for fact in entry["facts"]:
            # The canonical phrase of every fact must literally appear
            # in the document: a fact the source does not state is a
            # fabrication channel, and a source edited without its
            # facts fails here instead of silently rotting.
            assert fact["any_of"][0].lower() in docs[title].lower(), fact["fact_id"]
            assert fact["append"].rstrip(".").lower() in docs[title].lower(), fact["fact_id"]


class _StubLLM:
    def __init__(self, text: str):
        self._text = text

    async def generate(self, request):
        return LLMResponse(content=self._text)


class _StubKnowledge:
    async def search(self, query, limit=5):
        return [KnowledgeChunk(content=STORAGE_TEXT, document_title=STORAGE_TITLE)]


async def _ask(llm_text: str, question: str) -> str:
    advisor = ShoppingAdvisor(
        llm=_StubLLM(llm_text),
        gateway=None,
        knowledge=_StubKnowledge(),
        settings=Settings(),
    )
    store = MemoryConversationStore()
    conv = store.create("guest", None, 15)
    principal = Principal(customer_id=None, is_guest=True)
    text = ""
    async for event in advisor.turn_events(store, conv, principal, question):
        if event["type"] == "token":
            text += event["text"]
    return text


async def test_guard_appends_missing_storage_environment():
    # The R02 shape: the model answers two facts and drops the third.
    reply = await _ask(
        "Dạ, sau khi mở gói anh/chị buộc kín miệng túi và dùng trong 7 ngày ạ.",
        "Cách bảo quản mè xửng sau khi mở gói?",
    )
    assert "khô ráo" in reply
    assert "Lưu ý thêm từ cửa hàng" in reply


async def test_guard_noop_when_reply_is_complete():
    reply = await _ask(
        "Dạ, bảo quản nơi khô ráo, thoáng mát; sau khi mở gói dùng trong 7 ngày ạ.",
        "Cách bảo quản mè xửng sau khi mở gói?",
    )
    assert "Lưu ý thêm từ cửa hàng" not in reply


async def test_guard_only_repairs_the_asked_topic():
    # A storage answer must not grow a shelf-life sentence the customer
    # did not ask about (the chunk-dump failure in miniature).
    reply = await _ask(
        "Dạ, bảo quản nơi khô ráo; sau khi mở gói dùng trong 7 ngày ạ.",
        "Cách bảo quản mè xửng sau khi mở gói?",
    )
    assert "6 tháng" not in reply
