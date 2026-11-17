import Image from "next/image";
import type { ReactNode } from "react";

export function ShopSection({ children }: { children: ReactNode }) {
  return (
    <section
      id="shop"
      className="section-block"
      style={{ position: "relative" }}
    >
      <Image
        width={1536}
        height={1024}
        sizes="(max-width: 767px) 100vw, 50vw"
        src="/images/goi-hue/07-hat-me.png"
        alt=""
        aria-hidden="true"
        style={{
          position: "absolute",
          left: "0",
          bottom: "0",
          width: "clamp(200px, 18vw, 260px)",
          opacity: "0.36",
          pointerEvents: "none",
          zIndex: "0",
        }}
      />
      <Image
        width={1306}
        height={493}
        sizes="(max-width: 767px) 100vw, 50vw"
        src="/images/goi-hue/10-tuong-thanh-phai.png"
        alt=""
        aria-hidden="true"
        style={{
          position: "absolute",
          right: "0",
          top: "-6px",
          width: "clamp(380px, 36vw, 520px)",
          opacity: "0.24",
          maskImage: "linear-gradient(to left, #000 72%, transparent 100%)",
          WebkitMaskImage:
            "linear-gradient(to left, #000 72%, transparent 100%)",
          pointerEvents: "none",
          zIndex: "0",
        }}
      />
      <div className="wrap" style={{ position: "relative", zIndex: "1" }}>
        {children}
      </div>
    </section>
  );
}
