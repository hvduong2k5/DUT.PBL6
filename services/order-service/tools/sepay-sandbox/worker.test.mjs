import test from 'node:test';
import assert from 'node:assert/strict';
import worker from './worker.mjs';

const secret = 'synthetic-test-secret-not-a-live-credential';
const captureKey = 'ab'.repeat(32);
const encoder = new TextEncoder();
// Shape from SePay's bank webhook integration example, not the hosted gateway IPN.
const fixture = JSON.stringify({ id: 0, gateway: 'Vietcombank', transactionDate: '2026-10-24 12:00:00',
  accountNumber: '0000000000', code: 'OMP7K9M2N4P6R8T', content: 'OMP7K9M2N4P6R8T',
  transferType: 'in', transferAmount: 10000, accumulated: 0, subAccount: null,
  referenceCode: 'SYNTHETIC-CONNECTIVITY-PROBE', description: 'Sandbox probe' });
function database(fail = false) {
  const rows = new Map();
  return { rows, prepare(sql) { return {
    first: async () => { if (fail) throw Error('offline'); return { ready: 1 }; },
    bind(...args) { return { run: async () => {
      if (fail) throw Error('offline');
      assert.match(sql, /ON CONFLICT\(receipt_id\)/);
      const old = rows.get(args[0]);
      rows.set(args[0], old ? { ...old, deliveries: old.deliveries + 1 } : { args, deliveries: 1 });
      return { success: true };
    } }; },
  }; } };
}
async function signed(raw = fixture, age = 0) {
  const timestamp = String(Math.floor(Date.now() / 1000) + age);
  const key = await crypto.subtle.importKey('raw', encoder.encode(secret), { name: 'HMAC', hash: 'SHA-256' }, false, ['sign']);
  const mac = await crypto.subtle.sign('HMAC', key, encoder.encode(timestamp + '.' + raw));
  const signature = Array.from(new Uint8Array(mac), b => b.toString(16).padStart(2, '0')).join('');
  return new Request('https://sepay-test.example/api/v1/payments/sepay/webhook', { method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-SePay-Timestamp': timestamp, 'X-SePay-Signature': 'sha256=' + signature }, body: raw });
}
function env(db = database()) { return { WEBHOOK_SECRET: secret, CAPTURE_KEY: captureKey, DB: db }; }

test('authenticated capture encrypts exact raw bytes and deduplicates, retaining conflicts', async () => {
  const e = env();
  const first = await worker.fetch(await signed(), e);
  assert.equal(first.status, 200);
  assert.deepEqual(await first.json(), { success: true });
  assert.equal(first.headers.get('X-Payment-Processing'), 'disabled');
  const receiptID = first.headers.get('X-Capture-Receipt');
  assert.equal((await worker.fetch(await signed(), e)).status, 200);
  assert.equal(e.DB.rows.size, 1);
  const stored = e.DB.rows.get(receiptID);
  assert.equal(stored.deliveries, 2);
  const envelope = JSON.parse(stored.args[3]);
  assert.ok(!stored.args[3].includes('0000000000'));
  const key = await crypto.subtle.importKey('raw', Uint8Array.from({ length: 32 }, () => 0xab), 'AES-GCM', false, ['decrypt']);
  const plain = await crypto.subtle.decrypt({ name: 'AES-GCM', iv: Uint8Array.from(atob(envelope.nonce), c => c.charCodeAt(0)),
    additionalData: encoder.encode('sepay-sandbox:v1:' + receiptID) }, key, Uint8Array.from(atob(envelope.ciphertext), c => c.charCodeAt(0)));
  const evidence = JSON.parse(new TextDecoder().decode(plain));
  assert.equal(atob(evidence.raw_base64), fixture);
  assert.equal((await worker.fetch(await signed(fixture.replace('10000', '20000')), e)).status, 200);
  assert.equal(e.DB.rows.size, 2);
});
test('missing, stale, future and tampered signatures never persist', async () => {
  const e = env();
  for (const age of [-600, 600]) assert.equal((await worker.fetch(await signed(fixture, age), e)).status, 401);
  const request = await signed();
  request.headers.set('X-SePay-Signature', 'sha256=' + '00'.repeat(32));
  assert.equal((await worker.fetch(request, e)).status, 401);
  assert.equal((await worker.fetch(new Request('https://x/api/v1/payments/sepay/webhook', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: fixture }), e)).status, 401);
  assert.equal(e.DB.rows.size, 0);
});
test('bad JSON, unsupported event, oversized body, wrong route and method are rejected', async () => {
  const e = env();
  for (const raw of ['{', '{}', 'null', '[]', '{"id":-1,"transferType":"in"}']) {
    assert.equal((await worker.fetch(await signed(raw), e)).status, 400);
  }
  assert.equal((await worker.fetch(await signed(' '.repeat(65537)), e)).status, 413);
  assert.equal((await worker.fetch(new Request('https://x/api/v1/payments/sepay/webhook'), e)).status, 405);
  assert.equal((await worker.fetch(new Request('https://x/receipts'), e)).status, 404);
  const request = await signed(); request.headers.set('Content-Type', 'text/plain');
  assert.equal((await worker.fetch(request, e)).status, 415);
  assert.equal(e.DB.rows.size, 0);
});
test('storage or secret failure is never acknowledged as captured', async () => {
  assert.equal((await worker.fetch(await signed(), env(database(true)))).status, 503);
  assert.equal((await worker.fetch(await signed(), { DB: database() })).status, 503);
  assert.equal((await worker.fetch(new Request('https://x/healthz'), env())).status, 200);
  assert.equal((await worker.fetch(new Request('https://x/healthz'), env(database(true)))).status, 503);
});

test('bridge preserves signed bytes and ACKs only after the runtime observation', async (t) => {
  const e = { ...env(), FORWARD_ENABLED: 'true' };
  const raw = fixture.replace('"id":0', '"id":123');
  const request = await signed(raw);
  let calls = 0;
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    calls++;
    assert.equal(e.DB.rows.size, 1, 'capture must precede forwarding');
    assert.equal(url, 'https://sepay-origin-test.npanhbao.dev/api/v1/payments/sepay/webhook');
    assert.equal(new TextDecoder().decode(options.body), raw);
    assert.equal(options.redirect, 'manual');
    assert.equal(options.headers['X-SePay-Signature'], request.headers.get('X-SePay-Signature'));
    return Response.json({ success: true });
  });
  const response = await worker.fetch(request, e);
  assert.equal(response.status, 200);
  assert.equal(response.headers.get('X-Payment-Processing'), 'runtime-observation');
  assert.equal((await worker.fetch(await signed(), e)).status, 200, 'ID0 dashboard probe stays capture-only');
  assert.equal(calls, 1);
});

test('bridge failure retains capture but requests provider retry', async (t) => {
  const raw = fixture.replace('"id":0', '"id":123');
  for (const response of [new Response(null, { status: 302, headers: { Location: 'https://wrong.example' } }),
    new Response(null, { status: 503 }), Response.json({ success: false }), new Response('invalid')]) {
    const e = { ...env(), FORWARD_ENABLED: 'true' };
    const mock = t.mock.method(globalThis, 'fetch', async () => response);
    assert.equal((await worker.fetch(await signed(raw), e)).status, 503);
    assert.equal(e.DB.rows.size, 1);
    mock.mock.restore();
  }
  const mock = t.mock.method(globalThis, 'fetch', async () => { throw Error('offline'); });
  assert.equal((await worker.fetch(await signed(raw), { ...env(), FORWARD_ENABLED: 'true' })).status, 503);
  mock.mock.restore();
});
