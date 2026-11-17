"use client";

import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { Hero } from "@/components/storefront/hero";
import { Story } from "@/components/storefront/story";
import { RecommendationRail } from "@/components/recommendation/recommendation-rail";

export function HomeExperience() {
  const router = useRouter();
  return (
    <main
      id="main"
      className="hue-surface hue-home hue-watermark-canvas renovated-home"
    >
      <Hero paper={false} />
      <div className="home-discovery">
        <RecommendationRail placement="home" />
      </div>
      <section className="home-gifting wrap">
        <div>
          <span className="section-header__kicker">
            Một món quà, một tấm lòng
          </span>
          <h2>Chọn quà từ Huế</h2>
          <p>
            Mè xửng, trà thơm và những hộp quà chỉn chu dành cho người thương.
          </p>
          <Link className="btn btn--brown" href="/products?category=qua-bieu">
            Xem quà biếu →
          </Link>
        </div>
        <Link href="/products?category=qua-bieu" aria-label="Khám phá quà biếu">
          <Image
            src="/images/goi-hue/photo-cat-qua-bieu.jpg"
            width={640}
            height={400}
            alt="Hộp quà Gói Huế thắt nơ nâu"
          />
        </Link>
      </section>
      <div id="heritage-story">
        <Story />
      </div>
      <section className="home-origin wrap">
        <div>
          <h2>Mỗi thức quà đều có câu chuyện</h2>
          <p>Tra cứu mã trên nhãn để xem thông tin về lô sản phẩm.</p>
        </div>
        <form

          onSubmit={(e) => {
            e.preventDefault();
            const value = new FormData(e.currentTarget)
              .get("batch")
              ?.toString()
              .trim();
            if (value)
              router.push("/trace/" + encodeURIComponent(value));
          }}
        >
          <label htmlFor="home-batch">Mã trên nhãn sản phẩm</label>
          <div>
            <input id="home-batch" name="batch" required />
            <button className="btn btn--brown">Tra cứu</button>
          </div>
        </form>
      </section>
    </main>
  );
}
