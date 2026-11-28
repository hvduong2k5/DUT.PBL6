export async function readBoundedText(body: ReadableStream<Uint8Array> | null, limit: number): Promise<string> {
  if (!body) return "";
  const reader = body.getReader(); const chunks: Uint8Array[] = []; let bytes = 0;
  try {
    for (;;) { const next = await reader.read(); if (next.done) break; bytes += next.value.byteLength; if (bytes > limit) { await reader.cancel(); throw new RangeError("body limit"); } chunks.push(next.value); }
  } finally { reader.releaseLock(); }
  const result = new Uint8Array(bytes); let offset = 0;
  for (const chunk of chunks) { result.set(chunk, offset); offset += chunk.byteLength; }
  return new TextDecoder("utf-8", { fatal: true }).decode(result);
}
