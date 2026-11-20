import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { ShippingWorkspace } from "@/features/employee/shipping/shipping-workspace";
import Link from "next/link";

export const metadata: Metadata = { title: "Quản lý giao vận" };
export default function ShippingPage() { return <EmployeePermissionGate permission="SHIPMENT_MANAGEMENT_VIEW">{process.env.SHIPPING_COMPANY_BFF_ENABLED==="true"?<Link href="/admin/shipping/dispatch">Điều phối giao hàng nội bộ</Link>:null}<ShippingWorkspace /></EmployeePermissionGate>; }
