import type { Metadata } from "next";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { SupportTicketWorkspace } from "@/features/employee/support/support-ticket-workspace";
export const metadata: Metadata = { title: "Quản lý Ticket hỗ trợ" };
export default function SupportTicketPage() { return <EmployeePermissionGate permission="TICKET_QUEUE_VIEW"><SupportTicketWorkspace /></EmployeePermissionGate>; }
