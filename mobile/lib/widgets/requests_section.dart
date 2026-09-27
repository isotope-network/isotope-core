// mobile/lib/widgets/requests_section.dart
import 'package:flutter/material.dart';
import '../services/libp2p_service.dart';
import '../services/log_service.dart';

/// Виджет «Запросы» — секция над списком контактов.
/// Показывает входящие запросы на добавление в контакты.
/// Пусто — виджет не рисуется (SizedBox.shrink).
/// Не пусто — заголовок «ЗАПРОСЫ (N)» + список запросов.
class RequestsSection extends StatefulWidget {
  final VoidCallback? onAccepted;
  final VoidCallback? onRejected;

  const RequestsSection({
    super.key,
    this.onAccepted,
    this.onRejected,
  });

  @override
  State<RequestsSection> createState() => RequestsSectionState();
}

class RequestsSectionState extends State<RequestsSection> {
  List<Map<String, dynamic>> _requests = [];
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    loadRequests();
  }

  /// Загружает запросы из ядра.
  /// Публичный — чтобы родитель мог вызвать при обновлении.
  Future<void> loadRequests() async {
    if (_loading) return;
    _loading = true;
    try {
      final raw = await LibP2PService.getRequests();
      final list = raw
          .whereType<Map>()
          .map((m) => Map<String, dynamic>.from(m))
          .toList();
      if (mounted) {
        setState(() => _requests = list);
      }
      LogService.log('RequestsSection: загружено ${list.length} запросов');
    } catch (e) {
      LogService.log('RequestsSection: ошибка загрузки: $e');
    } finally {
      _loading = false;
    }
  }

  /// Показать bottom sheet с деталями запроса.
  Future<void> _openRequest(Map<String, dynamic> req) async {
    final result = await showModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      builder: (_) => _RequestDetailsSheet(request: req),
    );
    if (result == 'accepted' || result == 'rejected') {
      await loadRequests();
      if (result == 'accepted') {
        widget.onAccepted?.call();
      } else {
        widget.onRejected?.call();
      }
    }
  }

  String _shortPeerID(String peerID) {
    if (peerID.length <= 12) return peerID;
    return '${peerID.substring(0, 12)}…';
  }

  @override
  Widget build(BuildContext context) {
    if (_requests.isEmpty) return const SizedBox.shrink();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(4, 8, 4, 8),
          child: Row(
            children: [
              Icon(Icons.person_add_alt_1, size: 18, color: Colors.orange.shade700),
              const SizedBox(width: 6),
              Text(
                'ЗАПРОСЫ (${_requests.length})',
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.bold,
                  color: Colors.orange.shade800,
                  letterSpacing: 0.5,
                ),
              ),
            ],
          ),
        ),
        ..._requests.map((req) {
          final peerID = req['peerID'] as String? ?? '';
          final name = req['name'] as String? ?? '';
          final displayName = name.isNotEmpty ? name : _shortPeerID(peerID);
          return Container(
            margin: const EdgeInsets.only(bottom: 4),
            decoration: BoxDecoration(
              color: Colors.orange.shade50,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.orange.shade200),
            ),
            child: ListTile(
              leading: Icon(Icons.person_add, color: Colors.orange.shade700),
              title: Text(
                displayName,
                style: const TextStyle(fontWeight: FontWeight.w600),
              ),
              subtitle: Text(
                'Хочет добавить вас в контакты',
                style: TextStyle(fontSize: 12, color: Colors.grey.shade700),
              ),
              trailing: const Icon(Icons.chevron_right, color: Colors.grey),
              onTap: () => _openRequest(req),
            ),
          );
        }),
      ],
    );
  }
}

// ============================================================
// Bottom sheet — детали запроса.
// ============================================================

class _RequestDetailsSheet extends StatefulWidget {
  final Map<String, dynamic> request;

  const _RequestDetailsSheet({required this.request});

  @override
  State<_RequestDetailsSheet> createState() => _RequestDetailsSheetState();
}

class _RequestDetailsSheetState extends State<_RequestDetailsSheet> {
  bool _processing = false;

  String _shortPeerID(String peerID) {
    if (peerID.length <= 16) return peerID;
    return '${peerID.substring(0, 16)}…';
  }

  Future<void> _accept() async {
    final id = widget.request['id'] as String? ?? '';
    if (id.isEmpty) return;
    setState(() => _processing = true);
    try {
      final result = await LibP2PService.acceptRequestByID(id);
      if (result.containsKey('error')) {
        LogService.log('RequestsSection: accept failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Не удалось принять: ${result['error']}')),
          );
        }
        setState(() => _processing = false);
        return;
      }
      LogService.log('RequestsSection: принят $id');
      if (mounted) Navigator.pop(context, 'accepted');
    } catch (e) {
      LogService.log('RequestsSection: accept exception: $e');
      setState(() => _processing = false);
    }
  }

  Future<void> _reject() async {
    final id = widget.request['id'] as String? ?? '';
    if (id.isEmpty) return;
    setState(() => _processing = true);
    try {
      final result = await LibP2PService.rejectRequestByID(id);
      if (result.containsKey('error')) {
        LogService.log('RequestsSection: reject failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Не удалось отклонить: ${result['error']}')),
          );
        }
        setState(() => _processing = false);
        return;
      }
      LogService.log('RequestsSection: отклонён $id');
      if (mounted) Navigator.pop(context, 'rejected');
    } catch (e) {
      LogService.log('RequestsSection: reject exception: $e');
      setState(() => _processing = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final peerID = widget.request['peerID'] as String? ?? '';
    final name = widget.request['name'] as String? ?? '';
    final signature = widget.request['signature'] as String? ?? '';

    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Icon(Icons.person_add_alt_1, color: Colors.orange.shade700, size: 28),
                const SizedBox(width: 10),
                const Expanded(
                  child: Text(
                    'Запрос на контакт',
                    style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 20),
            Text(
              name.isNotEmpty ? name : 'Без имени',
              style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 4),
            Text(
              _shortPeerID(peerID),
              style: TextStyle(
                fontSize: 12,
                fontFamily: 'monospace',
                color: Colors.grey.shade600,
              ),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Icon(
                  signature.isNotEmpty ? Icons.verified : Icons.warning_amber_rounded,
                  size: 16,
                  color: signature.isNotEmpty ? Colors.green : Colors.orange,
                ),
                const SizedBox(width: 6),
                Text(
                  signature.isNotEmpty
                      ? 'Подпись проверена'
                      : 'Подпись отсутствует',
                  style: TextStyle(
                    fontSize: 12,
                    color: signature.isNotEmpty ? Colors.green : Colors.orange,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 24),
            FilledButton.icon(
              onPressed: _processing ? null : _accept,
              icon: _processing
                  ? const SizedBox(
                      width: 16, height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                    )
                  : const Icon(Icons.check),
              label: const Text('Принять'),
              style: FilledButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
            ),
            const SizedBox(height: 10),
            OutlinedButton.icon(
              onPressed: _processing ? null : _reject,
              icon: const Icon(Icons.close),
              label: const Text('Отклонить'),
              style: OutlinedButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
// mobile/lib/widgets/requests_section.dart