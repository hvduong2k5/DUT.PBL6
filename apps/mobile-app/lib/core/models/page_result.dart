class PageResult<T> {
  PageResult({
    required List<T> items,
    required this.total,
    required this.page,
    required this.pageSize,
  }) : items = List.unmodifiable(items) {
    if (total < 0 || page < 1 || pageSize < 1) {
      throw const FormatException('Invalid pagination metadata');
    }
  }
  final List<T> items;
  final int total, page, pageSize;
  bool get hasMore => page * pageSize < total;
}
