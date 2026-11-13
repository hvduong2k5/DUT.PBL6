import { notFound } from "next/navigation";
import { connection } from "next/server";
import { EmployeePermissionGate } from "@/components/auth/employee-permission-gate";
import { SupplyWorkspace } from "@/features/employee/analytics/supply-workspace";

export default async function SupplyAnalyticsPage() {
  await connection();
  if (!process.env.ANALYTICS_ADMIN_UPSTREAM_URL) notFound();
  return (
    <EmployeePermissionGate permission="INVENTORY_MANAGEMENT_VIEW">
      <EmployeePermissionGate permission="BATCH_VIEW">
        <SupplyWorkspace initialWarehouse={process.env.ANALYTICS_PILOT_WAREHOUSE ?? ""} />
      </EmployeePermissionGate>
    </EmployeePermissionGate>
  );
}
