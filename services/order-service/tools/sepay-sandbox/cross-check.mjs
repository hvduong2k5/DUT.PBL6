// Reconcile an operator-exported encrypted D1 snapshot with read-only API results.
// Evidence check only: no import to Order Core and no payment decision.
import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import assert from 'node:assert/strict';

const dir = process.env.SEPAY_PRIVATE_DIR || join(process.env.LOCALAPPDATA, 'PBL', 'sepay-test');
const rows = JSON.parse(await readFile(join(dir, 'audit-receipts-private.json'), 'utf8'));
const transactions = JSON.parse(await readFile(join(dir, 'verified-transactions-private.json'), 'utf8'));
const accounts = JSON.parse(await readFile(join(dir, 'bank-accounts-private.json'), 'utf8')).data;
const captureKey = (await readFile(join(dir, 'capture-key.txt'), 'utf8')).trim();
const secret = (await readFile(join(dir, 'webhook-secret.txt'), 'utf8')).trim();
const key = await crypto.subtle.importKey('raw', Buffer.from(captureKey, 'hex'), 'AES-GCM', false, ['decrypt']);
const hmac = await crypto.subtle.importKey('raw', new TextEncoder().encode(secret), { name: 'HMAC', hash: 'SHA-256' }, false, ['verify']);
const results = [];
for (const row of rows.filter(row => row.event_id !== '0')) {
  const envelope = JSON.parse(row.envelope);
  assert.equal(envelope.version, 1);
  const decoded = await crypto.subtle.decrypt({ name: 'AES-GCM', iv: Buffer.from(envelope.nonce, 'base64'),
    additionalData: new TextEncoder().encode('sepay-sandbox:v1:' + row.receipt_id) }, key, Buffer.from(envelope.ciphertext, 'base64'));
  const evidence = JSON.parse(new TextDecoder().decode(decoded));
  const raw = Buffer.from(evidence.raw_base64, 'base64');
  assert.equal(Buffer.from(await crypto.subtle.digest('SHA-256', raw)).toString('hex'), row.payload_hash);
  assert.ok(await crypto.subtle.verify('HMAC', hmac, Buffer.from(evidence.signature.slice(7), 'hex'),
    Buffer.concat([Buffer.from(evidence.timestamp + '.'), raw])), 'Stored signature invalid');
  // Check signature at receipt time, not current time (historic observations are allowed).
  assert.ok(Math.abs(Date.parse(row.first_seen) / 1000 - Number(evidence.timestamp)) <= 300);
  const hook = JSON.parse(raw);
  assert.equal(String(hook.id), row.event_id);
  assert.ok(hook.referenceCode, 'Bank reference required for unambiguous correlation');
  const candidates = transactions.filter(t => t.reference_number === hook.referenceCode &&
    t.account_number === hook.accountNumber && t.transfer_type === hook.transferType &&
    (t.transfer_type === 'in' ? t.amount_in : t.amount_out) === hook.transferAmount &&
    t.transaction_content === hook.content);
  assert.equal(candidates.length, 1, 'Must correlate by reference/receiver/direction/amount/content, not numeric ID vs UUID');
  const transaction = candidates[0];
  assert.ok(accounts.some(a => a.id === transaction.bank_account_id && a.account_number === hook.accountNumber));
  const bankTime = Date.parse(hook.transactionDate.replace(' ', 'T') + '+07:00');
  assert.equal(Date.parse(transaction.transaction_date), bankTime);
  assert.equal(transaction.webhook_success, 1);
  results.push({ result: 'PASS', valid_stored_signature: true, unique_api_match: true,
    bank_time_match: true, webhook_success: true, recognized_omp: /^OMP[A-Z0-9]{12}$/.test(transaction.code || '') });
}
assert.ok(results.length > 0, 'No non-probe deliveries available to correlate');
const summary = { result: 'PASS', non_probe_receipts_checked: results.length,
  api_transactions: transactions.length, matched: results.length,
  numeric_webhook_id_compared_to_uuid: false, order_payment_confirmed: false, checks: results };
if (process.argv[2]) await writeFile(process.argv[2], JSON.stringify(summary, null, 2));
console.log(JSON.stringify(summary, null, 2));
