import type { Metadata } from "next";
import { AuthPage } from "@/components/site-shell";
import { getSafeReturnUrl } from "@/lib/auth/return-url";
import { LoginForm } from "./login-form";

export const metadata: Metadata = { title: "Đăng nhập tài khoản Tri Kỷ" };

export default async function LoginPage({ searchParams }: { searchParams: Promise<{ returnUrl?: string | string[] }> }) {
  const params = await searchParams;
  return (
    <AuthPage eyebrow="⚿ Cổng thành viên Hoàng Gia" title="Đăng Nhập Tài Khoản Tri Kỷ" description="Điền thông tin để tiếp tục thưởng lãm và đặt quà biếu." asideTitle="Tri Kỷ Hội Cung" asideBody="Trân trọng kính mời Quý Tri Kỷ mở lại lối xưa, thưởng thức những mẻ bánh truyền thống được tinh chế theo bí truyền ngự trù." benefits={[{ icon: "◇", title: "Tích Điểm Di Sản Kim Bảo", body: "Nhận điểm phong vị theo từng tráp bánh." }, { icon: "♨", title: "Mẻ Bánh Mới Vừa Ra Lò", body: "Đặc quyền đặt trước những thức quà mùa vụ." }, { icon: "♙", title: "Mừng Sinh Nhật Ngự Phẩm", body: "Trao tặng hộp quà hoàng triều độc bản." }]}>
      <LoginForm returnUrl={getSafeReturnUrl(params.returnUrl)} />
    </AuthPage>
  );
}
