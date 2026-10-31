import { NextRequest } from "next/server";
import { handleReturnRequest } from "./return-handler";

export function POST(request: NextRequest) {
  return handleReturnRequest(request, []);
}
