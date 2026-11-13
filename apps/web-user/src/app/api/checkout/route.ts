import { handleCheckoutRequest } from "./checkout-handler";
import type { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

export function POST(request: NextRequest) {
  return handleCheckoutRequest(request, []);
}
