enum AppEnvironment { mock, development, production }

abstract final class EnvironmentConfig {
  static const _name = String.fromEnvironment('APP_ENV', defaultValue: 'mock');

  static AppEnvironment get current => switch (_name) {
    'production' => AppEnvironment.production,
    'development' => AppEnvironment.development,
    _ => AppEnvironment.mock,
  };
}
