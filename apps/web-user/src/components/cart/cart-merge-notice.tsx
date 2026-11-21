"use client";

import { useCartSummary } from "./cart-summary-provider";

export function CartMergeNotice() {
  const { mergeResult, mergeError, mergePending, retryMerge } = useCartSummary();
  if (mergePending) return <section className="status-notice info" role="status">Đang gộp giỏ hàng khách…</section>;
  if (mergeError) return <section className="status-notice error" role="alert"><p>{mergeError}</p><button className="secondary-button" type="button" onClick={() => void retryMerge()}>Thử gộp lại</button></section>;
  if (!mergeResult) return null;
  const reasons: Record<string,string> = { SKU_NOT_AVAILABLE: "Sản phẩm không còn bán.", QUANTITY_LIMIT: "Đã áp dụng giới hạn số lượng.", CART_ITEM_LIMIT: "Giỏ đã đủ số loại sản phẩm." };
  return <section className="status-notice info" aria-label="Kết quả gộp giỏ hàng"><strong>Đã gộp giỏ hàng khách</strong><ul>{mergeResult.items.map((item) => <li key={item.sku_code}>{mergeResult.cart.items.find((line) => line.sku_code===item.sku_code)?.product_name || item.sku_code}: {item.status==="dropped" ? "Đã bỏ" : item.status==="adjusted" ? `Đã điều chỉnh · Số lượng ${item.quantity}` : `Đã gộp · Số lượng ${item.quantity}`} {item.reason ? reasons[item.reason] ?? "Vui lòng kiểm tra lại sản phẩm." : ""}</li>)}</ul></section>;
}
