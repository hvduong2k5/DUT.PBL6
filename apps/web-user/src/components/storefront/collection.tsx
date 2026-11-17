import Image from "next/image";
import type { ReactNode } from "react";

export function Collection({ children }: { children: ReactNode }) {
  return (
    <section id="collection" className="section-block">
      <Image
        width={2172}
        height={724}
        sizes="(max-width: 767px) 100vw, 50vw"
        src="/images/goi-hue/05-may.png"
        alt=""
        aria-hidden="true"
        style={{
          position: "absolute",
          left: "0",
          top: "10px",
          width: "clamp(180px, 16vw, 240px)",
          opacity: "0.28",
          pointerEvents: "none",
          zIndex: "0",
        }}
      />
      <div className="wrap" style={{ position: "relative", zIndex: "1" }}>
        <div className="section-header section-header--center">
          <div className="section-header__line"></div>
          <h2 style={{ paddingInline: "16px" }}>Chọn thức quà</h2>
          <div className="section-header__line"></div>
        </div>
        <div className="categories-grid" id="categoriesGrid">
          {children}
        </div>
      </div>
    </section>
  );
}
