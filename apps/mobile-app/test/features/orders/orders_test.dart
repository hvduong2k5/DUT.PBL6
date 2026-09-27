import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/features/orders/data/mock_orders_repository.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/presentation/orders_cubit.dart';

void main() {
  group('OrderStatusUi', () {
    test('maps API aliases to the customer-facing status model', () {
      expect(OrderStatusUi.fromApi('PENDING_PAYMENT'), OrderStatus.pending);
      expect(OrderStatusUi.fromApi('PROCESSING'), OrderStatus.confirmed);
      expect(OrderStatusUi.fromApi('IN_TRANSIT'), OrderStatus.shipping);
      expect(OrderStatusUi.fromApi('DELIVERED'), OrderStatus.delivered);
      expect(OrderStatusUi.fromApi('CANCELLED'), OrderStatus.cancelled);
    });

    test('falls back safely for a future unknown API status', () {
      expect(OrderStatusUi.fromApi('NEW_BACKEND_STATUS'), OrderStatus.pending);
    });
  });

  test('CustomerOrder defaults to an empty item list', () {
    final order = CustomerOrder(
      id: 'OM-EMPTY',
      status: OrderStatus.pending,
      totalItems: 0,
      total: Money.vnd(0),
      createdAt: DateTime(2026),
    );

    expect(order.items, isEmpty);
  });

  test(
    'mock order item quantities and unit prices match order totals',
    () async {
      final orders = await MockOrdersRepository().orders();

      for (final order in orders) {
        final itemCount = order.items.fold<int>(
          0,
          (count, item) => count + item.quantity,
        );
        final itemTotal = order.items.fold<Money>(
          Money.vnd(0),
          (total, item) => total + item.lineTotal,
        );

        expect(itemCount, order.totalItems, reason: order.id);
        expect(itemTotal, order.total, reason: order.id);
      }
    },
  );

  test('OrdersCubit loads and filters order history', () async {
    final cubit = OrdersCubit(MockOrdersRepository());

    await cubit.load();
    expect(cubit.state.status, OrdersStatus.success);
    expect(cubit.state.orders, isNotEmpty);

    cubit.filter(OrderStatus.delivered);
    expect(cubit.state.visible, isNotEmpty);
    expect(
      cubit.state.visible.every(
        (order) => order.status == OrderStatus.delivered,
      ),
      isTrue,
    );

    cubit.filter(null);
    expect(cubit.state.visible.length, cubit.state.orders.length);
    await cubit.close();
  });
}
