import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/app/app_dependencies.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/account/data/rest_customer_account_repository.dart';
import 'package:mobile_app/features/account/presentation/profile_cubit.dart';

void main() {
  test(
    'profile uses only writable ZIP fields and loyalty retains server values',
    () async {
      final requests = <RequestOptions>[];
      final dio = Dio(BaseOptions(baseUrl: 'https://example.test'));
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests.add(options);
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: options.path.endsWith('/points')
                    ? {
                        'customer_id': 'C1',
                        'current_points': 321,
                        'tier': 'SILVER',
                        'points_to_vnd_rate': 500,
                      }
                    : {
                        'customer_id': 'C1',
                        'phone_number': '0905123456',
                        'full_name': 'An Nguyễn',
                        'email': 'an@example.com',
                        'avatar_url': '',
                        'membership_tier': 'SILVER',
                        'loyalty_points': 321,
                      },
              ),
            );
          },
        ),
      );
      final repository = RestCustomerAccountRepository(ApiClient(dio: dio));
      final profile = await repository.updateProfile(
        'An Nguyễn',
        'an@example.com',
      );
      expect(requests.last.method, 'PUT');
      expect(requests.last.path, '/api/v1/profile');
      expect(requests.last.data, {
        'full_name': 'An Nguyễn',
        'email': 'an@example.com',
      });
      expect(profile.user.phoneNumber, '0905123456');
      final loyalty = await repository.loyalty();
      expect(loyalty.points, 321);
      expect(loyalty.vndRate, 500);
      expect(loyalty.tierLabel, 'Bạc');
    },
  );

  test('mock account follows actual login, saved profile and logout', () async {
    final dependencies = RepositoryFactory.create(mode: DataSourceMode.mock);
    final account = dependencies.customerAccountRepository;
    await expectLater(account.profile(), throwsException);
    await dependencies.authRepository.login(
      phoneNumber: '0905123456',
      password: 'local-fixture',
    );
    final cubit = ProfileCubit(account);
    await cubit.load();
    await cubit.save('', 'bad');
    expect(cubit.state.errors.keys, containsAll(['full_name', 'email']));
    await cubit.save('Tên mới', 'new@example.com');
    expect(cubit.state.saved, isTrue);
    expect((await account.profile()).user.fullName, 'Tên mới');
    await dependencies.authRepository.logout();
    await expectLater(account.loyalty(), throwsException);
    await cubit.close();
  });
}
