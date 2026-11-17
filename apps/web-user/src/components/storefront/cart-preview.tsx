"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import { useCartSummary } from "@/components/cart/cart-summary-provider";
import { cartService } from "@/services/cart-service";
import type { Cart, CartItem } from "@/lib/cart/types";
import { money } from "./items";

export function CartPreview({ close }: { close: () => void }) {
  const { customerAccess, status } = useCustomerSession();
  const { updateFromCart } = useCartSummary();
  const [cart, setCart] = useState<Cart>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    if (!customerAccess) return;
    let active = true;
    cartService
      .get()
      .then((data) => {
        if (active) setCart(data);
      })
      .catch(() => {
        if (active) setError("Chưa tải được giỏ quà.");
      });
    return () => {
      active = false;
    };
  }, [customerAccess]);
  async function change(line: CartItem, quantity: number) {
    if (!cart || busy) return;
    setBusy(true);
    setError("");
    try {
      const next = quantity
        ? await cartService.updateItem(
            line.itemId,
            { quantity },
            cart.revision ?? 0,
          )
        : await cartService.removeItem(line.itemId, cart.revision);
      setCart(next);
      updateFromCart(next);
    } catch {
      setError(
        "Giỏ quà đã thay đổi hoặc chưa cập nhật được. Mở giỏ hàng để kiểm tra lại.",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="filter-form">
      {status === "loading" || (customerAccess && !cart && !error) ? (
        <p role="status">Đang tải giỏ quà…</p>
      ) : !customerAccess ? (
        <p>Đăng nhập để xem các thức quà đã chọn.</p>
      ) : cart?.items.length === 0 ? (
        <p>Giỏ quà đang trống.</p>
      ) : null}
      {cart?.items.map((line) => (
        <div className="between" key={line.itemId}>
          <div>
            <strong>{line.productName}</strong>
            <p>{line.skuLabel}</p>
          </div>
          <div className="row">
            <button
              className="chip-btn"
              disabled={busy}
              aria-label={"Giảm số lượng " + line.productName}
              onClick={() => change(line, line.quantity - 1)}
            >
              −
            </button>
            {line.quantity}
            <button
              className="chip-btn"
              disabled={busy}
              aria-label={"Tăng số lượng " + line.productName}
              onClick={() => change(line, line.quantity + 1)}
            >
              +
            </button>
          </div>
        </div>
      ))}
      {cart?.pricingComplete && (
        <strong>Tạm tính · {money(cart.subtotalVnd)}</strong>
      )}
      {error && <p role="alert">{error}</p>}
      <div className="row">
        <Link className="btn btn--paper" href="/cart" onClick={close}>
          Xem giỏ hàng
        </Link>
        {!!cart?.items.length && !cart.hasBlockingIssues && (
          <Link className="btn btn--brown" href="/checkout" onClick={close}>
            Tiến hành đặt quà →
          </Link>
        )}
      </div>
    </div>
  );
}
