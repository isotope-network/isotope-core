// mobile/lib/widgets/file_offer_bubble.dart
import 'package:flutter/material.dart';
import '../providers/chat_provider.dart';

/// Состояние оффера файла (v2.0.7 — право отказа).
enum FileOfferState { pending, accepted, declined, expired }

/// Плашка «предложение принять файл» (v2.0.7 — право отказа).
///
/// Получатель: state=pending, onAccept/onDecline — кнопки.
/// Отправитель: state=pending/accepted/declined/expired, onAccept/onDecline=null — статус.
class FileOfferBubble extends StatelessWidget {
  final FileOfferData offer;
  final FileOfferState state;
  final VoidCallback? onAccept;
  final VoidCallback? onDecline;

  const FileOfferBubble({
    super.key,
    required this.offer,
    this.state = FileOfferState.pending,
    this.onAccept,
    this.onDecline,
  });

  String _fmtSize(int bytes) {
    if (bytes < 1024) return '$bytes Б';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} КБ';
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} МБ';
  }

  bool get _hasButtons => onAccept != null || onDecline != null;

  @override
  Widget build(BuildContext context) {
    final name = offer.fileName.isNotEmpty ? offer.fileName : 'Файл';
    return Align(
      alignment: Alignment.centerLeft,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
        child: Container(
          constraints: BoxConstraints(
            maxWidth: MediaQuery.of(context).size.width * 0.7,
          ),
          decoration: const BoxDecoration(
            color: Color(0xFFFFF3E0),
            borderRadius: BorderRadius.only(
              topLeft: Radius.circular(16),
              topRight: Radius.circular(16),
              bottomLeft: Radius.circular(4),
              bottomRight: Radius.circular(16),
            ),
          ),
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    const SizedBox(
                      width: 40,
                      height: 40,
                      child: DecoratedBox(
                        decoration: BoxDecoration(
                          color: Color(0xFF4CAF50),
                          shape: BoxShape.circle,
                        ),
                        child: Icon(
                          Icons.insert_drive_file,
                          color: Colors.white,
                          size: 22,
                        ),
                      ),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            name,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                              fontSize: 14,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                          const SizedBox(height: 2),
                          Text(
                            _fmtSize(offer.fileSize),
                            style: const TextStyle(
                              fontSize: 12,
                              color: Colors.black54,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 10),
                _buildBody(),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildBody() {
    switch (state) {
      case FileOfferState.pending:
        if (!_hasButtons) {
          // Отправитель: ждём решения получателя.
          return const Row(
            children: [
              SizedBox(
                width: 16,
                height: 16,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
              SizedBox(width: 8),
              Text(
                'Ждём готовности принятия',
                style: TextStyle(fontSize: 13, color: Colors.black87),
              ),
            ],
          );
        }
        // Получатель: кнопки.
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Хочет отправить файл',
              style: TextStyle(fontSize: 13, color: Colors.black87),
            ),
            const SizedBox(height: 10),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton(
                  onPressed: onDecline,
                  style: TextButton.styleFrom(
                    foregroundColor: Colors.red,
                  ),
                  child: const Text('Не принять'),
                ),
                const SizedBox(width: 4),
                ElevatedButton(
                  onPressed: onAccept,
                  style: ElevatedButton.styleFrom(
                    backgroundColor: const Color(0xFF4CAF50),
                    foregroundColor: Colors.white,
                  ),
                  child: const Text('Принять'),
                ),
              ],
            ),
          ],
        );
      case FileOfferState.accepted:
        return const Row(
          children: [
            SizedBox(
              width: 16,
              height: 16,
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
            SizedBox(width: 8),
            Text(
              'Получатель принял. Отправка…',
              style: TextStyle(fontSize: 13, color: Colors.black87),
            ),
          ],
        );
      case FileOfferState.declined:
        return const Text(
          'Получатель не готов',
          style: TextStyle(fontSize: 13, color: Colors.red),
        );
      case FileOfferState.expired:
        return const Text(
          'Запрос истёк',
          style: TextStyle(fontSize: 13, color: Colors.black54),
        );
    }
  }
}
// mobile/lib/widgets/file_offer_bubble.dart