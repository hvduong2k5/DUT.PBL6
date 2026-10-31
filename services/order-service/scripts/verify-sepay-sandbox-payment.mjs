// Verify an existing public-checkout Intent against canonical SePay Sandbox.
// The user/authorized browser creates the TEST transaction in the Dashboard.
// This verifier never creates a bank transaction or changes a financial row.
import assert from 'node:assert/strict';
import { readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

const directory = process.argv[2];
const reportAt = process.argv.indexOf('--report-file');
const waitAt = process.argv.indexOf('--wait-seconds');
if (!directory || reportAt < 0 || !process.argv[reportAt + 1])
  throw Error('Usage: node verify-sepay-sandbox-payment.mjs <private-directory> --report-file <path> [--wait-seconds 180]');
const reportFile = process.argv[reportAt + 1];
const seconds = waitAt >= 0 ? Number(process.argv[waitAt + 1]) : 180;
assert.ok(Number.isInteger(seconds) && seconds > 0 && seconds <= 900);
const saved = JSON.parse(await readFile(path.join(directory, 'runtime-smoke-private.json'), 'utf8'));
const receiver = JSON.parse(await readFile(path.join(directory, 'runtime-receiver.json'), 'utf8'));
const instructions = saved.instructions;
const id = instructions.order_id;
const uuid = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i;
assert.match(id, uuid);
assert.match(instructions.payment_code, /^OMP[A-Z0-9]{12}$/);
const base = 'http://127.0.0.1:18004/api/v1/orders/' + id;
async function get(url, token) {
  const response = await fetch(url, { headers: { Accept: 'application/json', Authorization: 'Bearer ' + token },
    redirect: 'error', signal: AbortSignal.timeout(10000) });
  assert.equal(response.status, 200, 'Expected authenticated JSON GET to succeed');
  return response.json();
}
const report = { status: 'WAITING', mode: 'real-sepay-sandbox', order_id: id,
  payment_code: instructions.payment_code, expected_amount: instructions.amount, expires_at: instructions.expires_at,
  provider_base: 'https://userapi-sandbox.sepay.vn/v2', transitions: [], provider_mutations_by_verifier: false,
  upstreams: 'Inventory/Catalog/Profile/Shipping/Care TEST facades; actual SePay Sandbox API/webhook' };
let previous = '';
let payment;
let order;
try {
  const deadline = performance.now() + seconds * 1000;
  while (performance.now() < deadline) {
    payment = await get(base + '/payment', saved.token);
    order = await get(base, saved.token);
    const phase = JSON.stringify([payment.order_status, payment.display_state, order.stock_status, order.ready_authorized]);
    if (phase !== previous) {
      const transition = { observed_at: new Date().toISOString(), order_status: payment.order_status,
        display_state: payment.display_state, allocated_amount: payment.allocated_amount,
        stock_status: order.stock_status, ready_authorized: order.ready_authorized };
      report.transitions.push(transition);
      console.log(JSON.stringify(transition));
      previous = phase;
    }
    if (payment.order_status === 'PAID' && order.ready_authorized) break;
    assert.ok(!payment.order_status.startsWith('CANCELLED'), 'Order expired/cancelled before paid acceptance');
    await new Promise(resolve => setTimeout(resolve, 500));
  }
  assert.equal(payment?.order_status, 'PAID', 'Timed out before PAID; webhook ACK alone is insufficient');
  assert.equal(payment.display_state, 'PAID');
  assert.equal(payment.expected_amount, instructions.amount);
  assert.equal(payment.allocated_amount, instructions.amount);
  assert.equal(order.stock_status, 'COMMITTED');
  assert.equal(order.ready_authorized, true);

  const token = (await readFile(path.join(directory, 'api-token.txt'), 'utf8')).trim();
  const query = new URLSearchParams({ bank_account_id: receiver.bank_account_uuid, q: instructions.payment_code,
    per_page: '100', timestamp_format: 'iso8601' });
  // One provider read after PAID; do not compete with the runtime scanner by polling the account API.
  const canonical = await get('https://userapi-sandbox.sepay.vn/v2/transactions?' + query, token);
  assert.equal(canonical.status, 'success');
  const matches = canonical.data.filter(row => row.bank_account_id === receiver.bank_account_uuid &&
    row.transfer_type === 'in' && row.amount_in === instructions.amount && row.transaction_content.trim() === instructions.transfer_content);
  assert.equal(matches.length, 1, 'Expected exactly one canonical matching Sandbox transaction');
  assert.match(matches[0].id, uuid);

  const sql = `SELECT json_build_object(
    'order_status',o.status,'stock_status',o.stock_status,'ready_authorized',o.ready_authorized,
    'allocation_count',(SELECT count(*) FROM allocations WHERE order_id=o.id),
    'canonical_transaction_id',(SELECT r.transaction_id FROM receipts r JOIN allocations a ON a.receipt_id=r.id WHERE a.order_id=o.id LIMIT 1),
    'linked_observations',(SELECT count(*) FROM payment_provider_observations obs JOIN allocations a ON a.receipt_id=obs.linked_receipt WHERE a.order_id=o.id AND obs.status='LINKED'),
    'first_paid_count',(SELECT count(*) FROM order_milestones WHERE order_id=o.id AND kind='FIRST_PAID'),
    'paid_event_count',(SELECT count(*) FROM outbox_events WHERE aggregate_id=o.id AND event_type='vn.omama.order.paid.v3'),
    'ready_event_count',(SELECT count(*) FROM outbox_events WHERE aggregate_id=o.id AND event_type='vn.omama.order.fulfillment_ready.v3'),
    'pending_outbox',(SELECT count(*) FROM outbox_events WHERE aggregate_id=o.id AND publish_status<>'PUBLISHED')
    ) FROM orders o WHERE o.id='${id}' AND o.cohort='order-core-1'`;
  const { stdout } = await promisify(execFile)('docker', ['exec', 'codex-order-connected-20261024-db-1',
    'psql', '-U', 'order_test', '-d', 'order_runtime', '-t', '-A', '-c', sql]);
  const trace = JSON.parse(stdout.trim());
  assert.equal(trace.allocation_count, 1);
  assert.equal(trace.canonical_transaction_id, matches[0].id);
  assert.ok(trace.linked_observations >= 1, 'Provider webhook must link to the canonical receipt');
  assert.equal(trace.first_paid_count, 1);
  assert.equal(trace.paid_event_count, 1);
  assert.equal(trace.ready_event_count, 1);
  assert.equal(trace.pending_outbox, 0, 'Kafka publication must finish before acceptance');
  report.status = 'PASS';
  report.canonical_transaction_id = matches[0].id;
  report.durable_trace = trace;
  report.paid_acceptance = true;
  console.log(JSON.stringify({ status: 'PASS', order_id: id, canonical_transaction_id: matches[0].id,
    allocated_amount: payment.allocated_amount, stock_status: order.stock_status, ready_authorized: true }));
} catch (error) {
  report.status = 'FAIL';
  report.failure = error instanceof assert.AssertionError ? error.message : 'Verifier dependency/transport failure';
  process.exitCode = 1;
  console.error(report.failure);
} finally {
  report.finished_at = new Date().toISOString();
  await writeFile(reportFile, JSON.stringify(report, null, 2) + '\n', { mode: 0o600 });
}
