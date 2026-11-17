import Link from "next/link";
import Image from "next/image";

export function Story() {
  return (
    <section id="story" className="story-showcase">
      <Image
        width={1536}
        height={1024}
        sizes="(max-width: 767px) 100vw, 50vw"
        src="/images/goi-hue/02-canh-hoa.png"
        alt=""
        aria-hidden="true"
        style={{
          position: "absolute",
          left: "0",
          top: "0",
          width: "clamp(220px, 21vw, 300px)",
          opacity: "0.88",
          pointerEvents: "none",
          zIndex: "1",
        }}
      />

      <Image
        width={2172}
        height={724}
        sizes="(max-width: 767px) 100vw, 50vw"
        src="/images/goi-hue/01-hien-nha.png"
        alt=""
        aria-hidden="true"
        style={{
          position: "absolute",
          left: "clamp(90px, 10vw, 150px)",
          top: "8px",
          width: "clamp(340px, 32vw, 460px)",
          opacity: "0.24",
          maskImage:
            "linear-gradient(90deg, transparent 0%, #000 18%, #000 88%, transparent 100%)",
          WebkitMaskImage:
            "linear-gradient(90deg, transparent 0%, #000 18%, #000 88%, transparent 100%)",
          pointerEvents: "none",
          zIndex: "0",
        }}
      />

      <Image
        width={1306}
        height={493}
        sizes="(max-width: 767px) 100vw, 50vw"
        src="/images/goi-hue/10-tuong-thanh.png"
        alt=""
        aria-hidden="true"
        style={{
          position: "absolute",
          left: "0",
          bottom: "0",
          width: "clamp(360px, 34vw, 480px)",
          opacity: "0.22",
          maskImage: "linear-gradient(to right, #000 75%, transparent 100%)",
          WebkitMaskImage:
            "linear-gradient(to right, #000 75%, transparent 100%)",
          pointerEvents: "none",
          zIndex: "0",
        }}
      />

      <Image
        width={994}
        height={725}
        sizes="(max-width: 767px) 100vw, 50vw"
        src="/images/goi-hue/12-hoa-sen-cung-dinh-phai.png"
        alt=""
        aria-hidden="true"
        style={{
          position: "absolute",
          right: "0",
          bottom: "6px",
          width: "clamp(220px, 21vw, 300px)",
          opacity: "0.78",
          pointerEvents: "none",
          zIndex: "1",
        }}
      />
      <div className="wrap">
        <div
          className="hue-story-grid"
          style={{ marginBottom: "clamp(36px, 5vw, 64px)" }}
        >
          <div
            className="hue-story-copy"
            style={{ paddingTop: "clamp(64px, 7vw, 92px)" }}
          >
            <h2>Bên chén trà Huế</h2>
            <div
              style={{
                width: "64px",
                height: "3px",
                background: "var(--mat)",
                margin: "14px 0 18px",
              }}
            ></div>
            <p
              style={{
                fontSize: "1.05rem",
                color: "var(--muc)",
                marginBottom: "14px",
              }}
            >
              Thưởng chén trà Huế, cảm nhận hương vị thanh dịu và nét tinh tế
              trong những thức quà mộc mạc của cố đô.
            </p>
            <p style={{ color: "var(--phu)", marginBottom: "22px" }}>
              Một miếng mè xửng dẻo thơm mùi mạch nha, điểm hạt mè rang vàng
              óng, nhấp cùng ngụm trà sen Tịnh Tâm ấm nóng — đĩa quà nhỏ trên
              bàn gỗ là lời mời thong thả ngồi lại, kể nhau nghe đôi chuyện xứ
              kinh kỳ.
            </p>
            <Link className="btn btn--paper" href="/products?category=tra-cung-dinh" data-nav-group="1">
              Khám phá Trà Huế &amp; Mè xửng{" "}
              <span className="btn__arrow">→</span>
            </Link>
          </div>
          <div className="story-photo-frame">
            <Image
              width={504}
              height={354}
              sizes="(max-width: 767px) 100vw, 50vw"
              src="/images/goi-hue/photo-story-tra-hue.jpg"
              alt="Bàn tay rót trà Huế từ ấm sứ hoa lam bên đĩa mè xửng"
            />
          </div>
        </div>

        <div className="hue-story-grid">
          <div className="story-photo-frame">
            <Image
              width={470}
              height={260}
              sizes="(max-width: 767px) 100vw, 50vw"
              src="/images/goi-hue/photo-story-nha-ruong.jpg"
              alt="Không gian hiên nhà rường gỗ cổ truyền xứ Huế ngập nắng"
            />
          </div>
          <div className="hue-story-copy">
            <span className="section-header__kicker">
              Di sản kiến trúc &amp; Ẩm thực
            </span>
            <h2>Thức quà từ Huế</h2>
            <div
              style={{
                width: "64px",
                height: "3px",
                background: "var(--mat)",
                margin: "14px 0 18px",
              }}
            ></div>
            <p
              style={{
                fontSize: "1.05rem",
                color: "var(--muc)",
                marginBottom: "14px",
              }}
            >
              Gói Huế mang đến những thức quà mộc mạc mà tinh tế của xứ Huế: mè
              xửng dẻo thơm, trà Huế thanh dịu và những món quà biếu trang nhã,
              gửi trọn hương vị và tình cảm Cố đô.
            </p>
            <p style={{ color: "var(--phu)", marginBottom: "22px" }}>
              Mỗi hộp quà lấy cảm hứng từ nếp giấy Dó thủ công, nhịp mái hiên
              nhà rường và cổng thành cổ kính, để người nhận khi mở hộp như thấy
              cả một góc bình yên của đất thần kinh.
            </p>
            <Link className="btn btn--brown" href="/products?category=qua-bieu">
              Tự tay gói hộp quà Cố đô <span className="btn__arrow">→</span>
            </Link>
          </div>
        </div>
      </div>
    </section>
  );
}
