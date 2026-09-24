// mobile/lib/models/message.dart
class Message {
  final String id;
  final String text;
  final String plainText;
  final int version;
  final String sender;
  final String time;
  final bool isOwn;
  final int score;
  final double weight;
  final bool archived;
  final String? deliveryStatus;
  final String? channel;
  final int ttl;
  final DateTime? expiresAt;

  Message({
    required this.id,
    required this.text,
    this.plainText = '',
    this.version = 0,
    required this.sender,
    required this.time,
    required this.isOwn,
    required this.score,
    required this.weight,
    required this.archived,
    this.deliveryStatus,
    this.channel,
    this.ttl = 0,
    this.expiresAt,
  });

  factory Message.fromJson(Map<String, dynamic> json) {
    return Message(
      id: json['id'] ?? '',
      text: json['text'] ?? '',
      plainText: json['plainText'] ?? '',
      version: json['version'] ?? 0,
      sender: json['sender'] ?? '',
      time: json['time'] ?? '',
      isOwn: json['isOwn'] ?? false,
      score: json['score'] ?? 0,
      weight: (json['weight'] ?? 0.5).toDouble(),
      archived: json['archived'] ?? false,
      deliveryStatus: json['deliveryStatus'],
      channel: json['channel'],
      ttl: json['ttl'] ?? 0,
      expiresAt: json['expiresAt'] != null
          ? DateTime.tryParse(json['expiresAt'])
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'text': text,
      'plainText': plainText,
      'version': version,
      'sender': sender,
      'time': time,
      'isOwn': isOwn,
      'score': score,
      'weight': weight,
      'archived': archived,
      'deliveryStatus': deliveryStatus,
      'channel': channel,
      'ttl': ttl,
      'expiresAt': expiresAt?.toIso8601String(),
    };
  }

  /// Текст для отображения в UI.
  /// Для своих E2E-сообщений (Version=2) — PlainText (открытый).
  /// Для остальных — Text (входящие уже открытые, broadcast — открытые).
  String get displayText {
    if (isOwn && version == 2 && plainText.isNotEmpty) {
      return plainText;
    }
    return text;
  }

  /// Проверка: истекло ли сообщение
  bool get isExpired {
    if (expiresAt == null) return false;
    return DateTime.now().isAfter(expiresAt!);
  }

  /// Иконка статуса доставки
  String get statusIcon {
    switch (deliveryStatus) {
      case 'sent':
        return '📤';
      case 'delivered':
        return '✓✓';
      case 'partial':
        return '⚠';
      case 'filtered':
        return '🚫';
      default:
        return '';
    }
  }

  /// Цвет статуса
  int get statusColor {
    switch (deliveryStatus) {
      case 'sent':
        return 0xFF999999;
      case 'delivered':
        return 0xFF4CAF50;
      case 'partial':
        return 0xFFFF9800;
      case 'filtered':
        return 0xFFF44336;
      default:
        return 0xFF999999;
    }
  }

  /// Отформатированное время (ЧЧ:ММ) с конвертацией в локальный часовой пояс.
  ///
  /// Go-ядро отправляет время в UTC формате "2006-01-02T15:04:05" (без суффикса Z).
  /// Эта функция парсит строку как UTC (если нет TZ-суффикса) и конвертирует
  /// в локальное время устройства.
  String get formattedTime {
    try {
      var t = time;
      // Если нет TZ-суффикса — считаем UTC, добавляем Z
      if (!t.endsWith('Z') && !t.contains('+') && !_hasTimezoneOffset(t)) {
        t = '${t}Z';
      }
      final dt = DateTime.tryParse(t);
      if (dt == null) return time;

      final local = dt.toLocal();
      final hh = local.hour.toString().padLeft(2, '0');
      final mm = local.minute.toString().padLeft(2, '0');
      return '$hh:$mm';
    } catch (_) {
      return time;
    }
  }

  /// Проверяет наличие TZ-смещения после времени (формат "+HH:MM" или "-HH:MM")
  bool _hasTimezoneOffset(String t) {
    final tIndex = t.indexOf('T');
    if (tIndex < 0) return false;
    final after = t.substring(tIndex);
    return after.contains('+') || after.lastIndexOf('-') > 0;
  }

  /// Вес для бейджа
  String get weightLabel => '⚖${weight.toStringAsFixed(2)}';

  /// Тип отправителя: свой, чужой, сеть
  String get senderType {
    if (isOwn) return 'own';
    if (sender == '🌐 Сеть') return 'network';
    return 'peer';
  }

  /// TTL-метка
  String get ttlLabel {
    if (ttl == 0) return '∞';
    if (ttl < 60) return '${ttl}с';
    if (ttl < 3600) return '${ttl ~/ 60}м';
    if (ttl < 86400) return '${ttl ~/ 3600}ч';
    return '${ttl ~/ 86400}д';
  }
}
// mobile/lib/models/message.dart