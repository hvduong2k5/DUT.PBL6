import type { AuthenticatedSession } from "./types";

export interface CustomerCoreUser {
  customer_id: string;
  phone_number: string;
  full_name: string;
  email?: string;
  avatar_url?: string;
  membership_tier?: "BRONZE" | "SILVER" | "GOLD" | "DIAMOND";
  loyalty_points?: number;
}

export interface CustomerCoreTokens {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
}

export interface CustomerCoreAuthResponse {
  user: CustomerCoreUser;
  tokens: CustomerCoreTokens;
}

export function mapCustomerCoreSession(user: CustomerCoreUser): AuthenticatedSession {
  return {
    authenticated: true,
    customer: {
      id: user.customer_id,
      displayName: user.full_name,
      status: "ACTIVE",
      phoneNumber: user.phone_number,
      email: user.email,
      membershipTier: user.membership_tier,
      loyaltyPoints: user.loyalty_points
    }
  };
}

export function toCustomerCoreLogin(input: { phoneNumber: string; password: string }) {
  return { phone_number: input.phoneNumber, password: input.password };
}

export function toCustomerCoreRegistration(input: { fullName: string; phoneNumber: string; email?: string; password: string }) {
  return {
    phone_number: input.phoneNumber,
    password: input.password,
    full_name: input.fullName,
    ...(input.email ? { email: input.email } : {})
  };
}
