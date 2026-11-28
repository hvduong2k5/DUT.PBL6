import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { TraceabilityWorkspace } from "@/features/employee/traceability/workspace";

export default async function Page({ searchParams }: { searchParams: Promise<{ id?: string; batchId?: string; skuCode?: string }> }) {
  const initial = await searchParams;
  return <EmployeePermissionGate permission="traceability.read_internal"><TraceabilityWorkspace initial={initial} /></EmployeePermissionGate>;
}
