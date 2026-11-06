# Hướng Dẫn Tích Hợp Xác Thực & Phân Quyền Hợp Nhất Cho Frontend (Unified Web)
**Dự án PBL6 — Mè Xửng O Mạ**  
**Tài liệu bàn giao cho Đội ngũ Frontend (Web Next.js / Mobile)**  
**Kiến trúc:** **Backend-for-Frontend (BFF Pattern)** theo chuẩn **RFC 9700** & **RFC 7636 PKCE**  
**Cổng tiếp nhận duy nhất (Single Ingress):** **Kong API Gateway** (`https://api.omama.vn` hoặc `http://localhost:8000`)  
**Phiên bản:** 2.0.0 (Cập nhật chuẩn RESTful & IAM tập trung)

---

## 1. Triết Lý Thiết Kế: 1 Website Duy Nhất (Unified Web Portal)

Hệ thống triển khai theo mô hình **1 Web duy nhất** cho toàn bộ người dùng (Khách hàng D2C, Nhân viên kho, Nhân viên đóng gói, Nhân viên giao hàng, Quản lý cửa hàng và Super Admin):
- **Cùng 1 Domain & Port:** Chạy trên cùng một ứng dụng Next.js (ví dụ `https://omama.vn` hoặc `http://localhost:3000`).
- **Cùng 1 Cổng Xác thực (BFF + Keycloak):** Mọi đối tượng đều xác thực qua quy trình SSO OIDC chuẩn.
- **Phân quyền Động dựa trên Hồ sơ & Quyền hạn (RBAC + Scoped IAM):** Sau khi xác thực, hệ thống dựa vào danh sách `roles` và `permissions` từ backend (`GET /api/v1/users/me`) để quyết định các tính năng, menu và route hiển thị.

```
                               ┌────────────────────────────────┐
                               │   TRANG ĐĂNG NHẬP DUY NHẤT     │
                               │           (/login)             │
                               └───────────────┬────────────────┘
                                               │
                                               ▼
                                      Đăng nhập thành công
                                (Lưu Cookie bảo mật OMAMA_SESSION)
                                               │
                                               ▼
                  (Kiểm tra Session & Lấy Roles + Permissions qua /api/v1/users/me)
                        ┌───────────────────────┴───────────────────────┐
                        ▼                                               ▼
           Nếu principalType = "CUSTOMER"                  Nếu principalType = "EMPLOYEE"
        (Khách hàng mua kẹo thông thường)              (Admin, Warehouse, Packing, Delivery...)
                        │                                               │
   • Hiển thị Menu Mua hàng, Giỏ hàng, Đơn mua.     • Có thể xem Storefront ở chế độ chỉ đọc.
   • Trang cá nhân: Điểm thưởng, Địa chỉ.           • Header mở Kênh Quản Trị (/admin), ẩn Cart:
   • Nếu cố tình gõ URL /admin/**:                  - Có INVENTORY_VIEW    --> Hiện Tab Kho
     --> Middleware chặn & redirect về /               - Có PACKING_WORKBENCH --> Hiện Tab Đóng gói
                                                       - Có ROLE_READ         --> Hiện Tab Phân quyền
```

### 1.1. Mô hình Phân lập Danh tính Doanh nghiệp (Enterprise Persona Separation)
Để đảm bảo an toàn thông tin theo nguyên tắc Đặc quyền tối thiểu (Least Privilege), hệ thống áp dụng chuẩn phân lập tài khoản:
* **Tài khoản Công vụ (`EMPLOYEE`):** Sử dụng email công ty (`@omama.vn`) để đăng nhập và làm việc trên giao diện nội bộ `/admin/**`. Tài khoản vẫn được xem nội dung Storefront nhưng mọi action mua hàng bị ẩn và BFF từ chối lại bằng `EMPLOYEE_PURCHASE_NOT_ALLOWED`. Chỉ tài khoản này mới được gán quyền trong `iam_role_assignments`.
* **Tài khoản Tiêu dùng (`CUSTOMER`):** Sử dụng email cá nhân để mua kẹo mè xửng trên giao diện Storefront.
* **Quy tắc bảo mật bắt buộc:** `EMPLOYEE` không được dùng Cart, Checkout, Customer Order hoặc Address. Nhân viên mua sắm cá nhân phải sử dụng tài khoản `CUSTOMER` độc lập; legacy `customer_id` hoặc role `CUSTOMER` không mở quyền mua hàng cho Employee.

---

## 2. Các API Xác Thực & Phiên Làm Việc (BFF Pattern)

Hợp đồng chi tiết được đặc tả tại [`frontend-auth-contract.md`](./frontend-auth-contract.md) và [`identity-service.openapi.yaml`](./identity-service.openapi.yaml).

> **Bảo vệ Token theo RFC 9700:**
> - **Tuyệt đối không lưu Access Token hoặc Refresh Token trong `localStorage` hay `sessionStorage` của trình duyệt** để triệt tiêu hoàn toàn nguy cơ rò rỉ token khi bị tấn công XSS.
> - **Identity Service đóng vai trò BFF:** Quản lý Access Token và Refresh Token an toàn phía backend.
> - **Trình duyệt chỉ trao đổi qua Cookie HttpOnly:** Cookie `OMAMA_SESSION` (`HttpOnly`, `Secure`, `SameSite=Lax`).
> - Mọi request gọi API từ Frontend (Next.js/React/Axios/Fetch) bắt buộc phải cấu hình `credentials: 'include'` (hoặc `withCredentials: true`).

---

### 2.1. Khởi Tạo Đăng Nhập (`GET /api/v1/public/auth/login`)

Frontend gọi endpoint này khi người dùng bấm nút "Đăng nhập" hoặc chọn đăng nhập mạng xã hội (Google/Zalo):

- **Method & URL:** `GET https://api.omama.vn/api/v1/public/auth/login`
- **Query Parameters:**
  - `redirect_uri` *(optional)*: Đường dẫn trang Frontend cần quay lại sau khi đăng nhập (ví dụ: `https://omama.vn/checkout`). Mặc định: `https://omama.vn`.
  - `provider` *(optional)*: Tên nhà cung cấp SSO nếu muốn đi thẳng vào trang OAuth2 của bên thứ ba (`google`, `facebook`, `zalo`, `apple`). Để trống sẽ hiển thị trang đăng nhập Keycloak mang bản sắc Ô Mạ.
- **Response Web mặc định:** `302 Found` sang Keycloak/OIDC. Backend giữ PKCE verifier/state và tự thiết lập `OMAMA_SESSION` sau callback.
- **Hành động của Frontend:** điều hướng toàn trang; không gọi Axios và không nhận token.
  ```typescript
  window.location.href = `${apiBase}/public/auth/login?redirect_uri=${encodeURIComponent(returnUrl)}`;
  ```

---

### 2.2. Tiếp Nhận Callback & Thiết Lập Phiên (`GET /api/v1/public/auth/callback`)

- Endpoint này do **trình duyệt tự động chuyển hướng** sau khi người dùng xác thực thành công trên Keycloak / Google.
- BFF tự động đổi Authorization Code kèm PKCE Verifier lấy bộ Token JWT, ghi nhận JIT Provisioning (nếu là khách mới), lưu phiên vào `BffSessionStore` và trả về `HTTP 302 Redirect` về URL ban đầu kèm Header:
  ```http
  Set-Cookie: OMAMA_SESSION=d7b5b5c9-9481-4b72-9d3e-9080b06b2308; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax
  ```

---

### 2.3. Kiểm Tra Trạng Thái Phiên Đăng Nhập (`GET /api/v1/public/auth/session`)

Được Frontend gọi khi khởi động ứng dụng hoặc khi người dùng F5 tải lại trang:

- **Method & URL:** `GET https://api.omama.vn/api/v1/public/auth/session`
- **Request Options:** `credentials: 'include'`
- **Response `200 OK` (Đã đăng nhập):**
  ```json
  {
    "success": true,
    "code": "SUCCESS",
    "message": "Lấy thông tin phiên làm việc thành công",
    "data": {
      "sessionId": "d7b5b5c9-9481-4b72-9d3e-9080b06b2308",
      "principalId": "e9f8d7c6-1111-2222-3333-444455556666",
      "email": "customer@omama.vn",
      "authenticated": true,
      "expiresAt": "2026-11-01T15:30:00Z"
    }
  }
  ```
- **Response `200 OK` (Chưa đăng nhập hoặc phiên hết hạn):**
  ```json
  {
    "success": true,
    "code": "SUCCESS",
    "data": {
      "sessionId": null,
      "principalId": null,
      "email": null,
      "authenticated": false,
      "expiresAt": null
    }
  }
  ```

---

### 2.4. Nhận Diện Danh Sách Vai Trò & Quyền Hạn (`GET /api/v1/users/me`)

Khi phiên xác thực hợp lệ (`authenticated === true`), Frontend gọi endpoint này để nhận toàn bộ thông tin tài khoản, danh sách Vai trò (`roles`) và Quyền hạn (`permissions`):

- **Method & URL:** `GET https://api.omama.vn/api/v1/users/me`
- **Request Options:** `credentials: 'include'` (BFF Filter tự động chuyển Cookie sang JWT Principal).

#### Trường hợp A: Tài khoản là Khách hàng D2C (`CUSTOMER`)
```json
{
  "success": true,
  "code": "SUCCESS",
  "data": {
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
}
```

#### Trường hợp B: Tài khoản là Nhân viên / Quản lý (`EMPLOYEE`)
```json
{
  "success": true,
  "code": "SUCCESS",
  "data": {
    "principalId": "e9f8d7c6-1111-2222-3333-444455556666",
    "idpSubject": "kc-subject-uuid-employee",
    "email": "nhanvienkho@omama.vn",
    "principalType": "EMPLOYEE",
    "provisioningStatus": "ACTIVE",
    "roles": [
      {
        "roleCode": "CUSTOMER",
        "roleName": "Khách hàng thân thiết",
        "scopeType": "GLOBAL",
        "scopeId": null
      },
      {
        "roleCode": "WAREHOUSE_STAFF",
        "roleName": "Nhân viên kho vận",
        "scopeType": "WAREHOUSE",
        "scopeId": "WH_DANANG_01"
      },
      {
        "roleCode": "PACKING_STAFF",
        "roleName": "Nhân viên đóng gói",
        "scopeType": "STORE",
        "scopeId": "STORE_HUE_MAIN"
      }
    ],
    "permissions": [
      "INVENTORY_VIEW",
      "INVENTORY_RECEIPT_CREATE",
      "INVENTORY_ISSUE_CREATE",
      "INVENTORY_ADJUSTMENT_CREATE",
      "PACKING_WORKBENCH_VIEW",
      "PACKING_TASK_VIEW",
      "PACKING_TASK_UPDATE",
      "PACKING_CHECKLIST_CHECK"
    ]
  }
}
```

---

### 2.5. Đăng Xuất Tài Khoản (`POST /api/v1/public/auth/logout`)

- **Method & URL:** `POST https://api.omama.vn/api/v1/public/auth/logout`
- **Request Options:** `credentials: 'include'`
- **Hành vi Backend:** Hủy phiên trong `BffSessionStore`, thu hồi token tại Keycloak IdP, xóa Cookie qua Header:
  ```http
  Set-Cookie: OMAMA_SESSION=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax
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
  idpSubject: string;
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
  login: (provider?: string) => Promise<void>;
  reload: () => Promise<void>;
  logout: () => Promise<void>;
}

const AppSessionContext = createContext<AppSessionContextValue | null>(null);

export function AppSessionProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AppSessionContextValue["status"]>("loading");
  const [profile, setProfile] = useState<UserProfile | null>(null);

  const reload = useCallback(async () => {
    try {
      // 1. Kiểm tra session cookie HttpOnly
      const sessionRes = await fetch("/api/v1/public/auth/session", {
        headers: { Accept: "application/json" },
        credentials: "include", // Bắt buộc gửi OMAMA_SESSION cookie
        cache: "no-store",
      });

      if (!sessionRes.ok) {
        setProfile(null);
        setStatus("guest");
        return;
      }

      const sessionJson = await sessionRes.json();
      if (!sessionJson.data?.authenticated) {
        setProfile(null);
        setStatus("guest");
        return;
      }

      // 2. Lấy hồ sơ & phân quyền chi tiết
      const meRes = await fetch("/api/v1/users/me", {
        headers: { Accept: "application/json" },
        credentials: "include",
        cache: "no-store",
      });

      if (meRes.ok) {
        const meJson = await meRes.json();
        setProfile(meJson.data);
        setStatus("authenticated");
      } else {
        setProfile(null);
        setStatus("guest");
      }
    } catch (err) {
      console.error("Lỗi kiểm tra phiên làm việc:", err);
      setProfile(null);
      setStatus("error");
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const login = useCallback((provider?: string) => {
    const searchParams = new URLSearchParams({ redirect_uri: window.location.href });
    if (provider) searchParams.set("provider", provider);
    window.location.href = `/api/v1/public/auth/login?${searchParams.toString()}`;
  }, []);

  const isStaff = useMemo(() => {
    if (!profile) return false;
    return profile.principalType === "EMPLOYEE";
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
    try {
      await fetch("/api/v1/public/auth/logout", {
        method: "POST",
        credentials: "include",
      });
    } finally {
      setProfile(null);
      setStatus("guest");
      window.location.href = "/";
    }
  }, []);

  const value = useMemo(
    () => ({ status, profile, isStaff, hasRole, hasPermission, login, reload, logout }),
    [status, profile, isStaff, hasRole, hasPermission, login, reload, logout]
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
"use client";

import { useAppSession } from "@/components/auth/app-session-provider";
import Link from "next/link";

export function Header() {
  const { status, profile, isStaff, login, logout } = useAppSession();

  // Employee vẫn được xem catalog, nhưng header chỉ hiện Kênh Quản Trị;
  // Cart và các action mua hàng không được render.

  return (
    <header className="flex justify-between p-4 bg-white shadow">
      <Link href="/" className="font-bold text-xl text-amber-900">Mè Xửng O Mạ</Link>

      <div className="flex items-center gap-4">
        {isStaff ? <Link href="/admin">Kênh Quản Trị</Link> : <Link href="/cart">Giỏ hàng</Link>}

        {status === "loading" && <span>Đang tải...</span>}

        {status === "guest" && (
          <button onClick={() => login()} className="btn-primary">
            Đăng nhập
          </button>
        )}

        {status === "authenticated" && (
          <>
            <span>Xin chào, {profile?.email}</span>

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

### 3.3. Bảo Vệ Route Employee

Bảo vệ tất cả đường dẫn `/admin/**` bằng `EmployeeAccessBoundary` và kiểm tra lại ở `/api/admin/**`. Middleware dưới đây chỉ là lớp chuyển hướng sớm khi thiếu cookie; nó không thay thế việc gọi Identity để xác minh `principalType`.

```typescript
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Kiểm tra nếu truy cập vào khu vực Employee
  if (pathname.startsWith("/admin")) {
    const sessionCookie = request.cookies.get("OMAMA_SESSION")?.value;

    // Chưa đăng nhập -> đá về trang chủ hoặc kích hoạt flow đăng nhập
    if (!sessionCookie) {
      const loginUrl = new URL("/login", request.url);
      loginUrl.searchParams.set("redirect_uri", pathname);
      return NextResponse.redirect(loginUrl);
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/admin/:path*"],
};
```

---

### 3.4. Ẩn / Hiện Menu Con Trong Layout Employee (`/admin/layout.tsx`)

Trong trang quản trị, mỗi tab chức năng chỉ hiển thị khi tài khoản có quyền tương ứng trong danh sách `permissions`:

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

## 4. Tóm Tắt Cho Kỹ Sư Frontend

1. **Khởi tạo Đăng nhập:** Điều hướng toàn trang đến `GET /api/v1/public/auth/login`; backend trả `302` sang OIDC.
2. **Quản lý Phiên:** Dùng Cookie bảo mật `OMAMA_SESSION` (`HttpOnly`, `SameSite=Lax`). Luôn cấu hình `credentials: 'include'`.
3. **Kiểm tra Phiên & Quyền:**
   - `GET /api/v1/public/auth/session` -> kiểm tra `authenticated`.
   - `GET /api/v1/users/me` -> nhận danh mục `roles` và `permissions`.
4. **Phân biệt Khách hàng & Nhân sự:**
   - `principalType === 'EMPLOYEE'` $\longrightarrow$ **Nhân viên / Quản lý / Admin**.
   - `principalType === 'CUSTOMER'` $\longrightarrow$ **Khách hàng D2C**.
5. **Điều hướng & Trải nghiệm:**
   - Khách hàng D2C: Mua kẹo Ô Mạ, giỏ hàng, hồ sơ cá nhân.
   - Nhân viên: Có thể xem Storefront ở chế độ chỉ đọc; header hiện nút `/admin`, ẩn giỏ hàng và Product Detail thay action mua bằng cảnh báo. Menu nội bộ ẩn hiện theo `permissions`.
