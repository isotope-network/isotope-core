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
  final int? messageStatus; // 1=sent, 2=delivered, 3=hidden, 4=read (из Go-ядра)
  final String? pendingState; // null | 'pending' | 'draft' | 'error'
  final String? channel;
  final int ttlPeriodSeconds;
  final String ttlMode;
  final DateTime? expiresAt;
  /// PeerID получателя (для адресных сообщений).
  /// Пусто — broadcast или история без адресата.
  final String recipient;
  /// Я прочитал это входящее. false — не прочитано (для бейджа).
  /// Источник истины — Go. Восстанавливается при старте.
  final bool readLocally;

  /// Тип медиа: '' | 'text' | 'voice' | 'file'.
  /// '' — старое текстовое (обратная совместимость).
  final String mediaType;

  /// Длительность голосового (секунды). 0 — не голосовое.
  final int duration;

  /// Медиа-данные (base64). Не используем сейчас — Text уже содержит.
  /// Оставлено для совместимости с Go-JSON.
  final String mediaData;

  /// Имя файла (для файлов). Пока не используем.
  final String fileName;

  /// Размер файла (байты). Пока не используем.
  final int fileSize;

  /// MediaID — идентификатор файла (для чанков).
  final String mediaId;

  /// ChunkIndex — номер чанка (0, 1, 2, ...). 0 — если не чанк.
  final int chunkIndex;

  /// ChunkTotal — общее число чанков. 0 — если не чанк.
  final int chunkTotal;

  /// Локальный путь к собранному файлу. Только на устройстве.
  /// В сеть не передаётся. Пусто — файл не собран.
  final String localFilePath;

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
    this.messageStatus,
    this.pendingState,
    this.channel,
    this.ttlPeriodSeconds = 0,
    this.ttlMode = '',
    this.expiresAt,
    this.recipient = '',
    this.readLocally = false,
    this.mediaType = '',
    this.duration = 0,
    this.mediaData = '',
    this.fileName = '',
    this.fileSize = 0,
    this.mediaId = '',
    this.chunkIndex = 0,
    this.chunkTotal = 0,
    this.localFilePath = '',
  });

  /// Копия с обновлённым expiresAt.
  Message copyWith({DateTime? expiresAt}) {
    return Message(
      id: id,
      text: text,
      plainText: plainText,
      version: version,
      sender: sender,
      time: time,
      isOwn: isOwn,
      score: score,
      weight: weight,
      archived: archived,
      deliveryStatus: deliveryStatus,
      messageStatus: messageStatus,
      pendingState: pendingState,
      channel: channel,
      ttlPeriodSeconds: ttlPeriodSeconds,
      ttlMode: ttlMode,
      expiresAt: expiresAt ?? this.expiresAt,
      recipient: recipient,
      readLocally: readLocally,
      mediaType: mediaType,
      duration: duration,
      mediaData: mediaData,
      fileName: fileName,
      fileSize: fileSize,
      mediaId: mediaId,
      chunkIndex: chunkIndex,
      chunkTotal: chunkTotal,
      localFilePath: localFilePath,
    );
  }

  /// Копия с обновлённым messageStatus.
  Message withStatus(int? status) {
    return Message(
      id: id,
      text: text,
      plainText: plainText,
      version: version,
      sender: sender,
      time: time,
      isOwn: isOwn,
      score: score,
      weight: weight,
      archived: archived,
      deliveryStatus: deliveryStatus,
      messageStatus: status,
      pendingState: pendingState,
      channel: channel,
      ttlPeriodSeconds: ttlPeriodSeconds,
      ttlMode: ttlMode,
      expiresAt: expiresAt,
      recipient: recipient,
      readLocally: readLocally,
      mediaType: mediaType,
      duration: duration,
      mediaData: mediaData,
      fileName: fileName,
      fileSize: fileSize,
      mediaId: mediaId,
      chunkIndex: chunkIndex,
      chunkTotal: chunkTotal,
      localFilePath: localFilePath,
    );
  }

  /// Копия с обновлённым pendingState.
  Message withPendingState(String? state) {
    return Message(
      id: id,
      text: text,
      plainText: plainText,
      version: version,
      sender: sender,
      time: time,
      isOwn: isOwn,
      score: score,
      weight: weight,
      archived: archived,
      deliveryStatus: deliveryStatus,
      messageStatus: messageStatus,
      pendingState: state,
      channel: channel,
      ttlPeriodSeconds: ttlPeriodSeconds,
      ttlMode: ttlMode,
      expiresAt: expiresAt,
      recipient: recipient,
      readLocally: readLocally,
      mediaType: mediaType,
      duration: duration,
      mediaData: mediaData,
      fileName: fileName,
      fileSize: fileSize,
      mediaId: mediaId,
      chunkIndex: chunkIndex,
      chunkTotal: chunkTotal,
      localFilePath: localFilePath,
    );
  }

  /// Копия с обновлённым localFilePath.
  Message withLocalFilePath(String path) {
    return Message(
      id: id,
      text: text,
      plainText: plainText,
      version: version,
      sender: sender,
      time: time,
      isOwn: isOwn,
      score: score,
      weight: weight,
      archived: archived,
      deliveryStatus: deliveryStatus,
      messageStatus: messageStatus,
      pendingState: pendingState,
      channel: channel,
      ttlPeriodSeconds: ttlPeriodSeconds,
      ttlMode: ttlMode,
      expiresAt: expiresAt,
      recipient: recipient,
      readLocally: readLocally,
      mediaType: mediaType,
      duration: duration,
      mediaData: mediaData,
      fileName: fileName,
      fileSize: fileSize,
      mediaId: mediaId,
      chunkIndex: chunkIndex,
      chunkTotal: chunkTotal,
      localFilePath: path,
    );
  }

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
      messageStatus: json['messageStatus'],
      pendingState: json['pendingState'],
      channel: json['channel'],
      ttlPeriodSeconds: json['ttlPeriodSeconds'] ?? 0,
      ttlMode: json['ttlMode'] ?? '',
      expiresAt: Message.parseExpiresAt(json['expiresAt']),
      recipient: json['recipient'] ?? '',
      readLocally: json['read_locally'] ?? false,
      mediaType: json['media_type'] ?? '',
      duration: json['duration'] ?? 0,
      mediaData: json['media_data'] ?? '',
      fileName: json['file_name'] ?? '',
      fileSize: json['file_size'] ?? 0,
      mediaId: json['media_id'] ?? '',
      chunkIndex: json['chunk_index'] ?? 0,
      chunkTotal: json['chunk_total'] ?? 0,
      localFilePath: '', // локальное поле, не из Go
    );
  }

  /// Парсит expiresAt из Go-JSON.
  /// Go с omitempty на time.Time возвращает "0001-01-01T00:00:00Z"
  /// вместо пропуска поля. Трактуем как null.
  static DateTime? parseExpiresAt(dynamic raw) {
    if (raw == null) return null;
    final s = raw.toString();
    if (s.isEmpty) return null;
    if (s.startsWith('0001-01-01')) return null;
    final dt = DateTime.tryParse(s);
    return dt?.toLocal();
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
      'messageStatus': messageStatus,
      'pendingState': pendingState,
      'channel': channel,
      'ttlPeriodSeconds': ttlPeriodSeconds,
      'ttlMode': ttlMode,
      'expiresAt': expiresAt?.toIso8601String(),
      'recipient': recipient,
      'media_type': mediaType,
      'duration': duration,
      'media_data': mediaData,
      'file_name': fileName,
      'file_size': fileSize,
      'media_id': mediaId,
      'chunk_index': chunkIndex,
      'chunk_total': chunkTotal,
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

  /// Иконка статуса сообщения из Go-ядра (1/2/3/4).
  /// 1=sent (✓), 2=delivered (✓✓), 3=hidden (✓🔒), 4=read (✓✓).
  String get messageStatusIcon {
    switch (messageStatus) {
      case 1:
        return '✓';
      case 2:
        return '✓✓';
      case 3:
        return '✓🔒';
      case 4:
        return '✓✓';
      default:
        return '';
    }
  }

  /// Цвет статуса сообщения.
  /// 4 (read) — зелёный. Остальные — серый.
  int get messageStatusColor {
    switch (messageStatus) {
      case 4:
        return 0xFF4CAF50;
      default:
        return 0xFF999999;
    }
  }

  /// Тип отправителя: свой, чужой, сеть
  String get senderType {
    if (isOwn) return 'own';
    if (sender == '🌐 Сеть') return 'network';
    return 'peer';
  }

  /// Голосовое сообщение?
  bool get isVoice => mediaType == 'voice';

  /// Файл?
  bool get isFile => mediaType == 'file';

  /// Base64-данные медиа: у своих — PlainText, у входящих — Text.
  /// Для голосовых — base64 Opus/Ogg.
  String get mediaBase64 {
    if (isOwn && plainText.isNotEmpty) return plainText;
    return text;
  }

  /// TTL-метка
  String get ttlLabel {
    final t = ttlPeriodSeconds;
    if (t == 0) return '∞';
    if (t < 60) return '${t}с';
    if (t < 3600) return '${t ~/ 60}м';
    if (t < 86400) return '${t ~/ 3600}ч';
    return '${t ~/ 86400}д';
  }
}
// mobile/lib/models/message.dart