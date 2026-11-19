enum DataSourceMode {
  mock,
  mockoon,
  backend;

  static DataSourceMode parse(String? value) => switch (value?.toLowerCase()) {
    'mockoon' => DataSourceMode.mockoon,
    'backend' => DataSourceMode.backend,
    _ => DataSourceMode.mock,
  };

  bool get isRemote => this != DataSourceMode.mock;
}
