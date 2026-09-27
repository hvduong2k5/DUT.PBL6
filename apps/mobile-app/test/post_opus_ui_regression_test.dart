import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';

void main() {
  setUp(() {
    FlutterSecureStorage.setMockInitialValues({});
  });

  const routes = <String, String>{
    'Catalog': '/catalog',
    'Product detail': '/product/PROD-MX-GION-500',
    'Cart': '/cart',
    'Checkout': '/checkout',
    'Confirmation': '/order-confirmation/OM-TEST',
    'Orders': '/orders',
    'Order detail': '/orders/OM-8921',
    'Account': '/account',
    'Addresses': '/addresses',
    'Address form': '/addresses/form',
    'Notifications': '/notifications',
    'Login': '/login',
    'Register': '/register',
  };

  for (final entry in routes.entries) {
    testWidgets('${entry.key} has no initial overflow at 390x844', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      await tester.pumpWidget(OmaApp(initialLocation: entry.value));
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
    });
  }
}
