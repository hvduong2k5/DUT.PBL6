"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { PublicApiError } from "@/lib/api-contract";
import type { Cart, CartItem } from "@/lib/cart/types";
import { MAX_CART_ITEM_QUANTITY } from "@/lib/cart/validation";
import { cartService } from "@/services/cart-service";
import { useCartSummary } from "@/components/cart/cart-summary-provider";
import { ProductImage } from "@/components/product/product-image";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

function CartLoading() {
  return <div className="cart-loading" aria-busy="true" aria-label="Đang tải giỏ hàng"><div /><div /><div /></div>;
}

export function CartFlow() {
  const { updateFromCart } = useCartSummary();
  const router = useRouter();
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [cart, setCart] = useState<Cart | null>(null);
  const [loading, setLoading] = useState(true);
  const [pageError, setPageError] = useState("");
  const [pendingItemId, setPendingItemId] = useState<string>();
  const [lineErrors, setLineErrors] = useState<Record<string, string>>({});
  const [draftQuantities, setDraftQuantities] = useState<Record<string, string>>({});
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
        updateFromCart(payload);
        setDraftQuantities(Object.fromEntries(payload.items.map((item) => [item.itemId, String(item.quantity)])));
      }
    }).catch((cause: unknown) => {
      if (!active) return;
      setPageError(cause instanceof PublicApiError ? cause.message : "Không thể tải giỏ hàng lúc này.");
    }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [retryKey, scenario, updateFromCart]);

  async function updateQuantity(item: CartItem, quantity: number) {
    if (!Number.isInteger(quantity) || quantity < 1 || quantity > MAX_CART_ITEM_QUANTITY) {
      setLineErrors((current) => ({ ...current, [item.itemId]: `Số lượng phải từ 1 đến ${MAX_CART_ITEM_QUANTITY}.` }));
      return;
    }
    setPendingItemId(item.itemId);
    setLineErrors((current) => ({ ...current, [item.itemId]: "" }));
    setStatusMessage("");
    try {
      const nextCart = await cartService.updateItem(item.itemId, { quantity }, cart?.revision ?? 0, scenario);
      setCart(nextCart);
      updateFromCart(nextCart);
      setDraftQuantities(Object.fromEntries(nextCart.items.map((cartItem) => [cartItem.itemId, String(cartItem.quantity)])));
      setStatusMessage(`Đã cập nhật ${item.productName} thành ${quantity} sản phẩm.`);
    } catch (cause) {
      setLineErrors((current) => ({
        ...current,
        [item.itemId]: cause instanceof PublicApiError
          ? cause.status === 409 ? "Giỏ hàng đã thay đổi ở nơi khác. Hãy tải lại trước khi cập nhật." : cause.message
          : "Không thể cập nhật số lượng."
      }));
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
      updateFromCart(nextCart);
      setDraftQuantities(Object.fromEntries(nextCart.items.map((cartItem) => [cartItem.itemId, String(cartItem.quantity)])));
      setStatusMessage(`Đã xóa ${item.productName} khỏi giỏ hàng.`);
    } catch (cause) {
      setLineErrors((current) => ({ ...current, [item.itemId]: cause instanceof PublicApiError ? cause.message : "Không thể xóa sản phẩm." }));
    } finally {
      setPendingItemId(undefined);
    }
  }

  async function clearCart() {
    if (!window.confirm("Bạn có chắc muốn xóa toàn bộ sản phẩm khỏi giỏ hàng?")) return;
    setPendingItemId("__clear-cart__");
    setStatusMessage("");
    try {
      const nextCart = await cartService.clear(scenario);
      setCart(nextCart);
      updateFromCart(nextCart);
      setDraftQuantities({});
      setStatusMessage("Đã dọn toàn bộ giỏ hàng.");
    } catch (cause) {
      setPageError(cause instanceof PublicApiError ? cause.message : "Không thể dọn giỏ hàng lúc này.");
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

  function continueToCheckout() {
    if (!cart || !cart.items.length || cart.hasBlockingIssues) return;
    const params = new URLSearchParams();
    if (scenario) params.set("cartScenario", scenario);
    router.push(params.size ? `/checkout?${params}` : "/checkout");
  }

  return (
    <main className="cart-page">
      <nav className="cart-breadcrumbs" aria-label="Breadcrumb"><Link href="/">Trang chủ</Link><span aria-hidden="true">/</span><strong>Giỏ hàng</strong></nav>

      <ol className="cart-steps" aria-label="Tiến trình mua hàng">
        <li className="active" aria-current="step"><span>1</span><div><strong>Giỏ hàng</strong><small>Kiểm tra sản phẩm</small></div></li>
        <li><span>2</span><div><strong>Thông tin đặt hàng</strong><small>Giao hàng &amp; xác nhận</small></div></li>
        <li><span>3</span><div><strong>Hoàn tất</strong><small>Chờ thanh toán</small></div></li>
      </ol>

      <header className="cart-heading">
        <div><p className="eyebrow">Shopping Cart MVP</p><h1>Giỏ hàng của bạn</h1></div>
        <p>{cart.itemCount ? `${cart.itemCount} sản phẩm sẽ được báo giá cùng một revision` : "Chưa có sản phẩm nào trong giỏ"}</p>
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
            <div className="cart-selection-toolbar"><strong>Checkout toàn bộ giỏ · Revision {cart.revision ?? "—"}</strong><div className="cart-selection-actions"><small>{cart.items.length} dòng · {cart.itemCount} sản phẩm</small><button type="button" disabled={pendingItemId === "__clear-cart__"} onClick={clearCart}>{pendingItemId === "__clear-cart__" ? "Đang dọn…" : "Dọn giỏ hàng"}</button></div></div>
            {cart.items.map((item) => {
              const pending = pendingItemId === item.itemId;
              return (
                <article className={`cart-line selected ${!item.isAvailable ? "unavailable" : ""}`} key={item.itemId}>
                  <Link className="cart-line-image" href={`/products/${item.productSlug}`} aria-label={`Mở ${item.productName}`}>
                    <ProductImage src={item.imageUrl} alt={item.imageAlt} fill sizes="(max-width: 640px) 108px, 150px" />
                  </Link>
                  <div className="cart-line-info">
                    <span>{item.availabilityUnverified ? "Sẽ kiểm tra khi thanh toán" : item.isAvailable ? "Có thể mua" : "Cần xử lý"}</span>
                    <Link href={`/products/${item.productSlug}`}><h2>{item.productName}</h2></Link>
                    <p>{item.skuLabel}</p>
                    <small>{item.weightGrams ? `${item.weightGrams}g · ` : ""}Mã SKU: {item.skuId}</small>
                    {item.priceChanged ? <div className="cart-line-notice" role="status">Giá đã đổi từ <s>{VND.format(item.previousUnitPriceVnd ?? 0)}</s> sang giá hiện tại.</div> : null}
                    {!item.isAvailable ? <div className="cart-line-warning" role="alert">{item.unavailableReason || "Quy cách này không còn khả dụng."} <Link href={`/products/${item.productSlug}`}>Mở trang sản phẩm</Link></div> : null}
                    {lineErrors[item.itemId] ? <div className="cart-line-warning" role="alert">{lineErrors[item.itemId]}</div> : null}
                  </div>
                  <div className="cart-line-price"><span>Đơn giá</span><strong>{item.priceKnown === false ? "Chờ báo giá" : VND.format(item.unitPriceVnd)}</strong></div>
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
                  <div className="cart-line-total"><span>Thành tiền</span><strong>{item.lineSubtotalVnd === null ? "Chờ báo giá" : VND.format(item.lineSubtotalVnd ?? 0)}</strong></div>
                  <button className="cart-remove" type="button" disabled={pending} onClick={() => removeItem(item)} aria-label={`Xóa ${item.productName} khỏi giỏ`}>Xóa</button>
                </article>
              );
            })}
            <Link className="cart-continue" href="/products">← Tiếp tục chọn sản phẩm</Link>
          </section>

          <aside className="cart-summary" aria-labelledby="cart-summary-title">
            <p className="eyebrow">Tóm tắt</p>
            <h2 id="cart-summary-title">Tạm tính giỏ hàng</h2>
            <dl className="cart-overview-summary"><div><dt>Toàn bộ giỏ hàng</dt><dd>{cart.items.length} dòng · {cart.itemCount} sản phẩm</dd></div><div><dt>Tạm tính hiện tại</dt><dd>{cart.pricingComplete === false ? "Chờ báo giá" : VND.format(cart.subtotalVnd)}</dd></div></dl>
            <dl className="cart-selection-summary"><div><dt>Phạm vi Checkout</dt><dd>Toàn bộ giỏ</dd></div><div className="total"><dt>Revision gửi báo giá</dt><dd>{cart.revision ?? "—"}</dd></div></dl>
            {cart.hasBlockingIssues ? <div className="cart-blocking-note" role="alert"><strong>Cần xử lý giỏ hàng trước khi tiếp tục</strong><p>Checkout không hỗ trợ bỏ qua từng dòng. Hãy xóa hoặc thay thế sản phẩm không khả dụng rồi báo giá lại toàn bộ giỏ.</p></div> : null}
            <p className="cart-summary-disclaimer">Quote sẽ đọc toàn bộ dòng từ giỏ server tại revision hiện tại. Giỏ hàng chưa giữ tồn hoặc cố định giá; phí vận chuyển và tổng thanh toán do quote trả về.</p>
            <button type="button" className="cart-checkout" disabled={!cart.items.length || cart.hasBlockingIssues} onClick={continueToCheckout}>{cart.hasBlockingIssues ? "Xử lý sản phẩm không khả dụng" : `Thanh toán toàn bộ ${cart.items.length} dòng →`}</button>
            {statusMessage ? <p className="cart-checkout-note" role="status">{statusMessage}</p> : null}
            <div className="cart-boundary"><strong>Trước khi đặt hàng</strong><p>Checkout sẽ kiểm tra lại giá, khả dụng, địa chỉ và phí vận chuyển. Payment chỉ bắt đầu sau khi Order hợp lệ được tạo.</p></div>
          </aside>
        </div>
      )}
    </main>
  );
}
