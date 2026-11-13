import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { ShippingWorkspace } from "@/features/employee/shipping/shipping-workspace";

export const metadata: Metadata = { title: "Quản lý giao vận" };
export default function ShippingPage() { return <EmployeePermissionGate permission="SHIPMENT_MANAGEMENT_VIEW"><ShippingWorkspace /></EmployeePermissionGate>; }
