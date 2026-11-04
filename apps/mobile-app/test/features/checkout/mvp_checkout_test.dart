import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/checkout/data/mvp_checkout_store.dart';
import 'package:mobile_app/features/checkout/data/rest_mvp_checkout_repository.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_repository.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';
import 'package:mobile_app/features/checkout/presentation/mvp_checkout_cubit.dart';
import '../../support/mvp_checkout_fixtures.dart';

class _Repo implements MvpCheckoutRepository {
  final calls = <({MvpCheckoutInput input, String key})>[];
  final polls = <String>[];
  Future<MvpCheckoutOperation> Function()? response;
  Future<MvpCheckoutOperation> Function()? pollResponse;
  @override
  Future<MvpCheckoutOperation> accept(
    MvpCheckoutInput input, {
    required String key,
  }) {
    calls.add((input: input, key: key));
    return response?.call() ?? Future.value(checkoutOperation());
  }

  @override
  Future<MvpCheckoutOperation> operation(String id) {
    polls.add(id);
    return pollResponse?.call() ??
        Future.value(checkoutOperation(status: 'SUCCEEDED'));
  }
}

class _Store extends MemoryMvpCheckoutStore {
  bool failWrite = false, failRead = false;
  @override
  Future<void> write(MvpCheckoutAttempt attempt) async {
    if (failWrite) throw StateError('Storage full');
    return super.write(attempt);
  }

  @override
  Future<MvpCheckoutAttempt?> read(String owner, String scope) async {
    if (failRead) throw const FormatException('Corrupt record');
    return super.read(owner, scope);
  }
}

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'invalid quote identity cannot throw through UI or start an operation',
    () async {
      final r = _Repo();
      final c = MvpCheckoutCubit(
        MvpCheckoutRuntime(repository: r, store: _Store(), scope: 'b'),
        principal: () => 'a',
        now: () => checkoutNow,
      );
      await c.restore();
      await c.start(
        MvpQuote.fromJson({...checkoutQuoteJson(), 'quote_id': 'invalid-uuid'}),
      );
      expect(r.calls, isEmpty);
      expect(c.state.attempt, isNull);
      expect(c.state.message, isNotNull);
      await c.close();
    },
  );
  test(
    'operation schema rejects invented order identity, missing failure and unknown state',
    () {
      expect(checkoutOperation().orderId, isNull);
      expect(checkoutOperation(status: 'SUCCEEDED').orderId, orderId);
      for (final bad in [
        {...operationJson(), 'order_id': orderId},
        {...operationJson(status: 'SUCCEEDED'), 'order_id': null},
        {...operationJson(), 'status': 'FAILED'},
        {...operationJson(), 'status': 'UNKNOWN'},
        {...operationJson(), 'updated_at': '2026-10-30T00:00:00Z'},
      ]) {
        expect(() => MvpCheckoutOperation.fromJson(bad), throwsFormatException);
      }
    },
  );
  test(
    'REST sends exact public input/key, follows operation and preserves PublicError operation_id',
    () async {
      final requests = <RequestOptions>[];
      var error = false;
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (o, h) {
            requests.add(o);
            if (error) {
              h.reject(
                DioException(
                  requestOptions: o,
                  type: DioExceptionType.badResponse,
                  response: Response(
                    requestOptions: o,
                    statusCode: 503,
                    data: {
                      'code': 'PENDING',
                      'message': 'Pending',
                      'requestId': 'r1',
                      'operation_id': operationId,
                    },
                  ),
                ),
              );
            } else {
              h.resolve(
                Response(
                  requestOptions: o,
                  statusCode: o.method == 'POST' ? 202 : 200,
                  data: operationJson(),
                ),
              );
            }
          },
        ),
      );
      final r = RestMvpCheckoutRepository(ApiClient(dio: dio));
      await r.accept(
        MvpCheckoutInput.fromQuote(checkoutQuote()),
        key: 'fixture-key-123456',
      );
      expect(requests.single.data, {
        'quote_id': quoteId,
        'cart_revision': 2,
        'payment_method': 'VIETQR',
      });
      expect(requests.single.headers['Idempotency-Key'], 'fixture-key-123456');
      expect(requests.single.headers.containsKey('X-Idempotency-Key'), isFalse);
      await r.operation(operationId);
      expect(requests.last.path, '/api/v1/checkout-operations/$operationId');
      error = true;
      await expectLater(
        r.operation(operationId),
        throwsA(
          isA<ApiException>().having(
            (e) => e.failure.operationId,
            'preserved recovery identity',
            operationId,
          ),
        ),
      );
      expect(requests.length, 3);
    },
  );
  test(
    'secure store survives new instances and isolates owner/backend; no contact or tokens',
    () async {
      final attempt = MvpCheckoutAttempt(
        owner: 'a',
        scope: 'backend-a',
        key: 'fixture-key-123456',
        input: MvpCheckoutInput.fromQuote(checkoutQuote()),
      );
      await const SecureMvpCheckoutStore().write(attempt);
      final restored = await const SecureMvpCheckoutStore().read(
        'a',
        'backend-a',
      );
      expect(restored!.toJson(), attempt.toJson());
      expect(
        await const SecureMvpCheckoutStore().read('b', 'backend-a'),
        isNull,
      );
      expect(
        await const SecureMvpCheckoutStore().read('a', 'backend-b'),
        isNull,
      );
      expect(restored.toJson().toString(), isNot(contains('phone')));
      expect(restored.toJson().toString(), isNot(contains('token')));
    },
  );
  test(
    'unknown POST survives controller recreation; expired replay retains exact key/body',
    () async {
      final r = _Repo(), store = _Store();
      final runtime = MvpCheckoutRuntime(
        repository: r,
        store: store,
        scope: 'backend',
      );
      final c = MvpCheckoutCubit(
        runtime,
        principal: () => 'a',
        now: () => checkoutNow,
      );
      await c.restore();
      r.response = () async {
        expect((await store.read('a', 'backend'))!.key, r.calls.single.key);
        throw const ApiException(
          Failure(kind: FailureKind.timeout, message: 'Unknown outcome'),
        );
      };
      await c.start(checkoutQuote());
      expect(c.state.unresolved, isTrue);
      await c.start(checkoutQuote());
      expect(r.calls.length, 1);
      await c.close();
      final fresh = MvpCheckoutCubit(
        runtime,
        principal: () => 'a',
        now: () => checkoutNow.add(const Duration(days: 1)),
      );
      await fresh.restore();
      r.response = () async => checkoutOperation();
      await fresh.recover();
      expect(r.calls.length, 2);
      expect(r.calls.last.key, r.calls.first.key);
      expect(r.calls.last.input.toJson(), r.calls.first.input.toJson());
      expect(fresh.state.operation!.status, 'ACCEPTED');
      await fresh.recover();
      expect(fresh.state.operation!.status, 'SUCCEEDED');
      expect(r.polls, [operationId]);
      await fresh.start(checkoutQuote());
      expect(r.calls.length, 2);
      await fresh.close();
    },
  );
  test(
    'duplicate confirmation is serialized and failure to persist prevents POST',
    () async {
      final r = _Repo(), store = _Store();
      final c = MvpCheckoutCubit(
        MvpCheckoutRuntime(repository: r, store: store, scope: 'b'),
        principal: () => 'a',
        now: () => checkoutNow,
      );
      await c.restore();
      store.failWrite = true;
      await c.start(checkoutQuote());
      expect(r.calls, isEmpty);
      expect(c.state.storageBlocked, isTrue);
      final key = c.state.attempt!.key;
      store.failWrite = false;
      final pending = Completer<MvpCheckoutOperation>();
      r.response = () => pending.future;
      final request = c.recover();
      await Future<void>.delayed(Duration.zero);
      await c.start(checkoutQuote());
      await c.recover();
      expect(r.calls.length, 1);
      expect(r.calls.single.key, key);
      pending.complete(checkoutOperation());
      await request;
      await c.close();
    },
  );
  test(
    'corrupt storage fails closed and requires successful restore',
    () async {
      final r = _Repo(), store = _Store()..failRead = true;
      final c = MvpCheckoutCubit(
        MvpCheckoutRuntime(repository: r, store: store, scope: 'b'),
        principal: () => 'a',
        now: () => checkoutNow,
      );
      await c.restore();
      await c.start(checkoutQuote());
      expect(c.state.storageBlocked, isTrue);
      expect(r.calls, isEmpty);
      store.failRead = false;
      await c.recover();
      expect(c.state.canStart, isTrue);
      await c.close();
    },
  );
  test(
    'account switch discards delayed response and restores only its original retry record',
    () async {
      var owner = 'a';
      final identities = StreamController<String?>.broadcast(sync: true);
      final r = _Repo(), store = _Store();
      final pending = Completer<MvpCheckoutOperation>();
      r.response = () => pending.future;
      final c = MvpCheckoutCubit(
        MvpCheckoutRuntime(repository: r, store: store, scope: 'b'),
        principal: () => owner,
        identityChanges: identities.stream,
        now: () => checkoutNow,
      );
      await c.restore();
      final request = c.start(checkoutQuote());
      await Future<void>.delayed(Duration.zero);
      owner = 'b';
      identities.add(owner);
      await Future<void>.delayed(Duration.zero);
      pending.complete(checkoutOperation(status: 'SUCCEEDED'));
      await request;
      expect(c.state.attempt, isNull);
      expect(await store.read('b', 'b'), isNull);
      owner = 'a';
      identities.add(owner);
      await Future<void>.delayed(Duration.zero);
      expect(c.state.attempt!.key, r.calls.single.key);
      expect(c.state.operation, isNull);
      await c.close();
      await identities.close();
    },
  );
  test(
    'PublicError recovery identity is polled without an invented operation or repeated POST',
    () async {
      final r = _Repo();
      r.response = () async => throw const ApiException(
        Failure(
          kind: FailureKind.server,
          message: 'Unknown',
          operationId: operationId,
        ),
      );
      final c = MvpCheckoutCubit(
        MvpCheckoutRuntime(repository: r, store: _Store(), scope: 'b'),
        principal: () => 'a',
        now: () => checkoutNow,
      );
      await c.restore();
      await c.start(checkoutQuote());
      expect(c.state.operation, isNull);
      expect(c.state.attempt!.pollId, operationId);
      await c.recover();
      expect(r.calls.length, 1);
      expect(r.polls, [operationId]);
      expect(c.state.operation!.status, 'SUCCEEDED');
      await c.close();
    },
  );
  test(
    'mismatched/regressive polls do not replace authority, manual review does not auto-poll',
    () async {
      final r = _Repo();
      final c = MvpCheckoutCubit(
        MvpCheckoutRuntime(repository: r, store: _Store(), scope: 'b'),
        principal: () => 'a',
        now: () => checkoutNow,
      );
      await c.restore();
      await c.start(checkoutQuote());
      r.pollResponse = () async => MvpCheckoutOperation.fromJson({
        ...operationJson(),
        'operation_id': quoteId,
      });
      await c.recover();
      expect(c.state.operation!.id, operationId);
      expect(c.state.message, isNotNull);
      r.pollResponse = () async => MvpCheckoutOperation.fromJson({
        ...operationJson(),
        'updated_at': '2026-10-31T00:00:00Z',
      });
      await c.recover();
      expect(c.state.operation!.updatedAt, DateTime.utc(2026, 10, 31, 0, 0, 1));
      r.pollResponse = () async => checkoutOperation(status: 'MANUAL_REVIEW');
      await c.recover();
      final calls = r.polls.length;
      await c.recover(automatic: true);
      expect(r.polls.length, calls);
      expect(c.state.canStart, isFalse);
      await c.close();
    },
  );
}
