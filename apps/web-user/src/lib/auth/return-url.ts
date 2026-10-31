const CONTROL_OR_BACKSLASH = /[\\\u0000-\u001f\u007f]/u;

export function getSafeReturnUrl(value: string | string[] | undefined, fallback = "/"): string {
  const candidate = Array.isArray(value) ? value[0] : value;

  if (!candidate || !candidate.startsWith("/") || candidate.startsWith("//")) {
    return fallback;
  }

  if (CONTROL_OR_BACKSLASH.test(candidate)) {
    return fallback;
  }

  try {
    const decoded = decodeURIComponent(candidate);
    if (decoded.startsWith("//") || CONTROL_OR_BACKSLASH.test(decoded)) {
      return fallback;
    }
  } catch {
    return fallback;
  }

  return candidate;
}
