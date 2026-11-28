import type { NextRequest } from "next/server";
import { traceEvidenceURL } from "@/features/employee/traceability/media-server";
export const dynamic = "force-dynamic";
export async function GET(request: NextRequest, context: { params: Promise<{ profileId: string; evidenceId: string }> }) {
  const { profileId, evidenceId } = await context.params;
  return traceEvidenceURL(request, profileId, evidenceId);
}
