import 'package:flutter/foundation.dart';
import 'package:mobile_app/core/config/app_environment.dart';
import 'package:mobile_app/core/config/data_source_mode.dart';

abstract final class AppConfig {
  static const _dataSource = String.fromEnvironment(
    'DATA_SOURCE',
    defaultValue: 'mock',
  );
  static const _apiBaseUrl = String.fromEnvironment('API_BASE_URL');

  static DataSourceMode get dataSourceMode => DataSourceMode.parse(_dataSource);
  static AppEnvironment get appEnvironment => EnvironmentConfig.current;
  static String get apiBaseUrl => resolveApiBaseUrl(
    mode: dataSourceMode,
    environment: appEnvironment,
    explicitUrl: _apiBaseUrl,
    isWeb: kIsWeb,
    platform: defaultTargetPlatform,
  );

  @visibleForTesting
  static String resolveApiBaseUrl({
    required DataSourceMode mode,
    required AppEnvironment environment,
    String explicitUrl = '',
    bool isWeb = false,
    TargetPlatform platform = TargetPlatform.android,
  }) {
    final override = explicitUrl.trim();
    if (override.isNotEmpty) return override;
    if (mode == DataSourceMode.mock) return '';

    final host = isWeb || platform != TargetPlatform.android
        ? 'localhost'
        : '10.0.2.2';
    if (mode == DataSourceMode.mockoon) return 'http://$host:4010';

    return switch (environment) {
      AppEnvironment.production => 'https://api.omama.vn',
      AppEnvironment.development => 'http://$host:8000',
      AppEnvironment.mock => 'http://$host:8000',
    };
  }
}
