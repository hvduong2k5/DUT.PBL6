import type { Metadata } from "next";
import { Suspense } from "react";
import { OrdersFlow } from "./orders-flow";

export const metadata: Metadata = {
  title: "Đơn hàng của tôi",
  description: "Xem lịch sử và trạng thái các Order thuộc tài khoản Tri Kỷ."
};

export default function OrdersPage() {
  return <Suspense fallback={null}><OrdersFlow /></Suspense>;
}
