"use client";

import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { useAuth } from "@/components/auth/auth-provider";
import { PublicApiError } from "@/lib/api-contract";
import type { RecommendationConsent, RecommendationItem, Recommendations } from "@/lib/recommendation/types";
import { recommendationService, recommendationsEnabled } from "@/services/recommendation-service";

declare global {
  interface Window {
    dataLayer?: unknown[];
    gtag?: (...args: unknown[]) => void;
  }
}

interface RecommendationState {
  consent: RecommendationConsent | null;
  setConsent: (value: boolean) => Promise<void>;
  consentError: string;
  record: (sku: string, kind: "VIEW_ITEM" | "ADD_TO_CART") => void;
  click: (result: Recommendations, item: RecommendationItem) => void;
  impression: (result: Recommendations) => void;
  measurement: boolean;
  setMeasurement: (value: boolean) => void;
}

const Context = createContext<RecommendationState | null>(null);

export function RecommendationProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();
  const actor = auth.isCustomer ? auth.profile?.principalId : undefined;
  const [consentState, setConsentState] = useState<{ actor: string; value: RecommendationConsent } | null>(null);
  const consent = consentState && consentState.actor === actor ? consentState.value : null;
  const [consentError, setConsentError] = useState("");
  const [measurement, setMeasurementValue] = useState(false);
  const current = useRef({ actor, consent });
  useLayoutEffect(() => { current.current = { actor, consent }; }, [actor, consent]);
  const clicks = useRef(new Map<string, { result: Recommendations; at: number }>());

  useEffect(() => {
    let active = true;
    clicks.current.clear();
    if (!actor) document.cookie = "oma_rec_csrf=; Path=/; SameSite=Strict; Max-Age=0";
    void Promise.resolve().then(async () => {
      if (active) { setConsentState(null); setConsentError(""); }
      if (!recommendationsEnabled || !actor) return;
      try {
        const value = await recommendationService.consent();
        if (active) setConsentState({ actor, value });
      } catch {
        if (active) setConsentError("Chưa thể tải lựa chọn cá nhân hóa.");
      }
    });
    return () => { active = false; };
  }, [actor]);

  useEffect(() => {
    if (!recommendationsEnabled) return;
    void Promise.resolve().then(() => setMeasurementValue(document.cookie.split("; ").includes("oma_rec_measurement=granted")));
  }, []);

  const setMeasurement = useCallback((value: boolean) => {
    setMeasurementValue(value);
    document.cookie = `oma_rec_measurement=${value ? "granted" : "denied"}; Path=/; SameSite=Strict; Max-Age=31536000${location.protocol === "https:" ? "; Secure" : ""}`;
    window.gtag?.("consent", "update", { analytics_storage: value ? "granted" : "denied" });
  }, []);

  useEffect(() => {
    const id = process.env.NEXT_PUBLIC_GA4_MEASUREMENT_ID;
    if (!measurement || !recommendationsEnabled || !id || !/^G-[A-Z0-9]+$/u.test(id)) return;
    if (document.getElementById("recommendation-google-tag")) return;
    window.dataLayer ??= [];
    // Google tag's command queue uses Arguments objects.
    // eslint-disable-next-line prefer-rest-params
    window.gtag ??= function () { window.dataLayer?.push(arguments); };
    window.gtag("consent", "default", { analytics_storage: "granted", ad_storage: "denied", ad_user_data: "denied", ad_personalization: "denied" });
    window.gtag("js", new Date());
    window.gtag("config", id, { send_page_view: false });
    const script = document.createElement("script");
    script.id = "recommendation-google-tag";
    script.async = true;
    script.src = `https://www.googletagmanager.com/gtag/js?id=${id}`;
    document.head.appendChild(script);
  }, [measurement]);

  const measure = useCallback((name: string, result: Recommendations, items: RecommendationItem[]) => {
    if (!measurement) return;
    window.gtag?.("event", name, {
      item_list_id: `rec_${result.placement}_v1`, currency: "VND",
      items: items.map((item) => ({ item_id: item.skuCode, item_name: item.name, price: item.priceVnd, index: item.rank - 1 }))
    });
  }, [measurement]);

  const setConsent = useCallback(async (enabled: boolean) => {
    if (!actor || !consent) return;
    try {
      const result = await recommendationService.setConsent(enabled, consent);
      if (current.current.actor === actor) setConsentState({ actor, value: result });
      setConsentError("");
    } catch {
      setConsentError("Chưa lưu được lựa chọn. Hãy tải lại trang và thử lại.");
      throw new Error("Consent update failed");
    }
  }, [actor, consent]);

  const record = useCallback((sku: string, kind: "VIEW_ITEM" | "ADD_TO_CART") => {
    const state = current.current;
    const provenance = clicks.current.get(sku);
    if (kind === "ADD_TO_CART" && provenance && Date.now() - provenance.at < 30 * 60 * 1000) {
      const item = provenance.result.items.find((r) => r.skuCode === sku);
      if (item) measure("add_to_cart", provenance.result, [item]);
    }
    if (!recommendationsEnabled || !state.actor || !state.consent?.enabled) return;
    const body = { eventId: crypto.randomUUID(), eventType: kind, skuCode: sku,
      occurredAt: new Date().toISOString(), consentVersion: state.consent.consentVersion,
      ...(provenance && Date.now() - provenance.at < 30 * 60 * 1000 ? { recommendationId: provenance.result.recommendationId } : {}) };
    const send = async (attempt: number) => {
      if (current.current.actor !== state.actor || !current.current.consent?.enabled
        || current.current.consent.consentVersion !== body.consentVersion) return;
      try { await recommendationService.event(body); }
      catch (error) {
        if (attempt < 2 && (!(error instanceof PublicApiError) || error.status >= 500)) {
          setTimeout(() => { void send(attempt + 1); }, 250 * 2 ** attempt + Math.random() * 100);
        }
      }
    };
    void send(0);
  }, [measure]);

  return <Context.Provider value={{ consent, setConsent, consentError, record, measurement, setMeasurement,
    click: (result, item) => { clicks.current.set(item.skuCode, { result, at: Date.now() }); measure("select_item", result, [item]); },
    impression: (result) => measure("view_item_list", result, result.items)
  }}>{children}</Context.Provider>;
}

export function useRecommendation() {
  const state = useContext(Context);
  if (!state) throw new Error("RecommendationProvider is required");
  return state;
}
