import Image from "next/image";

export function Hero({ paper }: { paper: boolean }) {
  return (
    <section className="hero-section" id="hero">
      <Image
        width={1536}
        height={1024}
        sizes="(max-width: 767px) 100vw, 50vw"
        className="hero-ambient-blossom"
        src="/images/goi-hue/02-canh-hoa.png"
        alt=""
        aria-hidden="true"
      />
      <Image
        width={2172}
        height={724}
        sizes="(max-width: 767px) 100vw, 50vw"
        className="hero-ambient-veranda"
        src="/images/goi-hue/01-hien-nha-phai.png"
        alt=""
        aria-hidden="true"
      />
      <div className="wrap">
        <div id="heroDarkVariant" hidden={paper} className="hero-stage--dark">
          <div className="hero-dark__copy">
            <span className="hero-dark__kicker">
              Đặc sản &amp; Quà biếu Cố đô
            </span>
            <h1 className="hero-dark__title">Phong vị Huế</h1>
            <div className="hero-dark__rule"></div>
            <p className="hero-dark__lead">
              Mè xửng dẻo thơm, Trà Huế thanh dịu, quà biếu đậm tình miền Trung
              — trao gửi bằng cả tấm lòng.
            </p>
            <div className="row" style={{ gap: "14px" }}>
              <a className="btn" href="#collection">
                Khám phá bộ sưu tập <span className="btn__arrow">→</span>
              </a>
              <a
                className="btn btn--paper"
                href="#gift"
                style={{
                  background: "rgba(252,249,244,0.12)",
                  color: "var(--kem)",
                  borderColor: "rgba(252,249,244,0.45)",
                }}
              >
                Gói quà theo ý bạn
              </a>
            </div>
          </div>
          <div className="hero-dark__visual">
            <Image
              width={605}
              height={368}
              sizes="(max-width: 767px) 100vw, 50vw"
              src="/images/goi-hue/photo-hero-studio.jpg"
              alt="Bộ sưu tập hộp quà Gói Huế, lon Trà Sen và đĩa Mè xửng thủ công"
            />
          </div>
        </div>

        <div
          id="heroPaperVariant"
          className="hero-stage--paper"
          hidden={!paper}
        >
          <Image
            width={2172}
            height={724}
            sizes="(max-width: 767px) 100vw, 50vw"
            className="hero-paper__veranda"
            src="/images/goi-hue/01-hien-nha.png"
            alt=""
            aria-hidden="true"
          />
          <div className="hero-paper__plaque">
            <span className="section-header__kicker">Thức quà từ Cố đô</span>
            <h1
              style={{
                fontSize: "clamp(2.5rem, 4.4vw, 4rem)",
                color: "var(--nau-dam)",
                marginBottom: "14px",
              }}
            >
              Phong vị Huế
            </h1>
            <p
              style={{
                color: "var(--phu)",
                fontSize: "1.05rem",
                margin: "0 auto 24px",
                maxWidth: "34ch",
              }}
            >
              Mè xửng dẻo thơm, trà Huế thanh dịu cùng những thức quà đậm tình
              Cố đô.
            </p>
            <a
              className="btn btn--brown"
              href="#collection"
              style={{ minWidth: "190px" }}
            >
              Khám phá <span className="btn__arrow">→</span>
            </a>
          </div>
          <div className="hero-paper__frame">
            <Image
              width={594}
              height={416}
              sizes="(max-width: 767px) 100vw, 50vw"
              src="/images/goi-hue/photo-hero-tea-table.jpg"
              alt="Đĩa mè xửng Huế bên ấm trà sứ xanh trắng dưới nắng hiên nhà"
            />
          </div>
        </div>
      </div>
    </section>
  );
}
