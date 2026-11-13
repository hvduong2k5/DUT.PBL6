import type { Metadata } from "next";
import { AuthPage } from "@/components/layout/auth-page";
import { getSafeReturnUrl } from "@/lib/auth/return-url";
import { RegistrationFlow } from "./registration-flow";

export const metadata: Metadata = { title: "Đăng ký tài khoản Tri Kỷ" };

export default async function RegistrationPage({ searchParams }: { searchParams: Promise<{ returnUrl?: string | string[] }> }) {
  const params = await searchParams;
  return (
    <AuthPage eyebrow="Ghi danh hội viên mới" title="Khởi Tạo Thẻ Tri Kỷ" description="Đăng ký bằng email để bắt đầu mua sắm và quản lý đơn hàng. Sau khi đăng nhập, bạn chỉ cần thêm số điện thoại trong hồ sơ để kích hoạt đầy đủ quyền mua hàng." asideTitle="Trở thành Khách Tri Kỷ Ô Mạ" asideBody="Kết duyên cùng phong vị Cố Đô. Thưởng ngoạn trọn vẹn thức quà ngự thiện với các đặc quyền được chọn lọc." benefits={[{ icon: "◉", title: "Hoàn thiện hồ sơ nhanh", body: "Thêm số điện thoại sau khi đăng nhập để kích hoạt mua hàng." }, { icon: "%", title: "Đặc quyền mùa lễ", body: "Nhận tin về bộ sưu tập và dịp lễ truyền thống." }, { icon: "✎", title: "Thư pháp Cố Đô", body: "Cá nhân hóa thiệp quà trên hành trình sau." }]}>
      <RegistrationFlow returnUrl={getSafeReturnUrl(params.returnUrl)} />
    </AuthPage>
  );
}
