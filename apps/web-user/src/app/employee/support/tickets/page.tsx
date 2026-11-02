import type { Metadata } from "next";
import { SupportTicketWorkspace } from "./support-ticket-workspace";
export const metadata: Metadata = { title: "Quản lý Ticket hỗ trợ" };
export default function SupportTicketPage() { return <SupportTicketWorkspace />; }
