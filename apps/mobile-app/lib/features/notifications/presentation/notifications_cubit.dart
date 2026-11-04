import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:mobile_app/features/notifications/domain/app_notification.dart';

class NotificationsState extends Equatable {
  const NotificationsState({this.loading = true, this.items = const []});
  final bool loading;
  final List<AppNotification> items;
  @override
  List<Object?> get props => [loading, items];
}

class NotificationsCubit extends Cubit<NotificationsState> {
  NotificationsCubit(this._repository) : super(const NotificationsState());
  final NotificationsRepository _repository;
  Future<void> load() async => emit(
    NotificationsState(
      loading: false,
      items: await _repository.notifications(),
    ),
  );
}
