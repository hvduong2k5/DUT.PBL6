import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';

void main() {
  testWidgets('renders Ô Mạ foundation shell', (tester) async {
    await tester.pumpWidget(const OmaApp());
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.text('Ô Mạ'), findsOneWidget);
    expect(find.text('Trang Chủ'), findsOneWidget);
    expect(find.text('Thực Đơn'), findsOneWidget);
  });
}
