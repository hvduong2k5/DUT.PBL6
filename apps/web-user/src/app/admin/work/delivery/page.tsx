import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { DeliveryWorkbench } from "@/features/employee/shipping/delivery-workbench";
import { CompanyWorkbench } from "@/features/employee/shipping/company-workbench";

export const metadata: Metadata = { title: "Công việc giao hàng" };

export default function DeliveryWorkbenchPage() { return process.env.SHIPPING_COMPANY_BFF_ENABLED === "true" ? <EmployeePermissionGate permission="SHIPMENT_EXECUTE"><CompanyWorkbench /></EmployeePermissionGate> : <EmployeePermissionGate permission="DELIVERY_WORKBENCH_VIEW"><DeliveryWorkbench /></EmployeePermissionGate>; }
