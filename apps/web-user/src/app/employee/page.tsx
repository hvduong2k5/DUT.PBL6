import type { Metadata } from "next";

export const metadata: Metadata = { title: "Không gian nhân viên" };

export default function EmployeeHomePage() {
  return <section className="employee-dashboard-intro"><p className="eyebrow">Không gian nhân viên</p><h1>Tổng quan vận hành</h1><p>Các phân hệ và workbench theo permission sẽ được hợp nhất trong giai đoạn tiếp theo.</p></section>;
}
