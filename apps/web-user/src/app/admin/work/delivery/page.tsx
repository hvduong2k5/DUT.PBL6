import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { DeliveryWorkbench } from "@/features/employee/shipping/delivery-workbench";

export const metadata: Metadata = { title: "Công việc giao hàng" };

export default function DeliveryWorkbenchPage() { return <EmployeePermissionGate permission="DELIVERY_WORKBENCH_VIEW"><DeliveryWorkbench /></EmployeePermissionGate>; }
