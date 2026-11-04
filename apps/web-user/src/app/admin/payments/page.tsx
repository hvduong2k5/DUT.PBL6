import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { PaymentWorkspace } from "@/features/employee/payments/payment-workspace";

export const metadata: Metadata = { title: "Vận hành thanh toán" };
export default function PaymentPage() { return <EmployeePermissionGate permission="PAYMENT_VIEW"><PaymentWorkspace /></EmployeePermissionGate>; }
