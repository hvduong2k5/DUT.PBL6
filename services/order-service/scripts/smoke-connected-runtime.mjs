// Local TEST checkout with canonical SePay Sandbox reads. No transfer API is used.
// --probe also sends a synthetic signed webhook through the hosted bridge and
// proves that an observation alone cannot allocate money or mark the Order PAID.
import { readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { createHmac, randomUUID } from 'node:crypto';

const directory = process.argv[2];
if (!directory) throw new Error('Usage: node smoke-connected-runtime.mjs <private-directory> [--probe]');
const settings = JSON.parse(await readFile(path.join(directory, 'runtime-settings.json'), 'utf8'));
const receiver = JSON.parse(await readFile(path.join(directory, 'runtime-receiver.json'), 'utf8'));
const base = 'http://127.0.0.1:18004';
async function request(url, method = 'GET', body, headers = {}) {
  const response = await fetch(url, { method, headers: { 'Content-Type': 'application/json', ...headers },
    body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(10000), redirect: 'error' });
  const result = await response.json();
  if (!response.ok) throw new Error(`HTTP_${response.status}:${result.error?.code || 'REQUEST_FAILED'}`);
  return { result, response };
}
const { result: identity } = await request('http://127.0.0.1:18104/test/token', 'POST',
  { user_id: randomUUID(), role: 'CUSTOMER' }, { 'X-Internal-Token': settings.ORDER_INTERNAL_TOKEN });
const auth = { Authorization: 'Bearer ' + identity.token };
const { result: cart } = await request(base + '/api/v1/cart/items', 'POST', { sku_code: 'MX-GION-500G', quantity: 2, revision: 0 }, auth);
const { result: quote } = await request(base + '/api/v1/checkout/quote', 'POST', {
  cart_revision: cart.revision, payment_method: 'VIETQR', address: { recipient_name: 'Synthetic TEST', phone: '+84905123456',
    street: 'TEST only', ward_code: 'HUE-01', province_code: '75' } }, auth);
const { result: accepted } = await request(base + '/api/v1/checkout', 'POST', {
  quote_id: quote.quote_id, cart_revision: cart.revision, payment_method: 'VIETQR' }, { ...auth, 'X-Idempotency-Key': randomUUID() });
let operation;
const deadline = Date.now() + 30000;
while (Date.now() < deadline) {
  ({ result: operation } = await request(base + '/api/v1/checkout-operations/' + accepted.operation_id, 'GET', undefined, auth));
  if (operation.status === 'SUCCEEDED') break;
  if (['FAILED', 'MANUAL_REVIEW'].includes(operation.status)) throw new Error('CHECKOUT_' + operation.status + ':' + operation.error_code);
  await new Promise(resolve => setTimeout(resolve, 250));
}
if (operation?.status !== 'SUCCEEDED') throw new Error('CHECKOUT_TIMEOUT');
const id = operation.order_id;
const { result: instructions } = await request(base + '/api/v1/orders/' + id + '/payment-instructions', 'GET', undefined, auth);
const privateFile = path.join(directory, 'runtime-smoke-private.json');
await writeFile(privateFile, JSON.stringify({ token: identity.token, operation, instructions }), { mode: 0o600 });
let bridgeStatus = 'NOT_RUN';
if (process.argv.includes('--probe')) {
  const stamp = String(Math.floor(Date.now() / 1000));
  const bankTime = new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Ho_Chi_Minh', year: 'numeric', month: '2-digit',
    day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(new Date());
  const raw = JSON.stringify({ id: Date.now(), gateway: receiver.bank_code, transactionDate: bankTime,
    accountNumber: receiver.account_number, code: instructions.payment_code, content: instructions.transfer_content,
    transferType: 'in', transferAmount: instructions.amount, accumulated: 0, subAccount: null,
    referenceCode: 'SYNTHETIC-BRIDGE-' + randomUUID(), description: 'Connectivity probe; no canonical bank transaction' });
  const secret = await readFile(path.join(directory, 'webhook-secret.txt'), 'utf8');
  const signature = 'sha256=' + createHmac('sha256', secret).update(stamp + '.' + raw).digest('hex');
  const response = await fetch('https://sepay-test.npanhbao.dev/api/v1/payments/sepay/webhook', {
    method: 'POST', headers: { 'Content-Type': 'application/json', 'X-SePay-Timestamp': stamp, 'X-SePay-Signature': signature },
    body: raw, signal: AbortSignal.timeout(10000), redirect: 'error' });
  if (response.status !== 200 || response.headers.get('X-Payment-Processing') !== 'runtime-observation' || (await response.json()).success !== true)
    throw new Error('BRIDGE_NOT_ACKNOWLEDGED');
  bridgeStatus = 'RUNTIME_DURABLE_ACK';
}
const { result: payment } = await request(base + '/api/v1/orders/' + id + '/payment', 'GET', undefined, auth);
if (payment.order_status !== 'PENDING_PAYMENT' || payment.allocated_amount !== 0) throw new Error('WEBHOOK_BOOKED_UNVERIFIED_FUNDS');
console.log(JSON.stringify({ status: 'PASS', mode: 'connected-sandbox-runtime', order_id: id, operation_status: operation.status,
  order_status: payment.order_status, display_state: payment.display_state, allocated_amount: payment.allocated_amount,
  bridge: bridgeStatus, provider_transfer_mutations: false, real_paid_acceptance: false, expires_at: instructions.expires_at,
  private_instructions: privateFile }));
