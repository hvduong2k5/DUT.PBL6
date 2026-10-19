import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/app_mobile_header.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/features/support/presentation/support_cubit.dart';

class SupportRequestPage extends StatefulWidget {
  const SupportRequestPage({super.key});
  @override
  State<SupportRequestPage> createState() => _SupportRequestPageState();
}

class _SupportRequestPageState extends State<SupportRequestPage> {
  final _title = TextEditingController(), _message = TextEditingController();
  String? _subject, _priority;
  String _order = '';
  @override
  void dispose() {
    _title.dispose();
    _message.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Gửi yêu cầu hỗ trợ', showBack: true),
    body: SafeArea(
      child: BlocConsumer<SupportCubit, SupportState>(
        listener: (context, state) {
          if (state.createdId != null) {
            context.go(
              '/support/tickets/${Uri.encodeComponent(state.createdId!)}',
            );
          }
        },
        builder: (context, state) {
          final cubit = context.read<SupportCubit>(), data = state.context;
          if (state.loading) return const AppLoadingView();
          if (data == null) {
            return AppErrorView(
              message: state.error ?? 'Chưa thể mở trung tâm hỗ trợ.',
              onRetry: cubit.loadContext,
            );
          }
          final subject =
              data.subjects.where((o) => o.code == _subject).firstOrNull ??
              data.subjects.firstOrNull;
          final priority =
              data.priorities.where((o) => o.code == _priority).firstOrNull ??
              data.priorities.firstOrNull;
          return ListView(
            padding: const EdgeInsets.all(20),
            children: [
              Text(
                'Trao đổi cùng Ban Tri Kỷ',
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 12),
              Text(data.servicePromise),
              _SupportSection(
                title: '1 · Thông tin người gửi',
                children: [
                  Text(
                    data.guest
                        ? 'Mua sắm với tư cách khách'
                        : data.customerName,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  if (data.customerContact.isNotEmpty)
                    Text(data.customerContact),
                  if (data.guest)
                    const Text(
                      'Theo dõi trao đổi trong phiên hiện tại. Hãy giữ mã ticket để mở lại khi phiên còn hiệu lực.',
                    ),
                ],
              ),
              _SupportSection(
                title: '2 · Chủ đề và đơn liên quan',
                children: [
                  DropdownButtonFormField<String>(
                    initialValue: subject?.code,
                    isExpanded: true,
                    decoration: const InputDecoration(
                      labelText: 'Chủ đề cần hỗ trợ',
                    ),
                    items: [
                      for (final option in data.subjects)
                        DropdownMenuItem(
                          value: option.code,
                          child: Text(
                            option.label,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                    ],
                    onChanged: state.pending
                        ? null
                        : (value) => setState(() => _subject = value),
                  ),
                  if (subject != null)
                    Padding(
                      padding: const EdgeInsets.only(top: 8),
                      child: Text(subject.description),
                    ),
                  const SizedBox(height: 16),
                  DropdownButtonFormField<String>(
                    initialValue: _order,
                    isExpanded: true,
                    decoration: const InputDecoration(
                      labelText: 'Đơn hàng liên quan',
                    ),
                    items: [
                      const DropdownMenuItem(
                        value: '',
                        child: Text('Không liên kết đơn'),
                      ),
                      for (final order in data.orders)
                        DropdownMenuItem(
                          value: order.id,
                          child: Text(
                            '#${order.number} · ${order.status}',
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                    ],
                    onChanged: state.pending
                        ? null
                        : (value) => setState(() => _order = value ?? ''),
                  ),
                  if (data.orders.isEmpty)
                    const Padding(
                      padding: EdgeInsets.only(top: 8),
                      child: Text(
                        'Chỉ đơn thuộc quyền truy cập mới có thể liên kết.',
                      ),
                    ),
                ],
              ),
              _SupportSection(
                title: '3 · Ưu tiên mong muốn',
                children: [
                  DropdownButtonFormField<String>(
                    initialValue: priority?.code,
                    isExpanded: true,
                    decoration: const InputDecoration(
                      labelText: 'Mức độ ưu tiên',
                    ),
                    items: [
                      for (final option in data.priorities)
                        DropdownMenuItem(
                          value: option.code,
                          child: Text(
                            option.label,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                    ],
                    onChanged: state.pending
                        ? null
                        : (value) => setState(() => _priority = value),
                  ),
                  if (priority != null)
                    Padding(
                      padding: const EdgeInsets.only(top: 8),
                      child: Text(
                        '${priority.description}\n${priority.promise}',
                      ),
                    ),
                  const Text(
                    'CSKH xác định ưu tiên và thời hạn xử lý chính thức.',
                  ),
                ],
              ),
              _SupportSection(
                title: '4 · Nội dung yêu cầu',
                children: [
                  AppTextField(
                    controller: _title,
                    label: 'Tiêu đề yêu cầu',
                    enabled: !state.pending,
                    maxLength: 160,
                    textInputAction: TextInputAction.next,
                  ),
                  const SizedBox(height: 16),
                  AppTextField(
                    controller: _message,
                    label: 'Nội dung cần hỗ trợ',
                    maxLines: 6,
                    maxLength: 3000,
                    enabled: !state.pending,
                  ),
                  const SizedBox(height: 12),
                  const Text(
                    'Tệp đính kèm: chưa hỗ trợ tải tệp trong phiên bản này.',
                    style: TextStyle(color: AppColors.textMuted, fontSize: 12),
                  ),
                ],
              ),
              if (state.error != null) _SupportError(state.error!),
              AppPrimaryButton(
                label: 'Gửi yêu cầu đến Ban Tri Kỷ',
                loading: state.pending,
                onPressed: subject == null || priority == null
                    ? null
                    : () => cubit.create(
                        subject: subject.code,
                        title: _title.text,
                        message: _message.text,
                        priority: priority.code,
                        orderId: _order,
                      ),
              ),
              TextButton(
                onPressed: state.pending
                    ? null
                    : () => context.go(AppRoutes.home),
                child: const Text('Hủy bỏ & về trang chủ'),
              ),
            ],
          );
        },
      ),
    ),
  );
}

class SupportTicketPage extends StatefulWidget {
  const SupportTicketPage({super.key, required this.ticketId});
  final String ticketId;
  @override
  State<SupportTicketPage> createState() => _SupportTicketPageState();
}

class _SupportTicketPageState extends State<SupportTicketPage> {
  final _message = TextEditingController();
  @override
  void dispose() {
    _message.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: const AppMobileHeader(title: 'Trao đổi hỗ trợ', showBack: true),
    body: SafeArea(
      child: BlocBuilder<SupportCubit, SupportState>(
        builder: (context, state) {
          final cubit = context.read<SupportCubit>(), ticket = state.ticket;
          if (state.loading) return const AppLoadingView();
          if (ticket == null) {
            return AppErrorView(
              message: state.error ?? 'Chưa thể mở ticket.',
              onRetry: () => cubit.loadTicket(widget.ticketId),
              action: TextButton(
                onPressed: () => context.go(AppRoutes.supportRequest),
                child: const Text('Gửi yêu cầu mới'),
              ),
            );
          }
          return ListView(
            padding: const EdgeInsets.all(20),
            children: [
              SelectableText(
                '#${ticket.number}',
                style: const TextStyle(color: AppColors.textMuted),
              ),
              const SizedBox(height: 8),
              Text(
                ticket.title,
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 12),
              Align(
                alignment: Alignment.centerLeft,
                child: Chip(label: Text(ticket.status)),
              ),
              Text('${ticket.subject} · ${ticket.priority}'),
              _SupportSection(
                title: 'Thông tin tiếp nhận',
                children: [
                  Text(ticket.promise),
                  Text(ticket.assignee),
                  Text(ticket.description),
                  if (ticket.order != null)
                    TextButton(
                      onPressed: () => context.push(
                        '/orders/${Uri.encodeComponent(ticket.order!.id)}',
                      ),
                      child: Text('Xem đơn #${ticket.order!.number}'),
                    ),
                ],
              ),
              Text(
                'Nội dung trao đổi',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              if (ticket.messages.isEmpty)
                const Padding(
                  padding: EdgeInsets.all(16),
                  child: Text('Chưa có trao đổi công khai.'),
                ),
              for (final message in ticket.messages)
                Card(
                  color: message.role == 'CUSTOMER' ? AppColors.cream : null,
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          message.sender,
                          style: const TextStyle(fontWeight: FontWeight.w600),
                        ),
                        Text(
                          DateFormat(
                            'dd/MM/yyyy HH:mm',
                          ).format(message.time.toLocal()),
                          style: const TextStyle(
                            fontSize: 11,
                            color: AppColors.textMuted,
                          ),
                        ),
                        const SizedBox(height: 10),
                        SelectableText(message.message),
                        for (final file in message.attachments)
                          Padding(
                            padding: const EdgeInsets.only(top: 8),
                            child: Text(
                              file,
                              style: const TextStyle(fontSize: 12),
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
              _SupportSection(
                title: 'Phản hồi thêm',
                children: [
                  if (ticket.canReply) ...[
                    AppTextField(
                      controller: _message,
                      label: 'Lời nhắn gửi Ban Tri Kỷ',
                      maxLines: 5,
                      maxLength: 3000,
                      enabled: !state.pending,
                    ),
                    const SizedBox(height: 16),
                    if (state.error != null) _SupportError(state.error!),
                    AppPrimaryButton(
                      label: 'Gửi phản hồi',
                      loading: state.pending,
                      onPressed: () async {
                        final sent = await cubit.reply(_message.text);
                        if (sent && context.mounted) {
                          _message.clear();
                          FocusScope.of(context).unfocus();
                        }
                      },
                    ),
                  ] else
                    const Text(
                      'Ticket đã đóng trao đổi. Hãy tạo yêu cầu mới nếu cần hỗ trợ thêm.',
                    ),
                ],
              ),
              TextButton(
                onPressed: () => context.go(AppRoutes.supportRequest),
                child: const Text('Gửi yêu cầu khác'),
              ),
            ],
          );
        },
      ),
    ),
  );
}

class _SupportSection extends StatelessWidget {
  const _SupportSection({required this.title, required this.children});
  final String title;
  final List<Widget> children;
  @override
  Widget build(BuildContext context) => Card(
    margin: const EdgeInsets.symmetric(vertical: 12),
    child: Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 16),
          ...children,
        ],
      ),
    ),
  );
}

class _SupportError extends StatelessWidget {
  const _SupportError(this.message);
  final String message;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 12),
    child: Semantics(
      liveRegion: true,
      child: Text(message, style: const TextStyle(color: AppColors.darkRed)),
    ),
  );
}
