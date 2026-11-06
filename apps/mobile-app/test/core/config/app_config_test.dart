import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/config/app_config.dart';
import 'package:mobile_app/core/config/app_environment.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';

void main() {
  group('DataSourceMode.parse', () {
    test('missing and invalid values safely use mock', () {
      expect(DataSourceMode.parse(null), DataSourceMode.mock);
      expect(DataSourceMode.parse(''), DataSourceMode.mock);
      expect(DataSourceMode.parse('unexpected'), DataSourceMode.mock);
    });

    test('parses every supported value', () {
      expect(DataSourceMode.parse('mock'), DataSourceMode.mock);
      expect(DataSourceMode.parse('mockoon'), DataSourceMode.mockoon);
      expect(DataSourceMode.parse('backend'), DataSourceMode.backend);
    });
  });

  group('AppConfig URL resolution', () {
    test('explicit URL always has highest priority', () {
      expect(
        AppConfig.resolveApiBaseUrl(
          mode: DataSourceMode.mockoon,
          environment: AppEnvironment.mock,
          explicitUrl: 'http://192.168.1.20:4010',
        ),
        'http://192.168.1.20:4010',
      );
    });

    test('local mock needs no remote URL', () {
      expect(
        AppConfig.resolveApiBaseUrl(
          mode: DataSourceMode.mock,
          environment: AppEnvironment.mock,
        ),
        isEmpty,
      );
    });

    test('Mockoon is platform aware', () {
      expect(
        AppConfig.resolveApiBaseUrl(
          mode: DataSourceMode.mockoon,
          environment: AppEnvironment.mock,
          platform: TargetPlatform.android,
        ),
        'http://10.0.2.2:4010',
      );
      expect(
        AppConfig.resolveApiBaseUrl(
          mode: DataSourceMode.mockoon,
          environment: AppEnvironment.mock,
          isWeb: true,
        ),
        'http://localhost:4010',
      );
    });

    test('backend derives only from the selected environment', () {
      expect(
        AppConfig.resolveApiBaseUrl(
          mode: DataSourceMode.backend,
          environment: AppEnvironment.development,
          platform: TargetPlatform.android,
        ),
        'http://10.0.2.2:8000',
      );
      expect(
        AppConfig.resolveApiBaseUrl(
          mode: DataSourceMode.backend,
          environment: AppEnvironment.production,
        ),
        'https://api.omama.vn',
      );
    });
  });
}
