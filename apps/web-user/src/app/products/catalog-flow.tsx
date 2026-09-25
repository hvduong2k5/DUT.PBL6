"use client";

import Image from "next/image";
import Link from "next/link";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import type { CatalogResult, DiscoveryConfig, PricePreset, ProductSummary, ProductType } from "@/lib/catalog/types";
import { CatalogApiError } from "@/lib/catalog/types";
import { catalogService } from "@/services/catalog-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const SORT_OPTIONS = [
  ["CURATED", "Tuyển chọn"],
  ["RELEVANCE", "Liên quan nhất"],
  ["PRICE_ASC", "Giá thấp đến cao"],
  ["PRICE_DESC", "Giá cao đến thấp"],
  ["NAME_ASC", "Tên A–Z"]
] as const;

function samePricePreset(params: URLSearchParams, preset: PricePreset): boolean {
  return (params.get("minPrice") ?? undefined) === (preset.minPrice?.toString() ?? undefined)
    && (params.get("maxPrice") ?? undefined) === (preset.maxPrice?.toString() ?? undefined);
}

function ProductCard({ product }: { product: ProductSummary }) {
  const offer = product.matchedOffer;
  return (
    <article className="product-card">
      <Link className="product-image" href={`/products/${product.slug}`} aria-label={`Xem chi tiết ${product.name}`}>
        <Image src={product.imageUrl} alt={product.name} fill sizes="(max-width: 760px) 100vw, (max-width: 1100px) 50vw, 30vw" />
        <div className="product-badges">
          {product.ocopStars ? <span>OCOP {product.ocopStars} sao</span> : null}
          {product.badges.slice(0, 1).map((badge) => <span className="soft" key={badge}>{badge}</span>)}
        </div>
        {!offer.isAvailable ? <span className="sold-out-badge">Tạm hết hàng</span> : null}
      </Link>
      <div className="product-card-content">
        <p className="product-category">{product.categoryName}</p>
        <h2><Link href={`/products/${product.slug}`}>{product.name}</Link></h2>
        <p className="product-description">{product.shortDescription}</p>
        <div className="product-offer-row">
          <div><strong>{VND.format(offer.priceVnd)}</strong><span>{offer.label}</span></div>
          <span className={offer.isAvailable ? "stock-status" : "stock-status unavailable"}>{offer.isAvailable ? "Còn hàng" : "Hết hàng"}</span>
        </div>
        <Link className="product-detail-link" href={`/products/${product.slug}`}>Xem chi tiết &amp; chọn quy cách <span aria-hidden="true">→</span></Link>
      </div>
    </article>
  );
}

export function CatalogFlow() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const queryString = searchParams.toString();
  const [keyword, setKeyword] = useState(searchParams.get("q") ?? "");
  const [searchError, setSearchError] = useState("");
  const [config, setConfig] = useState<DiscoveryConfig | null>(null);
  const [result, setResult] = useState<CatalogResult | null>(null);
  const [configError, setConfigError] = useState("");
  const [catalogError, setCatalogError] = useState("");
  const [loadingConfig, setLoadingConfig] = useState(true);
  const [loadingCatalog, setLoadingCatalog] = useState(true);
  const [retryKey, setRetryKey] = useState(0);

  const params = useMemo(() => new URLSearchParams(queryString), [queryString]);
  const scenario = params.get("mockScenario") ?? undefined;

  useEffect(() => {
    const urlKeyword = params.get("q") ?? "";
    Promise.resolve().then(() => setKeyword(urlKeyword));
  }, [params]);

  useEffect(() => {
    let active = true;
    Promise.resolve().then(() => {
      if (!active) return Promise.reject(new Error("cancelled"));
      setLoadingConfig(true);
      setConfigError("");
      return catalogService.getConfig(scenario);
    })
      .then((payload) => { if (active) setConfig(payload); })
      .catch((error: unknown) => { if (active) setConfigError(error instanceof Error ? error.message : "Không thể tải bộ lọc."); })
      .finally(() => { if (active) setLoadingConfig(false); });
    return () => { active = false; };
  }, [scenario, retryKey]);

  useEffect(() => {
    let active = true;
    Promise.resolve().then(() => {
      if (!active) return Promise.reject(new Error("cancelled"));
      setLoadingCatalog(true);
      setCatalogError("");
      return catalogService.getProducts(new URLSearchParams(queryString));
    })
      .then((payload) => { if (active) setResult(payload); })
      .catch((error: unknown) => {
        if (!active) return;
        setCatalogError(error instanceof CatalogApiError ? error.message : "Không thể tải danh mục sản phẩm.");
      })
      .finally(() => { if (active) setLoadingCatalog(false); });
    return () => { active = false; };
  }, [queryString, retryKey]);

  function navigate(mutator: (next: URLSearchParams) => void, resetPage = true) {
    const next = new URLSearchParams(queryString);
    mutator(next);
    if (resetPage) next.delete("page");
    const nextQuery = next.toString();
    router.push(nextQuery ? `${pathname}?${nextQuery}` : pathname, { scroll: false });
  }

  function submitSearch(event: FormEvent) {
    event.preventDefault();
    const normalized = keyword.trim().replace(/\s+/gu, " ");
    if (normalized.length < 2) {
      setSearchError("Vui lòng nhập ít nhất 2 ký tự để tìm kiếm.");
      return;
    }
    setSearchError("");
    navigate((next) => {
      next.set("q", normalized);
      if (!next.has("sort") || next.get("sort") === "CURATED") next.set("sort", "RELEVANCE");
    });
  }

  function toggleWeight(weight: number) {
    navigate((next) => {
      const current = new Set((next.get("weights") ?? "").split(",").filter(Boolean).map(Number));
      if (current.has(weight)) current.delete(weight); else current.add(weight);
      const value = [...current].sort((a, b) => a - b).join(",");
      if (value) next.set("weights", value); else next.delete("weights");
    });
  }

  function setPrice(preset?: PricePreset) {
    navigate((next) => {
      next.delete("minPrice");
      next.delete("maxPrice");
      if (preset?.minPrice !== undefined) next.set("minPrice", String(preset.minPrice));
      if (preset?.maxPrice !== undefined) next.set("maxPrice", String(preset.maxPrice));
    });
  }

  function resetFilters(clearKeyword = false) {
    navigate((next) => {
      ["category", "productType", "minPrice", "maxPrice", "weights", "inStock"].forEach((key) => next.delete(key));
      if (clearKeyword) {
        next.delete("q");
        next.delete("sort");
        setKeyword("");
      }
    });
  }

  const chips = useMemo(() => {
    if (!config) return [];
    const items: Array<{ key: string; label: string; clear: (next: URLSearchParams) => void }> = [];
    const category = config.categories.find((item) => item.slug === params.get("category"));
    const productType = config.productTypes.find((item) => item.value === params.get("productType"));
    const weights = (params.get("weights") ?? "").split(",").filter(Boolean).map(Number);
    const price = config.pricePresets.find((preset) => samePricePreset(params, preset));
    if (params.get("q")) items.push({ key: "q", label: `“${params.get("q")}”`, clear: (next) => { next.delete("q"); next.delete("sort"); setKeyword(""); } });
    if (category) items.push({ key: "category", label: category.name, clear: (next) => next.delete("category") });
    if (productType) items.push({ key: "productType", label: productType.label, clear: (next) => next.delete("productType") });
    if (price) items.push({ key: "price", label: price.label, clear: (next) => { next.delete("minPrice"); next.delete("maxPrice"); } });
    weights.forEach((weight) => items.push({ key: `weight-${weight}`, label: `${weight}g`, clear: (next) => {
      const remaining = weights.filter((item) => item !== weight);
      if (remaining.length) next.set("weights", remaining.join(",")); else next.delete("weights");
    } }));
    if (params.get("inStock") === "true") items.push({ key: "inStock", label: "Chỉ sản phẩm còn hàng", clear: (next) => next.delete("inStock") });
    return items;
  }, [config, params]);

  const currentPage = Number(params.get("page") ?? 1);
  const pageSize = Number(params.get("pageSize") ?? 9);
  const totalPages = Math.max(1, Math.ceil((result?.filteredCount ?? 0) / pageSize));

  return (
    <main className="catalog-page">
      <div className="catalog-breadcrumbs"><Link href="/">Trang chủ</Link><span>/</span><strong>Khám phá sản phẩm</strong></div>
      <section className="catalog-hero">
        <div><span className="eyebrow">Tinh tuyển di sản</span><h1>Khám phá thức quà xứ Huế</h1><p>Tìm món quà vừa ý từ những hương vị được gìn giữ qua nhiều thế hệ.</p></div>
        <form className="catalog-search" onSubmit={submitSearch} noValidate>
          <label htmlFor="catalog-keyword">Tìm trong kho thức quà</label>
          <div><input id="catalog-keyword" value={keyword} onChange={(event) => { setKeyword(event.target.value); setSearchError(""); }} placeholder="Thử “mè xửng”, “hạt sen”, “trà”…" aria-invalid={Boolean(searchError)} /><button type="submit">Tìm kiếm</button></div>
          {searchError ? <small role="alert">{searchError}</small> : null}
        </form>
      </section>

      <div className="catalog-layout">
        <aside className="catalog-filters" aria-label="Bộ lọc sản phẩm">
          <div className="filter-title"><div><span aria-hidden="true">⌁</span><h2>Bộ lọc</h2></div><button type="button" onClick={() => resetFilters()}>Đặt lại</button></div>
          {loadingConfig ? <div className="filter-loading">Đang tải bộ lọc…</div> : null}
          {configError ? <div className="filter-error"><p>{configError}</p><button type="button" onClick={() => setRetryKey((value) => value + 1)}>Thử lại</button></div> : null}
          {config ? <>
            <fieldset><legend>Danh mục</legend><label className={!params.get("category") ? "selected" : ""}><input type="radio" name="category" checked={!params.get("category")} onChange={() => navigate((next) => next.delete("category"))} />Tất cả thức quà</label>{config.categories.map((category) => <label className={params.get("category") === category.slug ? "selected" : ""} key={category.slug}><input type="radio" name="category" checked={params.get("category") === category.slug} onChange={() => navigate((next) => next.set("category", category.slug))} />{category.name}</label>)}</fieldset>
            <fieldset><legend>Khoảng giá</legend><label className={!params.get("minPrice") && !params.get("maxPrice") ? "selected" : ""}><input type="radio" name="price" checked={!params.get("minPrice") && !params.get("maxPrice")} onChange={() => setPrice()} />Tất cả mức giá</label>{config.pricePresets.map((preset) => <label className={samePricePreset(params, preset) ? "selected" : ""} key={preset.label}><input type="radio" name="price" checked={samePricePreset(params, preset)} onChange={() => setPrice(preset)} />{preset.label}</label>)}</fieldset>
            <fieldset><legend>Khối lượng</legend><div className="weight-options">{config.weightOptions.map((weight) => { const checked = (params.get("weights") ?? "").split(",").includes(String(weight)); return <label className={checked ? "selected" : ""} key={weight}><input type="checkbox" checked={checked} onChange={() => toggleWeight(weight)} />{weight}g</label>; })}</div></fieldset>
            <fieldset><legend>Loại sản phẩm</legend>{config.productTypes.map((type) => <label className={params.get("productType") === type.value ? "selected" : ""} key={type.value}><input type="checkbox" checked={params.get("productType") === type.value} onChange={() => navigate((next) => { if (next.get("productType") === type.value) next.delete("productType"); else next.set("productType", type.value satisfies ProductType); })} />{type.label}</label>)}</fieldset>
            <fieldset className="availability-filter"><legend>Khả dụng</legend><label className={params.get("inStock") === "true" ? "selected" : ""}><input type="checkbox" checked={params.get("inStock") === "true"} onChange={(event) => navigate((next) => { if (event.target.checked) next.set("inStock", "true"); else next.delete("inStock"); })} /><span><b>Chỉ sản phẩm còn hàng</b><small>Không tính lô hết hạn</small></span></label></fieldset>
            <div className="rating-coming"><span>★</span><div><b>Đánh giá khách hàng</b><small>Sẽ mở khi có dữ liệu thật từ hệ thống đánh giá.</small></div></div>
          </> : null}
        </aside>

        <section className="catalog-results" aria-live="polite" aria-busy={loadingCatalog}>
          <div className="results-toolbar">
            <div><span className="eyebrow">Bộ sưu tập Ô Mạ</span><h2>{params.get("q") ? `Kết quả cho “${params.get("q")}”` : "Tất cả sản phẩm"}</h2><p>{loadingCatalog ? "Đang tìm thức quà phù hợp…" : `${result?.filteredCount ?? 0} sản phẩm phù hợp`}</p></div>
            <label>Sắp xếp<select value={params.get("sort") ?? (params.get("q") ? "RELEVANCE" : "CURATED")} onChange={(event) => navigate((next) => next.set("sort", event.target.value))}>{SORT_OPTIONS.map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select></label>
          </div>

          {chips.length ? <div className="filter-chips" aria-label="Bộ lọc đang dùng">{chips.map((chip) => <button key={chip.key} type="button" onClick={() => navigate(chip.clear)}>{chip.label}<span aria-hidden="true">×</span></button>)}<button className="clear-filters" type="button" onClick={() => resetFilters(true)}>Xóa tất cả</button></div> : null}

          {loadingCatalog ? <div className="catalog-loading"><span className="large-spinner" /><p>Đang tìm thức quà phù hợp…</p></div> : null}
          {!loadingCatalog && catalogError ? <div className="catalog-state error-state"><span aria-hidden="true">!</span><h2>Kho thức quà đang gián đoạn</h2><p>{catalogError}</p><button className="primary-button" type="button" onClick={() => setRetryKey((value) => value + 1)}>Thử tải lại</button></div> : null}
          {!loadingCatalog && !catalogError && result?.items.length === 0 ? <div className="catalog-state"><span aria-hidden="true">⌕</span><h2>Chưa tìm thấy thức quà phù hợp</h2><p>Hãy thử từ khóa rộng hơn hoặc bớt một vài bộ lọc.</p><button className="primary-button" type="button" onClick={() => resetFilters(true)}>Xem toàn bộ sản phẩm</button></div> : null}
          {!loadingCatalog && !catalogError && result?.items.length ? <div className="product-grid">{result.items.map((product) => <ProductCard product={product} key={product.id} />)}</div> : null}

          {!loadingCatalog && !catalogError && result && result.filteredCount > pageSize ? <nav className="catalog-pagination" aria-label="Phân trang sản phẩm"><button type="button" disabled={currentPage <= 1} onClick={() => navigate((next) => next.set("page", String(currentPage - 1)), false)}>← Trước</button><span>Trang <strong>{currentPage}</strong> / {totalPages}</span><button type="button" disabled={currentPage >= totalPages} onClick={() => navigate((next) => next.set("page", String(currentPage + 1)), false)}>Sau →</button></nav> : null}
        </section>
      </div>
    </main>
  );
}
