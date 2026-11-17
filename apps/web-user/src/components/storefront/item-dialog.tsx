"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useAuth } from "@/components/auth/auth-provider";
import { useCartSummary } from "@/components/cart/cart-summary-provider";
import { useRecommendation } from "@/components/recommendation/recommendation-provider";
import { ProductImage } from "@/components/product/product-image";
import { cartService } from "@/services/cart-service";
import { productDetailService } from "@/services/product-detail-service";
import type { ProductDetail } from "@/lib/product-detail/types";
import { StorefrontDialog } from "./dialog";
import { type HomeItem, money } from "./items";

export function ItemDialog({
  item,
  close,
}: {
  item: HomeItem;
  close: () => void;
}) {
  const [detail, setDetail] = useState<ProductDetail>();
  const [failed, setFailed] = useState(false);
  const [skuId, select] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const { isEmployee } = useAuth();
  const { updateFromCart } = useCartSummary();
  const tracking = useRecommendation();
  useEffect(() => {
    let active = true;
    productDetailService
      .getBySlug(item.slug)
      .then((result) => {
        if (active) {
          setDetail(result);
          select(
            result.skus.find((s) => s.availability === "AVAILABLE")?.skuId ??
              "",
          );
        }
      })
      .catch(() => {
        if (active) setFailed(true);
      });
    return () => {
      active = false;
    };
  }, [item.slug]);
  const sku = detail?.skus.find((s) => s.skuId === skuId);
  async function add() {
    if (isEmployee || !sku || sku.availability !== "AVAILABLE" || busy) return;
    setBusy(true);
    try {
      const cart = await cartService.addItem({ skuId: sku.skuId, quantity: 1 });
      updateFromCart(cart);
      tracking.record(sku.skuId, "ADD_TO_CART");
      setMessage("Đã thêm vào giỏ quà.");
    } catch {
      setMessage("Chưa thể thêm vào giỏ. Vui lòng thử lại.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <StorefrontDialog title="Chi tiết thức quà Cố đô" close={close} fullPage>
      <div className="item-detail-grid">
        <ProductImage
          src={item.image}
          width={450}
          height={360}
          alt={item.name}
        />
        <div>
          <h3>{detail?.name ?? item.name}</h3>
          <p className="price-tag">{money(sku?.priceVnd ?? item.amount)}</p>
          <p>{detail?.shortDescription ?? item.meta}</p>
          {detail?.longDescription && <p>{detail.longDescription}</p>}
          {detail && (
            <label className="field-group">
              Chọn quy cách đóng gói
              <select
                className="field"
                value={skuId}
                onChange={(e) => {
                  select(e.target.value);
                  setMessage("");
                }}
              >
                {!skuId && (
                  <option value="">Chưa có quy cách có thể mua</option>
                )}
                {detail.skus.map((s) => (
                  <option
                    key={s.skuId}
                    value={s.skuId}
                    disabled={s.availability !== "AVAILABLE"}
                  >
                    {s.label}
                  </option>
                ))}
              </select>
            </label>
          )}
          {!detail && (
            <p role="status">
              {failed
                ? "Chưa tải được quy cách sản phẩm."
                : "Đang tải quy cách…"}
            </p>
          )}
          <div className="detail-actions">
            {!isEmployee && (
              <button
                className="btn btn--brown"
                disabled={busy || !sku}
                onClick={add}
              >
                Thêm vào giỏ quà →
              </button>
            )}
            <Link className="btn btn--paper" href={"/products/" + item.slug}>
              Trang sản phẩm
            </Link>
          </div>
          <p role="status">{message}</p>
        </div>
      </div>
    </StorefrontDialog>
  );
}
