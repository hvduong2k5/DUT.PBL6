import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/orders/data/rest_mvp_orders_repository.dart';
import 'package:mobile_app/features/orders/data/mvp_cancellation_store.dart';
import 'package:mobile_app/features/orders/domain/mvp_order_models.dart';
import 'package:mobile_app/features/orders/domain/mvp_orders_repository.dart';
import 'package:mobile_app/features/orders/presentation/mvp_order_cubit.dart';
import 'package:mobile_app/features/checkout/presentation/mvp_payment_cubit.dart';
import '../../support/mvp_checkout_fixtures.dart';
import '../../support/mvp_order_fixtures.dart';

class _Orders implements MvpOrdersRepository {
  Map<String, dynamic> orderData = orderJson(),
      paymentData = paymentJson(),
      instructionsData = instructionsJson();
  int instructionCalls = 0;
  Future<MvpPaymentStatus> Function()? paymentResponse;
  Future<MvpOrder> Function()? orderResponse;
  Future<MvpCancellation> Function()? cancelResponse;
  final cancellations = <MvpCancellationCommand>[];
  @override
  Future<MvpOrderPageData> firstPage() async => MvpOrderPageData.fromJson({
    'items': [orderData],
    'next_cursor': 'opaque',
  });
  @override
  Future<MvpOrder> order(String id) =>
      orderResponse?.call() ?? Future.value(MvpOrder.fromJson(orderData));
  @override
  Future<MvpPaymentStatus> payment(String id) =>
      paymentResponse?.call() ??
      Future.value(MvpPaymentStatus.fromJson(paymentData));
  @override
  Future<MvpPaymentInstructions> instructions(String id) async {
    instructionCalls++;
    return MvpPaymentInstructions.fromJson(instructionsData);
  }

  @override
  Future<MvpCancellation> cancel(MvpCancellationCommand command) {
    cancellations.add(command);
    return cancelResponse?.call() ??
        Future.value(MvpCancellation.fromJson(cancellationJson()));
  }
}

class _Store extends MemoryMvpCancellationStore {
  bool fail = false;
  @override
  Future<void> write(MvpCancellationCommand command) async {
    if (fail) throw StateError('Storage full');
    return super.write(command);
  }
}

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'MVP money/action projections do not guess missing values or confirm independent money',
    () {
      final order = MvpOrder.fromJson(orderJson());
      expect(order.canPay, isTrue);
      expect(order.snapshot, isNull);
      expect(
        MvpOrder.fromJson({
          ...orderJson(),
          'final_amount': 1000000000001,
        }).total.units,
        BigInt.from(1000000000001),
      ); // Order int64 does not inherit Payment's upper bound.
      expect(() => order.actions.clear(), throwsUnsupportedError);
      expect(
        () => MvpOrder.fromJson({...orderJson(), 'final_amount': null}),
        throwsFormatException,
      );
      expect(
        () => MvpPaymentStatus.fromJson({
          ...paymentJson(),
          'expected_amount': 1.5,
        }),
        throwsFormatException,
      );
      expect(
        () =>
            MvpPaymentStatus.fromJson({...paymentJson(), 'refunded_amount': 1}),
        throwsFormatException,
      );
      final processing = MvpOrder.fromJson({
        ...orderJson(),
        'status': 'PAYMENT_FINALIZING',
        'payment_status': 'CONFIRMED',
      });
      expect(processing.statusLabel, 'Đang hoàn tất thanh toán');
      expect(processing.canPay, isFalse);
      expect(
        MvpCancellation.fromJson(
          cancellationJson(status: 'SUCCEEDED'),
        ).compensationStatus,
        'PENDING',
      );
    },
  );
  test(
    'instructions retain leading zero account and server URL, reject malformed or unsafe data',
    () {
      final instructions = MvpPaymentInstructions.fromJson(instructionsJson());
      expect(instructions.account, '001234');
      for (final bad in [
        {...instructionsJson(), 'qr_url': 'http://example.invalid/qr'},
        {
          ...instructionsJson(),
          'qr_url': 'https://user:secret@example.invalid/qr',
        },
        {...instructionsJson(), 'can_pay': false},
        {...instructionsJson(), 'amount': 1000000000001},
      ]) {
        expect(
          () => MvpPaymentInstructions.fromJson(bad),
          throwsFormatException,
        );
      }
    },
  );
  test(
    'REST uses first-page without legacy queries and exact versioned cancellation body/header',
    () async {
      final requests = <RequestOptions>[];
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (o, h) {
            requests.add(o);
            final data = o.path.endsWith('/cancel')
                ? cancellationJson()
                : o.path.endsWith('/payment-instructions')
                ? instructionsJson()
                : o.path.endsWith('/payment')
                ? paymentJson()
                : o.path == '/api/v1/orders'
                ? {
                    'items': [orderJson()],
                    'next_cursor': 'opaque',
                  }
                : orderJson();
            h.resolve(
              Response(
                requestOptions: o,
                statusCode: o.method == 'POST' ? 202 : 200,
                data: data,
              ),
            );
          },
        ),
      );
      final r = RestMvpOrdersRepository(ApiClient(dio: dio));
      expect((await r.firstPage()).nextCursor, 'opaque');
      expect(requests.single.queryParameters, isEmpty);
      await r.order(orderId);
      await r.payment(orderId);
      await r.instructions(orderId);
      final command = MvpCancellationCommand(
        owner: 'a',
        scope: 'b',
        orderId: orderId,
        key: 'fixture-cancel-123456',
        reason: 'Fixture',
        version: 1,
      );
      final result = await r.cancel(command);
      expect(result.status, 'PENDING');
      expect(requests.last.data, {'reason': 'Fixture', 'active': true});
      expect(requests.last.headers['If-Match'], '1');
      expect(requests.last.headers['Idempotency-Key'], command.key);
      expect(requests.where((r) => r.method == 'POST').length, 1);
    },
  );
  test(
    'payment deadline uses server time plus monotonic elapsed and hides at 30-second margin',
    () async {
      final r = _Orders();
      var elapsed = Duration.zero;
      final c = MvpPaymentCubit(r, orderId, () => 'a', elapsed: () => elapsed);
      await c.load();
      expect(c.instructionsUsable, isTrue);
      expect(r.instructionCalls, 1);
      elapsed = const Duration(minutes: 9, seconds: 29);
      expect(c.instructionsUsable, isTrue);
      elapsed += const Duration(seconds: 1);
      expect(c.instructionsUsable, isFalse);
      expect(c.serverNow, checkoutNow.add(elapsed));
      await c.close();
    },
  );
  test(
    'verification, delay, finalization and paid states never show transfer instructions',
    () async {
      for (final display in [
        'VERIFYING',
        'VERIFICATION_DELAYED',
        'FINALIZING',
        'PAID',
        'RECONCILIATION_REQUIRED',
        'CANCELLED',
        'EXPIRED',
      ]) {
        final r = _Orders()
          ..paymentData = paymentJson(display: display, canPay: true);
        final c = MvpPaymentCubit(r, orderId, () => 'a');
        await c.load();
        expect(c.instructionsUsable, isFalse, reason: display);
        expect(r.instructionCalls, 0, reason: display);
        expect(c.state.status!.displayState, display);
        await c.close();
      }
    },
  );
  test(
    'payment instructions race, version regression and refresh failure hide old QR',
    () async {
      final r = _Orders();
      final c = MvpPaymentCubit(r, orderId, () => 'a');
      await c.load();
      expect(c.instructionsUsable, isTrue);
      final pending = Completer<MvpPaymentStatus>();
      r.paymentResponse = () => pending.future;
      final refresh = c.load();
      await Future<void>.delayed(Duration.zero);
      expect(c.instructionsUsable, isFalse);
      pending.completeError(StateError('Offline'));
      await refresh;
      expect(c.instructionsUsable, isFalse);
      r.paymentResponse = () async =>
          MvpPaymentStatus.fromJson(paymentJson(version: 2));
      r.instructionsData = {...instructionsJson(), 'intent_id': quoteId};
      await c.load();
      expect(c.instructionsUsable, isFalse);
      expect(c.state.message, isNotNull);
      r.instructionsData = instructionsJson();
      await c.load();
      expect(c.instructionsUsable, isTrue);
      r.paymentResponse = () async =>
          MvpPaymentStatus.fromJson(paymentJson(version: 1));
      await c.load();
      expect(c.instructionsUsable, isFalse);
      expect(c.state.status!.version, 2);
      await c.close();
    },
  );
  test(
    'payment actions, server poll stop and stale account responses are guarded',
    () async {
      final r = _Orders()
        ..orderData = {
          ...orderJson(),
          'allowed_actions': ['CONTACT_SUPPORT'],
        };
      var elapsed = Duration.zero, owner = 'a';
      final c = MvpPaymentCubit(
        r,
        orderId,
        () => owner,
        elapsed: () => elapsed,
      );
      await c.load();
      expect(r.instructionCalls, 0);
      elapsed = const Duration(seconds: 3);
      expect(c.shouldPoll, isTrue);
      r.paymentData = {...paymentJson(), 'next_poll_after_seconds': 0};
      await c.load();
      elapsed = const Duration(hours: 1);
      expect(c.shouldPoll, isFalse);
      final pending = Completer<MvpPaymentStatus>();
      r.paymentResponse = () => pending.future;
      final refresh = c.load();
      await Future<void>.delayed(Duration.zero);
      owner = 'b';
      pending.complete(MvpPaymentStatus.fromJson(paymentJson(display: 'PAID')));
      await refresh;
      expect(c.state.status!.displayState, 'WAITING_PAYMENT');
      expect(c.instructionsUsable, isFalse);
      await c.close();
    },
  );
  test(
    'cancellation unknown outcome replays original version/reason/key after controller recreation',
    () async {
      final r = _Orders(), store = _Store();
      final runtime = MvpOrdersRuntime(
        repository: r,
        cancellations: store,
        scope: 'b',
      );
      final c = MvpOrderCubit(runtime, orderId, () => 'a');
      await c.load();
      r.cancelResponse = () async => throw StateError('Unknown transport');
      await c.cancel('Original reason');
      expect(c.state.command!.result, isNull);
      await c.cancel('Changed reason');
      expect(r.cancellations.length, 1);
      await c.close();
      r.orderData = {
        ...orderJson(version: 2),
        'allowed_actions': [],
        'cancellation_status': 'PENDING',
        'compensation_status': 'PENDING',
      };
      final fresh = MvpOrderCubit(runtime, orderId, () => 'a');
      await fresh.load();
      r.cancelResponse = () async =>
          MvpCancellation.fromJson(cancellationJson());
      await fresh.recoverCancellation();
      expect(r.cancellations.length, 2);
      expect(r.cancellations.last.toJson(), r.cancellations.first.toJson());
      expect(fresh.state.order!.cancellationStatus, 'PENDING');
      expect(fresh.state.order!.status, 'PENDING_PAYMENT');
      await fresh.recoverCancellation();
      expect(r.cancellations.length, 2);
      await fresh.close();
    },
  );
  test(
    'cancellation storage failure prevents POST and new version requires review before confirmation',
    () async {
      final r = _Orders(), store = _Store();
      final c = MvpOrderCubit(
        MvpOrdersRuntime(repository: r, cancellations: store, scope: 'b'),
        orderId,
        () => 'a',
      );
      await c.load();
      r.orderData = orderJson(version: 2);
      await c.cancel('Fixture');
      expect(r.cancellations, isEmpty);
      expect(c.state.order!.version, 2);
      store.fail = true;
      await c.cancel('Fixture');
      expect(r.cancellations, isEmpty);
      expect(c.state.storageBlocked, isTrue);
      final original = c.state.command!;
      store.fail = false;
      r.cancelResponse = () async =>
          MvpCancellation.fromJson({...cancellationJson(), 'version': 3});
      r.orderData = {
        ...orderJson(version: 3),
        'cancellation_status': 'PENDING',
        'allowed_actions': [],
      };
      await c.recoverCancellation();
      expect(r.cancellations.single.key, original.key);
      expect(r.cancellations.single.version, 2);
      await c.close();
    },
  );
  test(
    'secure cancellation command is isolated by principal/backend/order across instances',
    () async {
      final command = MvpCancellationCommand(
        owner: 'a',
        scope: 'b',
        orderId: orderId,
        key: 'fixture-cancel-123456',
        reason: 'Fixture',
        version: 1,
      );
      await const SecureMvpCancellationStore().write(command);
      expect(
        (await const SecureMvpCancellationStore().read(
          'a',
          'b',
          orderId,
        ))!.toJson(),
        command.toJson(),
      );
      expect(
        await const SecureMvpCancellationStore().read('x', 'b', orderId),
        isNull,
      );
      expect(
        await const SecureMvpCancellationStore().read('a', 'x', orderId),
        isNull,
      );
      expect(
        await const SecureMvpCancellationStore().read('a', 'b', quoteId),
        isNull,
      );
    },
  );
  for (final closeBeforeReply in [false, true]) {
    test(
      'late cancellation outcome is discarded on ${closeBeforeReply ? 'dispose' : 'account change'}',
      () async {
        var owner = 'a';
        final r = _Orders(), store = _Store();
        final c = MvpOrderCubit(
          MvpOrdersRuntime(repository: r, cancellations: store, scope: 'b'),
          orderId,
          () => owner,
        );
        await c.load();
        final pending = Completer<MvpCancellation>();
        r.cancelResponse = () => pending.future;
        final request = c.cancel('Fixture');
        await Future<void>.delayed(Duration.zero);
        expect(r.cancellations.length, 1);
        if (closeBeforeReply) {
          await c.close();
        } else {
          owner = 'b';
        }
        pending.complete(
          MvpCancellation.fromJson(cancellationJson(status: 'SUCCEEDED')),
        );
        await request;
        expect((await store.read('a', 'b', orderId))!.result, isNull);
        if (!c.isClosed) await c.close();
      },
    );
  }
}
