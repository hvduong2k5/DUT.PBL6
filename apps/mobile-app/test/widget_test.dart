import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';

void main() {
  testWidgets('renders Gói Huế foundation shell', (tester) async {
    await tester.pumpWidget(const OmaApp());
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.text('Gói Huế'), findsOneWidget);
    expect(find.text('Trang chủ'), findsOneWidget);
    expect(find.text('Danh mục'), findsOneWidget);
  });
}
