export type CustomerStatus = "ACTIVE" | "PENDING_VERIFICATION";

export interface CustomerSummary {
  id: string;
  displayName: string;
  status: CustomerStatus;
  phoneNumber?: string;
  email?: string;
  membershipTier?: "BRONZE" | "SILVER" | "GOLD" | "DIAMOND";
  loyaltyPoints?: number;
}

export type PrincipalType = "CUSTOMER" | "EMPLOYEE";

export interface PrincipalRole {
  roleCode: string;
  roleName?: string;
  scopeType?: "GLOBAL" | "STORE" | "WAREHOUSE" | "REGION" | "DEPARTMENT" | "BRANCH" | "BUSINESS_UNIT";
  scopeId?: string | null;
}

export interface PrincipalSummary {
  principalId: string;
  principalType: PrincipalType;
  displayName: string;
  email?: string;
  roles: PrincipalRole[];
  permissions: string[];
  customerId?: string;
}

export interface AuthenticatedSession {
  authenticated: true;
  principal: PrincipalSummary;
  customer: CustomerSummary | null;
  customerAccess: boolean;
  sessionExpiresAt?: string;
}

export interface RegistrationResult {
  status: "PENDING_VERIFICATION";
  verificationId: string;
  delivery: {
    channel: "EMAIL";
    maskedDestination: string;
  };
  verificationExpiresAt: string;
  resendAvailableAt: string;
}

export interface RegistrationConfirmation extends AuthenticatedSession {
  status: "ACTIVE";
}

export interface AcceptedResult {
  accepted: true;
  messageCode: "RECOVERY_REQUEST_ACCEPTED" | "VERIFICATION_EMAIL_ACCEPTED";
  resendAvailableAt?: string;
}

export interface RecoveryVerification {
  verified: true;
  resetProof: string;
  expiresAt: string;
}

export interface ApiFieldError {
  field: string;
  code: string;
  message: string;
}

export interface ApiErrorBody {
  code?: string;
  message?: string;
  error_code?: string;
  user_message?: string;
  errors?: ApiFieldError[];
  requestId?: string;
  retryAfter?: string;
}

export class AuthApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly errors: ApiFieldError[];
  readonly requestId?: string;
  readonly retryAfter?: string;

  constructor(status: number, body: ApiErrorBody) {
    super(body.user_message || body.message || "Không thể xử lý yêu cầu lúc này.");
    this.name = "AuthApiError";
    this.status = status;
    this.code = body.error_code || body.code || "UNKNOWN_ERROR";
    this.errors = body.errors ?? [];
    this.requestId = body.requestId;
    this.retryAfter = body.retryAfter;
  }
}
