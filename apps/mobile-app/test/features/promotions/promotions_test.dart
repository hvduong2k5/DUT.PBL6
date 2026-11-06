import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/promotions/data/rest_promotions_repository.dart';
import 'package:mobile_app/features/promotions/domain/promotions_repository.dart';
import 'package:mobile_app/features/promotions/presentation/promotions_cubit.dart';

class PromotionsRepositoryMock extends Mock implements PromotionsRepository {}

void main() {
  test(
    'REST voucher validation uses exactly ZIP request and money response',
    () async {
      RequestOptions? request;
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            request = options;
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {
                  'is_valid': true,
                  'discount_amount': {
                    'currency_code': 'VND',
                    'units': 30000,
                    'nanos': 0,
                  },
                  'message': 'Hợp lệ',
                },
              ),
            );
          },
        ),
      );
      final result = await RestPromotionsRepository(
        ApiClient(dio: dio),
      ).validate('CODE', 250000);
      expect(request!.path, '/api/v1/promotions/validate');
      expect(request!.data, {
        'voucher_code': 'CODE',
        'subtotal_amount': 250000,
      });
      expect(result.discount, Money.vnd(30000));
    },
  );

  test(
    'changing code or subtotal rejects stale validation responses',
    () async {
      final repository = PromotionsRepositoryMock();
      final pending = Completer<VoucherValidation>();
      when(
        () => repository.validate('CODE', 250000),
      ).thenAnswer((_) => pending.future);
      final cubit = PromotionsCubit(repository, subtotal: 250000);
      cubit.setCode(' code ');
      final request = cubit.validate();
      cubit.setSubtotal('100000');
      pending.complete(
        VoucherValidation(
          valid: true,
          discount: Money.vnd(30000),
          message: 'Hợp lệ',
        ),
      );
      await request;
      expect(cubit.state.result, isNull);
      expect(cubit.state.validating, isFalse);
      expect(cubit.state.subtotal, 100000);
      cubit.setCode('NEW');
      expect(cubit.state.result, isNull);
      await cubit.close();
    },
  );

  test('voucher list empty and retry do not fabricate offers', () async {
    final repository = PromotionsRepositoryMock();
    when(repository.vouchers).thenThrow(StateError('offline'));
    final cubit = PromotionsCubit(repository);
    await cubit.load();
    expect(cubit.state.error, isNotNull);
    cubit.setCode('CODE');
    expect(cubit.state.error, isNotNull);
    when(repository.vouchers).thenAnswer((_) async => []);
    await cubit.load();
    expect(cubit.state.vouchers, isEmpty);
    expect(cubit.state.error, isNull);
    await cubit.close();
  });
}
