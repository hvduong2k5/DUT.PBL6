import type {Metadata} from "next";
import {EmployeePermissionGate} from "@/components/auth/employee-permission-gate";
import {CompanyDispatcher} from "@/features/employee/shipping/company-dispatcher";

export const metadata:Metadata={title:"Điều phối giao hàng nội bộ"};
export default function CompanyDispatchPage(){return <EmployeePermissionGate permission="SHIPMENT_ASSIGN">{process.env.SHIPPING_COMPANY_BFF_ENABLED==="true"?<CompanyDispatcher/>:<p>Điều phối giao hàng nội bộ chưa được bật.</p>}</EmployeePermissionGate>;}
