import { handleOrderRequest } from "./order-handler";
import type { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

export function GET(request: NextRequest) {
  return handleOrderRequest(request, []);
}
