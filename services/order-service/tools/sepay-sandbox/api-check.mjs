// Read-only SePay sandbox checks. Never calls the Live host or mutation endpoints.
import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import assert from 'node:assert/strict';

const dir = process.env.SEPAY_PRIVATE_DIR || join(process.env.LOCALAPPDATA, 'PBL', 'sepay-test');
const token = (await readFile(join(dir, 'api-token.txt'), 'utf8')).trim();
const base = 'https://userapi-sandbox.sepay.vn/v2';
const checks = [];
const uuid = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i;
async function get(path, auth = true) {
  await new Promise(resolve => setTimeout(resolve, 400));
  for (let attempt = 0; attempt < 3; attempt++) {
    const response = await fetch(base + path, { headers: { Accept: 'application/json',
      ...(auth ? { Authorization: 'Bearer ' + token } : {}) }, signal: AbortSignal.timeout(20000) });
    if (response.status === 429 && attempt < 2) {
      const seconds = Math.min(10, Math.max(1, Number(response.headers.get('Retry-After')) || 1));
      await response.arrayBuffer();
      await new Promise(resolve => setTimeout(resolve, seconds * 1000));
      continue;
    }
    return { status: response.status, json: await response.json() };
  }
}
async function success(path) {
  const result = await get(path);
  assert.equal(result.status, 200, path + ' must return 200');
  assert.equal(result.json.status, 'success', path + ' envelope');
  checks.push({ path, status: 'PASS', http: result.status });
  return result.json.data;
}
const accounts = await success('/bank-accounts?per_page=100');
assert.ok(Array.isArray(accounts) && accounts.length > 0, 'At least one sandbox account');
for (const account of accounts) assert.match(account.id, uuid);
const account = await success('/bank-accounts/' + accounts[0].id);
assert.equal(account.id, accounts[0].id);
const transactions = await success('/transactions?per_page=100&timestamp_format=iso8601');
assert.ok(Array.isArray(transactions));
for (const transaction of transactions) {
  assert.match(transaction.id, uuid);
  assert.ok(['in', 'out'].includes(transaction.transfer_type));
  assert.ok(Number.isSafeInteger(transaction.amount_in) && transaction.amount_in >= 0);
  assert.ok(Number.isSafeInteger(transaction.amount_out) && transaction.amount_out >= 0);
}
for (const transaction of transactions.slice(0, 2)) {
  const detail = await success('/transactions/' + transaction.id);
  for (const field of ['id', 'bank_account_id', 'amount_in', 'amount_out', 'reference_number', 'transaction_content']) {
    assert.equal(detail[field], transaction[field], 'Detail/list mismatch: ' + field);
  }
}
const filtered = await success('/transactions?per_page=100&bank_account_id=' + accounts[0].id);
assert.ok(filtered.every(t => t.bank_account_id === accounts[0].id));
if (transactions.length > 1) {
  const first = await success('/transactions?per_page=1&page=1');
  const second = await success('/transactions?per_page=1&page=2');
  assert.equal(first.length, 1); assert.equal(second.length, 1);
  assert.notEqual(first[0].id, second[0].id);
}
const missing = await get('/transactions/00000000-0000-4000-8000-000000000000');
assert.equal(missing.status, 404); checks.push({ check: 'unknown UUID', status: 'PASS', http: 404 });
const unauthorized = await get('/bank-accounts', false);
assert.equal(unauthorized.status, 401); checks.push({ check: 'missing token', status: 'PASS', http: 401 });
await writeFile(join(dir, 'verified-transactions-private.json'), JSON.stringify(transactions));
// Result deliberately excludes token, account numbers, names and transaction contents.
const summary = { result: 'PASS', base_url: base, mode: 'read-only', account_count: accounts.length,
  transaction_count: transactions.length, recognized_omp_count: transactions.filter(t => /^OMP[A-Z0-9]{12}$/.test(t.code || '')).length,
  detail_checked: Math.min(transactions.length, 2), checks, live_api_called: false,
  order_payment_processing: 'NOT_IMPLEMENTED' };
if (process.argv[2]) await writeFile(process.argv[2], JSON.stringify(summary, null, 2));
console.log(JSON.stringify(summary, null, 2));
