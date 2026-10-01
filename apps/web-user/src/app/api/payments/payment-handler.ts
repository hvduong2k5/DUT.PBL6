import { PAYMENT_ACCESS_COOKIE, decodePaymentAccess } from "@/lib/payment/access";
import type { OrderPaymentState, PaymentStatus } from "@/lib/payment/types";
import { parsePaymentRoute } from "@/lib/payment/validation";
import { NextRequest, NextResponse } from "next/server";

function errorResponse(code: string, message: string, status: number) {
  return NextResponse.json({ code, message, requestId: `BFF-PAYMENT-${code}` }, { status });
}

function claimStatus(method: "BANK_TRANSFER" | "COD", expiresAt: string): { status: PaymentStatus; orderState: OrderPaymentState } {
  if (method === "COD") return { status: "COD_PENDING_COLLECTION", orderState: "PAYMENT_ON_DELIVERY" };
  if (Date.parse(expiresAt) <= Date.now()) return { status: "EXPIRED", orderState: "PAYMENT_EXPIRED" };
  return { status: "PENDING", orderState: "PENDING_PAYMENT" };
}

export async function handlePaymentRequest(request: NextRequest, path: string[]): Promise<NextResponse> {
  const route = parsePaymentRoute(request.method, path, request.nextUrl.searchParams);
  if (route.error || !route.operation || !route.orderId) {
    const notFound = route.error?.field === "path";
    return errorResponse(notFound ? "NOT_FOUND" : "INVALID_REQUEST", route.error?.message ?? "Yêu cầu Payment không hợp lệ.", notFound ? 404 : 400);
  }
  if (process.env.NODE_ENV === "production" && route.mockScenario) {
    return errorResponse("INVALID_REQUEST", "Mock scenario không khả dụng trong production.", 400);
  }

  const encodedClaim = request.cookies.get(PAYMENT_ACCESS_COOKIE)?.value;
  if (!encodedClaim) return errorResponse("PAYMENT_ACCESS_REQUIRED", "Không tìm thấy phiên truy cập Payment. Vui lòng tạo lại Order từ Checkout.", 401);
  const claim = decodePaymentAccess(encodedClaim);
  if (!claim) return errorResponse("PAYMENT_ACCESS_INVALID", "Phiên truy cập Payment đã hết hạn hoặc không hợp lệ. Vui lòng tạo lại Order từ Checkout.", 401);
  if (claim.orderId !== route.orderId) return errorResponse("PAYMENT_NOT_FOUND", "Không tìm thấy thông tin thanh toán phù hợp.", 404);
  const current = claimStatus(claim.method, claim.paymentExpiresAt);

  if (route.operation === "summary") {
    return NextResponse.json({
      orderId: claim.orderId,
      orderNumber: claim.orderNumber,
      method: claim.method,
      amountVnd: claim.amountVnd,
      currency: "VND",
      paymentId: `checkout-${claim.orderId}`,
      status: current.status,
      orderPaymentState: current.orderState,
      expiresAt: claim.paymentExpiresAt,
      instructions: null,
      checkedAt: new Date().toISOString(),
      vietQrUrl: claim.vietQrUrl
    });
  }

  if (route.operation === "status") {
    return NextResponse.json({
      paymentId: `checkout-${claim.orderId}`,
      status: current.status,
      orderPaymentState: current.orderState,
      verifiedAt: null,
      message: current.status === "EXPIRED"
        ? "Thời hạn VietQR đã kết thúc. Vui lòng kiểm tra trạng thái đơn trước khi thao tác tiếp."
        : claim.method === "COD"
          ? "Đơn hàng đang chờ thu tiền khi giao hàng."
          : "Chưa có xác nhận thanh toán mới. Trạng thái chính thức được cập nhật trong Order sau callback VietQR.",
      canRetry: false
    });
  }

  return errorResponse("PAYMENT_RETRY_NOT_SUPPORTED", "API customer hiện chưa hỗ trợ tạo lại một lần thanh toán VietQR.", 409);
}
