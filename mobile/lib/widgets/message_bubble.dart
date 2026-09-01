import 'package:flutter/material.dart';
import '../models/message.dart';

class MessageBubble extends StatelessWidget {
  final Message message;
  final VoidCallback? onLike;
  final VoidCallback? onDislike;

  const MessageBubble({
    super.key,
    required this.message,
    this.onLike,
    this.onDislike,
  });

  @override
  Widget build(BuildContext context) {
    final isOwn = message.senderType == 'own';
    final isNetwork = message.senderType == 'network';

    return Align(
      alignment: isOwn ? Alignment.centerRight : Alignment.centerLeft,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
        child: Column(
          crossAxisAlignment: isOwn ? CrossAxisAlignment.end : CrossAxisAlignment.start,
          children: [
            Container(
              constraints: BoxConstraints(
                maxWidth: MediaQuery.of(context).size.width * 0.7,
              ),
              decoration: BoxDecoration(
                color: isOwn
                    ? const Color(0xFFDCF8C6)
                    : isNetwork
                        ? const Color(0xFFFFF3E0)
                        : Colors.white,
                borderRadius: BorderRadius.only(
                  topLeft: const Radius.circular(16),
                  topRight: const Radius.circular(16),
                  bottomLeft: Radius.circular(isOwn ? 16 : 4),
                  bottomRight: Radius.circular(isOwn ? 4 : 16),
                ),
                border: (!isOwn && !isNetwork)
                    ? Border.all(color: Colors.grey.shade200)
                    : null,
              ),
              child: Padding(
                padding: const EdgeInsets.all(10),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // Имя полностью
                    Text(
                      isOwn ? 'Вы' : message.sender,
                      style: TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                        color: Colors.black54,
                      ),
                    ),
                    const SizedBox(height: 2),
                    // Вес + время
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        _weightBadge(message.weight),
                        const SizedBox(width: 6),
                        Text(
                          message.formattedTime,
                          style: const TextStyle(fontSize: 11, color: Colors.black45),
                        ),
                      ],
                    ),
                    const SizedBox(height: 6),
                    // Текст
                    Text(
                      message.text,
                      style: const TextStyle(fontSize: 15),
                    ),
                    if (isOwn && message.deliveryStatus != null) ...[
                      const SizedBox(height: 4),
                      Align(
                        alignment: Alignment.centerRight,
                        child: Text(
                          '${message.statusIcon} ${_statusText(message.deliveryStatus!)}',
                          style: TextStyle(
                            fontSize: 10,
                            color: Color(message.statusColor),
                          ),
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ),
            // Кнопки 👍/👎 под сообщением с отступом
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  _iconButton(
                    icon: Icons.thumb_up_alt_outlined,
                    color: message.score > 0 ? Colors.green : Colors.grey,
                    onTap: onLike,
                    size: 18,
                  ),
                  const SizedBox(width: 16),
                  _iconButton(
                    icon: Icons.thumb_down_alt_outlined,
                    color: message.score < 0 ? Colors.red : Colors.grey,
                    onTap: onDislike,
                    size: 18,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _weightBadge(double weight) {
    Color bg;
    Color fg;
    if (weight >= 0.7) {
      bg = const Color(0xFFC8E6C9);
      fg = const Color(0xFF2E7D32);
    } else if (weight <= 0.3) {
      bg = const Color(0xFFFFCDD2);
      fg = const Color(0xFFC62828);
    } else {
      bg = const Color(0xFFFFF9C4);
      fg = const Color(0xFFF57F17);
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        '⚖${weight.toStringAsFixed(2)}',
        style: TextStyle(fontSize: 10, fontWeight: FontWeight.w600, color: fg),
      ),
    );
  }

  Widget _iconButton({
    required IconData icon,
    required Color color,
    VoidCallback? onTap,
    double size = 16,
  }) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(4),
        child: Icon(icon, size: size, color: color),
      ),
    );
  }

  String _statusText(String status) {
    switch (status) {
      case 'sent':
        return 'Отправлено';
      case 'delivered':
        return 'Доставлено';
      case 'partial':
        return 'Частично';
      case 'filtered':
        return 'Тема вне интересов';
      default:
        return '';
    }
  }
}