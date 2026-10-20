import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/app/app.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/widgets/public_https_button.dart';
import 'package:mobile_app/features/trace/data/trace_repositories.dart';
import 'package:mobile_app/features/trace/domain/trace_repository.dart';
import 'package:mobile_app/features/trace/presentation/trace_cubit.dart';

class _Trace extends Mock implements TraceRepository {}

void main() {
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test('public media links refuse executable, local and credential URLs', () {
    for (final url in [
      'javascript:alert(1)',
      'file:///data/test',
      'http://video.example.com/a',
      'https://user:password@example.com/a',
      'https://localhost/a',
    ]) {
      expect(publicHttpsUri(url), isNull);
    }
    expect(
      publicHttpsUri('https://video.example.com/crafting.mp4')?.host,
      'video.example.com',
    );
  });
  test(
    'trace REST escapes code and parses complete stable projection',
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
                  'qr_code': 'QR/08',
                  'product_name': 'Mè xửng',
                  'batch_code': 'LOT08',
                  'ocop_star': 4,
                  'ocop_certificate_no': 'CERT',
                  'production_date': '2026-10-15',
                  'expiry_date': '2027-04-15',
                  'workshop_location': 'Huế',
                  'artisan_name': 'Nghệ nhân',
                  'food_safety_cert': 'ATTP',
                  'raw_materials_origin': 'Mè',
                  'crafting_video_url': 'https://video.example.com/a.mp4',
                },
              ),
            );
          },
        ),
      );
      final record = await RestTraceRepository(
        ApiClient(dio: dio),
      ).lookup('QR/08');
      expect(request!.path, '/api/v1/trace/QR%2F08');
      expect(record.batch, 'LOT08');
      expect(record.expiryDate, '2027-04-15');
      expect(record.origin, 'Mè');
    },
  );
  test('stale trace response never replaces a new lookup failure', () async {
    final repository = _Trace(), pending = Completer<TraceRecord>();
    when(() => repository.lookup('old')).thenAnswer((_) => pending.future);
    when(() => repository.lookup('new')).thenThrow(StateError('offline'));
    final cubit = TraceCubit(repository);
    final old = cubit.lookup('old');
    await cubit.lookup('new');
    pending.complete(
      await MockTraceRepository().lookup('QR-OMA-20261015-LOT08'),
    );
    await old;
    expect(cubit.state.code, 'new');
    expect(cubit.state.record, isNull);
    expect(cubit.state.error, isNotNull);
    await cubit.close();
  });
  testWidgets(
    'trace lookup recovers from unknown code and shows real record fields',
    (tester) async {
      await tester.pumpWidget(const OmaApp(initialLocation: '/trace'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'UNKNOWN');
      await tester.tap(find.text('Truy xuất ngay'));
      await tester.pumpAndSettle();
      expect(
        find.textContaining('Không tìm thấy mã truy xuất'),
        findsOneWidget,
      );
      await tester.tap(find.text('Xem hồ sơ mẫu local'));
      await tester.pumpAndSettle();
      expect(find.textContaining('Không tìm thấy mã truy xuất'), findsNothing);
      await tester.scrollUntilVisible(
        find.text('LOT-20261015-G08'),
        250,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.text('LOT-20261015-G08'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
