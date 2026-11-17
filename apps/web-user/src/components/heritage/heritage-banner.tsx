"use client";

import Link from "next/link";
import { useEffect, useState, type ReactNode } from "react";
import { ProductImage } from "@/components/product/product-image";

export interface BannerSlide {
  kicker: string;
  title: ReactNode;
  lead?: ReactNode;
  image: { src: string; alt: string };
  cta?: { href: string; label: string };
}

interface HeritageBannerProps {
  /** Single-slide shorthand. Ignored when `slides` has entries. */
  kicker?: string;
  title?: ReactNode;
  lead?: ReactNode;
  image?: { src: string; alt: string };
  /** 2–3 slides turn the banner into an onboarding-style carousel with dots and autoplay. */
  slides?: BannerSlide[];
  titleId: string;
  /** h1 for page heroes, h2 for banners inside a page. */
  level?: 1 | 2;
  priority?: boolean;
  /** Persistent content under the slide text (forms, buttons); it does not change between slides. */
  children?: ReactNode;
  className?: string;
  autoPlayMs?: number;
}

/**
 * Dark lacquer stage with the kit's double stepped frame (khung-doi). Copy sits on the left and the photo
 * fades into the lacquer from the right. Shared by the home hero, traceability, promotions and catalog heroes.
 */
export function HeritageBanner({ kicker, title, lead, image, slides, titleId, level = 2, priority = false, children, className = "", autoPlayMs = 3000 }: HeritageBannerProps) {
  const items: BannerSlide[] = slides?.length ? slides : [{ kicker: kicker ?? "", title, lead, image: image ?? { src: "", alt: "" } }];
  const multi = items.length > 1;
  const [index, setIndex] = useState(0);
  const [reduceMotion, setReduceMotion] = useState(false);
  
  useEffect(() => {
    const query = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = () => setReduceMotion(query.matches);
    update();
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);

  useEffect(() => {
    if (!multi || reduceMotion) return;
    const timer = setInterval(() => setIndex((value) => (value + 1) % items.length), autoPlayMs);
    return () => clearInterval(timer);
  }, [multi, reduceMotion, items.length, autoPlayMs, index]);

  const step = (delta: number) => setIndex((value) => (value + delta + items.length) % items.length);
  const arrow = (delta: number) => <button type="button" className={`gh-banner__arrow gh-banner__arrow--${delta < 0 ? "prev" : "next"}`}
    aria-label={delta < 0 ? "Trang trước" : "Trang kế tiếp"} onClick={() => step(delta)}><span aria-hidden="true">{delta < 0 ? "←" : "→"}</span></button>;

  return <div className="gh-banner-shell">
    <div
    className={`gh-banner gh-surface-imperial${multi ? " gh-banner--multi" : ""} ${className}`}
    {...(multi ? { role: "group", "aria-roledescription": "carousel", "aria-label": "Giới thiệu nổi bật" } : {})}>
    <div className="gh-banner__copy">
      <div className="gh-banner__slides" aria-live="off">
        {items.map((slide, i) => { const Heading = i === 0 && level === 1 ? "h1" : "h2"; return <div className={`gh-banner__slide${i === index ? " is-active" : ""}`} key={i}
          aria-hidden={i === index ? undefined : true} inert={i === index ? undefined : true}
          {...(multi ? { role: "group", "aria-roledescription": "slide", "aria-label": `${i + 1} / ${items.length}` } : {})}>
          <span className="gh-banner__kicker">{slide.kicker}</span>
          <Heading id={i === 0 ? titleId : `${titleId}-${i}`} className="gh-banner__title">{slide.title}</Heading>
          <div className="gh-banner__rule" aria-hidden="true" />
          {slide.lead ? <p className="gh-banner__lead">{slide.lead}</p> : null}
          {slide.cta ? <Link className="gh-banner__cta" href={slide.cta.href}>{slide.cta.label} <span aria-hidden="true">→</span></Link> : null}
        </div>; })}
      </div>
      {children}
    </div>
    <div className="gh-banner__visual">
      {items.map((slide, i) => <div className={`gh-banner__photo${i === index ? " is-active" : ""}`} key={slide.image.src + i}>
        <ProductImage fill priority={priority && i === 0} sizes="(max-width: 960px) 100vw, 55vw" src={slide.image.src} alt={i === index ? slide.image.alt : ""} />
      </div>)}
    </div>
    {multi ? <>
      {arrow(-1)}
      {arrow(1)}
      <div className="gh-banner__dots" role="group" aria-label="Chọn trang giới thiệu">
        {items.map((_, i) => <button key={i} type="button" className={i === index ? "is-active" : undefined}
          aria-label={`Trang ${i + 1} / ${items.length}`} aria-current={i === index ? "true" : undefined} onClick={() => setIndex(i)}><span /></button>)}
      </div>
    </> : null}
  </div>
  </div>;
}
