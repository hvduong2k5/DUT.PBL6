import 'package:equatable/equatable.dart';

class AppNotification extends Equatable {
  const AppNotification({
    required this.id,
    required this.title,
    required this.message,
    required this.timeLabel,
    this.unread = false,
  });
  final String id;
  final String title;
  final String message;
  final String timeLabel;
  final bool unread;
  @override
  List<Object?> get props => [id, title, message, timeLabel, unread];
}

abstract interface class NotificationsRepository {
  Future<List<AppNotification>> notifications();
}
