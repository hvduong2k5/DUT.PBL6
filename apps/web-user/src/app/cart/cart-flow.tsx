"use client";

import Image from "next/image";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { getInitiallySelectedItemIds, getSelectedCartSummary, reconcileSelectedItemIds } from "@/lib/cart/selection";
import type { Cart, CartItem } from "@/lib/cart/types";
import { CartApiError } from "@/lib/cart/types";
import { MAX_CART_ITEM_QUANTITY } from "@/lib/cart/validation";
import { cartService } from "@/services/cart-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

function CartLoading() {
  return <div className="cart-loading" aria-busy="true" aria-label="Đang tải giỏ hàng"><div /><div /><div /></div>;
}

export function CartFlow() {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [cart, setCart] = useState<Cart | null>(null);
  const [loading, setLoading] = useState(true);
  const [pageError, setPageError] = useState("");
  const [pendingItemId, setPendingItemId] = useState<string>();
  const [lineErrors, setLineErrors] = useState<Record<string, string>>({});
  const [draftQuantities, setDraftQuantities] = useState<Record<string, string>>({});
  const [selectedItemIds, setSelectedItemIds] = useState<string[]>([]);
  const [statusMessage, setStatusMessage] = useState("");
  const [retryKey, setRetryKey] = useState(0);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setPageError("");
    setStatusMessage("");
    cartService.get(scenario).then((payload) => {
      if (active) {
        setCart(payload);
        setDraftQuantities(Object.fromEntries(payload.items.map((item) => [item.itemId, String(item.quantity)])));
        setSelectedItemIds(getInitiallySelectedItemIds(payload.items));
      }
    }).catch((cause: unknown) => {
      if (!active) return;
      setPageError(cause instanceof CartApiError ? cause.message : "Không thể tải giỏ hàng lúc này.");
    }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [retryKey, scenario]);

  async function updateQuantity(item: CartItem, quantity: number) {
    if (!Number.isInteger(quantity) || quantity < 1 || quantity > MAX_CART_ITEM_QUANTITY) {
      setLineErrors((current) => ({ ...current, [item.itemId]: `Số lượng phải từ 1 đến ${MAX_CART_ITEM_QUANTITY}.` }));
      return;
    }
    setPendingItemId(item.itemId);
    setLineErrors((current) => ({ ...current, [item.itemId]: "" }));
    setStatusMessage("");
    try {
      const nextCart = await cartService.updateItem(item.itemId, { quantity }, scenario);
      setCart(nextCart);
      setDraftQuantities(Object.fromEntries(nextCart.items.map((cartItem) => [cartItem.itemId, String(cartItem.quantity)])));
      setSelectedItemIds((current) => reconcileSelectedItemIds(nextCart.items, current));
      setStatusMessage(`Đã cập nhật ${item.productName} thành ${quantity} sản phẩm.`);
    } catch (cause) {
      setLineErrors((current) => ({ ...current, [item.itemId]: cause instanceof CartApiError ? cause.message : "Không thể cập nhật số lượng." }));
    } finally {
      setPendingItemId(undefined);
    }
  }

  async function removeItem(item: CartItem) {
    setPendingItemId(item.itemId);
    setLineErrors((current) => ({ ...current, [item.itemId]: "" }));
    setStatusMessage("");
    try {
      const nextCart = await cartService.removeItem(item.itemId, scenario);
      setCart(nextCart);
      setDraftQuantities(Object.fromEntries(nextCart.items.map((cartItem) => [cartItem.itemId, String(cartItem.quantity)])));
      setSelectedItemIds((current) => reconcileSelectedItemIds(nextCart.items, current));
      setStatusMessage(`Đã xóa ${item.productName} khỏi giỏ hàng.`);
    } catch (cause) {
      setLineErrors((current) => ({ ...current, [item.itemId]: cause instanceof CartApiError ? cause.message : "Không thể xóa sản phẩm." }));
    } finally {
      setPendingItemId(undefined);
    }
  }

  if (loading) return <main className="cart-page"><CartLoading /></main>;

  if (pageError) {
    return (
      <main className="cart-page">
        <section className="cart-error-state" role="alert">
          <span aria-hidden="true">!</span>
          <p className="eyebrow">Kết nối gián đoạn</p>
          <h1>Chưa thể mở giỏ hàng</h1>
          <p>{pageError}</p>
          <div><button type="button" className="primary-button" onClick={() => setRetryKey((value) => value + 1)}>Thử tải lại</button><Link href="/products">Tiếp tục chọn sản phẩm</Link></div>
        </section>
      </main>
    );
  }

  if (!cart) return null;

  const availableItemIds = cart.items.filter((item) => item.isAvailable).map((item) => item.itemId);
  const allAvailableSelected = availableItemIds.length > 0 && availableItemIds.every((itemId) => selectedItemIds.includes(itemId));
  const selectedSummary = getSelectedCartSummary(cart.items, selectedItemIds);

  function toggleItemSelection(item: CartItem) {
    if (!item.isAvailable) return;
    setSelectedItemIds((current) => current.includes(item.itemId)
      ? current.filter((itemId) => itemId !== item.itemId)
      : [...current, item.itemId]);
    setStatusMessage("");
  }

  function toggleAllAvailable() {
    setSelectedItemIds(allAvailableSelected ? [] : availableItemIds);
    setStatusMessage("");
  }

  return (
    <main className="cart-page">
      <nav className="cart-breadcrumbs" aria-label="Breadcrumb"><Link href="/">Trang chủ</Link><span aria-hidden="true">/</span><strong>Giỏ hàng</strong></nav>

      <ol className="cart-steps" aria-label="Tiến trình mua hàng">
        <li className="active" aria-current="step"><span>1</span><div><strong>Giỏ hàng</strong><small>Kiểm tra sản phẩm</small></div></li>
        <li><span>2</span><div><strong>Thông tin đặt hàng</strong><small>Feature tiếp theo</small></div></li>
        <li><span>3</span><div><strong>Hoàn tất</strong><small>Chưa phát sinh Order</small></div></li>
      </ol>

      <header className="cart-heading">
        <div><p className="eyebrow">Shopping Cart MVP</p><h1>Giỏ hàng của bạn</h1></div>
        <p>{cart.itemCount ? `${cart.itemCount} sản phẩm theo đúng quy cách đã chọn` : "Chưa có sản phẩm nào được chọn"}</p>
      </header>

      <p className="sr-only" aria-live="polite">{statusMessage}</p>

      {cart.items.length === 0 ? (
        <section className="cart-empty-state">
          <span aria-hidden="true">◇</span>
          <p className="eyebrow">Giỏ hàng trống</p>
          <h2>Chọn một thức quà Huế phù hợp</h2>
          <p>Sản phẩm chỉ được thêm sau khi bạn xác nhận rõ SKU và số lượng tại trang chi tiết.</p>
          <Link className="primary-button" href="/products">Khám phá sản phẩm</Link>
        </section>
      ) : (
        <div className="cart-layout">
          <section className="cart-lines" aria-label="Sản phẩm trong giỏ">
            <div className="cart-selection-toolbar">
              <label>
                <input type="checkbox" checked={allAvailableSelected} onChange={toggleAllAvailable} aria-label="Chọn tất cả sản phẩm khả dụng để thanh toán" />
                Chọn tất cả sản phẩm khả dụng
              </label>
              <small>Đã chọn {selectedSummary.lineCount}/{availableItemIds.length} dòng</small>
            </div>
            {cart.items.map((item) => {
              const pending = pendingItemId === item.itemId;
              const selected = selectedItemIds.includes(item.itemId);
              return (
                <article className={`cart-line ${selected ? "selected" : "not-selected"} ${!item.isAvailable ? "unavailable" : ""}`} key={item.itemId}>
                  <label className="cart-line-selector">
                    <input type="checkbox" checked={selected} disabled={!item.isAvailable} onChange={() => toggleItemSelection(item)} aria-label={`Chọn ${item.productName} để thanh toán`} />
                  </label>
                  <Link className="cart-line-image" href={`/products/${item.productSlug}`} aria-label={`Mở ${item.productName}`}>
                    <Image src={item.imageUrl} alt={item.imageAlt} fill sizes="(max-width: 640px) 108px, 150px" />
                  </Link>
                  <div className="cart-line-info">
                    <span>{item.isAvailable ? "Có thể mua" : "Cần xử lý"}</span>
                    <Link href={`/products/${item.productSlug}`}><h2>{item.productName}</h2></Link>
                    <p>{item.skuLabel}</p>
                    <small>{item.weightGrams}g · {item.flavor || "Nguyên bản"} · {item.packageType}</small>
                    {item.priceChanged ? <div className="cart-line-notice" role="status">Giá đã đổi từ <s>{VND.format(item.previousUnitPriceVnd ?? 0)}</s> sang giá hiện tại.</div> : null}
                    {!item.isAvailable ? <div className="cart-line-warning" role="alert">{item.unavailableReason || "Quy cách này không còn khả dụng."} <Link href={`/products/${item.productSlug}`}>Mở trang sản phẩm</Link></div> : null}
                    {lineErrors[item.itemId] ? <div className="cart-line-warning" role="alert">{lineErrors[item.itemId]}</div> : null}
                  </div>
                  <div className="cart-line-price"><span>Đơn giá</span><strong>{VND.format(item.unitPriceVnd)}</strong></div>
                  <div className="cart-line-quantity">
                    <span>Số lượng</span>
                    <div>
                      <button type="button" aria-label={`Giảm số lượng ${item.productName}`} disabled={pending || !item.isAvailable || item.quantity <= 1} onClick={() => updateQuantity(item, item.quantity - 1)}>−</button>
                      <input aria-label={`Số lượng ${item.productName}`} type="number" min="1" max={MAX_CART_ITEM_QUANTITY} value={draftQuantities[item.itemId] ?? String(item.quantity)} disabled={pending || !item.isAvailable} onChange={(event) => setDraftQuantities((current) => ({ ...current, [item.itemId]: event.currentTarget.value }))} onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); updateQuantity(item, Number(event.currentTarget.value)); } }} />
                      <button type="button" aria-label={`Tăng số lượng ${item.productName}`} disabled={pending || !item.isAvailable || item.quantity >= MAX_CART_ITEM_QUANTITY} onClick={() => updateQuantity(item, item.quantity + 1)}>+</button>
                      <button className="cart-quantity-apply" type="button" aria-label={`Áp dụng số lượng cho ${item.productName}`} disabled={pending || !item.isAvailable || draftQuantities[item.itemId] === String(item.quantity)} onClick={() => updateQuantity(item, Number(draftQuantities[item.itemId]))}>✓</button>
                    </div>
                    {pending ? <small role="status">Đang cập nhật…</small> : null}
                  </div>
                  <div className="cart-line-total"><span>Thành tiền</span><strong>{item.lineSubtotalVnd === null ? "Không tính" : VND.format(item.lineSubtotalVnd ?? 0)}</strong></div>
                  <button className="cart-remove" type="button" disabled={pending} onClick={() => removeItem(item)} aria-label={`Xóa ${item.productName} khỏi giỏ`}>Xóa</button>
                </article>
              );
            })}
            <Link className="cart-continue" href="/products">← Tiếp tục chọn sản phẩm</Link>
          </section>

          <aside className="cart-summary" aria-labelledby="cart-summary-title">
            <p className="eyebrow">Tóm tắt</p>
            <h2 id="cart-summary-title">Tạm tính giỏ hàng</h2>
            <dl className="cart-overview-summary"><div><dt>Toàn bộ giỏ hàng</dt><dd>{cart.itemCount} sản phẩm</dd></div><div><dt>Giá trị hàng có thể mua</dt><dd>{VND.format(cart.subtotalVnd)}</dd></div></dl>
            <dl className="cart-selection-summary"><div><dt>Đã chọn thanh toán</dt><dd>{selectedSummary.lineCount} dòng · {selectedSummary.itemCount} sản phẩm</dd></div><div className="total"><dt>Tạm tính hàng đã chọn</dt><dd>{VND.format(selectedSummary.subtotalVnd)}</dd></div></dl>
            {cart.hasBlockingIssues ? <div className="cart-blocking-note" role="status"><strong>Có sản phẩm không khả dụng</strong><p>Các dòng này không thể chọn và không được tính vào phần chuẩn bị Checkout.</p></div> : null}
            <p className="cart-summary-disclaimer">Chưa gồm phí vận chuyển, ưu đãi và tổng thanh toán. Chỉ dòng đã chọn được chuẩn bị cho Checkout; giỏ hàng không giữ tồn hoặc cố định giá.</p>
            <button type="button" className="cart-checkout" disabled={selectedSummary.lineCount === 0} onClick={() => setStatusMessage(`Đã chọn ${selectedSummary.lineCount} dòng (${selectedSummary.itemCount} sản phẩm). Checkout sẽ được hoàn thiện ở feature tiếp theo; chưa có Order, reservation hay payment nào được tạo.`)}>{selectedSummary.lineCount === 0 ? "Chọn sản phẩm để tiếp tục" : `Checkout ${selectedSummary.lineCount} dòng — sắp ra mắt`}</button>
            {statusMessage ? <p className="cart-checkout-note" role="status">{statusMessage}</p> : null}
            <div className="cart-boundary"><strong>Phạm vi hiện tại</strong><p>Giá và khả dụng được kiểm tra lại. Shipping, voucher, payment và đặt hàng chưa hoạt động.</p></div>
          </aside>
        </div>
      )}
    </main>
  );
}
