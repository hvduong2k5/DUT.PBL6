export interface CompanyShipment {
  shipment_id: string; shipment_number: string; order_id: string; package_id: string;
  status: string; revision: number; assignment_version: number; accepted: boolean;
}
export interface CompanyDetail extends CompanyShipment {
  recipient: { name: string; phone: string; street: string; ward: string; district: string; province: string };
  parcel: { weight_grams: number }; tracking_code: string;
}
export interface CompanyList { items: CompanyShipment[]; total: number; limit: number; offset: number }
export class CompanyApiError extends Error {
  constructor(readonly status:number,message:string){super(`${status}: ${message}`);}
}
export function nextCompanyStatuses(status: string): string[] {
  return ({ HANDED_OVER: ["IN_TRANSIT"], IN_TRANSIT: ["OUT_FOR_DELIVERY"], OUT_FOR_DELIVERY: ["DELIVERED", "DELIVERY_FAILED"], DELIVERY_FAILED: ["OUT_FOR_DELIVERY", "RETURNING"], RETURNING: ["RETURNED"] } as Record<string, string[]>)[status] ?? [];
}
export async function companyRequest<T>(path: string, body?: object, key?: string): Promise<T> {
  const response = await fetch(`/api/admin/shipping/company/shipments${path}`, { method: body ? "POST" : "GET", credentials: "same-origin", cache: "no-store", headers: body ? { "Content-Type": "application/json", "Idempotency-Key": key! } : undefined, body: body ? JSON.stringify(body) : undefined });
  const payload = await response.json();
  if (!response.ok) throw new CompanyApiError(response.status,payload.message ?? "Không thể xử lý yêu cầu");
  return payload as T;
}
