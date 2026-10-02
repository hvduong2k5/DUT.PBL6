import type { Metadata } from "next";
import { ShippingWorkspace } from "./shipping-workspace";

export const metadata: Metadata = { title: "Quản lý giao vận" };
export default function ShippingPage() { return <ShippingWorkspace />; }
