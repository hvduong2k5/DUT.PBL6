export const SHIPMENT_STATUSES = ["CREATING", "CREATED", "READY_FOR_HANDOVER", "HANDED_OVER", "IN_TRANSIT", "OUT_FOR_DELIVERY", "DELIVERED", "DELIVERY_FAILED", "RETURNING", "RETURNED", "CANCELLED", "CREATION_FAILED"] as const;
export type ShipmentStatus = (typeof SHIPMENT_STATUSES)[number];
export type ShipmentExceptionState = "NONE" | "ATTENTION" | "BLOCKED";
export type ShipmentMappingResult = "APPLIED" | "IGNORED_DUPLICATE" | "IGNORED_STALE" | "UNMAPPED" | "REJECTED_SOURCE";

export interface ShipmentSummary {
  shipmentId: string;
  shipmentNumber: string;
  orderId: string;
  orderNumber: string;
  orderSource: string;
  orderSourceLabel: string;
  packageReference: string;
  providerCode: string;
  providerLabel: string;
  serviceLabel: string;
  trackingCode: string | null;
  status: ShipmentStatus;
  statusLabel: string;
  deliveryAssignee: { employeeId: string; displayName: string } | null;
  orderStatus: string;
  orderStatusLabel: string;
  codAmountVnd: number;
  codStatusLabel: string;
  exceptionState: ShipmentExceptionState;
  exceptionLabel: string;
  exceptionReason: string | null;
  estimatedDeliveryAt: string | null;
  lastEventAt: string | null;
  updatedAt: string;
  revision: number;
}

export interface ShipmentList {
  items: ShipmentSummary[];
  statusCounts: Array<{ status: "ALL" | ShipmentStatus; label: string; count: number }>;
  providers: Array<{ code: string; label: string }>;
  sources: Array<{ code: string; label: string }>;
  calculatedAt: string;
}

export interface ShipmentTrackingEvent {
  eventId: string;
  providerStatusCode: string;
  providerStatusLabel: string;
  mappedStatus: ShipmentStatus | null;
  mappedStatusLabel: string;
  mappingResult: ShipmentMappingResult;
  mappingResultLabel: string;
  eventAt: string;
  receivedAt: string;
  sourceLabel: string;
  publicDescription: string;
  internalNote: string | null;
}

export interface ShipmentDetail extends ShipmentSummary {
  recipient: { displayName: string; phoneDisplay: string; addressDisplay: string; masked: boolean };
  package: { weightGrams: number; lengthCm: number | null; widthCm: number | null; heightCm: number | null; packedAt: string | null };
  fees: { checkoutFeeVnd: number; providerEstimatedFeeVnd: number; actualFeeVnd: number | null };
  handover: { ready: boolean; handedOverAt: string | null; confirmedBy: string | null; note: string | null };
  trackingEvents: ShipmentTrackingEvent[];
}

interface AdminShippingErrorBody { code?: string; message?: string; errors?: Array<{ field: string; message: string }> }
export class AdminShippingApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly errors: Array<{ field: string; message: string }>;
  constructor(status: number, body: AdminShippingErrorBody) {
    super(body.message || "Không thể xử lý yêu cầu quản lý giao vận.");
    this.name = "AdminShippingApiError";
    this.status = status;
    this.code = body.code || "UNKNOWN_ERROR";
    this.errors = body.errors ?? [];
  }
}
