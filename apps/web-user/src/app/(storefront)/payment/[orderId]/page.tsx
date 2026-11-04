import type { Metadata } from "next";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { PaymentFlow } from "./payment-flow";

export const metadata: Metadata = {
  title: "Thanh toán đơn hàng",
  description: "Theo dõi hướng dẫn và trạng thái thanh toán cho đơn hàng Ô Mạ Huế."
};

export default async function PaymentPage({ params }: { params: Promise<{ orderId: string }> }) {
  const { orderId } = await params;
  return <><SiteHeader /><PaymentFlow orderId={orderId} /><SiteFooter /></>;
}
