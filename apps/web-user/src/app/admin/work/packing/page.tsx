import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { PackingWorkbench } from "@/features/employee/packing/packing-workbench";
import { FulfillmentWorkspace } from "@/features/employee/packing/fulfillment-workspace";
import { usesIdentityBff } from "@/lib/auth/oidc-config";

export const metadata: Metadata = { title: "Workbench đóng gói" };
export const dynamic = "force-dynamic";

export default function PackingWorkbenchPage() { return usesIdentityBff() ? <FulfillmentWorkspace /> : <EmployeePermissionGate permission="PACKING_WORKBENCH_VIEW"><PackingWorkbench /></EmployeePermissionGate>; }
