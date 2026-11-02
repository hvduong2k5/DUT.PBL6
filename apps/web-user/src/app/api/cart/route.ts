import { handleCartRequest } from "./cart-handler";
import type { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

export function GET(request: NextRequest) {
  return handleCartRequest(request, []);
}
