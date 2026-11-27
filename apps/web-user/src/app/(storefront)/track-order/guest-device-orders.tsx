"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { listGuestOrders, type GuestDeviceOrder } from "@/lib/orders/guest-orders";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

export function GuestDeviceOrders() {
  const [orders, setOrders] = useState<GuestDeviceOrder[]>([]);

  useEffect(() => { void Promise.resolve().then(() => setOrders(listGuestOrders())); }, []);

  if (!orders.length) return null;

  return (
    <section className="guest-device-orders" aria-labelledby="guest-device-orders-title">
      <h2 id="guest-device-orders-title">Đơn đã đặt trên thiết bị này</h2>
      <p>Các đơn bạn đã đặt với tư cách khách trên trình duyệt này. Bấm vào một đơn để xem chi tiết — chi tiết mở được khi phiên khách còn hiệu lực (24 giờ từ lúc đặt hàng).</p>
      <ul>
        {orders.map((order) => (
          <li key={order.orderId}>
            <Link href={`/orders/${encodeURIComponent(order.orderId)}`}>
              <strong>{order.orderCode}</strong>
              <span>{new Date(order.placedAt).toLocaleString("vi-VN")}{order.totalVnd !== undefined ? ` · ${VND.format(order.totalVnd)}` : ""}</span>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
