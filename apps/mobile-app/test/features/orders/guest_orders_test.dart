import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/orders/data/mock_orders_repository.dart';

void main() {
  test(
    'neutral challenge reveals no order; OTP grants exactly one order',
    () async {
      final dependencies = RepositoryFactory.create(mode: DataSourceMode.mock);
      final orders = dependencies.ordersRepository;
      final guest = dependencies.guestOrdersRepository;
      await expectLater(orders.order('OM-8921'), throwsA(isA<ApiException>()));
      final challenge = await guest.challenge(
        'OM-8921',
        '0905123456',
        idempotencyKey: 'one',
      );
      final unknown = await guest.challenge(
        'UNKNOWN',
        '0905123456',
        idempotencyKey: 'two',
      );
      expect(unknown.message, challenge.message);
      await expectLater(
        guest.verify(challenge.id, '000000', idempotencyKey: 'wrong'),
        throwsA(isA<ApiException>()),
      );
      final access = await guest.verify(
        challenge.id,
        '789214',
        idempotencyKey: 'valid',
      );
      expect((await orders.order(access.orderId)).id, 'OM-8921');
      await expectLater(orders.order('OM-8874'), throwsA(isA<ApiException>()));
      await expectLater(orders.orders(), throwsA(isA<ApiException>()));
      await expectLater(
        guest.verify(challenge.id, '789214', idempotencyKey: 'again'),
        throwsA(isA<ApiException>()),
      );
      (orders as MockOrdersRepository).grantGuestAccess(
        access.orderId,
        DateTime.now().subtract(const Duration(seconds: 1)),
      );
      await expectLater(
        orders.order(access.orderId),
        throwsA(isA<ApiException>()),
      );
    },
  );
}
