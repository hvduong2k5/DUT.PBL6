// Signed synthetic connectivity probes, never a bank transfer or payment decision.
import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import assert from 'node:assert/strict';

const url = 'https://sepay-test.npanhbao.dev/api/v1/payments/sepay/webhook';
const secretFile = process.env.SEPAY_SECRET_FILE || join(process.env.LOCALAPPDATA, 'PBL', 'sepay-test', 'webhook-secret.txt');
const secret = (await readFile(secretFile, 'utf8')).trim();
const encoder = new TextEncoder();
const key = await crypto.subtle.importKey('raw', encoder.encode(secret), { name: 'HMAC', hash: 'SHA-256' }, false, ['sign']);
const raw = JSON.stringify({ id: 0, gateway: 'Vietcombank', transactionDate: '2026-10-24 12:00:00',
  accountNumber: '0000000000', subAccount: '', code: 'OMP7K9M2N4P6R8T', content: 'OMP7K9M2N4P6R8T',
  transferType: 'in', description: 'Synthetic HTTPS connectivity probe', transferAmount: 10000,
  accumulated: 0, referenceCode: 'LOCAL-PROBE-' + new Date().toISOString() });
async function post({ age = 0, tamper = false } = {}) {
  const timestamp = String(Math.floor(Date.now() / 1000) + age);
  const mac = await crypto.subtle.sign('HMAC', key, encoder.encode(timestamp + '.' + raw));
  const hex = Array.from(new Uint8Array(mac), b => b.toString(16).padStart(2, '0')).join('');
  return fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json',
    'X-SePay-Timestamp': timestamp, 'X-SePay-Signature': 'sha256=' + (tamper ? '00'.repeat(32) : hex) }, body: raw });
}
const health = await fetch(new URL('/healthz', url));
assert.equal(health.status, 200);
assert.equal((await health.json()).payment_processing, false);
const first = await post();
assert.equal(first.status, 200);
assert.deepEqual(await first.json(), { success: true });
const receipt = first.headers.get('X-Capture-Receipt');
assert.equal(first.headers.get('X-Payment-Processing'), 'disabled');
const replay = await post();
assert.equal(replay.status, 200);
assert.equal(replay.headers.get('X-Capture-Receipt'), receipt);
assert.equal((await post({ tamper: true })).status, 401);
assert.equal((await post({ age: -600 })).status, 401);
assert.equal((await fetch(url)).status, 405);
assert.equal((await fetch(new URL('/receipts', url))).status, 404);
console.log(JSON.stringify({ result: 'PASS', url, receipt_id: receipt,
  checks: ['health', 'signed capture', 'same-receipt replay', 'wrong signature', 'expired signature', 'POST only', 'no public receipts'],
  actual_sepay_delivery: false }, null, 2));
