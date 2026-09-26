"use client";

import Link from "next/link";
import Image from "next/image";
import { FormEvent, useState } from "react";

const HERO_IMAGE = "https://lh3.googleusercontent.com/aida-public/AB6AXuBgDTnNDpDqWayHGmfgdpMZ7xrIpepsmqvZgohQd66eSgDGKr7krs5KknIa7b4nRGLRpZ1FQt5ye-7UfbqbiPvJvJk1WwXIrbwaCwIrTdJFFNr3ykdI5S8DWq6OdVN3-EKG57DFKRSrBew_wdq8j-i8jLch_hKQsGgnqbqtyN9t9H5K_e3TDwZ3WbFQPlBv1Hz4dJd1EJUaLXqGEKZh5C-4YeKNuxpklujrvtUvGDeZTK552UkmRfHd";

const categories = [
  { href: "/products?category=banh-cung-dinh", count: "12 Món Tiến Vua", name: "Bánh Phục Linh Hoàng Gia", description: "Bột bình tinh hảo hạng hòa nước cốt dừa nạo tay, đúc khuôn hoa sen tinh xảo tan êm đầu lưỡi.", action: "Khám phá bộ trà", image: "https://lh3.googleusercontent.com/aida-public/AB6AXuDK-6ehuDAERW9uqUbt344yaZtQgpADHd2yyez8RxefWSEEt8Cl6UaJfz_0au_UUu6yxVNFeO1zRYS1PbXSu_SUA9RtTV21dSC3qTr6cvsqN0w156pAoesP9q2ajpptyhQc9UzyMVFwV85nrD9RQg0HfEz0r8HZwCqc-BX99furgLpI0ADgC8mJbbFLcQ60lqJdhxY7w6xD5hKJJtLgzb9jwwsUt6Vjs_C4ey1xXvvpFYkT-z5WGgNt" },
  { href: "/products?category=me-xung-keo-hue", count: "16 Đặc Sản Mè", name: "Mè Xửng Giòn & Dẻo", description: "Nấu bằng mật mía nguyên chất, đậu phộng rang cát thủ công và mè lụa thơm lừng bên sông An Cựu.", action: "Xem hương vị", image: "https://lh3.googleusercontent.com/aida-public/AB6AXuBngxjLVXu5j5lYo9poeYnYxkWNbjQB7rmRCdOLJg_WiE6a3lur8V0DM3udlIw75cHUtu8s7rx7-xci0aZO__dPVWZPNfnLOMtzfsczVECqaJbZdiue4-mXBgBO_8UM38tx_TeNhpp933bdKa4OHfAC02I6l13-8-JhpvR-YGVkDQ1WT7qVw10fb5xmEFr_qI_ZxBF0LJSKuwoGanlscxyH98e98QrJ6SnfCheThb0Tpmw7dAtRcqDx" },
  { href: "/products?category=mut-hat-sen", count: "08 Loại Mứt Cố Đô", name: "Mứt Gừng Kim Long", description: "Gừng củ nhỏ vùng đất phù sa Kim Long cay nồng đậm đà, sao lửa củi đều tay giữ trọn vị cay ấm.", action: "Tìm hiểu mứt gừng", image: "https://lh3.googleusercontent.com/aida-public/AB6AXuBGOAWav2VEBt0lobU0eBf3r004msCLIKQcrkVdwoZ5-ym-8-iOwjcgmaX8SFdm0A9On6v377BxdIdin8LzJ5rx90lBA2xmafMOXM_GnU0r3QCObcf_Q0Pn7e4VBKcLmd98cz1-dp63pUXZWOvouj3mUHd7fRda5EOlodErw5nyfvOxIEdCjVRXgA_KHQ-IGxf6PDMVala-gb33sLTm3jWUvoyXja2kqN17KDEvTxNk6BymzMnNAhMq" },
  { href: "/products?category=tra-cung-dinh", count: "06 Thức Ngự Trà", name: "Trà Sen Tịnh Tâm", description: "Sen Bách Diệp hái từ tờ mờ sương sớm tại hồ Tịnh Tâm, ướp trà nõn tôm Thái Nguyên thượng hạng.", action: "Thưởng ngự trà", image: "https://lh3.googleusercontent.com/aida-public/AB6AXuDC--esaFbpVCCm10NkQ3a9IFObGRsE1vdAZ08UPfzNVNbcsk6ksizCXb0NSCqtUeG_cZTHOEAM-GL7lb1nPiW2NHxPtDI4kyHD6Y03oHrm8m23sI36OURSdl_e4V_9aXlpPO0MFnWMM7HEttBEwqTIGdFs68xsiP3syB8umGugWu0WJUxKK0C_-qIi0F2mM7v4Qrn2U-7nsM2siTiy0NJU71irRZK8VuUPnQsBLbDH7S_g14v191bM" }
];

const products = [
  { skuId: "SKU-009-500", slug: "bo-qua-hoang-trieu", meta: "Trọng lượng: 500g", name: "Hộp Quà Cố Đô Ngự Thiện", description: "Tuyển tập 5 vị bánh mứt cung đình trong hộp sơn mài đỏ thẫm đài các.", price: "450.000đ", oldPrice: "520.000đ", image: "https://lh3.googleusercontent.com/aida-public/AB6AXuCa21IRbyb84mPhvaAT5JZ_5v_4VEsiqBmM_5xFn1wAIcdtn-AnlNJtQlbIcw5-823yoRkTAtXo64QmCjw31a8M2px23in5QgC4E9ml3X3ZueUQxeBD0k5YHQP_l210Ic-06gzpkr2OyztLHyktx9qSOtMTKW7WXYJugl9K5o9hUCSCadplPNa2oqpl3VBOCgFbqp7BUMrhnkli6RhTHCTYO95jeyUMCC4BTA3aETpcKEJF7m62CylO" },
  { skuId: "SKU-002-200", slug: "me-xung-deo-ngu-thien", meta: "Hộp cao cấp: 250g", name: "Mè Xửng Dẻo Ngự Trà Thượng Hạng", description: "Độ dẻo đạt chuẩn hoàng gia, không dính răng, vị ngọt thanh đượm từ mật mía.", price: "85.000đ", oldPrice: "95.000đ", image: "https://lh3.googleusercontent.com/aida-public/AB6AXuDUX-enjOmLZIn6BKCzIQB19BmC_BtSpVJKsZ-qnoeZoGS2Mb2b99G8-L1MPVvr_vHR6Ps1OmEvKU16F3lzBeKN_Pr5LzRofe0APa73JxCIKqroeNzLDeZwN1S0EUP4sOsu47yGGU3v1dfacuXBMhvvTf0yaRfKkfu0LmVWv6x2-o6WPG_uxGM2uSRr79rdTJTxxlnk980lrhMhRgbcj7i9r85OlBgzrJLN2EO374eB_zNTYj6ZI5aE" },
  { skuId: "SKU-006-300", slug: "banh-dau-xanh-trai-cay", meta: "Hộp sơn mài: 12 cái", name: "Bánh Đậu Xanh Trái Cây Nghệ Nhân", description: "Nặn tay tỉ mỉ hình trái cây cung đình, tráng lớp rau câu tự nhiên giòn mọng.", price: "120.000đ", oldPrice: "140.000đ", image: "https://lh3.googleusercontent.com/aida-public/AB6AXuAGdlKPl8besUprgcIcaSNI-VXXEzt-aIkDTw6zE0If6NpcKmXbUXA58HO_oZbPDdSW8L_PwvVi7s3-n4fiI7yFLqHyWKiYM8zP30qlRMzUz2rDLvUDKnxRxHBMBN_TpgSNM9ciBOUI-kxDrnOfioqEgb9W16eSmlaDDnvGH_p6MK5CYsqFmT0ggFzCTnCpTmUHqbI7VNUkbm47hgj65LWqY9tqc9nNXP4jGIkreI45AUAse_CgYT8E" },
  { skuId: "SKU-011-150", slug: "tra-sen-tinh-tam", meta: "Hộp thiếc cao cấp: 100g", name: "Trà Sen Bách Diệp Hồ Tịnh Tâm", description: "Gạo sen thu hái từ 1000 bông sen Bách Diệp cổ, hương thơm tinh khiết an thần.", price: "380.000đ", oldPrice: "420.000đ", image: "https://lh3.googleusercontent.com/aida-public/AB6AXuCrlxWqIevpSqaBVb4E-_s0vLAPmThBrLkhA3jA95yYbw2rpp4vn2LQAsN0XLqg8WoX9mz9pxezbtCPPcEj0FqXm9NqQC2udbOZle5ePcwycVj6IpNoogvGrScIDwXsATP4_xHTg0ViRdPDFAP7Dn3LyiG09FWYbhk0G_-sg7bnl1WqSSYVUn1AGogMqbo2VNkNYuUDXcqeeJJzdGBHVFVrs-dw8_uSkfav3JZz3eVYzZjyBCP6kMao" }
];

export function HomeExperience() {
  const [toast, setToast] = useState("");
  const [batchCode, setBatchCode] = useState("");
  const [traceVisible, setTraceVisible] = useState(false);
  const [newsletterSent, setNewsletterSent] = useState(false);

  function showToast(message: string) {
    setToast(message);
    globalThis.setTimeout(() => setToast(""), 3200);
  }

  function previewAddToCart(name: string) {
    showToast(`Đã chọn “${name}”. Mở chi tiết sản phẩm để xác nhận quy cách trước khi thêm vào giỏ.`);
  }

  function trace(event: FormEvent) {
    event.preventDefault();
    if (!batchCode.trim()) return;
    setTraceVisible(true);
  }

  function newsletter(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    event.currentTarget.reset();
    setNewsletterSent(true);
  }

  return <main className="heritage-home">
    <section className="heritage-hero" style={{ backgroundImage: `linear-gradient(90deg, rgba(60,34,20,.98) 0%, rgba(76,43,24,.86) 47%, rgba(63,35,20,.28) 100%), url(${HERO_IMAGE})` }}>
      <div className="heritage-shell hero-grid"><div className="hero-copy"><span className="heritage-pill">✺ Di sản Ẩm thực Hoàng Cung Triều Nguyễn</span><h1>Tinh Hoa Bánh Mứt Cung Đình Huế<em>Gói Trọn Hương Vị Kinh Kỳ</em></h1><p>Tái hiện chuẩn phong vị yến tiệc cung đình qua bàn tay nghệ nhân ba đời xứ Huế. Nguyên liệu thanh thuần từ sen hồ Tịnh Tâm, gừng cay Kim Long đượm nồng tình cố đô.</p><div className="hero-buttons"><Link href="/products">Khám Phá Bộ Sưu Tập <span>→</span></Link><a href="#ocop-traceability">⌗ Truy Xuất Nguồn Gốc OCOP</a></div><dl className="hero-metrics"><div><dt>100%</dt><dd>Nguyên liệu tự nhiên</dd></div><div><dt>3 Đời</dt><dd>Nghệ nhân gia truyền</dd></div><div><dt>4 Sao</dt><dd>Chứng nhận OCOP Tỉnh</dd></div></dl></div>
      <article className="hero-feature"><div><Image fill priority sizes="(max-width: 760px) 100vw, 360px" src="https://lh3.googleusercontent.com/aida-public/AB6AXuDwdTKbXuXb6nWweFc-2PBl726WKIK1TazWrs7KwBcE-fGSju_T95LHe0AuoEEmunEVket3Hj14nQL_o5jH5S7mfU5NBqJH8HQzupuIlqPtvsv2WEzTFu92FYBE4ftxEo37pyrLiZRvVg7MzPfoLTct5Pg9DpO2DTlDp_je8ldqpu2UESMeuZrdDqxNjA2gOqTnxK1tjz-mh6zhplrJTwlgFMUMV-wbHOLbvZZCcc_shSqG-bxCsWyR" alt="Hộp sơn mài Ngự Triều" /><span>★ Đặc Sản Tiến Vua</span></div><header><h2>Hộp Sơn Mài Ngự Triều</h2><b>680.000đ</b></header><p>Bao gồm 4 vị tiến vua: Mè xửng bọc vàng, Bánh phục linh hoa sen, Trà ướp sen Tịnh Tâm &amp; Mứt gừng tiến chúa.</p><Link href="/products?category=qua-bieu">♧ Thưởng Thức Ngay</Link></article>
      </div>
    </section>

    <section className="home-section"><div className="heritage-shell"><header className="home-heading"><div><span>Tứ Đại Danh Vị</span><h2>Danh Mục Di Sản Cố Đô</h2></div><p>Tuyển lựa theo quy chuẩn mỹ vị Cung đình Nguyễn: thanh tao, chuẩn mực, lưu giữ tinh chất dược thảo xứ Huế.</p></header><div className="heritage-categories">{categories.map((category) => <Link href={category.href} key={category.name}><div><Image fill sizes="(max-width: 520px) 100vw, (max-width: 1050px) 50vw, 25vw" src={category.image} alt={category.name} /><span>OCOP 4 Sao</span></div><small>{category.count}</small><h3>{category.name}</h3><p>{category.description}</p><b>{category.action} →</b></Link>)}</div></div></section>

    <section className="home-section popular-section"><div className="heritage-shell"><header className="home-heading"><div><span>✧ Tuyển Chọn Đắt Khách Nhất</span><h2>Ngự Phẩm Được Ưa Chuộng</h2></div><div className="carousel-controls"><button aria-label="Sản phẩm trước">‹</button><button aria-label="Sản phẩm tiếp theo">›</button></div></header><div className="heritage-products">{products.map((product) => <article key={product.skuId}><Link className="product-visual" href={`/products/${product.slug}`}><Image fill sizes="(max-width: 520px) 100vw, (max-width: 1050px) 50vw, 25vw" src={product.image} alt={product.name} /><span>✺ OCOP 4 Sao</span><i>♡</i></Link><small>{product.meta}</small><Link className="product-name" href={`/products/${product.slug}`}>{product.name}</Link><p>{product.description}</p><div><b>{product.price}</b><del>{product.oldPrice}</del></div><button onClick={() => previewAddToCart(product.name)}>♧ Thêm Vào Giỏ</button></article>)}</div></div></section>

    <section className="home-section story-section" id="heritage-story"><div className="heritage-shell story-grid"><div className="story-image"><Image width={900} height={730} sizes="(max-width: 760px) 100vw, 50vw" src="https://lh3.googleusercontent.com/aida-public/AB6AXuBiLL5AFZW-DydVfKZ7zKeSltMseWtTFpfWLjT5ZzSJycGotLnk0OhH8AoxyW55RkoPKmec5YgpO1fkuEU6yUKpsZpOAvCr0xatSiDqMbTQtISycTRhD59e62URjIywRXc48YOMhvQ4ltiaXVclO7zhN7zHoXrYcpH2kmHwdAGoHN8-caFlsCGbiAk1JDmwyld8Si4P5xrykFNkek9KccyrMcc_0fCnhUxdbeOakGqdoVt2oSmG1m9F" alt="Nghệ nhân Ô Mạ làm bánh trong nhà vườn Kim Long" /><aside><span>◎</span><div><strong>Chuẩn VSATTP Quốc Tế</strong><small>ISO 22000:2018</small><p>100% không chất bảo quản công nghiệp, giữ vị bằng phương pháp ngào đường cát tự nhiên.</p></div></aside></div><div className="story-copy"><span className="section-kicker">▧ Chuyện Làng Nghề Kim Long</span><h2>Ba Đời Gìn Giữ Mật Mã Hương Vị Của Các Bậc Ngự Trù</h2><p>Khởi nguồn từ xưởng bánh thủ công nhỏ ven bờ sông Hương thuộc làng cổ Kim Long vào những năm 1950, Ô Mạ được nuôi dưỡng bởi công thức bí truyền của cụ cố từng phục vụ tại ngự trù Nội đình Huế.</p><p>Từng chiếc mè xửng vàng óng, từng đĩa mứt gừng cay đượm không đơn thuần là món điểm tâm thưởng trà, mà là tác phẩm kết tinh từ tính cách trầm mặc, tỉ mỉ và tôn kính cội nguồn.</p><div className="story-notes"><article><span>♧</span><div><strong>Nguồn Gốc Bản Địa</strong><small>Hạt sen Tịnh Tâm, mè rang cát An Cựu, mạch nha Kim Long.</small></div></article><article><span>✣</span><div><strong>Kỹ Nghệ Thủ Công</strong><small>Ngào lửa củi thanh trà, nhào nặn khuôn gỗ chạm trổ hoa văn.</small></div></article></div><Link href="/products">Đọc toàn bộ biên niên sử Ô Mạ Huế →</Link></div></div></section>

    <section className="trace-section" id="ocop-traceability"><div className="heritage-shell"><div className="trace-panel"><div className="trace-copy"><span className="heritage-pill">⌗ Hệ Thống Số Hóa Nông Sản Cố Đô</span><h2>Minh Bạch Nguồn Gốc Chuẩn OCOP 4 Sao</h2><p>Mỗi hộp bánh mứt Ô Mạ khi xuất xưởng đều gắn liền mã định danh độc bản. Nhập mã số Batch trên nhãn hộp hoặc quét mã QR để thẩm định vùng canh tác sen Tịnh Tâm, nhật ký ngào mứt và kết quả kiểm nghiệm VSATTP.</p><form onSubmit={trace}><div><input value={batchCode} onChange={(event) => { setBatchCode(event.target.value); setTraceVisible(false); }} placeholder="Nhập mã lô (VD: HUE-2025-BATCH09)" required /><button type="button" onClick={() => setBatchCode("HUE-2025-BATCH09")}>Mã mẫu</button></div><button>⌕ Kiểm Tra Ngay</button></form>{traceVisible ? <div className="trace-result"><strong>✓ Lô hàng hợp lệ: {batchCode}</strong><p>Thu hái tại vùng sen Hồ Tịnh Tâm; kiểm nghiệm VSATTP đạt chuẩn OCOP 4 sao.</p></div> : null}</div><div className="trace-proofs"><article><span>♧</span><strong>100% Thuần Tự Nhiên</strong><small>Không phẩm màu nhân tạo</small></article><article><span>✺</span><strong>Chuẩn OCOP 4 Sao</strong><small>Chứng nhận cấp Tỉnh</small></article><article><span>⌾</span><strong>Bảo Hộ Chỉ Dẫn Địa Lý</strong><small>Đặc sản sông Hương</small></article><article><span>✈</span><strong>Đạt Chuẩn Xuất Khẩu</strong><small>Nhật Bản, Pháp &amp; Hoa Kỳ</small></article></div></div></div></section>

    <section className="newsletter-section"><div className="heritage-shell newsletter-panel"><div><span>♧ Món Quà Tri Kỷ Bốn Phương</span><h2>Đăng Ký Nhận Hương Trà Cố Đô &amp; Giảm 10% Cho Đơn Hàng Đầu Tiên</h2><p>Trở thành tri âm của Ô Mạ để nhận cẩm nang thưởng trà theo mùa cung đình, những câu chuyện di sản kinh thành và ưu đãi đặc quyền dịp lễ Tết.</p></div><div><form onSubmit={newsletter}><input type="email" required placeholder="Nhập địa chỉ thư điện tử của bạn…" /><button>Đăng Ký →</button></form>{newsletterSent ? <strong>✓ Cảm ơn quý khách! Mã TRIKY10 đã được gửi đến email.</strong> : <small>Chúng tôi cam kết tôn trọng quyền riêng tư và không gửi thư rác.</small>}</div></div></section>
    {toast ? <div className="home-toast" role="status">✓ {toast}</div> : null}
  </main>;
}
