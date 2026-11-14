import type { Metadata } from "next";
import { AuthPage } from "@/components/layout/auth-page";
import { getSafeReturnUrl } from "@/lib/auth/return-url";
import Link from "next/link";
import { LoginForm } from "./login-form";
import { usesOidc, usesIdentityBff } from "@/lib/auth/oidc-config";

export const metadata: Metadata = { title: "Đăng nhập tài khoản Tri Kỷ" };

export default async function LoginPage({ searchParams }: { searchParams: Promise<{ returnUrl?: string | string[] }> }) {
  const params = await searchParams;
  return (
    <AuthPage eyebrow="⚿ Cổng thành viên Hoàng Gia" title="Đăng Nhập Tài Khoản Tri Kỷ" description="Đăng nhập để tiếp tục thưởng lãm và đặt quà biếu." asideTitle="Tri Kỷ Hội Cung" asideBody="Trân trọng kính mời Quý Tri Kỷ mở lại lối xưa, thưởng thức những mẻ bánh truyền thống được tinh chế theo bí truyền ngự trù." benefits={[{ icon: "◇", title: "Tích Điểm Di Sản Kim Bảo", body: "Nhận điểm phong vị theo từng tráp bánh." }, { icon: "♨", title: "Mẻ Bánh Mới Vừa Ra Lò", body: "Đặc quyền đặt trước những thức quà mùa vụ." }, { icon: "♙", title: "Mừng Sinh Nhật Ngự Phẩm", body: "Trao tặng hộp quà hoàng triều độc bản." }]}>
      {(usesOidc() || usesIdentityBff()) ? <div className="auth-form"><p>Tiếp tục tới trang xác thực để đăng nhập tài khoản của bạn.</p><a className="primary-button" href={`/api/auth/oidc/login?returnUrl=${encodeURIComponent(getSafeReturnUrl(params.returnUrl))}`}>Tiếp tục đăng nhập</a><p className="switch-flow">Chưa có tài khoản Tri Kỷ? <Link href={`/register?returnUrl=${encodeURIComponent(getSafeReturnUrl(params.returnUrl))}`}>Đăng ký ngay</Link></p><p className="security-note">Bạn sẽ được đưa trở lại cửa hàng sau khi đăng nhập.</p></div> : <LoginForm returnUrl={getSafeReturnUrl(params.returnUrl)} />}
    </AuthPage>
  );
}
