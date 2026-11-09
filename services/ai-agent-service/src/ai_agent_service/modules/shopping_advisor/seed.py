"""Starter knowledge corpus for fixture/dev mode.

Static FAQ entries mirror the topics the AI proposal doc (section 5.3)
assigns to the customer FAQ layer. Product stories are generated from the
fixture catalog descriptions. In production the corpus is maintained
through the ingestion API, not this file.
"""
from __future__ import annotations

from ...integrations.fixtures import FixtureCatalog
from ...platform.knowledge import KnowledgeStore

FAQ_DOCUMENTS = [
    (
        "Hạn sử dụng và cách bảo quản mè xửng",
        "faq",
        "Mè xửng có hạn sử dụng 6 tháng kể từ ngày sản xuất, in trên bao bì. "
        "Bảo quản nơi khô ráo, thoáng mát, tránh ánh nắng trực tiếp và nơi nóng ẩm "
        "để mè không bị chảy dầu. Sau khi mở gói, nên dùng trong 7 ngày và buộc kín miệng túi.",
    ),
    (
        "Khác nhau giữa mè xửng dẻo và mè xửng giòn",
        "faq",
        "Mè xửng dẻo mềm, dai nhẹ, vị ngọt đậm truyền thống, hợp người lớn tuổi và người "
        "thích vị cổ truyền. Mè xửng giòn cứng và rụm hơn, thường ít ngọt hơn, hợp người "
        "trẻ và người thích nhai giòn. Cả hai đều làm từ mè rang, đậu phộng và mạch nha.",
    ),
    (
        "Làng nghề mè xửng Kim Long và Nam Phổ",
        "faq",
        "Mè xửng là đặc sản lâu đời của Huế, gắn với các làng nghề như Kim Long và Nam Phổ. "
        "Nghề làm mè xửng thủ công được truyền qua nhiều thế hệ, với bí quyết ở khâu rang mè "
        "và thử đường bằng nước lạnh để kẹo đạt độ dẻo vừa ý.",
    ),
    (
        "Địa chỉ xưởng và cửa hàng",
        "faq",
        "Xưởng sản xuất đặt tại Hương Thủy, Thừa Thiên Huế. Khách có thể mua trực tiếp tại "
        "cửa hàng giới thiệu sản phẩm ở trung tâm thành phố Huế hoặc đặt hàng qua website.",
    ),
    (
        "Gợi ý chọn quà Tết",
        "faq",
        "Quà Tết nên ưu tiên hộp quà có bao bì trang nhã, hạn sử dụng dài và hương vị "
        "truyền thống hợp người lớn tuổi. Hộp quà Chuyện Huế kết hợp mè xửng dẻo, mè xửng "
        "giòn và trà sen, kèm thiệp, là lựa chọn phổ biến cho ngân sách khoảng 150.000đ đến 200.000đ.",
    ),
]


async def seed_knowledge(store: KnowledgeStore, catalog: FixtureCatalog | None = None) -> int:
    total = 0
    for title, source, text in FAQ_DOCUMENTS:
        total += await store.ingest(title, source, text)
    if catalog is not None:
        for product in catalog.popular(50):
            text = f"{product.name}. {product.description} Giá bán {product.price_vnd} đồng."
            total += await store.ingest(
                f"Sản phẩm: {product.name}", "catalog-fixture", text, product_id=product.id
            )
    return total
