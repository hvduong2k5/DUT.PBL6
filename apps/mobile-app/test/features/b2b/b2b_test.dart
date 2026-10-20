import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';
import 'package:mobile_app/features/b2b/data/rest_b2b_repository.dart';
import 'package:mobile_app/features/b2b/domain/b2b_repository.dart';
import 'package:mobile_app/features/b2b/presentation/b2b_cubits.dart';

class _Repository extends Mock implements B2bRepository {}

Future<AppDependencies> verified() async {
  final dependencies = RepositoryFactory.create(mode: DataSourceMode.mock);
  await dependencies.authRepository.login(
    phoneNumber: MockAuthRepository.b2bFixturePhone,
    password: 'legacy',
  );
  return dependencies;
}

B2bQuote changed(
  B2bQuote q, {
  int? version,
  String? status,
  bool? canAccept,
  DateTime? expires,
}) => B2bQuote(
  id: q.id,
  number: q.number,
  version: version ?? q.version,
  status: status ?? q.status,
  label: q.label,
  issued: q.issued,
  expires: expires ?? q.expires,
  organization: q.organization,
  contact: q.contact,
  items: q.items,
  customizations: q.customizations,
  totals: q.totals,
  terms: q.terms,
  payment: q.payment,
  canAccept: canAccept ?? q.canAccept,
);
B2bRequestDraft requestDraft(B2bContext context, {int quantity = 20}) =>
    B2bRequestDraft(
      organizationId: context.organization.id,
      purpose: 'VIP_CUSTOMER_GIFT',
      date: DateTime.now()
          .add(const Duration(days: 14))
          .toIso8601String()
          .substring(0, 10),
      provinceCode: context.provinces.first.code,
      provinceName: context.provinces.first.name,
      quantities: {context.products.first.sku: quantity},
      branding: {for (final key in b2bBranding.keys) key: false},
      invoice: true,
      sample: false,
    );
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  setUpAll(
    () => registerFallbackValue(
      const B2bRequestDraft(
        organizationId: '',
        purpose: '',
        date: '',
        provinceCode: '',
        provinceName: '',
        quantities: {},
        branding: {},
        invoice: false,
        sample: false,
      ),
    ),
  );
  test('B2B request and acceptance send only stable ZIP body/header', () async {
    final requests = <RequestOptions>[];
    final dio = Dio(BaseOptions(baseUrl: 'http://localhost'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requests.add(options);
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 201,
              data: options.path.endsWith('/accept')
                  ? {
                      'orderId': 'O1',
                      'orderNumber': 'B2B-1',
                      'message': 'Chờ đặt cọc',
                    }
                  : {
                      'requestId': 'R1',
                      'requestNumber': 'YC1',
                      'message': 'Đã tiếp nhận',
                    },
            ),
          );
        },
      ),
    );
    final repository = RestB2bRepository(ApiClient(dio: dio));
    final context = await (await verified()).b2bRepository.context(),
        draft = requestDraft(context);
    await repository.request(draft, idempotencyKey: 'request-key');
    expect(requests.single.data, {
      'organizationId': 'LOCAL-ORG',
      'purpose': 'VIP_CUSTOMER_GIFT',
      'requestedDeliveryDate': draft.date,
      'deliveryProvinceCode': 'HUE',
      'deliveryProvinceName': 'Thành phố Huế',
      'items': [
        {'skuId': 'LOCAL-GIFT', 'quantity': 20},
      ],
      'branding': {
        'engraveLogo': false,
        'customSleeve': false,
        'greetingCard': false,
        'ribbon': false,
        'attachments': [],
      },
      'invoiceRequested': true,
      'sampleRequested': false,
      'idempotencyKey': 'request-key',
    });
    final accepted = await repository.accept(
      'Q/1',
      idempotencyKey: 'accept-key',
    );
    expect(requests.last.path, '/api/v1/b2b/quotes/Q%2F1/accept');
    expect(requests.last.data, {
      'confirmation': true,
      'idempotencyKey': 'accept-key',
    });
    expect(requests.last.headers['Idempotency-Key'], 'accept-key');
    expect(accepted.orderId, 'O1');
  });
  test(
    'unverified actors are denied; verified mock acceptance creates one unpaid order',
    () async {
      final dependencies = RepositoryFactory.create(mode: DataSourceMode.mock),
          repository = dependencies.b2bRepository;
      await expectLater(repository.context(), throwsA(isA<ApiException>()));
      await dependencies.authRepository.login(
        phoneNumber: '0905123456',
        password: 'legacy',
      );
      await expectLater(
        repository.quote('LOCAL-QUOTE-001'),
        throwsA(isA<ApiException>()),
      );
      await dependencies.authRepository.login(
        phoneNumber: MockAuthRepository.b2bFixturePhone,
        password: 'legacy',
      );
      expect((await dependencies.capabilitiesRepository.load()).actor, 'B2B');
      final first = await repository.accept(
        'LOCAL-QUOTE-001',
        idempotencyKey: 'same',
      );
      expect(
        (await repository.accept(
          'LOCAL-QUOTE-001',
          idempotencyKey: 'same',
        )).orderId,
        first.orderId,
      );
      final order = await dependencies.ordersRepository.order(first.orderId);
      expect(order.rawStatus, 'PENDING_PAYMENT');
      expect(
        (await dependencies.ordersRepository.orders())
            .where((o) => o.id == first.orderId)
            .length,
        1,
      );
      expect((await repository.quote('LOCAL-QUOTE-001')).canAccept, isFalse);
    },
  );
  test(
    'MOQ validation blocks low quantity; failed request keeps retry key and edit renews it',
    () async {
      final context = await (await verified()).b2bRepository.context(),
          repository = _Repository();
      final keys = <String>[];
      when(repository.context).thenAnswer((_) async => context);
      when(
        () => repository.request(
          any(),
          idempotencyKey: any(named: 'idempotencyKey'),
        ),
      ).thenAnswer((i) async {
        keys.add(i.namedArguments[#idempotencyKey] as String);
        throw StateError('offline');
      });
      final cubit = B2bRequestCubit(repository);
      await cubit.load();
      await cubit.submit(requestDraft(context, quantity: 1));
      expect(keys, isEmpty);
      await cubit.submit(requestDraft(context));
      await cubit.submit(requestDraft(context));
      await cubit.submit(requestDraft(context, quantity: 40));
      expect(keys[0], keys[1]);
      expect(keys[2], isNot(keys[1]));
      await cubit.close();
    },
  );
  test(
    'changed quote version clears agreement and never accepts old content',
    () async {
      final original = await (await verified()).b2bRepository.quote(
            'LOCAL-QUOTE-001',
          ),
          repository = _Repository();
      var reads = 0;
      when(() => repository.quote(original.id)).thenAnswer(
        (_) async => ++reads == 1 ? original : changed(original, version: 2),
      );
      final cubit = B2bQuoteCubit(repository, mayAccept: () => true);
      await cubit.load(original.id);
      cubit.agree(true);
      await cubit.accept();
      expect(cubit.state.quote?.version, 2);
      expect(cubit.state.confirmed, isFalse);
      verifyNever(
        () => repository.accept(
          any(),
          idempotencyKey: any(named: 'idempotencyKey'),
        ),
      );
      await cubit.close();
    },
  );
  test(
    'uncertain acceptance replays same key after quote becomes accepted',
    () async {
      final original = await (await verified()).b2bRepository.quote(
            'LOCAL-QUOTE-001',
          ),
          repository = _Repository();
      final keys = <String>[];
      when(() => repository.quote(original.id)).thenAnswer(
        (_) async => keys.isEmpty
            ? original
            : changed(original, status: 'ACCEPTED', canAccept: false),
      );
      when(
        () => repository.accept(
          original.id,
          idempotencyKey: any(named: 'idempotencyKey'),
        ),
      ).thenAnswer((i) async {
        keys.add(i.namedArguments[#idempotencyKey] as String);
        if (keys.length == 1) {
          throw const ApiException(
            Failure(
              kind: FailureKind.network,
              message: 'Mất kết nối sau khi gửi',
            ),
          );
        }
        return const B2bAcceptance(
          orderId: 'O1',
          orderNumber: 'B2B-1',
          message: 'Chờ đặt cọc',
        );
      });
      final cubit = B2bQuoteCubit(repository, mayAccept: () => true);
      await cubit.load(original.id);
      cubit.agree(true);
      await cubit.accept();
      expect(cubit.state.retryingAcceptance, isTrue);
      await cubit.accept();
      expect(keys.length, 2);
      expect(keys[0], keys[1]);
      expect(cubit.state.accepted?.orderId, 'O1');
      await cubit.accept();
      expect(keys.length, 2);
      await cubit.close();
    },
  );
  test(
    'revoking capability during quote precheck prevents financial POST',
    () async {
      final original = await (await verified()).b2bRepository.quote(
            'LOCAL-QUOTE-001',
          ),
          repository = _Repository();
      final pending = Completer<B2bQuote>();
      var reads = 0, allowed = true;
      when(
        () => repository.quote(original.id),
      ).thenAnswer((_) async => ++reads == 1 ? original : pending.future);
      final cubit = B2bQuoteCubit(repository, mayAccept: () => allowed);
      await cubit.load(original.id);
      cubit.agree(true);
      final action = cubit.accept();
      allowed = false;
      pending.complete(original);
      await action;
      verifyNever(
        () => repository.accept(
          any(),
          idempotencyKey: any(named: 'idempotencyKey'),
        ),
      );
      expect(cubit.state.error, contains('Quyền'));
      expect(cubit.state.pending, isFalse);
      await cubit.close();
    },
  );
  test(
    'expired quote and missing acceptance capability never send POST',
    () async {
      final original = await (await verified()).b2bRepository.quote(
            'LOCAL-QUOTE-001',
          ),
          repository = _Repository();
      when(() => repository.quote(original.id)).thenAnswer(
        (_) async => changed(
          original,
          expires: DateTime.now().subtract(const Duration(seconds: 1)),
        ),
      );
      final cubit = B2bQuoteCubit(repository, mayAccept: () => true);
      await cubit.load(original.id);
      cubit.agree(true);
      await cubit.accept();
      expect(cubit.state.error, contains('hết hạn'));
      verifyNever(
        () => repository.accept(
          any(),
          idempotencyKey: any(named: 'idempotencyKey'),
        ),
      );
      await cubit.close();
      final denied = B2bQuoteCubit(repository, mayAccept: () => false);
      await denied.load(original.id);
      denied.agree(true);
      await denied.accept();
      expect(denied.state.error, contains('quyền'));
      await denied.close();
    },
  );
}
