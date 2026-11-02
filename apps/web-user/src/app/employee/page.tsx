"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAdminSession } from "@admin/components/auth/admin-session-provider";
import type { AdminPermission } from "@admin/lib/auth/types";

const FIRST_WORKSPACES: Array<{ permission: AdminPermission; href: string }> = [
  { permission: "WAREHOUSE_WORKBENCH_VIEW", href: "/employee/work/warehouse" },
  { permission: "PACKING_WORKBENCH_VIEW", href: "/employee/work/packing" },
  { permission: "DELIVERY_WORKBENCH_VIEW", href: "/employee/work/delivery" },
  { permission: "B2B_REQUEST_VIEW", href: "/employee/b2b/quotes" },
  { permission: "PRODUCT_VIEW", href: "/employee/catalog/products" },
  { permission: "INVENTORY_MANAGEMENT_VIEW", href: "/employee/inventory" },
  { permission: "PACKING_MANAGEMENT_VIEW", href: "/employee/packing" },
  { permission: "SHIPMENT_MANAGEMENT_VIEW", href: "/employee/shipping" },
  { permission: "ORDER_VIEW", href: "/employee/orders" },
  { permission: "PAYMENT_VIEW", href: "/employee/payments" },
  { permission: "RETURN_CASE_VIEW", href: "/employee/returns" },
  { permission: "TICKET_QUEUE_VIEW", href: "/employee/support/tickets" },
  { permission: "EMPLOYEE_ACCOUNT_VIEW", href: "/employee/administration/employees" },
  { permission: "ROLE_VIEW", href: "/employee/administration/roles" },
  { permission: "PERMISSION_VIEW", href: "/employee/administration/permissions" }
];

export default function EmployeeHome() {
  const router = useRouter();
  const { session, hasPermission } = useAdminSession();
  const firstWorkspace = FIRST_WORKSPACES.find(({ permission }) => hasPermission(permission));

  useEffect(() => {
    if (firstWorkspace) router.replace(firstWorkspace.href);
  }, [firstWorkspace, router]);

  if (firstWorkspace) return <main className="admin-state" aria-busy="true"><span className="admin-spinner" /><h1>{"\u0110ang m\u1edf kh\u00f4ng gian l\u00e0m vi\u1ec7c..."}</h1></main>;
  return <main className="admin-state"><span>!</span><h1>{"Ch\u01b0a c\u00f3 kh\u00f4ng gian l\u00e0m vi\u1ec7c \u0111\u01b0\u1ee3c c\u1ea5p quy\u1ec1n"}</h1><p>{session?.employee.displayName} {"ch\u01b0a \u0111\u01b0\u1ee3c c\u1ea5p permission xem trang."}</p></main>;
}
