import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { SupportTicketFlow } from "./support-ticket-flow";

export const metadata: Metadata = { title: "Chi tiết Ticket hỗ trợ", description: "Theo dõi Ticket và trao đổi cùng Ban Chăm Sóc Tri Kỷ Ô Mạ." };

export default async function SupportTicketPage({ params }: { params: Promise<{ ticketId: string }> }) {
  const { ticketId } = await params;
  return <><SiteHeader /><Suspense fallback={null}><SupportTicketFlow ticketId={ticketId} /></Suspense><SiteFooter /></>;
}
