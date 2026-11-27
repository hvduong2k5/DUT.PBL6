"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useAuth } from "@/components/auth/auth-provider";
import { ProductImage } from "@/components/product/product-image";
import { useRecommendation } from "./recommendation-provider";
import { recommendationService, recommendationsEnabled } from "@/services/recommendation-service";
import type { RecommendationContext, Recommendations } from "@/lib/recommendation/types";
import styles from "./recommendation.module.css";

const SKELETON_CARDS = 4;

export function RecommendationRail(props: RecommendationContext) {
  const auth = useAuth();
  const tracking = useRecommendation();
  const [result, setResult] = useState<Recommendations | null>(null);
  const container = useRef<HTMLElement>(null);
  const shown = useRef(new Set<string>());
  const context = JSON.stringify(props);
  const version = tracking.consent?.consentVersion;
  const actor = auth.profile?.principalId;
  const requestKey = JSON.stringify([context, actor, version]);
  const [resolvedKey, setResolvedKey] = useState("");
  const [failedKey, setFailedKey] = useState("");

  useEffect(() => {
    if (!recommendationsEnabled || auth.isLoading || auth.isEmployee) return;
    const controller = new AbortController();
    const delay = props.placement === "cart" ? 200 : 0;
    let active = true;
    const timer = setTimeout(() => {
      setResult(null);
      void recommendationService.list(JSON.parse(context) as RecommendationContext, controller.signal)
        .then((value) => { if (active) { setResult(value); setResolvedKey(requestKey); } })
        .catch(() => { if (active) { setResult(null); setFailedKey(requestKey); } });
    }, delay);
    return () => { active = false; clearTimeout(timer); controller.abort(); };
  }, [context, version, actor, auth.isLoading, auth.isEmployee, props.placement, requestKey]);

  useEffect(() => {
    const node = container.current;
    if (!node || !result || !tracking.measurement || shown.current.has(result.recommendationId)) return;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const observer = new IntersectionObserver(([entry]) => {
      if (timer) clearTimeout(timer);
      if (entry.isIntersecting && entry.intersectionRatio >= 0.5) {
        timer = setTimeout(() => {
          if (!shown.current.has(result.recommendationId)) {
            shown.current.add(result.recommendationId);
            tracking.impression(result);
          }
        }, 1000);
      }
    }, { threshold: [0, 0.5, 1] });
    observer.observe(node);
    return () => { observer.disconnect(); if (timer) clearTimeout(timer); };
  }, [result, tracking]);

  if (!recommendationsEnabled || auth.isEmployee || failedKey === requestKey
    || props.placement === "cart" && !props.skuCodes?.length) return null;

  const ready = Boolean(result) && resolvedKey === requestKey;
  if (ready && !result?.items.length) return null;

  let title = "Sản phẩm tương tự";
  if (props.placement === "cart") title = result?.strategy === "BOUGHT_TOGETHER" ? "Thường mua cùng" : "Gợi ý thêm cho bạn";
  if (props.placement === "home") title = result?.strategy === "PERSONALIZED" ? "Dành cho bạn" : "Sản phẩm nổi bật";

  return <section className={styles.rail} ref={container} aria-label={title} aria-busy={!ready}>
    <div className={styles.header}>
      <div className={styles.line} aria-hidden="true" />
      <h2>{title}</h2>
      <div className={styles.line} aria-hidden="true" />
    </div>
    <div className={styles.grid}>
      {ready && result ? result.items.map((item) => <Link
        className={`gh-frame-card ${styles.card}`} key={item.productId} href={`/products/${encodeURIComponent(item.slug)}`}
        onClick={() => tracking.click(result, item)}>
        <div className={`gh-card-media ${styles.media}`}>
          {item.thumbnailUrl
            ? <ProductImage src={item.thumbnailUrl} alt="" fill sizes="(max-width: 760px) 45vw, 210px" />
            : <span className={styles.fallback} aria-hidden="true">Gói Huế</span>}
        </div>
        <div className={styles.body}>
          <strong className={styles.name}>{item.name}</strong>
          <span className={styles.price}>{new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND" }).format(item.priceVnd)}</span>
        </div>
      </Link>) : Array.from({ length: SKELETON_CARDS }, (_, index) => <div className={`gh-frame-card ${styles.card} ${styles.skeleton}`} key={index} aria-hidden="true">
        <div className={`gh-card-media ${styles.media}`} />
        <div className={styles.body}><div className={styles.bar} /><div className={`${styles.bar} ${styles.short}`} /></div>
      </div>)}
    </div>
  </section>;
}

