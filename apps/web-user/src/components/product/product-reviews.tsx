"use client";

import Link from "next/link";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import type { ProductReview, ProductReviews } from "@/lib/customer-core/types";
import { CustomerCoreApiError } from "@/lib/customer-core/types";
import { customerCoreService } from "@/services/customer-core-service";

const DATE = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "long", year: "numeric" });

function Stars({ value, label }: { value: number; label: string }) {
  return (
    <span className="review-stars" aria-label={label}>
      {Array.from({ length: 5 }, (_, index) => <span className={index < Math.round(value) ? "filled" : ""} aria-hidden="true" key={index}>★</span>)}
    </span>
  );
}

function ReviewCard({ review }: { review: ProductReview }) {
  return (
    <article className="review-card">
      <header>
        <div className="review-avatar" aria-hidden="true">{review.customer_name.trim().charAt(0).toUpperCase()}</div>
        <div><strong>{review.customer_name}</strong><Stars value={review.rating} label={`${review.rating} trên 5 sao`} /></div>
        <time dateTime={review.created_at}>{DATE.format(new Date(review.created_at))}</time>
      </header>
      {review.is_verified_purchase ? <span className="verified-review">✓ Đã mua sản phẩm</span> : null}
      <p>{review.comment}</p>
      {review.media_urls.length ? <div className="review-media-links">{review.media_urls.map((url, index) => <a href={url} target="_blank" rel="noreferrer" key={url}>Xem ảnh đánh giá {index + 1} ↗</a>)}</div> : null}
      {review.seller_reply ? <blockquote><strong>Ô Mạ phản hồi</strong><p>{review.seller_reply}</p></blockquote> : null}
    </article>
  );
}

export function ProductReviewsSection({ productId, productName, mockScenario }: { productId: string; productName: string; mockScenario?: string }) {
  const { status, capabilityStatus, hasCapability } = useCustomerSession();
  const [data, setData] = useState<ProductReviews>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [retryKey, setRetryKey] = useState(0);
  const [formOpen, setFormOpen] = useState(false);
  const [rating, setRating] = useState(5);
  const [comment, setComment] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitMessage, setSubmitMessage] = useState("");

  useEffect(() => {
    let active = true;
    customerCoreService.getProductReviews(productId, mockScenario)
      .then((payload) => { if (active) setData(payload); })
      .catch((cause: unknown) => {
        if (!active) return;
        setError(cause instanceof CustomerCoreApiError ? cause.message : "Chưa thể tải đánh giá sản phẩm.");
      })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [mockScenario, productId, retryKey]);

  function retryLoadingReviews() {
    setLoading(true);
    setError("");
    setRetryKey((value) => value + 1);
  }

  const averageLabel = useMemo(() => {
    if (!data) return "Chưa có điểm đánh giá";
    return `${data.average_rating.toLocaleString("vi-VN", { maximumFractionDigits: 1 })} trên 5`;
  }, [data]);

  async function submitReview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!hasCapability("REVIEW_CREATE")) {
      setSubmitMessage("Tài khoản hiện chưa có quyền gửi đánh giá.");
      return;
    }
    const normalized = comment.trim();
    if (normalized.length < 10) {
      setSubmitMessage("Nội dung đánh giá cần ít nhất 10 ký tự.");
      return;
    }
    setSubmitting(true);
    setSubmitMessage("");
    try {
      const created = await customerCoreService.createReview({ product_id: productId, rating, comment: normalized }, mockScenario);
      setData((current) => {
        if (!current) return { average_rating: created.rating, total_reviews: 1, reviews: [created] };
        const alreadyPresent = current.reviews.some((review) => review.review_id === created.review_id);
        return {
          ...current,
          total_reviews: alreadyPresent ? current.total_reviews : current.total_reviews + 1,
          reviews: alreadyPresent
            ? current.reviews.map((review) => review.review_id === created.review_id ? created : review)
            : [created, ...current.reviews]
        };
      });
      setComment("");
      setFormOpen(false);
      setSubmitMessage("Đánh giá đã được ghi nhận. Cảm ơn bạn đã chia sẻ trải nghiệm.");
    } catch (cause) {
      setSubmitMessage(cause instanceof CustomerCoreApiError ? cause.message : "Chưa thể gửi đánh giá lúc này.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="product-reviews" aria-labelledby="product-reviews-title">
      <div className="reviews-heading">
        <div><p className="eyebrow">Đánh giá đã xác thực</p><h2 id="product-reviews-title">Tri kỷ nói gì về thức quà này?</h2><p>Chia sẻ từ khách hàng đã trải nghiệm {productName}.</p></div>
        {capabilityStatus === "ready" && hasCapability("REVIEW_CREATE") ? <button type="button" className="secondary-button" onClick={() => { setFormOpen((open) => !open); setSubmitMessage(""); }}>{formOpen ? "Đóng biểu mẫu" : "Viết đánh giá"}</button> : status === "guest" ? <Link className="secondary-link" href="/login">Đăng nhập để đánh giá</Link> : null}
      </div>

      {formOpen ? <form className="review-form" onSubmit={submitReview}>
        <fieldset><legend>Mức độ hài lòng</legend><div className="rating-picker">{[1, 2, 3, 4, 5].map((value) => <button type="button" className={value <= rating ? "active" : ""} aria-label={`${value} sao`} aria-pressed={rating === value} onClick={() => setRating(value)} key={value}>★</button>)}</div></fieldset>
        <label htmlFor="review-comment">Cảm nhận của bạn</label>
        <textarea id="review-comment" value={comment} minLength={10} maxLength={1000} required placeholder="Hương vị, đóng gói hoặc trải nghiệm nào khiến bạn nhớ nhất?" onChange={(event) => setComment(event.target.value)} />
        <div><small>{comment.trim().length}/1000 ký tự</small><button type="submit" className="primary-button" disabled={submitting}>{submitting ? "Đang gửi…" : "Gửi đánh giá"}</button></div>
      </form> : null}
      {submitMessage ? <p className="review-submit-message" role="status">{submitMessage}</p> : null}

      {loading ? <div className="reviews-loading" aria-busy="true"><span /><span /><span /></div> : null}
      {!loading && error ? <div className="reviews-error" role="alert"><p>{error}</p><button type="button" onClick={retryLoadingReviews}>Thử lại</button></div> : null}
      {!loading && !error && data ? <div className="reviews-layout">
        <aside className="reviews-summary"><strong>{data.average_rating.toLocaleString("vi-VN", { maximumFractionDigits: 1 })}</strong><Stars value={data.average_rating} label={averageLabel} /><p>{data.total_reviews.toLocaleString("vi-VN")} đánh giá đã gửi</p><small>Đánh giá xác thực được gắn nhãn “Đã mua sản phẩm”.</small></aside>
        <div className="reviews-list">{data.reviews.length ? data.reviews.map((review) => <ReviewCard review={review} key={review.review_id} />) : <div className="reviews-empty"><strong>Chưa có đánh giá</strong><p>Hãy là người đầu tiên chia sẻ trải nghiệm về sản phẩm này.</p></div>}</div>
      </div> : null}
    </section>
  );
}
