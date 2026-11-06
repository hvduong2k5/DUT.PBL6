export const TICKET_STATUSES = ["NEW", "TRIAGED", "ASSIGNED", "IN_PROGRESS", "WAITING_CUSTOMER", "WAITING_INTERNAL", "RESOLVED", "CLOSED"] as const;
export type TicketStatus = (typeof TICKET_STATUSES)[number];
export type TicketPriority = "NORMAL" | "IMPORTANT" | "URGENT";
export type TicketSlaState = "ON_TRACK" | "AT_RISK" | "OVERDUE" | "PAUSED" | "COMPLETE";

export interface SupportTicketSummary {
  ticketId: string; ticketNumber: string; subjectCode: string; subjectLabel: string; title: string; customerLabel: string;
  customerMode: "GUEST" | "REGISTERED"; channelCode: string; channelLabel: string; status: TicketStatus; statusLabel: string;
  priority: TicketPriority; priorityLabel: string; queueLabel: string; assigneeLabel: string | null;
  firstResponseSla: { state: TicketSlaState; stateLabel: string; dueAt: string | null };
  resolutionSla: { state: TicketSlaState; stateLabel: string; dueAt: string | null };
  linkedReferenceCount: number; unreadCount: number; lastMessageAt: string; updatedAt: string; revision: number;
}
export interface SupportTicketList {
  items: SupportTicketSummary[];
  statusCounts: Array<{ status: "ALL" | TicketStatus; label: string; count: number }>;
  subjects: Array<{ code: string; label: string }>;
  channels: Array<{ code: string; label: string }>;
  calculatedAt: string;
}
export interface SupportTicketDetail extends SupportTicketSummary {
  createdAt: string;
  customer: { displayName: string; contactDisplay: string; profileLabel: string; masked: boolean };
  messages: Array<{ messageId: string; kind: "PUBLIC" | "INTERNAL_NOTE"; senderRole: string; senderLabel: string; channelLabel: string; deliveryStateLabel: string; content: string; sentAt: string }>;
  conversationMasked: boolean;
  context: { order: { reference: string; statusLabel: string } | null; payment: { statusLabel: string } | null; shipment: { statusLabel: string } | null; returnCase: { reference: string; statusLabel: string } | null; sourcesUpdatedAt: string } | null;
  contextMasked: boolean;
  history: Array<{ eventId: string; label: string; actorLabel: string; occurredAt: string; detail: string | null }>;
}
interface AdminSupportErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
export class AdminSupportApiError extends Error {
  readonly code: string; readonly status: number; readonly errors: Array<{ field: string; message: string }>;
  constructor(status: number, body: AdminSupportErrorBody) { super(body.message || "Không thể xử lý yêu cầu quản lý Ticket."); this.name = "AdminSupportApiError"; this.status = status; this.code = body.code || "UNKNOWN_ERROR"; this.errors = body.errors ?? []; }
}
