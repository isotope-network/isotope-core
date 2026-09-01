class Message {
  final String id;
  final String text;
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

  /// Отформатированное время (ЧЧ:ММ)
  String get formattedTime {
    try {
      final match = RegExp(r'T(\d{2}:\d{2}):\d{2}').firstMatch(time);
      if (match != null) return match.group(1)!;
      if (RegExp(r'^\d{2}:\d{2}').hasMatch(time)) return time.substring(0, 5);
      return time;
    } catch (_) {
      return time;
    }
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