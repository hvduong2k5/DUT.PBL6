"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useAuth } from "@/components/auth/auth-provider";
import { ProductImage } from "@/components/product/product-image";
import { useRecommendation } from "./recommendation-provider";
import {
  recommendationService,
  recommendationsEnabled,
} from "@/services/recommendation-service";
import type {
  RecommendationContext,
  Recommendations,
} from "@/lib/recommendation/types";
import styles from "./recommendation.module.css";

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
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);

  useEffect(() => {
    if (!recommendationsEnabled || auth.isLoading || auth.isEmployee) return;
    const controller = new AbortController();
    const delay = props.placement === "cart" ? 200 : 0;
    let active = true;
    const timer = setTimeout(() => {
      setResult(null);
      setFailed(false);
      void recommendationService
        .list(JSON.parse(context) as RecommendationContext, controller.signal)
        .then((value) => {
          if (active) {
            setResult(value);
            setResolvedKey(requestKey);
          }
        })
        .catch(() => {
          if (active) {
            setResult(null);
            setFailed(true);
            setResolvedKey(requestKey);
          }
        });
    }, delay);
    return () => {
      active = false;
      clearTimeout(timer);
      controller.abort();
    };
  }, [
    context,
    version,
    actor,
    auth.isLoading,
    auth.isEmployee,
    props.placement,
    requestKey,
    retry,
  ]);

  useEffect(() => {
    const node = container.current;
    if (
      !node ||
      !result ||
      !tracking.measurement ||
      shown.current.has(result.recommendationId)
    )
      return;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (timer) clearTimeout(timer);
        if (entry.isIntersecting && entry.intersectionRatio >= 0.5) {
          timer = setTimeout(() => {
            if (!shown.current.has(result.recommendationId)) {
              shown.current.add(result.recommendationId);
              tracking.impression(result);
            }
          }, 1000);
        }
      },
      { threshold: [0, 0.5, 1] },
    );
    observer.observe(node);
    return () => {
      observer.disconnect();
      if (timer) clearTimeout(timer);
    };
  }, [result, tracking]);

  if (
    !recommendationsEnabled ||
    auth.isEmployee ||
    (props.placement === "cart" && !props.skuCodes?.length)
  )
    return null;
  const title =
    props.placement === "home"
      ? "Sản phẩm nổi bật"
      : props.placement === "cart"
        ? "Gợi ý thêm cho bạn"
        : "Sản phẩm tương tự";
  const loading = !failed && resolvedKey !== requestKey;
  return (
    <section
      className={styles.rail}
      ref={container}
      aria-label={title}
      aria-busy={loading}
    >
      <div className={styles.heading}>
        <h2>{title}</h2>
        <Link href="/products">Xem tất cả sản phẩm →</Link>
      </div>
      {loading ? (
        <div className={styles.grid} role="status" aria-label="Đang tải gợi ý">
          {[0, 1, 2, 3].map((i) => (
            <div className={styles.loading} key={i} />
          ))}
        </div>
      ) : failed ? (
        <div className={styles.status}>
          <p>Chưa tải được gợi ý sản phẩm.</p>
          <button
            onClick={() => {
              setFailed(false);
              setResolvedKey("");
              setRetry((v) => v + 1);
            }}
          >
            Thử lại
          </button>
          <Link href="/products">Khám phá sản phẩm</Link>
        </div>
      ) : !result?.items.length ? (
        <div className={styles.status}>
          <p>Gợi ý đang được cập nhật.</p>
          <Link href="/products">Khám phá sản phẩm</Link>
        </div>
      ) : (
        <>
          <div className={styles.grid}>
            {result!.items.map((item) => (
              <Link
                className={styles.card}
                key={item.productId}
                href={`/products/${encodeURIComponent(item.slug)}`}
                onClick={() => tracking.click(result!, item)}
              >
                <div className={styles.image}>
                  {item.thumbnailUrl ? (
                    <ProductImage
                      src={item.thumbnailUrl}
                      alt={item.name}
                      fill
                      sizes="(max-width: 760px) 45vw, 180px"
                    />
                  ) : (
                    <span aria-hidden="true">Gói Huế</span>
                  )}
                </div>
                <strong>{item.name}</strong>
                <span className={styles.price}>
                  {new Intl.NumberFormat("vi-VN", {
                    style: "currency",
                    currency: "VND",
                  }).format(item.priceVnd)}
                </span>
              </Link>
            ))}
          </div>
        </>
      )}
      <RecommendationPreferences />
    </section>
  );
}

export function RecommendationPreferences() {
  const auth = useAuth();
  const state = useRecommendation();
  const [saving, setSaving] = useState(false);
  if (!recommendationsEnabled || auth.isEmployee) return null;
  async function toggle(value: boolean) {
    setSaving(true);
    try {
      await state.setConsent(value);
    } catch {
      /* The provider renders the retry message. */
    } finally {
      setSaving(false);
    }
  }
  return (
    <details className={styles.preferences}>
      <summary>Tùy chọn gợi ý và quyền riêng tư</summary>
      <p>
        Gợi ý phổ biến dành cho mọi người. Khi bật cá nhân hóa, lịch sử xem,
        thêm giỏ và mua hàng được dùng để chọn sản phẩm phù hợp với bạn.
      </p>
      {!auth.isCustomer && (
        <p>
          <Link href="/login?returnUrl=%2Faccount%2Fprofile">Đăng nhập</Link> để
          bật cá nhân hóa.
        </p>
      )}
      {auth.isCustomer && !state.consent && !state.consentError && (
        <p role="status">Đang tải lựa chọn cá nhân hóa…</p>
      )}
      {auth.isCustomer && state.consent ? (
        <label>
          <input
            type="checkbox"
            checked={state.consent.enabled}
            disabled={saving}
            onChange={(event) => {
              void toggle(event.target.checked);
            }}
          />{" "}
          Cá nhân hóa gợi ý bằng lịch sử xem, thêm giỏ và mua hàng của tôi.
        </label>
      ) : null}
      <label>
        <input
          type="checkbox"
          checked={state.measurement}
          onChange={(event) => state.setMeasurement(event.target.checked)}
        />{" "}
        Cho phép đo hiệu quả gợi ý bằng Google Analytics.
      </label>
      {state.consentError ? <p role="status">{state.consentError}</p> : null}
    </details>
  );
}
