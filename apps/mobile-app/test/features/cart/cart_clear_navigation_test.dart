import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  testWidgets(
    'clear cart can be cancelled and only confirmed action deletes lines',
    (tester) async {
      final deps = RepositoryFactory.create(mode: DataSourceMode.mock);
      await tester.runAsync(() async {
        final p = await deps.catalogRepository.product('PROD-MX-GION-500');
        await deps.cartRepository.add(p, p.variants.first, 2);
      });
      await tester.pumpWidget(
        OmaApp(initialLocation: '/cart', dependencies: deps),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('Dọn giỏ hàng'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Giữ lại'));
      await tester.pumpAndSettle();
      expect(find.text('Giỏ hàng trống'), findsNothing);
      await tester.tap(find.text('Dọn giỏ hàng'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Xóa toàn bộ'));
      await tester.pumpAndSettle();
      expect(find.text('Giỏ hàng trống'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
