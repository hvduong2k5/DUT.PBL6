import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/core/storage/search_history_store.dart';

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));

  test('search history keeps the 5 most recent unique queries', () async {
    final store = SearchHistoryStore();
    for (final q in ['mè', 'trà', 'mứt', 'quà', 'bánh', 'sen']) {
      await store.add(q);
    }
    await store.add('TRÀ');
    expect(await store.read(), ['TRÀ', 'sen', 'bánh', 'quà', 'mứt']);
  });

  testWidgets('home search bar opens a separate results screen', (
    tester,
  ) async {
    await tester.pumpWidget(const OmaApp());
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const Key('search-bar')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('search-history')), findsOneWidget);
    expect(find.byKey(const Key('home-content-scroll')), findsNothing);

    await tester.enterText(find.byKey(const Key('search-input')), 'mè xửng');
    await tester.testTextInput.receiveAction(TextInputAction.search);
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('search-results')), findsOneWidget);

    // Searching again from the same screen replaces the results.
    await tester.enterText(find.byKey(const Key('search-input')), 'zzzz');
    await tester.testTextInput.receiveAction(TextInputAction.search);
    await tester.pumpAndSettle();
    expect(find.text('Không tìm thấy sản phẩm'), findsOneWidget);

    expect(await SearchHistoryStore().read(), ['zzzz', 'mè xửng']);
  });
}
