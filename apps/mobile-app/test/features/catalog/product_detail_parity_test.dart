import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/features/catalog/data/rest_catalog_remote_data_source.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'detail keeps supplied gallery, certificate and exact weight/original price; stock null stays unknown',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
      final price = {'currency_code': 'VND', 'units': 85000, 'nanos': 0};
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (o, h) => h.resolve(
            Response(
              requestOptions: o,
              statusCode: 200,
              data: {
                'product_id': 'P',
                'name': 'Quà',
                'slug': 'qua',
                'description': 'Mô tả',
                'images': [
                  'https://example.com/a.jpg',
                  'https://example.com/b.jpg',
                ],
                'ocop_star': 4,
                'ocop_certificate_no': 'CERT',
                'story': 'Câu chuyện',
                'variants': [
                  {
                    'variant_id': 'V',
                    'sku_code': 'SKU',
                    'name': 'Hộp',
                    'weight_gram': 500,
                    'price': price,
                    'original_price': {...price, 'units': 95000},
                    'stock_available': null,
                  },
                ],
              },
            ),
          ),
        ),
      );
      final p = await RestCatalogRemoteDataSource(
        ApiClient(dio: dio),
      ).product('P');
      expect(p.images.length, 2);
      expect(p.certificateNumber, 'CERT');
      expect(p.variants.single.weightGrams, 500);
      expect(p.variants.single.originalPrice!.units, BigInt.from(95000));
      expect(p.stockKnown, isFalse);
      expect(p.availabilityLabel, 'Đang cập nhật tồn kho');
    },
  );
  testWidgets(
    'single variant still requires explicit choice; cart action adds chosen SKU',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final deps = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.pumpWidget(
        OmaApp(
          initialLocation: '/product/PROD-MX-GION-500',
          dependencies: deps,
        ),
      );
      await tester.pumpAndSettle();
      AppPrimaryButton action() =>
          tester.widget<AppPrimaryButton>(find.byType(AppPrimaryButton).first);
      expect(action().onPressed, isNull);
      final chip = find.byType(ChoiceChip).first;
      await tester.scrollUntilVisible(
        chip,
        200,
        scrollable: find.byType(Scrollable).first,
      );
      await Scrollable.ensureVisible(tester.element(chip), alignment: .3);
      await tester.pumpAndSettle();
      await tester.tap(chip);
      await tester.pumpAndSettle();
      expect(action().onPressed, isNotNull);
      await tester.tap(find.text('Thêm vào giỏ'));
      await tester.pumpAndSettle();
      expect(
        (await deps.cartRepository.getCart()).items.single.variant.sku,
        'MX-GION-500G',
      );
      expect(tester.takeException(), isNull);
    },
  );
}
