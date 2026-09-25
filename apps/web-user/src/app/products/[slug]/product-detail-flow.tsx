"use client";

import Image from "next/image";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import type { ProductDetail, ProductSku } from "@/lib/product-detail/types";
import { ProductDetailApiError } from "@/lib/product-detail/types";
import { productDetailService } from "@/services/product-detail-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

function formatPrice(product: ProductDetail, selectedSku?: ProductSku): string {
  if (selectedSku) return VND.format(selectedSku.priceVnd);
  const prices = product.skus.map((sku) => sku.priceVnd);
  if (!prices.length) return "Chưa công bố";
  const min = Math.min(...prices);
  const max = Math.max(...prices);
  return min === max ? VND.format(min) : `${VND.format(min)} – ${VND.format(max)}`;
}

function ProductDetailLoading() {
  return (
    <div className="detail-loading" aria-busy="true" aria-label="Đang tải chi tiết sản phẩm">
      <div className="detail-loading-gallery" />
      <div className="detail-loading-copy"><span /><span /><span /><span /></div>
    </div>
  );
}

export function ProductDetailFlow({ slug }: { slug: string }) {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [product, setProduct] = useState<ProductDetail | null>(null);
  const [selectedSkuId, setSelectedSkuId] = useState<string>();
  const [activeImageUrl, setActiveImageUrl] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ status?: number; message: string }>();
  const [selectionMessage, setSelectionMessage] = useState("");
  const [retryKey, setRetryKey] = useState(0);

  useEffect(() => {
    let active = true;
    Promise.resolve().then(() => {
      if (!active) return Promise.reject(new Error("cancelled"));
      setLoading(true);
      setError(undefined);
      setProduct(null);
      setSelectedSkuId(undefined);
      setSelectionMessage("");
      return productDetailService.getBySlug(slug, scenario);
    }).then((payload) => {
      if (!active) return;
      setProduct(payload);
      setActiveImageUrl(payload.images[0]?.url);
      if (payload.skus.length === 1 && payload.skus[0].isAvailable) {
        setSelectedSkuId(payload.skus[0].skuId);
      }
    }).catch((cause: unknown) => {
      if (!active) return;
      if (cause instanceof ProductDetailApiError) setError({ status: cause.status, message: cause.message });
      else setError({ message: "Không thể tải chi tiết sản phẩm lúc này." });
    }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [retryKey, scenario, slug]);

  const selectedSku = useMemo(
    () => product?.skus.find((sku) => sku.skuId === selectedSkuId),
    [product, selectedSkuId]
  );
  const hasAvailableSku = product?.skus.some((sku) => sku.isAvailable) ?? false;

  function selectSku(sku: ProductSku) {
    if (!sku.isAvailable) return;
    setSelectedSkuId(sku.skuId);
    setSelectionMessage("");
    if (sku.imageUrl) setActiveImageUrl(sku.imageUrl);
  }

  function confirmIntegrationPoint() {
    if (!selectedSku) {
      setSelectionMessage("Vui lòng chọn một quy cách đang có thể mua trước khi tiếp tục.");
      return;
    }
    setSelectionMessage(`Đã chọn ${selectedSku.label} (${selectedSku.skuId}). Tính năng giỏ hàng sẽ được hoàn thiện trong EPIC 05; sản phẩm chưa được thêm vào giỏ.`);
  }

  if (loading) return <main className="product-detail-page"><ProductDetailLoading /></main>;

  if (error) {
    const notFound = error.status === 404;
    return (
      <main className="product-detail-page">
        <section className="detail-state" role="alert">
          <span aria-hidden="true">{notFound ? "⌕" : "!"}</span>
          <p className="eyebrow">{notFound ? "Không tìm thấy" : "Kết nối gián đoạn"}</p>
          <h1>{notFound ? "Sản phẩm chưa sẵn sàng để xem" : "Chưa thể mở chi tiết sản phẩm"}</h1>
          <p>{notFound ? "Sản phẩm có thể không tồn tại hoặc chưa được phép công khai." : error.message}</p>
          <div>{!notFound ? <button className="primary-button" type="button" onClick={() => setRetryKey((value) => value + 1)}>Thử tải lại</button> : null}<Link href="/products">Trở về danh mục</Link></div>
        </section>
      </main>
    );
  }

  if (!product) return null;

  const activeImage = activeImageUrl ?? product.images[0]?.url;
  return (
    <main className="product-detail-page" aria-live="polite">
      <nav className="detail-breadcrumbs" aria-label="Breadcrumb">
        <Link href="/">Trang chủ</Link><span aria-hidden="true">/</span>
        <Link href={`/products?category=${product.category.slug}`}>{product.category.name}</Link><span aria-hidden="true">/</span>
        <strong>{product.name}</strong>
      </nav>

      <section className="detail-hero">
        <div className="product-gallery">
          <div className="gallery-main">
            {activeImage ? <Image src={activeImage} alt={product.images.find((image) => image.url === activeImage)?.alt ?? product.name} fill priority loading="eager" sizes="(max-width: 860px) 100vw, 52vw" /> : <div className="gallery-empty">Chưa có ảnh sản phẩm</div>}
            {product.ocopCertification ? <span className="gallery-certification">✺ {product.ocopCertification.label}</span> : null}
          </div>
          {product.images.length > 1 ? <div className="gallery-thumbnails" aria-label="Ảnh sản phẩm">{product.images.map((image) => <button type="button" className={activeImage === image.url ? "active" : ""} aria-pressed={activeImage === image.url} onClick={() => setActiveImageUrl(image.url)} key={image.id}><Image src={image.url} alt={image.alt} fill sizes="92px" /></button>)}</div> : null}
        </div>

        <div className="detail-purchase-panel">
          <p className="eyebrow">{product.category.name}</p>
          <h1>{product.name}</h1>
          <p className="detail-lead">{product.shortDescription}</p>

          <div className="detail-price" aria-live="polite">
            <span>{selectedSku ? "Giá bán của quy cách đã chọn" : "Khoảng giá theo quy cách"}</span>
            <strong>{formatPrice(product, selectedSku)}</strong>
            <small>Giá bán Website, chưa áp dụng khuyến mãi.</small>
          </div>

          <fieldset className="sku-picker" aria-describedby="sku-guidance sku-message">
            <legend>Chọn quy cách sản phẩm</legend>
            <p id="sku-guidance">{product.skus.length > 1 ? "Chọn rõ một SKU trước khi chuyển sang luồng mua." : "Sản phẩm này có một quy cách công khai."}</p>
            <div>{product.skus.map((sku) => (
              <label className={`${selectedSkuId === sku.skuId ? "selected" : ""} ${!sku.isAvailable ? "unavailable" : ""}`} key={sku.skuId}>
                <input type="radio" name="product-sku" value={sku.skuId} checked={selectedSkuId === sku.skuId} disabled={!sku.isAvailable} onChange={() => selectSku(sku)} />
                <span className="sku-radio" aria-hidden="true" />
                <span className="sku-copy"><strong>{sku.label}</strong><small>{sku.weightGrams}g · {sku.flavor || "Hương vị nguyên bản"} · {sku.packageType}</small><b>{VND.format(sku.priceVnd)}</b>{!sku.isAvailable ? <em>{sku.unavailableReason || "Tạm thời không khả dụng"}</em> : null}</span>
              </label>
            ))}</div>
          </fieldset>

          <div className="selected-sku-summary">
            <span>Quy cách đang xác nhận</span>
            {selectedSku ? <dl><div><dt>Khối lượng</dt><dd>{selectedSku.weightGrams}g</dd></div><div><dt>Hương vị</dt><dd>{selectedSku.flavor || "Nguyên bản"}</dd></div><div><dt>Đóng gói</dt><dd>{selectedSku.packageType}</dd></div><div><dt>Khả dụng</dt><dd className="available">Có thể chọn</dd></div></dl> : <p>Chưa chọn SKU. Không có quy cách nào được ngầm xác nhận.</p>}
          </div>

          <button className="cart-integration-button" type="button" disabled={!hasAvailableSku} onClick={confirmIntegrationPoint}>{hasAvailableSku ? (selectedSku ? "Giỏ hàng sắp ra mắt" : "Chọn SKU để tiếp tục") : "Tạm thời không thể mua"}</button>
          {selectionMessage ? <p id="sku-message" className={selectedSku ? "integration-note" : "selection-error"} role={selectedSku ? "status" : "alert"}>{selectionMessage}</p> : <p id="sku-message" className="integration-caption">Integration point cho Shopping Cart — chưa phát sinh yêu cầu thêm giỏ.</p>}
        </div>
      </section>

      {!hasAvailableSku ? <section className="availability-banner" role="status"><span aria-hidden="true">◇</span><div><h2>Sản phẩm đang tạm hết khả dụng</h2><p>Bạn vẫn có thể xem thông tin sản phẩm. Các SKU sẽ không thể chọn cho đến khi trạng thái bán được cập nhật.</p></div></section> : null}

      <section className="product-story" aria-labelledby="product-story-title">
        <div><p className="eyebrow">Thông tin đã công bố</p><h2 id="product-story-title">Câu chuyện của thức quà</h2><p>{product.longDescription}</p></div>
        <aside><span>Phân loại</span><strong>{product.category.name}</strong><span>Dòng sản phẩm</span><strong>{product.productType}</strong>{product.ocopCertification ? <><span>Chứng nhận</span><strong>{product.ocopCertification.label}</strong></> : null}</aside>
      </section>

      <section className="food-information" aria-labelledby="food-title">
        <div className="section-heading"><p className="eyebrow">Thông tin thực phẩm</p><h2 id="food-title">Hiểu rõ trước khi lựa chọn</h2><p>Dữ liệu catalog chung; ngày cụ thể của Batch/Lot không được hiển thị tại đây.</p></div>
        <div className="food-grid">
          <article><span aria-hidden="true">✦</span><h3>Thành phần</h3><p>{product.foodInformation.ingredients}</p>{product.foodInformation.allergenStatement ? <div className="allergen-note"><strong>Cảnh báo dị ứng</strong><p>{product.foodInformation.allergenStatement}</p></div> : null}</article>
          <article><span aria-hidden="true">⌂</span><h3>Bảo quản</h3><p>{product.foodInformation.storageInstructions}</p></article>
          <article><span aria-hidden="true">◷</span><h3>Ngày sản xuất &amp; hạn dùng</h3><dl><dt>Ngày sản xuất</dt><dd>{product.foodInformation.manufacturingDatePolicy}</dd><dt>Hạn sử dụng catalog</dt><dd>{product.foodInformation.shelfLifeDescription}</dd></dl><small>Đây là chính sách shelf-life, không phải ngày của lô hàng sẽ giao.</small></article>
        </div>
      </section>
    </main>
  );
}
