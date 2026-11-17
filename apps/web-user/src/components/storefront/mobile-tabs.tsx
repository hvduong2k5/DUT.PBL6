import Link from "next/link";

type Props = {
  onMenu: () => void;
  onSearch: () => void;
  onCart: () => void;
  count: number | null;
  employee: boolean;
  onHome: boolean;
};
export function MobileTabs({
  onMenu,
  onSearch,
  onCart,
  count,
  employee,
  onHome,
}: Props) {
  return (
    <nav className="mobile-bottom-nav" aria-label="Thanh điều hướng điện thoại">
      <Link
        className={onHome ? "mobile-nav-btn is-active" : "mobile-nav-btn"}
        href="/"
      >
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          aria-hidden="true"
        >
          <path d="M3 10.5L12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1v-9.5z" />
        </svg>
        <span>Trang chủ</span>
      </Link>
      <button type="button" className="mobile-nav-btn" onClick={onMenu}>
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          aria-hidden="true"
        >
          <rect x="3" y="3" width="7" height="7" />
          <rect x="14" y="3" width="7" height="7" />
          <rect x="3" y="14" width="7" height="7" />
          <rect x="14" y="14" width="7" height="7" />
        </svg>
        <span>Danh mục</span>
      </button>
      <Link className="mobile-nav-btn" href="/#gift">
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          aria-hidden="true"
        >
          <rect x="3" y="8" width="18" height="13" rx="1" />
          <path d="M12 8v13M3 12h18M12 8c-2-3-5-3-5-1s3 1 5 1 5 1 5-1-3-2-5 1z" />
        </svg>
        <span>Gói quà</span>
      </Link>
      <button type="button" className="mobile-nav-btn" onClick={onSearch}>
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          aria-hidden="true"
        >
          <circle cx="11" cy="11" r="7" />
          <path d="M16 16l4.5 4.5" strokeLinecap="round" />
        </svg>
        <span>Tìm kiếm</span>
      </button>
      <button
        type="button"
        className="mobile-nav-btn"
        onClick={onCart}
        hidden={employee}
      >
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          aria-hidden="true"
        >
          <path d="M4 8h16l-1.2 12H5.2L4 8z" />
          <path d="M8 8V6a4 4 0 0 1 8 0v2" />
        </svg>
        <span>Giỏ quà</span>
        <span className="cart-badge count" hidden={count === null}>
          {count}
        </span>
      </button>
    </nav>
  );
}
