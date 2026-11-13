import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { PackingWorkspace } from "@/features/employee/packing/packing-workspace";
import { FulfillmentWorkspace } from "@/features/employee/packing/fulfillment-workspace";
import { usesIdentityBff } from "@/lib/auth/oidc-config";

export const metadata: Metadata = { title: "Quản lý đóng gói" };
export const dynamic = "force-dynamic";
export default function PackingPage() { return usesIdentityBff() ? <FulfillmentWorkspace management /> : <EmployeePermissionGate permission="PACKING_MANAGEMENT_VIEW"><PackingWorkspace /></EmployeePermissionGate>; }
