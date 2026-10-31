export function usesOidc(): boolean {
  const mode = process.env.WEB_AUTH_MODE ?? (process.env.NODE_ENV === "production" ? "oidc" : "legacy");
  if (mode !== "oidc" && mode !== "legacy") throw new Error("Invalid WEB_AUTH_MODE");
  if (mode === "legacy" && process.env.NODE_ENV === "production") throw new Error("Legacy auth is local development only");
  return mode === "oidc";
}

export function oidcConfig() {
  if (!usesOidc()) throw new Error("OIDC is not enabled");
  const allowHTTP = process.env.WEB_OIDC_ALLOW_LOCAL_HTTP === "true";
  const checked = (raw: string | undefined) => {
    if (!raw) throw new Error("Missing OIDC configuration");
    const url = new URL(raw);
    const local = ["localhost", "127.0.0.1", "[::1]"].includes(url.hostname);
    if (url.username || url.password || url.search || url.hash || (url.protocol !== "https:" && !(allowHTTP && local && url.protocol === "http:"))) throw new Error("OIDC requires HTTPS or explicit loopback TEST HTTP");
    return url;
  };
  const origin = checked(process.env.WEB_PUBLIC_URL);
  if (origin.pathname !== "/") throw new Error("WEB_PUBLIC_URL must be an origin");
  const issuer = checked(process.env.WEB_OIDC_ISSUER).href.replace(/\/$/u, "");
  const clientId = process.env.WEB_OIDC_CLIENT_ID;
  const secret = process.env.WEB_OIDC_COOKIE_SECRET;
  if (!clientId || !secret || Buffer.byteLength(secret) < 32) throw new Error("OIDC requires client ID and cookie secret >=32 bytes");
  return { origin: origin.origin, issuer, clientId, secret, audience: process.env.WEB_OIDC_API_AUDIENCE ?? "omama-api", secure: origin.protocol === "https:" };
}
