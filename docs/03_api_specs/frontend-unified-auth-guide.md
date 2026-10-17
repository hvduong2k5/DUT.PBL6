# Hướng Dẫn Tích Hợp Xác Thực & Phân Quyền Hợp Nhất Cho Frontend (Unified Web)
**Dự án PBL6 — Mè Xửng O Mạ**  
**Tài liệu bàn giao cho Đội ngũ Frontend (FE)**  
**Cập nhật:** 2026-10-17

---

## 1. Triết Lý Thiết Kế: 1 Website Duy Nhất (Unified Web Portal)

Hệ thống triển khai theo mô hình **1 Web duy nhất** cho toàn bộ người dùng (Khách hàng D2C, Nhân viên kho, Nhân viên đóng gói, Nhân viên giao hàng, Quản lý cửa hàng và Super Admin):
- **Cùng 1 Domain & Port:** Chạy trên cùng một ứng dụng Next.js (ví dụ `https://omama.vn` hoặc `http://localhost:3000`).
- **Cùng 1 Trang Đăng nhập (`/login`):** Mọi đối tượng đều đăng nhập qua một trang chung.
- **Phân quyền Động dựa trên Token & Quyền hạn (RBAC + Scoped IAM):** Sau khi đăng nhập, hệ thống dựa vào danh sách `roles` và `permissions` từ backend để quyết định các tính năng, menu và route hiển thị.

```
                               ┌────────────────────────────────┐
                               │   TRANG ĐĂNG NHẬP DUY NHẤT     │
                               │           (/login)             │
                               └───────────────┬────────────────┘
                                               │
                                               ▼
                                      Đăng nhập thành công
                               (Lưu Token vào HttpOnly Cookie)
                                               │
                                               ▼
                                   Gọi GET /api/v1/users/me
                       ┌───────────────────────┴───────────────────────┐
                       ▼                                               ▼
          Nếu principalType = "CUSTOMER"                  Nếu principalType = "EMPLOYEE"
       (Khách hàng mua kẹo thông thường)              (Admin, Warehouse, Packing, Delivery...)
                       │                                               │
  • Hiển thị Menu Mua hàng, Giỏ hàng, Đơn mua.     • Header xuất hiện nút "Kênh Quản Trị" (/admin).
  • Trang cá nhân: Điểm thưởng, Địa chỉ.           • Chuyển sang khu vực /admin:
  • Nếu cố tình gõ URL /admin/**:                     - Có INVENTORY_VIEW    --> Hiện Tab Kho
    --> Middleware chặn & redirect về /               - Có PACKING_WORKBENCH --> Hiện Tab Đóng gói
                                                      - Có ROLE_READ         --> Hiện Tab Phân quyền
```

---

## 2. Các API Xác Thực & Phiên Làm Việc

Toàn bộ hợp đồng chi tiết nằm trong file [`docs/03_api_specs/customer-stable.openapi.yaml`](./customer-stable.openapi.yaml).

### 2.1. Đăng nhập (`POST /api/v1/auth/login`)
Frontend gửi thông tin đăng nhập:
```json
POST /api/v1/auth/login
Content-Type: application/json
Idempotency-Key: <UUID>

{
  "identifier": "0905123456", // Hoặc email: nhanvien@omama.vn
  "password": "Password@123"
}
```

**Response (HTTP 200):**
```json
{
  "tokens": {
    "access_token": "eyJhbGciOiJSUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJSUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  },
  "user": {
    "customer_id": "CUST-8820",
    "phone_number": "0905123456",
    "full_name": "Nguyễn Văn An",
    "email": "an.nguyen@example.com",
    "membership_tier": "GOLD",
    "loyalty_points": 450
  }
}
```
*Gợi ý FE:* Lưu `access_token` và `refresh_token` vào Cookie (HttpOnly) hoặc secure storage.

---

### 2.2. Nhận diện Phiên & Danh sách Quyền (`GET /api/v1/users/me`)

Ngay sau khi đăng nhập hoặc mỗi khi F5/tải trang, FE gọi endpoint này kèm header:
`Authorization: Bearer <access_token>`

#### Trường hợp A: Tài khoản là Khách hàng (`CUSTOMER`)
```json
{
  "principalId": "b1a2c3d4-0000-1111-2222-333344445555",
  "idpSubject": "kc-subject-uuid-customer",
  "email": "khachhang@gmail.com",
  "principalType": "CUSTOMER",
  "provisioningStatus": "ACTIVE",
  "roles": [
    {
      "roleCode": "CUSTOMER",
      "roleName": "Khách hàng",
      "scopeType": "GLOBAL",
      "scopeId": null
    }
  ],
  "permissions": []
}
```

#### Trường hợp B: Tài khoản là Nhân viên / Quản lý (`EMPLOYEE`)
```json
{
  "principalId": "e9f8d7c6-1111-2222-3333-444455556666",
  "idpSubject": "kc-subject-uuid-employee",
  "email": "nhanvienkho@omama.vn",
  "principalType": "EMPLOYEE",
  "provisioningStatus": "ACTIVE",
  "roles": [
    {
      "roleCode": "WAREHOUSE_STAFF",
      "roleName": "Nhân viên kho vận",
      "scopeType": "WAREHOUSE",
      "scopeId": "WH_DANANG"
    }
  ],
  "permissions": [
    "INVENTORY_VIEW",
    "INVENTORY_RECEIPT_CREATE",
    "INVENTORY_ISSUE_CREATE",
    "PACKING_WORKBENCH_VIEW"
  ]
}
```

---

## 3. Hướng Dẫn Code Mẫu Cho Frontend (Next.js / React)

### 3.1. Hook Quản Lý Phiên Hợp Nhất (`useAppSession`)

Tạo file `src/components/auth/app-session-provider.tsx`:

```tsx
"use client";

import React, { createContext, useContext, useEffect, useState, useMemo, useCallback } from "react";

export interface UserRoleScope {
  roleCode: string;
  roleName?: string;
  scopeType: "GLOBAL" | "STORE" | "WAREHOUSE";
  scopeId?: string | null;
}

export interface UserProfile {
  principalId: string;
  email: string;
  principalType: "CUSTOMER" | "EMPLOYEE";
  provisioningStatus: string;
  roles: UserRoleScope[];
  permissions: string[];
}

interface AppSessionContextValue {
  status: "loading" | "authenticated" | "guest" | "error";
  profile: UserProfile | null;
  isStaff: boolean;
  hasRole: (roleCode: string) => boolean;
  hasPermission: (permissionCode: string) => boolean;
  reload: () => Promise<void>;
  logout: () => Promise<void>;
}

const AppSessionContext = createContext<AppSessionContextValue | null>(null);

export function AppSessionProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AppSessionContextValue["status"]>("loading");
  const [profile, setProfile] = useState<UserProfile | null>(null);

  const reload = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/users/me", {
        headers: { Accept: "application/json" },
        cache: "no-store",
      });

      if (res.ok) {
        const data: UserProfile = await res.json();
        setProfile(data);
        setStatus("authenticated");
      } else if (res.status === 401) {
        setProfile(null);
        setStatus("guest");
      } else {
        setProfile(null);
        setStatus("error");
      }
    } catch {
      setProfile(null);
      setStatus("error");
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const isStaff = useMemo(() => {
    if (!profile) return false;
    return profile.principalType === "EMPLOYEE" || profile.roles.some((r) => r.roleCode !== "CUSTOMER");
  }, [profile]);

  const hasRole = useCallback(
    (roleCode: string) => Boolean(profile?.roles.some((r) => r.roleCode === roleCode)),
    [profile]
  );

  const hasPermission = useCallback(
    (perm: string) => Boolean(profile?.permissions.includes(perm)),
    [profile]
  );

  const logout = useCallback(async () => {
    await fetch("/api/v1/auth/logout", { method: "POST" });
    setProfile(null);
    setStatus("guest");
    window.location.href = "/";
  }, []);

  const value = useMemo(
    () => ({ status, profile, isStaff, hasRole, hasPermission, reload, logout }),
    [status, profile, isStaff, hasRole, hasPermission, reload, logout]
  );

  return <AppSessionContext.Provider value={value}>{children}</AppSessionContext.Provider>;
}

export function useAppSession() {
  const ctx = useContext(AppSessionContext);
  if (!ctx) throw new Error("useAppSession must be used within AppSessionProvider");
  return ctx;
}
```

---

### 3.2. Hiển Thị Header Động Dựa Vào Quyền

Trên thanh điều hướng chính (`src/components/layout/header.tsx`):

```tsx
import { useAppSession } from "@/components/auth/app-session-provider";
import Link from "next/link";

export function Header() {
  const { status, profile, isStaff, logout } = useAppSession();

  return (
    <header className="flex justify-between p-4 bg-white shadow">
      <Link href="/" className="font-bold text-xl">Mè Xửng O Mạ</Link>

      <div className="flex items-center gap-4">
        {/* Khách hàng thường & Chưa đăng nhập luôn thấy Giỏ hàng */}
        <Link href="/cart">Giỏ hàng</Link>

        {status === "guest" && (
          <Link href="/login" className="btn-primary">Đăng nhập</Link>
        )}

        {status === "authenticated" && (
          <>
            <span>Xin chào, {profile?.email}</span>

            {/* NÚT KÊNH QUẢN TRỊ: Chỉ hiển thị nếu tài khoản là nhân viên/admin */}
            {isStaff && (
              <Link 
                href="/admin" 
                className="bg-amber-600 text-white px-3 py-1.5 rounded font-medium hover:bg-amber-700"
              >
                🛠️ Kênh Quản Trị
              </Link>
            )}

            <Link href="/account/profile">Tài khoản</Link>
            <button onClick={logout} className="text-red-600">Đăng xuất</button>
          </>
        )}
      </div>
    </header>
  );
}
```

---

### 3.3. Bảo Vệ Route Quản Trị Bằng Middleware (`middleware.ts`)

Bảo vệ tất cả các đường dẫn `/admin/**` để khách thường không vào được:

```typescript
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Kiểm tra nếu truy cập vào khu vực /admin
  if (pathname.startsWith("/admin")) {
    const accessToken = request.cookies.get("oma_access_token")?.value;

    // Chưa đăng nhập -> đá về trang Login kèm redirect callback
    if (!accessToken) {
      const loginUrl = new URL("/login", request.url);
      loginUrl.searchParams.set("callbackUrl", pathname);
      return NextResponse.redirect(loginUrl);
    }

    // Gợi ý: Giải mã JWT payload nhanh hoặc kiểm tra header
    // Nếu claim roles không chứa role nhân viên -> đá về trang chủ
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/admin/:path*"],
};
```

---

### 3.4. Ẩn / Hiện Menu Con Trong Layout Quản Trị (`/admin/layout.tsx`)

Trong trang quản trị, mỗi tab chức năng chỉ hiển thị khi tài khoản có quyền tương ứng:

```tsx
"use client";

import { useAppSession } from "@/components/auth/app-session-provider";
import Link from "next/link";
import { redirect } from "next/navigation";

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const { isStaff, hasPermission, status } = useAppSession();

  if (status === "loading") return <div>Đang kiểm tra quyền truy cập...</div>;
  if (!isStaff) {
    redirect("/"); // Khách thường cố tình vào sẽ bị đẩy về trang chủ
  }

  return (
    <div className="flex h-screen">
      <aside className="w-64 bg-slate-900 text-white p-4 space-y-2">
        <h2 className="text-lg font-bold text-amber-400">Bàn Làm Việc</h2>

        {/* Tab Đơn hàng */}
        {hasPermission("ORDER_READ") && (
          <Link href="/admin/orders" className="block p-2 hover:bg-slate-800 rounded">
            📦 Quản lý Đơn hàng
          </Link>
        )}

        {/* Tab Kho vận */}
        {hasPermission("INVENTORY_VIEW") && (
          <Link href="/admin/inventory" className="block p-2 hover:bg-slate-800 rounded">
            🏬 Quản lý Kho hàng
          </Link>
        )}

        {/* Tab Đóng gói */}
        {hasPermission("PACKING_WORKBENCH_VIEW") && (
          <Link href="/admin/work/packing" className="block p-2 hover:bg-slate-800 rounded">
            🎁 Bàn Đóng Gói
          </Link>
        )}

        {/* Tab Giao hàng */}
        {hasPermission("DELIVERY_WORKBENCH_VIEW") && (
          <Link href="/admin/work/delivery" className="block p-2 hover:bg-slate-800 rounded">
            🚚 Bàn Giao Hàng
          </Link>
        )}

        {/* Tab Quản trị Role & Phân quyền (Chỉ Super Admin) */}
        {hasPermission("ROLE_READ") && (
          <Link href="/admin/administration/roles" className="block p-2 hover:bg-slate-800 rounded">
            🛡️ Quản lý Vai Trò & Phân Quyền
          </Link>
        )}
      </aside>

      <main className="flex-1 p-6 overflow-auto bg-slate-50">{children}</main>
    </div>
  );
}
```

---

## 4. Tóm Tắt Cho FE Dev

1. **Đăng nhập:** Gọi `POST /api/v1/auth/login` (chung cho mọi user).
2. **Lấy Quyền:** Gọi `GET /api/v1/users/me` -> nhận `roles` và `permissions`.
3. **Phân biệt:**
   - `roles.some(r => r.roleCode !== 'CUSTOMER')` $\longrightarrow$ **Nhân viên / Admin**.
   - `roles.every(r => r.roleCode === 'CUSTOMER')` $\longrightarrow$ **Khách hàng D2C**.
4. **Điều hướng:**
   - Khách hàng: Mua sắm, giỏ hàng, hồ sơ cá nhân.
   - Nhân viên: Có thêm nút **"Kênh Quản Trị"** dẫn vào `/admin` và các menu con ẩn hiện theo `permissions`.
