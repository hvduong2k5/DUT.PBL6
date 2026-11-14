import type { Metadata } from "next";
import { FulfillmentWorkspace } from "@/features/employee/packing/fulfillment-workspace";

export const metadata: Metadata = { title: "Quản lý đóng gói" };
export const dynamic = "force-dynamic";
export default function PackingPage() { return <FulfillmentWorkspace management />; }
