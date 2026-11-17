"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import {
  RecommendationPreferences,
  RecommendationRail,
} from "@/components/recommendation/recommendation-rail";
import { ProductImage } from "@/components/product/product-image";
import { collections } from "@/components/layout/storefront-links";
import { Hero } from "@/components/storefront/hero";
import { Story } from "@/components/storefront/story";
import { GiftBuilder } from "@/components/storefront/gift-builder";
import { Collection } from "@/components/storefront/collection";
import { ShopSection } from "@/components/storefront/shop-section";
import { CustomerServices } from "@/components/storefront/customer-services";
import { GroupOptions } from "@/components/storefront/group-options";
import { StorefrontDialog } from "@/components/storefront/dialog";
import { ItemDialog } from "@/components/storefront/item-dialog";
import { useStorefrontView } from "@/components/storefront/view-mode";
import { items, money, type HomeItem } from "@/components/storefront/items";

function Divider() {
  return (
    <div className="wrap">
      <div className="divider-hue" aria-hidden="true">
        <span className="divider-hue__knot" />
      </div>
    </div>
  );
}

export function HomeExperience() {
  const { paper, compact, group, setGroup } = useStorefrontView();
  const [sort, setSort] = useState("featured");
  const [filter, setFilter] = useState<"popup" | "sheet" | null>(null);
  const [detail, setDetail] = useState<HomeItem | null>(null);
  const [gift, setGift] = useState("");
  const bar = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const groupName = collections[group]?.name;
  useEffect(() => {
    if (!filter) return;
    const outside = (e: PointerEvent) => {
      if (filter === "popup" && !bar.current?.contains(e.target as Node))
        setFilter(null);
    };
    const key = (e: KeyboardEvent) => {
      if (e.key === "Escape" && filter === "popup") {
        setFilter(null);
        trigger.current?.focus();
      }
    };
    const media = window.matchMedia("(max-width: 767px)");
    const resize = () => setFilter(null);
    media.addEventListener("change", resize);
    document.addEventListener("pointerdown", outside);
    document.addEventListener("keydown", key);
    return () => {
      media.removeEventListener("change", resize);
      document.removeEventListener("pointerdown", outside);
      document.removeEventListener("keydown", key);
    };
  }, [filter]);
  const visible = items.filter((item) => group < 0 || item.group === group);
  if (sort !== "featured")
    visible.sort((a, b) =>
      sort === "price-asc" ? a.amount - b.amount : b.amount - a.amount,
    );
  function choose(index: number) {
    setGroup(index);
    document.getElementById("shop")?.scrollIntoView();
  }
  const filters = (
    <div className="filter-form">
      <label>
        Nhóm thức quà
        <select
          className="field"
          value={group}
          onChange={(e) => setGroup(Number(e.target.value))}
        >
          <option value={-1}>Tất cả</option>
          {collections.map((c, i) => (
            <option key={c.name} value={i}>
              {c.name}
            </option>
          ))}
        </select>
      </label>
      <label>
        Sắp xếp
        <select
          className="field"
          value={sort}
          onChange={(e) => setSort(e.target.value)}
        >
          <option value="featured">Tinh tuyển</option>
          <option value="price-asc">Giá tăng dần</option>
          <option value="price-desc">Giá giảm dần</option>
        </select>
      </label>
      <button className="btn btn--brown" onClick={() => setFilter(null)}>
        Áp dụng
      </button>
    </div>
  );
  return (
    <main id="main" className="hue-surface hue-home hue-watermark-canvas">
      <Hero paper={paper} />
      <Collection>
        <GroupOptions choose={choose} selected={group} />
      </Collection>
      <Divider />
      <ShopSection>
        <div className="shop-toolbar">
          <div>
            <span className="section-header__kicker">Tinh tuyển từ Huế</span>
            <h2>
              {groupName
                ? "Bộ sưu tập · " + groupName
                : "Khám phá bộ sưu tập Gói Huế"}
            </h2>
          </div>
          <div className="row filter-anchor" ref={bar}>
            <div
              className="filter-pills"
              role="group"
              aria-label="Lọc nhanh theo nhóm món"
            >
              {["Tất cả", ...collections.map((c) => c.name)].map((name, i) => (
                <button
                  className={"chip-btn" + (group === i - 1 ? " is-active" : "")}
                  aria-pressed={group === i - 1}
                  key={name}
                  onClick={() => setGroup(i - 1)}
                >
                  {name}
                </button>
              ))}
            </div>
            <button
              className="btn btn--paper"
              ref={trigger}
              aria-expanded={!!filter}
              onClick={() =>
                setFilter(
                  filter
                    ? null
                    : compact || window.innerWidth < 768
                      ? "sheet"
                      : "popup",
                )
              }
            >
              Bộ lọc &amp; Sắp xếp ⌄
            </button>
            {filter === "popup" && <div className="ux-dropdown">{filters}</div>}
          </div>
        </div>
        <div className="products-grid">
          {visible.map((item) => (
            <article key={item.skuId} className="hue-product-card">
              <button
                className="product-card__media"
                onClick={() => setDetail(item)}
                aria-label={"Xem chi tiết " + item.name}
              >
                <ProductImage
                  src={item.image}
                  width={480}
                  height={350}
                  alt={item.name}
                />
              </button>
              <div className="product-card__body">
                <span className="product-card__meta">
                  {collections[item.group]?.name ?? "Bánh Huế"} · {item.meta}
                </span>
                <h3 className="product-card__title">{item.name}</h3>
                <div className="product-card__foot">
                  <span className="price-tag">{money(item.amount)}</span>
                  <div className="row">
                    <button
                      className="btn btn--paper"
                      onClick={() => setDetail(item)}
                    >
                      Chi tiết
                    </button>
                    <button className="btn" onClick={() => setDetail(item)}>
                      {item.group === 2 ? "Chọn hộp" : "+ Giỏ quà"}
                    </button>
                  </div>
                </div>
              </div>
            </article>
          ))}
        </div>
      </ShopSection>
      <Divider />
      <GiftBuilder onReview={setGift} />
      <Divider />
      <div
        id="heritage-story"
        onClick={(e) => {
          if ((e.target as HTMLElement).closest("[data-nav-group]"))
            setGroup(1);
        }}
      >
        <Story />
      </div>
      <div className="wrap">
        <RecommendationPreferences />
        <RecommendationRail placement="home" />
      </div>
      <CustomerServices />
      {filter === "sheet" && (
        <StorefrontDialog
          title="Bộ lọc & Sắp xếp"
          close={() => setFilter(null)}
        >
          {filters}
        </StorefrontDialog>
      )}
      {detail && <ItemDialog item={detail} close={() => setDetail(null)} />}
      {gift && (
        <StorefrontDialog title="Hộp quà theo ý bạn" close={() => setGift("")}>
          <p>{gift}</p>
          <p>Hộp phối riêng cần được xác nhận trước khi đặt.</p>
          <Link className="btn btn--brown" href="/support/request">
            Liên hệ đặt hộp quà
          </Link>
        </StorefrontDialog>
      )}
    </main>
  );
}
