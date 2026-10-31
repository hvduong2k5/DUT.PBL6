import type { Metadata } from "next";
import { OrderWorkspace } from "./order-workspace";

export const metadata: Metadata = { title: "Quản lý đơn hàng" };

export default function OrderPage() { return <OrderWorkspace />; }
