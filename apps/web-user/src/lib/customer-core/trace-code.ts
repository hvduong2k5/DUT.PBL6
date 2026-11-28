/** Accept a label token or a link issued for this storefront, without following arbitrary links. */
export function traceCode(input: string, origins: string[]): string | undefined {
  const value = input.trim();
  if (/^[A-Za-z0-9_-]{1,160}$/u.test(value)) return value;
  try {
    const url = new URL(value);
    if (!origins.includes(url.origin) || url.username || url.password || url.search || url.hash) return undefined;
    const match = /^\/trace\/([A-Za-z0-9_-]{1,160})\/?$/u.exec(url.pathname);
    return match?.[1];
  } catch { return undefined; }
}
