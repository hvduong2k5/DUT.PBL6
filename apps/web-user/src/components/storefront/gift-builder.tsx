"use client";

import Image from "next/image";
import { useState, type CSSProperties } from "react";

const colors = [
  ["Nâu gỗ Cố đô", "#563A26", "#FCF9F4", "#382315"],
  ["Vàng mật Cung đình", "#D4A24C", "#2F241C", "#A6782B"],
  ["Kem ngà Giấy Dó", "#FCF9F4", "#563A26", "#C9B79C"],
  ["Đỏ Chu Sa Lễ Tết", "#7D2319", "#FCF9F4", "#4F140E"],
];
const combos = [
  {
    label: "Hộp Đoàn Viên Chuẩn (02 Mè xửng + 01 Trà Sen Tịnh Tâm) — 265.000 ₫",
    price: 265000,
    items: [
      "Khoang 1: Mè xửng dẻo",
      "Khoang 2: Trà sen Tịnh Tâm",
      "Khoang 3: Mè xửng giòn",
    ],
  },
  {
    label:
      "Hộp Ngự Trà Hoàng Gia (02 Trà Huế + 01 Mè xửng hạt sen) — 310.000 ₫",
    price: 310000,
    items: [
      "Khoang 1: Trà Sen Thượng Hạng",
      "Khoang 2: Trà Cung Đình",
      "Khoang 3: Mè xửng hạt sen",
    ],
  },
  {
    label:
      "Hộp Thức Quà Quê Hương (Mè xửng dẻo, giòn & Hạt sen sấy) — 245.000 ₫",
    price: 245000,
    items: [
      "Khoang 1: Mè xửng dẻo",
      "Khoang 2: Mè xửng giòn",
      "Khoang 3: Kẹo cau & Hạt sen",
    ],
  },
];

export function GiftBuilder({
  onReview,
}: {
  onReview: (summary: string) => void;
}) {
  const [color, setColor] = useState(0);
  const [index, setIndex] = useState(0);
  const [note, setNote] = useState("Gửi trao phong vị Huế ấm nồng");
  const combo = combos[index];
  return (
    <section
      id="gift"
      className="section-block"
      style={{ position: "relative" }}
    >
      <Image
        width={1303}
        height={669}
        sizes="(max-width: 767px) 100vw, 50vw"
        src="/images/goi-hue/11-cau-truong-tien-phai.png"
        alt=""
        aria-hidden="true"
        style={{
          position: "absolute",
          right: "0",
          top: "-18px",
          width: "clamp(360px, 34vw, 480px)",
          opacity: "0.22",
          maskImage: "linear-gradient(to left, #000 72%, transparent 100%)",
          WebkitMaskImage:
            "linear-gradient(to left, #000 72%, transparent 100%)",
          pointerEvents: "none",
          zIndex: "0",
        }}
      />
      <div className="wrap" style={{ position: "relative", zIndex: "1" }}>
        <div className="gift-builder">
          <div
            className="gift-stage"
            id="giftPicture"
            style={
              {
                "--paper": colors[color][1],
                "--letter": colors[color][2],
                "--box-side": colors[color][3],
              } as CSSProperties
            }
          >
            <div className="gift-box-preview">
              <div className="gift-box__ribbon"></div>
              <div className="gift-box__tag">
                <small>QUÀ BIẾU ĐẶC SẢN CỐ ĐÔ</small>
                <strong>Gói Huế</strong>
                <span
                  id="boxSubLabel"
                  style={{ fontSize: "0.8rem", opacity: "0.9" }}
                >
                  {combo.label.split(" (")[0]}
                </span>
                <div className="gift-box__note-preview" id="boxGreetingPreview">
                  {note && "“" + note + "”"}
                </div>
              </div>
            </div>
            <div
              className="gift-compartments-summary"
              id="giftCompartmentsPreview"
              aria-live="polite"
            >
              {combo.items.map((item) => (
                <span className="compartment-pill" key={item}>
                  {item}
                </span>
              ))}
            </div>
          </div>

          <div
            style={{
              display: "flex",
              flexDirection: "column",
              justifyContent: "center",
              gap: "18px",
              position: "relative",
              zIndex: "2",
            }}
          >
            <div>
              <span className="section-header__kicker">
                Nghệ thuật trao tặng
              </span>
              <h2>Gói quà theo ý bạn</h2>
              <p style={{ color: "var(--phu)", marginTop: "6px" }}>
                Tự tay chọn sắc giấy Dó bồi cứng, phối thức quà trong hộp 3
                khoang và viết đôi dòng gửi người trân quý.
              </p>
            </div>

            <fieldset style={{ border: "0", padding: "0", margin: "0" }}>
              <legend className="hue-field-label">
                1. Chọn sắc giấy hộp quà ·{" "}
                <strong id="colorName" style={{ color: "var(--do-chu-sa)" }}>
                  {colors[color][0]}
                </strong>
              </legend>
              <div className="swatch-list" id="swatches">
                {colors.map(([name, hex, tick], i) => (
                  <label
                    key={name}
                    className="swatch-item"
                    style={{ "--chip": hex, "--tick": tick } as CSSProperties}
                  >
                    <input
                      type="radio"
                      name="gift-color"
                      aria-label={name}
                      checked={color === i}
                      onChange={() => setColor(i)}
                    />
                    <span className="swatch-dot" />
                    <span>{name}</span>
                  </label>
                ))}
              </div>
            </fieldset>

            <div className="field-group">
              <label className="hue-field-label" htmlFor="giftComboSelect">
                2. Chọn tổ hợp 3 khoang thức quà bên trong
              </label>
              <select
                id="giftComboSelect"
                className="field"
                value={index}
                onChange={(e) => setIndex(Number(e.target.value))}
              >
                {combos.map((opt, i) => (
                  <option key={opt.label} value={i}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            <div className="field-group">
              <label className="hue-field-label" htmlFor="giftNoteInput">
                3. Lời nhắn trên Thiệp chúc Cố đô (Miễn phí thiệp giấy Dó)
              </label>
              <input
                type="text"
                id="giftNoteInput"
                className="field"
                maxLength={70}
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="Nhập lời chúc gửi người nhận..."
              />
            </div>
            <div
              className="between"
              style={{ paddingTop: "8px", borderTop: "1px solid var(--vien)" }}
            >
              <div>
                <small style={{ color: "var(--phu)", display: "block" }}>
                  Giá hộp quà hoàn chỉnh
                </small>
                <strong
                  className="price-tag"
                  id="giftTotalPrice"
                  style={{ fontSize: "1.35rem" }}
                >
                  {combo.price.toLocaleString("vi-VN")} ₫
                </strong>
              </div>
              <button
                type="button"
                className="btn btn--brown btn--fold"
                id="addGiftBtn"
                onClick={() =>
                  onReview(
                    [colors[color][0], combo.label, note]
                      .filter(Boolean)
                      .join(" · "),
                  )
                }
              >
                Xem hộp quà đã phối <span className="btn__arrow">→</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
