import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/support/data/rest_support_repository.dart';
import 'package:mobile_app/features/support/domain/support_repository.dart';
import 'package:mobile_app/features/support/presentation/support_cubit.dart';

class _Repository extends Mock implements SupportRepository {}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  test(
    'support REST uses exact stable fields and retry header, no contact/files',
    () async {
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
                data: options.path.endsWith('/messages')
                    ? {
                        'messageId': 'm1',
                        'senderRole': 'CUSTOMER',
                        'senderDisplayName': 'Khách',
                        'message': 'Thông tin bổ sung',
                        'occurredAt': '2026-10-19T00:00:00Z',
                        'attachments': [],
                      }
                    : {'ticketId': 'ticket/id'},
              ),
            );
          },
        ),
      );
      final repository = RestSupportRepository(ApiClient(dio: dio));
      expect(
        await repository.create(
          subject: 'OTHER',
          title: 'Hỏi về bảo quản',
          message: 'Cần hướng dẫn bảo quản bánh.',
          priority: 'NORMAL',
          orderId: '',
          idempotencyKey: 'create-key',
        ),
        'ticket/id',
      );
      expect(requests.single.data, {
        'subjectCode': 'OTHER',
        'title': 'Hỏi về bảo quản',
        'message': 'Cần hướng dẫn bảo quản bánh.',
        'requestedPriority': 'NORMAL',
        'linkedOrderId': '',
        'attachments': [],
        'idempotencyKey': 'create-key',
      });
      expect(requests.single.headers['Idempotency-Key'], 'create-key');
      final message = await repository.reply(
        'ticket/id',
        'Thông tin bổ sung',
        idempotencyKey: 'reply-key',
      );
      expect(message.id, 'm1');
      expect(
        requests.last.path,
        '/api/v1/support/tickets/ticket%2Fid/messages',
      );
      expect(requests.last.data, {
        'message': 'Thông tin bổ sung',
        'attachments': [],
        'idempotencyKey': 'reply-key',
      });
    },
  );

  test(
    'support retries preserve idempotency, edits renew it; invalid linkage never submits',
    () async {
      final source = RepositoryFactory.create(
        mode: DataSourceMode.mock,
      ).supportRepository;
      final ctx = await source.context();
      final repository = _Repository();
      when(repository.context).thenAnswer((_) async => ctx);
      final keys = <String>[];
      when(
        () => repository.create(
          subject: any(named: 'subject'),
          title: any(named: 'title'),
          message: any(named: 'message'),
          priority: any(named: 'priority'),
          orderId: any(named: 'orderId'),
          idempotencyKey: any(named: 'idempotencyKey'),
        ),
      ).thenAnswer((invocation) async {
        keys.add(invocation.namedArguments[#idempotencyKey] as String);
        throw StateError('offline');
      });
      final cubit = SupportCubit(repository);
      await cubit.loadContext();
      Future<void> send(String title, {String order = ''}) => cubit.create(
        subject: 'OTHER',
        title: title,
        message: 'Cần hướng dẫn bảo quản bánh.',
        priority: 'NORMAL',
        orderId: order,
      );
      await send('Hỏi bảo quản', order: 'someone-else');
      expect(keys, isEmpty);
      await send('Hỏi bảo quản');
      await send('Hỏi bảo quản');
      await send('Hỏi thưởng trà');
      expect(keys[0], keys[1]);
      expect(keys[2], isNot(keys[1]));
      expect(cubit.state.context, same(ctx));
      expect(cubit.state.pending, isFalse);
      await cubit.close();
    },
  );

  test('local ticket/reply are idempotent and ownership-scoped', () async {
    final dependencies = RepositoryFactory.create(mode: DataSourceMode.mock);
    final repository = dependencies.supportRepository;
    Future<String> create() => repository.create(
      subject: 'OTHER',
      title: 'Cách bảo quản',
      message: 'Xin hướng dẫn bảo quản bánh.',
      priority: 'NORMAL',
      orderId: '',
      idempotencyKey: 'one',
    );
    final id = await create();
    expect(await create(), id);
    final first = await repository.reply(
      id,
      'Tôi cần bảo quản 2 ngày.',
      idempotencyKey: 'reply',
    );
    expect(
      (await repository.reply(
        id,
        'Tôi cần bảo quản 2 ngày.',
        idempotencyKey: 'reply',
      )).id,
      first.id,
    );
    expect((await repository.ticket(id)).messages.length, 2);
    await expectLater(
      repository.ticket('unknown'),
      throwsA(isA<ApiException>()),
    );
    await dependencies.authRepository.login(
      phoneNumber: '0905123456',
      password: 'local-fixture',
    );
    await expectLater(repository.ticket(id), throwsA(isA<ApiException>()));
  });
}
