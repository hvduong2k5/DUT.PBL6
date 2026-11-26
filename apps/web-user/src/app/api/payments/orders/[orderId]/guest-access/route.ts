import { NextRequest } from "next/server";
import { handleGuestPaymentAccess } from "@/lib/payment/guest-server";

export const dynamic = "force-dynamic";
export async function POST(request: NextRequest, context: { params: Promise<{ orderId: string }> }) {
  return handleGuestPaymentAccess(request, (await context.params).orderId);
}
