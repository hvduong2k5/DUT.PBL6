"use client";

import Link from "next/link";
import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { AccountShell } from "@/components/account/account-shell";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import {
  AddressInput,
  AddressType,
  AdministrativeArea,
  CustomerAddress,
  CustomerApiError,
  CustomerProfile
} from "@/lib/customer/types";
import { FieldErrors, normalizePhone, validateAddress } from "@/lib/customer/validation";
import { customerService } from "@/services/customer-service";

interface AreaOption {
  province: AdministrativeArea;
  districts: Array<{ area: AdministrativeArea; wards: AdministrativeArea[] }>;
}

const AREAS: AreaOption[] = [
  {
    province: { code: "VN-HUE", name: "Thành phố Huế" },
    districts: [
      { area: { code: "HUE-CENTER", name: "Khu vực trung tâm Huế" }, wards: [
        { code: "PHU-HOI", name: "Phường Phú Hội" }, { code: "THUAN-HOA", name: "Phường Thuận Hóa" }, { code: "AN-CUU", name: "Phường An Cựu" }
      ] }
    ]
  },
  {
    province: { code: "VN-DNG", name: "Thành phố Đà Nẵng" },
    districts: [
      { area: { code: "NGU-HANH-SON", name: "Ngũ Hành Sơn" }, wards: [
        { code: "HOA-HAI", name: "Phường Hòa Hải" }, { code: "MY-AN", name: "Phường Mỹ An" }
      ] }
    ]
  }
];

const EMPTY_AREA = { code: "", name: "" };
const EMPTY_FORM: AddressInput = {
  label: "", type: "HOME", recipientName: "", recipientPhone: "", province: EMPTY_AREA,
  district: null, ward: EMPTY_AREA, addressLine: "", deliveryNote: null, isDefault: false
};

const TYPE_LABEL: Record<AddressType, string> = { HOME: "Nhà riêng", OFFICE: "Văn phòng", GIFT: "Biếu tặng", OTHER: "Khác" };

function addressToInput(address: CustomerAddress): AddressInput {
  return {
    label: address.label, type: address.type, recipientName: address.recipientName,
    recipientPhone: address.recipientPhone, province: address.province, district: address.district,
    ward: address.ward, addressLine: address.addressLine, deliveryNote: address.deliveryNote,
    isDefault: address.isDefault
  };
}

export function AddressBookFlow() {
  const started = useRef(false);
  const [profile, setProfile] = useState<CustomerProfile | null>(null);
  const [addresses, setAddresses] = useState<CustomerAddress[]>([]);
  const [form, setForm] = useState<AddressInput>(EMPTY_FORM);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [errors, setErrors] = useState<FieldErrors>({});
  const [notice, setNotice] = useState<{ kind: "success" | "error"; text: string } | null>(null);
  const [loading, setLoading] = useState(true);
  const [busyAction, setBusyAction] = useState<string | null>(null);

  const selectedProvince = useMemo(() => AREAS.find((item) => item.province.code === form.province.code), [form.province.code]);
  const selectedDistrict = useMemo(() => selectedProvince?.districts.find((item) => item.area.code === form.district?.code), [selectedProvince, form.district?.code]);

  async function load() {
    setLoading(true);
    setNotice(null);
    try {
      const [profileResult, addressResult] = await Promise.all([customerService.getProfile(), customerService.getAddresses()]);
      setProfile(profileResult);
      setAddresses(addressResult.items);
    } catch (error) {
      setNotice({ kind: "error", text: error instanceof Error ? error.message : "Không thể tải sổ địa chỉ." });
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    Promise.all([customerService.getProfile(), customerService.getAddresses()])
      .then(([profileResult, addressResult]) => { setProfile(profileResult); setAddresses(addressResult.items); })
      .catch((error: unknown) => setNotice({ kind: "error", text: error instanceof Error ? error.message : "Không thể tải sổ địa chỉ." }))
      .finally(() => setLoading(false));
  }, []);

  function update<K extends keyof AddressInput>(key: K, value: AddressInput[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setErrors((current) => ({ ...current, [key]: "" }));
    setNotice(null);
  }

  function openCreate() {
    setEditingId(null);
    setForm({ ...EMPTY_FORM, recipientName: profile?.fullName ?? "", recipientPhone: profile?.phone ?? "", isDefault: addresses.length === 0 });
    setErrors({});
    setNotice(null);
    setShowForm(true);
  }

  function openEdit(address: CustomerAddress) {
    setEditingId(address.id);
    setForm(addressToInput(address));
    setErrors({});
    setNotice(null);
    setShowForm(true);
    window.setTimeout(() => document.getElementById("address-editor")?.scrollIntoView({ behavior: "smooth", block: "start" }), 0);
  }

  function closeForm() {
    setShowForm(false);
    setEditingId(null);
    setErrors({});
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const clientErrors = validateAddress(form);
    setErrors(clientErrors);
    if (Object.keys(clientErrors).length) return;

    const payload: AddressInput = {
      ...form,
      label: form.label.trim(),
      recipientName: form.recipientName.trim(),
      recipientPhone: normalizePhone(form.recipientPhone),
      addressLine: form.addressLine.trim(),
      deliveryNote: form.deliveryNote?.trim() || null
    };
    setBusyAction("save");
    try {
      if (editingId) {
        const current = addresses.find((item) => item.id === editingId);
        if (!current) return;
        const result = await customerService.updateAddress(editingId, payload, current.version);
        const merged = { ...result, ...payload, id: editingId, version: result.version || current.version + 1 } as CustomerAddress;
        setAddresses((items) => items.map((item) => item.id === editingId ? merged : payload.isDefault ? { ...item, isDefault: false } : item));
        setNotice({ kind: "success", text: "Địa chỉ đã được cập nhật." });
      } else {
        const result = await customerService.createAddress(payload);
        const created = { ...result, ...payload, id: result.id } as CustomerAddress;
        setAddresses((items) => [...(payload.isDefault ? items.map((item) => ({ ...item, isDefault: false })) : items), created]);
        setNotice({ kind: "success", text: "Địa chỉ mới đã được thêm vào sổ." });
      }
      closeForm();
    } catch (error) {
      if (error instanceof CustomerApiError) {
        setErrors(Object.fromEntries(error.errors.map((item) => [item.field, item.message])));
        setNotice({ kind: "error", text: error.code === "VERSION_CONFLICT" ? "Địa chỉ vừa thay đổi ở nơi khác. Hãy tải lại trước khi lưu." : error.message });
      } else setNotice({ kind: "error", text: "Không thể lưu địa chỉ lúc này." });
    } finally {
      setBusyAction(null);
    }
  }

  async function remove(address: CustomerAddress) {
    if (!window.confirm(`Xóa địa chỉ “${address.label}”? Đơn hàng đã tạo vẫn giữ nguyên thông tin giao nhận.`)) return;
    setBusyAction(`delete-${address.id}`);
    setNotice(null);
    try {
      await customerService.deleteAddress(address.id, address.version);
      setAddresses((items) => {
        const remaining = items.filter((item) => item.id !== address.id);
        if (address.isDefault && remaining.length) remaining[0] = { ...remaining[0], isDefault: true };
        return remaining;
      });
      if (editingId === address.id) closeForm();
      setNotice({ kind: "success", text: "Địa chỉ đã được xóa khỏi sổ. Đơn hàng cũ không bị ảnh hưởng." });
    } catch (error) {
      setNotice({ kind: "error", text: error instanceof Error ? error.message : "Không thể xóa địa chỉ." });
    } finally { setBusyAction(null); }
  }

  async function setDefault(address: CustomerAddress) {
    setBusyAction(`default-${address.id}`);
    setNotice(null);
    try {
      await customerService.setDefaultAddress(address.id, address.version);
      setAddresses((items) => items.map((item) => ({ ...item, isDefault: item.id === address.id })));
      setNotice({ kind: "success", text: `“${address.label}” hiện là địa chỉ mặc định.` });
    } catch (error) {
      setNotice({ kind: "error", text: error instanceof Error ? error.message : "Không thể đổi địa chỉ mặc định." });
    } finally { setBusyAction(null); }
  }

  return <><SiteHeader /><AccountShell active="addresses" customerName={profile?.fullName} customerEmail={profile?.email}>
    <section className="account-hero address-hero">
      <div><span className="eyebrow">Giao nhận thuận tiện</span><h1>Sổ địa chỉ nhận hàng</h1><p>Quản lý địa chỉ của bạn và người thân để dùng nhanh ở Checkout.</p></div>
      <button className="primary-button" onClick={openCreate}>＋ Thêm địa chỉ mới</button>
    </section>
    {notice && <div className={`status-notice account-notice ${notice.kind}`} role="status">{notice.text}</div>}

    {loading ? <section className="account-card center-state"><span className="large-spinner" /><p>Đang tải những điểm đến quen thuộc…</p></section> : !profile ?
      <section className="account-card center-state"><span className="state-icon">!</span><h2>Chưa thể mở sổ địa chỉ</h2><p>{notice?.text}</p><div className="state-actions"><button className="secondary-button" onClick={() => void load()}>Thử lại</button><Link className="primary-link" href="/login?returnUrl=/account/addresses">Đăng nhập lại</Link></div></section> :
      <section aria-labelledby="address-list-title">
        <div className="section-title"><div><span className="eyebrow">Địa chỉ đang áp dụng</span><h2 id="address-list-title">{addresses.length} điểm nhận hàng</h2></div><span>Mặc định được ưu tiên khi Checkout</span></div>
        {addresses.length === 0 ? <div className="account-card empty-address"><span>⌖</span><h3>Sổ địa chỉ còn trống</h3><p>Thêm địa chỉ đầu tiên để rút ngắn bước nhập thông tin nhận hàng.</p><button className="primary-button" onClick={openCreate}>Thêm địa chỉ đầu tiên</button></div> :
          <div className="address-list">{addresses.map((address) => <article className={`address-card ${address.isDefault ? "default" : ""}`} key={address.id}>
            <div className="address-card-top"><div className="address-labels"><span className={`address-type ${address.type.toLowerCase()}`}>{TYPE_LABEL[address.type]}</span>{address.isDefault && <span className="default-chip">Mặc định</span>}</div><div className="address-actions">{!address.isDefault && <button disabled={Boolean(busyAction)} onClick={() => void setDefault(address)}>{busyAction === `default-${address.id}` ? "Đang đặt…" : "Đặt mặc định"}</button>}<button onClick={() => openEdit(address)}>Chỉnh sửa</button><button className="danger-link" disabled={Boolean(busyAction)} onClick={() => void remove(address)}>{busyAction === `delete-${address.id}` ? "Đang xóa…" : "Xóa"}</button></div></div>
            <h3>{address.recipientName} <span>· {address.recipientPhone}</span></h3>
            <p className="address-line">⌖ {address.addressLine}, {address.ward.name}{address.district ? `, ${address.district.name}` : ""}, {address.province.name}</p>
            {address.deliveryNote && <p className="delivery-note">Ghi chú: {address.deliveryNote}</p>}
          </article>)}</div>}
      </section>}

    {showForm && profile && <form id="address-editor" className="account-card address-form" onSubmit={save} noValidate>
      <header className="card-heading"><span className="heading-icon">⌖</span><div><h2>{editingId ? "Chỉnh sửa địa chỉ" : "Thiết lập địa chỉ nhận hàng mới"}</h2><p>Thông tin này sẽ được kiểm tra lại trước khi xác nhận đơn hàng.</p></div><span className="privacy-chip">Bảo mật thông tin</span></header>
      <fieldset className="address-type-choices"><legend>Tính chất địa chỉ</legend>{(["HOME", "OFFICE", "GIFT", "OTHER"] as AddressType[]).map((type) => <label className={form.type === type ? "selected" : ""} key={type}><input type="radio" name="addressType" checked={form.type === type} onChange={() => update("type", type)} /><span>{TYPE_LABEL[type]}</span><small>{type === "HOME" ? "Tư gia, dinh thự" : type === "OFFICE" ? "Văn phòng làm việc" : type === "GIFT" ? "Gửi tặng người thân" : "Điểm nhận khác"}</small></label>)}</fieldset>
      <div className="address-form-grid">
        <label className="account-field"><span>Tên gợi nhớ <b>*</b></span><input value={form.label} placeholder="Ví dụ: Nhà riêng" onChange={(event) => update("label", event.target.value)} aria-invalid={Boolean(errors.label)} />{errors.label && <small>{errors.label}</small>}</label>
        <label className="account-field"><span>Họ và tên người nhận <b>*</b></span><input value={form.recipientName} onChange={(event) => update("recipientName", event.target.value)} aria-invalid={Boolean(errors.recipientName)} />{errors.recipientName && <small>{errors.recipientName}</small>}</label>
        <label className="account-field"><span>Số điện thoại liên lạc <b>*</b></span><input inputMode="tel" value={form.recipientPhone} onChange={(event) => update("recipientPhone", event.target.value)} aria-invalid={Boolean(errors.recipientPhone)} />{errors.recipientPhone && <small>{errors.recipientPhone}</small>}</label>
        <label className="account-field"><span>Tỉnh / Thành phố <b>*</b></span><select value={form.province.code} onChange={(event) => { const next = AREAS.find((item) => item.province.code === event.target.value); update("province", next?.province ?? EMPTY_AREA); update("district", null); update("ward", EMPTY_AREA); }}><option value="">Chọn tỉnh / thành phố</option>{AREAS.map((item) => <option key={item.province.code} value={item.province.code}>{item.province.name}</option>)}</select>{errors.province && <small>{errors.province}</small>}</label>
        <label className="account-field"><span>Khu vực / Quận huyện</span><select value={form.district?.code ?? ""} disabled={!selectedProvince} onChange={(event) => { const next = selectedProvince?.districts.find((item) => item.area.code === event.target.value); update("district", next?.area ?? null); update("ward", EMPTY_AREA); }}><option value="">Chọn khu vực</option>{selectedProvince?.districts.map((item) => <option key={item.area.code} value={item.area.code}>{item.area.name}</option>)}</select></label>
        <label className="account-field"><span>Phường / Xã <b>*</b></span><select value={form.ward.code} disabled={!selectedDistrict} onChange={(event) => update("ward", selectedDistrict?.wards.find((item) => item.code === event.target.value) ?? EMPTY_AREA)}><option value="">Chọn phường / xã</option>{selectedDistrict?.wards.map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}</select>{errors.ward && <small>{errors.ward}</small>}</label>
        <label className="account-field wide"><span>Số nhà, ngõ, tên đường <b>*</b></span><input value={form.addressLine} placeholder="Ví dụ: 54 Lê Lợi" onChange={(event) => update("addressLine", event.target.value)} aria-invalid={Boolean(errors.addressLine)} />{errors.addressLine && <small>{errors.addressLine}</small>}</label>
        <label className="account-field wide"><span>Lưu ý giao hàng</span><textarea value={form.deliveryNote ?? ""} placeholder="Giờ nhận hàng, điểm dễ nhận biết…" onChange={(event) => update("deliveryNote", event.target.value || null)} aria-invalid={Boolean(errors.deliveryNote)} />{errors.deliveryNote && <small>{errors.deliveryNote}</small>}</label>
      </div>
      <label className="checkbox address-default"><input type="checkbox" checked={Boolean(form.isDefault)} onChange={(event) => update("isDefault", event.target.checked)} /> Đặt địa chỉ này làm mặc định cho các đơn hàng tương lai</label>
      <footer className="form-footer"><span>Đơn hàng đã tạo luôn giữ snapshot địa chỉ riêng.</span><div><button type="button" className="secondary-button" onClick={closeForm}>Hủy bỏ</button><button className="primary-button" disabled={busyAction === "save"}>{busyAction === "save" && <span className="spinner" />}{busyAction === "save" ? "Đang lưu…" : "Lưu địa chỉ"}</button></div></footer>
    </form>}
  </AccountShell><SiteFooter /></>;
}
