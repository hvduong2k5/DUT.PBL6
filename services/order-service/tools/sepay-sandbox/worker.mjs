// Sandbox delivery capture only. Never writes to the Order/payment ledger.
const route = '/api/v1/payments/sepay/webhook';
const encoder = new TextEncoder();
const limit = 64 * 1024;

function reply(status, data, headers = {}) {
  return Response.json(data, { status, headers: { 'Cache-Control': 'no-store', ...headers } });
}
function fromHex(text) {
  if (!/^[a-f0-9]{64}$/i.test(text || '')) throw new Error('Invalid key');
  return Uint8Array.from(text.match(/../g), value => parseInt(value, 16));
}
function toHex(bytes) {
  return Array.from(new Uint8Array(bytes), value => value.toString(16).padStart(2, '0')).join('');
}
function base64(bytes) {
  return btoa(String.fromCharCode(...new Uint8Array(bytes)));
}
async function rawBody(request) {
  if (Number(request.headers.get('content-length')) > limit) throw new Error('BODY_TOO_LARGE');
  const reader = request.body?.getReader();
  if (!reader) return new Uint8Array();
  const chunks = [];
  let size = 0;
  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    size += value.byteLength;
    if (size > limit) {
      await reader.cancel();
      throw new Error('BODY_TOO_LARGE');
    }
    chunks.push(value);
  }
  const result = new Uint8Array(size);
  let offset = 0;
  for (const chunk of chunks) { result.set(chunk, offset); offset += chunk.byteLength; }
  return result;
}
async function authenticated(request, secret, raw) {
  const timestamp = request.headers.get('X-SePay-Timestamp') || '';
  const signature = request.headers.get('X-SePay-Signature') || '';
  if (!/^\d{1,12}$/.test(timestamp) || Math.abs(Date.now() / 1000 - Number(timestamp)) > 300 ||
      !/^sha256=[a-f0-9]{64}$/i.test(signature)) return false;
  const prefix = encoder.encode(timestamp + '.');
  const message = new Uint8Array(prefix.length + raw.length);
  message.set(prefix); message.set(raw, prefix.length);
  const key = await crypto.subtle.importKey('raw', encoder.encode(secret), { name: 'HMAC', hash: 'SHA-256' }, false, ['verify']);
  return crypto.subtle.verify('HMAC', key, fromHex(signature.slice(7)), message);
}
async function handle(request, env) {
  const path = new URL(request.url).pathname;
  if (path !== route && path !== '/healthz') return reply(404, { success: false });
  if (path === '/healthz') {
    if (request.method !== 'GET') return reply(405, { success: false }, { Allow: 'GET' });
    if (!env.WEBHOOK_SECRET || !env.CAPTURE_KEY || !env.DB) return reply(503, { success: false });
    await env.DB.prepare('SELECT 1 AS ready').first();
    return reply(200, { ready: true, mode: 'sandbox-capture', payment_processing: false });
  }
  if (request.method !== 'POST') return reply(405, { success: false }, { Allow: 'POST' });
  if (!env.WEBHOOK_SECRET || !env.CAPTURE_KEY || !env.DB) return reply(503, { success: false });
  if ((request.headers.get('Content-Type') || '').split(';')[0].trim().toLowerCase() !== 'application/json') {
    return reply(415, { success: false, error: 'JSON_REQUIRED' });
  }
  let raw;
  try { raw = await rawBody(request); } catch (error) {
    if (error.message === 'BODY_TOO_LARGE') return reply(413, { success: false });
    return reply(400, { success: false });
  }
  if (!await authenticated(request, env.WEBHOOK_SECRET, raw)) return reply(401, { success: false, error: 'INVALID_SIGNATURE' });
  let payload;
  try { payload = JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(raw)); } catch {
    return reply(400, { success: false, error: 'INVALID_JSON' });
  }
  // ID 0 is deliberately permitted for the Dashboard's documented send-test payload.
  if (!payload || Array.isArray(payload) || !Number.isSafeInteger(payload.id) || payload.id < 0 ||
      !['in', 'out'].includes(payload.transferType)) return reply(400, { success: false, error: 'INVALID_EVENT' });
  const hash = toHex(await crypto.subtle.digest('SHA-256', raw));
  const receiptID = toHex(await crypto.subtle.digest('SHA-256', encoder.encode(`${payload.id}:${hash}`)));
  const key = await crypto.subtle.importKey('raw', fromHex(env.CAPTURE_KEY), 'AES-GCM', false, ['encrypt']);
  const nonce = crypto.getRandomValues(new Uint8Array(12));
  const aad = `sepay-sandbox:v1:${receiptID}`;
  const evidence = encoder.encode(JSON.stringify({ raw_base64: base64(raw),
    signature: request.headers.get('X-SePay-Signature'), timestamp: request.headers.get('X-SePay-Timestamp') }));
  const ciphertext = await crypto.subtle.encrypt({ name: 'AES-GCM', iv: nonce, additionalData: encoder.encode(aad) }, key, evidence);
  const envelope = JSON.stringify({ version: 1, nonce: base64(nonce), ciphertext: base64(ciphertext) });
  const now = new Date().toISOString();
  const result = await env.DB.prepare(`INSERT INTO receipts
    (receipt_id,event_id,payload_hash,envelope,first_seen,last_seen,deliveries)
    VALUES (?1,?2,?3,?4,?5,?5,1)
    ON CONFLICT(receipt_id) DO UPDATE SET last_seen=excluded.last_seen,deliveries=receipts.deliveries+1`)
    .bind(receiptID, String(payload.id), hash, envelope, now).run();
  if (!result.success) throw new Error('Capture write failed');
  // ACK only after D1 persisted the authenticated observation; not a payment decision.
  return reply(200, { success: true }, { 'X-Capture-Receipt': receiptID, 'X-Payment-Processing': 'disabled' });
}
export default {
  async fetch(request, env) {
    try { return await handle(request, env); } catch {
      console.error(JSON.stringify({ event: 'sepay_capture_failed' }));
      return reply(503, { success: false, error: 'CAPTURE_UNAVAILABLE' });
    }
  },
};
