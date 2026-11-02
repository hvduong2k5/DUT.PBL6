import type { Metadata } from "next";
import { AuthPage } from "@/components/layout/auth-page";
import { getSafeReturnUrl } from "@/lib/auth/return-url";
import { LoginForm } from "./login-form";

export const metadata: Metadata = { title: "Đăng nhập tài khoản Tri Kỷ" };

export default async function LoginPage({ searchParams }: { searchParams: Promise<{ returnUrl?: string | string[] }> }) {
  const params = await searchParams;
  return (
    <AuthPage eyebrow="Cổng thành viên O Mạ" title="Đăng Nhập Tài Khoản Tri Kỷ" description="Tiếp tục qua cổng định danh an toàn để mua sắm và quản lý tài khoản." asideTitle="Tri Kỷ Hội Cung" asideBody="Một lần đăng nhập cho cả trải nghiệm mua sắm và không gian làm việc nội bộ được cấp quyền." benefits={[{ icon: "◇", title: "Đăng nhập một lần", body: "Dùng cùng một phiên cho Customer và Employee." }, { icon: "♢", title: "Cookie HttpOnly", body: "Token không được lưu trong trình duyệt hoặc mã giao diện." }, { icon: "♙", title: "Đúng quyền được cấp", body: "Role và permission được tải từ Identity sau đăng nhập." }]}>
      <LoginForm returnUrl={getSafeReturnUrl(params.returnUrl)} />
    </AuthPage>
  );
}
