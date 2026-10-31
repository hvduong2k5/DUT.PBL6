import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';

const quoteId = '11111111-1111-4111-8111-111111111111';
const operationId = '22222222-2222-4222-8222-222222222222';
const orderId = '33333333-3333-4333-8333-333333333333';
final checkoutNow = DateTime.utc(2026, 10, 31);
Map<String, dynamic> operationJson({String status = 'ACCEPTED'}) => {
  'operation_id': operationId,
  'status': status,
  'stage': status == 'ACCEPTED' ? 'VALIDATING' : 'DONE',
  'order_id': status == 'SUCCEEDED' ? orderId : null,
  if (status == 'FAILED') 'error_code': 'OUT_OF_STOCK',
  'accepted_at': '2026-10-31T00:00:00Z',
  'updated_at': '2026-10-31T00:00:01Z',
};
MvpCheckoutOperation checkoutOperation({String status = 'ACCEPTED'}) =>
    MvpCheckoutOperation.fromJson(operationJson(status: status));
MvpQuote checkoutQuote() => MvpQuote.fromJson({
  'quote_id': quoteId,
  'cart_revision': 2,
  'payment_method': 'VIETQR',
  'items': [
    {'sku_code': 'SKU', 'quantity': 2, 'unit_price': 50000},
  ],
  'address': {
    'recipient_name': 'Fixture',
    'phone': '0900000000',
    'street': 'Fixture',
    'province_code': '01',
    'ward_code': '00001',
  },
  'catalog_snapshot_id': 'fixture',
  'shipping_quote_reference': 'fixture',
  'subtotal': 100000,
  'shipping_fee': 15000,
  'final_amount': 115000,
  'expires_at': '2026-10-31T00:10:00Z',
  'catalog_expires_at': '2026-10-31T00:10:00Z',
  'shipping_expires_at': '2026-10-31T00:10:00Z',
});
