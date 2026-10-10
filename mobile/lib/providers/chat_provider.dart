// mobile/lib/providers/chat_provider.dart
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/message.dart';
import '../services/api_service.dart';
import '../services/ws_service.dart';
import '../services/p2p_service.dart';
import '../services/ethics_service.dart';
import '../services/libp2p_service.dart';
import '../services/media_storage.dart';
import '../services/log_service.dart';

/// Событие получения [CONTACT_HELLO_ACK] от B.
/// Содержит публичные ключи B — после этого A может слать [CONTACT_REQUEST] (E2E).
class HelloAck {
  final String peerID;
  final String ed25519Pub;
  final String x25519Pub;
  final String signature;

  HelloAck({
    required this.peerID,
    required this.ed25519Pub,
    required this.x25519Pub,
    required this.signature,
  });
}

class ChatProvider extends ChangeNotifier {
  late ApiService api;
  late WsService ws;
  P2PService? p2p;

  final Map<String, Message> _messagesMap = {};
  final Set<String> _ownMessageIds = {};
  String _activeChannel = 'general';
  String _currentNodeIp = '';
  List<Map<String, dynamic>> _channels = [];
  bool _wsConnected = false;
  bool _loading = false;
  String? _error;
  String _ttlPeriod = 'never';
  String _ttlMode = '';

  bool _libp2pStarted = false;
  bool _libp2pAvailable = false;
  String _libp2pPeerId = '';
  StreamSubscription? _messageSub;

  // Непрочитанные — по чату (peerID).
  // Открыл чат A → сбросил только A.
  final Map<String, int> _unreadByPeer = {};
  final Map<String, int> _unreadSnapshotByPeer = {};
  // Текущий открытый чат (peerID). null — чат закрыт.
  String? _currentOpenPeerID;

  // Кэш: PeerID → последнее сообщение от этого пира
  final Map<String, Message> _lastMessageByPeer = {};

  // Кэш: PeerID → отображаемое имя контакта.
  // Приоритет: локальное Name → RemoteName → короткий PeerID.
  // Заполняется loadPeerNames() из Go-контактов.
  final Map<String, String> _peerNames = {};

  // Флаг разовой миграции: перенос _readSent → ReadLocally (Go).
  static const String _readSentKeyLegacy = 'read_sent_ids';
  static const String _readSentMigratedKey = 'read_sent_migrated';
  bool _readSentMigrated = false;

  // Статусы своих сообщений: msg_id → 1/2/3.
  // 1=sent, 2=delivered, 3=read. Заполняется из Go-ядра (этап 1.5).
  final Map<String, int> _messageStatuses = {};
  Timer? _statusTimer;

  // ==== PENDING (таймер отправки) ====
  // Сообщения, ожидающие отправки через N секунд.
  // ID → Message (с pendingState = 'pending').
  final Map<String, Message> _pendingMessages = {};
  // ID → Timer, который отправит.
  final Map<String, Timer> _pendingTimers = {};
  // ID → секунды до отправки (для UI).
  final Map<String, int> _pendingSeconds = {};

  // Задержка отправки (секунды). 0 — без задержки.
  int _sendDelay = 0;
  static const String _sendDelayKey = 'send_delay';

  // Черновики — по одному на чат. Ключ — PeerID получателя.
  // Хранятся в SharedPreferences как JSON-строка под ключом 'drafts'.
  final Map<String, String> _drafts = {};
  static const String _draftsKey = 'drafts';

  // Контакт-протокол: [CONTACT_HELLO_ACK] от B.
  // При получении ChatProvider сам делает AddContact + sendContactRequest,
  // а также эмитит событие в _helloAckController — для UI.
  final StreamController<HelloAck> _helloAckController =
      StreamController<HelloAck>.broadcast();
  Stream<HelloAck> get helloAckStream => _helloAckController.stream;

  // Контакт-протокол: [CONTACT_REQUEST] от A.
  // Go уже сохранил запрос в requests store. Dart читает при событии.
  // Не polling — push. Событие-триггер с peerID отправителя.
  final StreamController<String> _contactRequestController =
      StreamController<String>.broadcast();
  Stream<String> get contactRequestStream => _contactRequestController.stream;

  // Контакт-протокол: [CONTACT_ACCEPT] от B — наш запрос принят.
  // Go обновил contact.confirmed = true. UI перечитывает контакты.
  final StreamController<String> _contactAcceptController =
      StreamController<String>.broadcast();
  Stream<String> get contactAcceptStream => _contactAcceptController.stream;

  // Контакт-протокол: [CONTACT_REJECT] от B — наш запрос отклонён.
  final StreamController<String> _contactRejectController =
      StreamController<String>.broadcast();
  Stream<String> get contactRejectStream => _contactRejectController.stream;

  // Событие: пир был замечен в сети (при отправке ему сообщения).
  // UI обновляет status узла на alive — устраняет асимметрию,
  // когда входящее сообщение обновляет статус, а исходящее — нет.
  final StreamController<String> _peerSeenController =
      StreamController<String>.broadcast();
  Stream<String> get peerSeenStream => _peerSeenController.stream;

  // Событие: тап по уведомлению → открыть чат с peerID.
  final StreamController<String> _openChatController =
      StreamController<String>.broadcast();
  Stream<String> get openChatStream => _openChatController.stream;

  // Разовое представление для [CONTACT_REQUEST].
  // Пользователь вводит в диалоге — сохраняем до получения ACK,
  // потом передаём в sendContactRequest. Не меняет MyDisplayName.
  final Map<String, String> _pendingContactNames = {};

  List<Message> get allMessages {
    final list = _messagesMap.values
        .where((m) => m.sender != '🌐 Сеть')
        .where((m) => !m.isExpired)
        .map((m) {
          return Message(
            id: m.id,
            text: m.text,
            plainText: m.plainText,
            version: m.version,
            sender: m.sender,
            time: m.time,
            isOwn: _ownMessageIds.contains(m.id) || m.isOwn,
            score: m.score,
            weight: m.weight,
            archived: m.archived,
            deliveryStatus: m.deliveryStatus,
            messageStatus: _messageStatuses[m.id],
            channel: m.channel,
            ttlPeriodSeconds: m.ttlPeriodSeconds,
            ttlMode: m.ttlMode,
            expiresAt: m.expiresAt,
            recipient: m.recipient,
          );
        })
        .toList();
    list.sort((a, b) => a.time.compareTo(b.time));
    return list;
  }

  List<Message> get messages => allMessages;

  /// Сообщения для конкретного чата (peerID).
  /// Свои: recipient == peerID. Входящие: sender == peerID.
  /// Файловые чанки (mediaType=file, chunkTotal>0) группируются по mediaId —
  /// в UI показывается одно сообщение на файл.
  List<Message> messagesFor(String peerID) {
    if (peerID.isEmpty) return allMessages;
    final filtered = _messagesMap.values
        .where((m) => m.sender != '🌐 Сеть')
        .where((m) => !m.isExpired)
        .where((m) {
          if (m.isOwn || _ownMessageIds.contains(m.id)) {
            return m.recipient == peerID;
          }
          return m.sender == peerID;
        })
        .toList();

    // Группировка файловых чанков по mediaId.
    final fileGroups = <String, List<Message>>{};
    final others = <Message>[];
    for (final m in filtered) {
      if (m.isFile && m.chunkTotal > 0) {
        fileGroups.putIfAbsent(m.mediaId, () => []).add(m);
      } else {
        others.add(m);
      }
    }
    for (final group in fileGroups.values) {
      group.sort((a, b) => a.chunkIndex.compareTo(b.chunkIndex));
      // Статус файла = минимум по всем чанкам группы.
      // 1=sent, 2=delivered, 3=hidden, 4=read.
      // hidden — терминальный: если есть хоть один, статус = hidden.
      int minStatus = 4;
      bool anyHidden = false;
      bool anyStatus = false;
      for (final c in group) {
        final s = _messageStatuses[c.id] ?? c.messageStatus ?? 0;
        if (s == 3) anyHidden = true;
        if (s > 0) {
          anyStatus = true;
          if (s < minStatus) minStatus = s;
        }
      }
      if (anyHidden) {
        minStatus = 3;
      } else if (!anyStatus) {
        minStatus = 0;
      }
      others.add(group.first.withStatus(minStatus));
    }

    final list = others
        .map((m) {
          return Message(
            id: m.id,
            text: m.text,
            plainText: m.plainText,
            version: m.version,
            sender: m.sender,
            time: m.time,
            isOwn: _ownMessageIds.contains(m.id) || m.isOwn,
            score: m.score,
            weight: m.weight,
            archived: m.archived,
            deliveryStatus: m.deliveryStatus,
            messageStatus: _messageStatuses[m.id],
            channel: m.channel,
            ttlPeriodSeconds: m.ttlPeriodSeconds,
            ttlMode: m.ttlMode,
            expiresAt: m.expiresAt,
            recipient: m.recipient,
            mediaType: m.mediaType,
            duration: m.duration,
            mediaData: m.mediaData,
            fileName: m.fileName,
            fileSize: m.fileSize,
            mediaId: m.mediaId,
            chunkIndex: m.chunkIndex,
            chunkTotal: m.chunkTotal,
            localFilePath: m.localFilePath,
          );
        })
        .toList();
    list.sort((a, b) => a.time.compareTo(b.time));
    return list;
  }

  String get activeChannel => _activeChannel;
  String get currentNodeIp => _currentNodeIp;
  String get ttlPeriod => _ttlPeriod;
  String get ttlMode => _ttlMode;

  /// Возвращает период TTL в секундах для UI.
  int get currentTtl => _ttlPeriodSeconds(_ttlPeriod);

  int _ttlPeriodSeconds(String period) {
    switch (period) {
      case '10s': return 10;
      case '30s': return 30;
      case '1m': return 60;
      case '5m': return 300;
      case '15m': return 900;
      case '30m': return 1800;
      case '1h': return 3600;
      case '4h': return 14400;
      case '24h': return 86400;
      case 'never':
      case 'forever':
      default:
        return 0;
    }
  }

  List<Map<String, dynamic>> get channels => _channels;
  bool get wsConnected => _wsConnected;
  bool get loading => _loading;
  String? get error => _error;
  bool get libp2pStarted => _libp2pStarted;
  bool get libp2pAvailable => _libp2pAvailable;
  String get libp2pPeerId => _libp2pPeerId;

  /// Непрочитанные для указанного чата.
  int unreadFor(String peerID) => _unreadByPeer[peerID] ?? 0;

  /// Снимок непрочитанных (до обнуления) для указанного чата.
  int unreadSnapshotFor(String peerID) => _unreadSnapshotByPeer[peerID] ?? 0;

  List<String> get logs => LogService.logs;

  // ==== PENDING / DELAY / DRAFT ====

  /// Сообщения, ожидающие отправки (pending).
  List<Message> get pendingMessages => _pendingMessages.values.toList();

  /// Есть ли pending сообщение (для блокировки ввода).
  bool get hasPending => _pendingMessages.isNotEmpty;

  /// Текущее значение задержки (сек).
  int get sendDelay => _sendDelay;

  /// Секунды до отправки для указанного ID.
  int? pendingSecondsFor(String id) => _pendingSeconds[id];

  /// Есть ли черновик для указанного чата.
  bool hasDraftFor(String peerID) =>
      peerID.isNotEmpty && (_drafts[peerID]?.isNotEmpty ?? false);

  /// Текст черновика для указанного чата (или пустая строка).
  String draftFor(String peerID) => _drafts[peerID] ?? '';

  /// Возвращает последнее сообщение от указанного пира (O(1))
  Message? getLastMessageForPeer(String peerID) => _lastMessageByPeer[peerID];

  /// Возвращает отображаемое имя контакта.
  /// Приоритет: Name → RemoteName → короткий PeerID.
  String nameFor(String peerID) {
    if (peerID.isEmpty) return '';
    final name = _peerNames[peerID];
    if (name != null && name.isNotEmpty) return name;
    return peerID.length > 12 ? '${peerID.substring(0, 12)}…' : peerID;
  }

  /// Устанавливает локальное имя контакта (после RenameContact).
  void setPeerName(String peerID, String name) {
    if (peerID.isEmpty) return;
    if (name.isEmpty) {
      _peerNames.remove(peerID);
    } else {
      _peerNames[peerID] = name;
    }
    _safeNotify();
  }

  /// Возвращает количество известных пиров с историей (для отладки)
  int get knownPeersCount => _lastMessageByPeer.length;

  /// Устанавливает P2PService (вызывается из ConnectScreen до входа в чат).
  /// Нужно для добавления libp2p-пиров в список узлов, когда приходит сообщение
  /// от нового пира (например, в LTE-сети, где NSD не работает).
  void setP2P(P2PService p2pService) {
    p2p = p2pService;
    LogService.log('ChatProvider: setP2P вызван');
  }

  /// Открывает/закрывает чат с указанным peerID.
  /// При открытии — снимок непрочитанных, сброс счётчика,
  /// один батч [READ] со всеми непрочитанными от этого peerID.
  void setChatOpen(bool open, String peerID) {
    if (open) {
      _currentOpenPeerID = peerID;
      _unreadSnapshotByPeer[peerID] = _unreadByPeer[peerID] ?? 0;
      _unreadByPeer[peerID] = 0;
      _sendReadBatchFor(peerID);
    } else {
      _currentOpenPeerID = null;
    }
    _safeNotify();
  }

  /// Собирает все непрочитанные msg_id от указанного peerID,
  /// помечает их прочитанными локально (Go) и отправляет
  /// одним батчем [READ] собеседнику.
  void _sendReadBatchFor(String peerID) {
    if (peerID.isEmpty) return;
    final refs = <String>[];
    for (final msg in _messagesMap.values) {
      if (msg.isOwn) continue;
      if (msg.sender != peerID) continue;
      if (msg.readLocally) continue;
      refs.add(msg.id);
    }
    if (refs.isEmpty) return;
    // Локально — пометить прочитанными (источник истины — Go).
    LibP2PService.markReadLocally(refs: refs).then((r) {
      if (r.containsKey('error')) {
        LogService.log('P2P: markReadLocally failed for $peerID: ${r['error']}');
      }
    }).catchError((e) {
      LogService.log('P2P: markReadLocally exception for $peerID: $e');
    });
    // Fire-and-forget: не блокируем UI.
    LibP2PService.sendReadBatch(refs: refs, recipient: peerID).then((r) {
      if (r.containsKey('error')) {
        LogService.log('P2P: sendReadBatch failed for $peerID: ${r['error']}');
      }
    }).catchError((e) {
      LogService.log('P2P: sendReadBatch exception for $peerID: $e');
    });
  }

  void initialize({String bootstrapPeers = ''}) {
    LogService.log('ChatProvider: initialize() CALLED');
    try {
      _loadOwnMessageIds();
      _loadSendDelay();
      _loadDrafts();
      _startLibP2P(bootstrapPeers: bootstrapPeers);
    } catch (e) {
      LogService.log('ChatProvider: initialize() ERROR: $e');
    }
  }

  /// Загружает TTL по умолчанию из Go-настроек.
  /// Вызывается после старта libp2p (когда Go готов).
  Future<void> loadTtl() async {
    try {
      final ttl = await LibP2PService.getTtl();
      var period = ttl['ttl_period'] ?? 'never';
      if (period == 'forever' || period.isEmpty) {
        period = 'never';
      }
      _ttlPeriod = period;
      _ttlMode = ttl['ttl_mode'] ?? '';
      LogService.log('ChatProvider: loadTtl period=$_ttlPeriod mode=$_ttlMode');
      _safeNotify();
    } catch (e) {
      LogService.log('ChatProvider: loadTtl ERROR: $e');
    }
  }

  void configure({
    required ApiService api,
    required WsService ws,
    P2PService? p2p,
    String nodeIp = '',
  }) {
    this.api = api;
    this.ws = ws;
    this.p2p = p2p;

    _currentNodeIp = _extractPeerID(nodeIp);
    LogService.log('ChatProvider.configure: nodeIp=$nodeIp, peerID=$_currentNodeIp');
  }

  /// Извлекает PeerID из строки. Поддерживает:
  /// - "libp2p://QmX..."                 → QmX...
  /// - "/ip4/.../p2p/QmX..."             → QmX...
  /// - "/ip4/.../p2p-circuit/p2p/QmX..." → QmX...
  /// - "QmX..."                          → QmX...
  /// - "QmX...:8081"                     → QmX...
  String _extractPeerID(String input) {
    if (input.isEmpty) return '';

    var clean = input.trim();

    // Убираем префикс libp2p://
    if (clean.startsWith('libp2p://')) {
      clean = clean.substring('libp2p://'.length);
    }

    // multiaddr: PeerID — всегда после последнего /p2p/
    if (clean.contains('/p2p/')) {
      clean = clean.split('/p2p/').last;
    }

    // На случай "QmX...:8081" — отрезаем порт, если есть ":". 
    // PeerID (base58btc) содержит только буквы и цифры, без ":".
    if (clean.contains(':')) {
      clean = clean.split(':')[0];
    }

    return clean;
  }

  Future<void> _loadOwnMessageIds() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final ids = prefs.getStringList('own_message_ids') ?? [];
      _ownMessageIds.addAll(ids);
      LogService.log('ChatProvider: загружено своих сообщений: ${ids.length}');
    } catch (e) {
      LogService.log('ChatProvider: _loadOwnMessageIds ERROR: $e');
    }
  }

  Future<void> _saveOwnMessageIds() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setStringList('own_message_ids', _ownMessageIds.toList());
    } catch (e) {
      LogService.log('ChatProvider: _saveOwnMessageIds ERROR: $e');
    }
  }

  /// Разовая миграция _readSent (Dart, SharedPreferences) → ReadLocally (Go).
  /// Выполняется после старта libp2p и loadMessages.
  /// Помечает все ранее прочитанные ID как ReadLocally=true в Go.
  Future<void> _migrateReadSentIfNeeded() async {
    if (_readSentMigrated) return;
    try {
      final prefs = await SharedPreferences.getInstance();
      if (prefs.getBool(_readSentMigratedKey) == true) {
        _readSentMigrated = true;
        return;
      }
      final legacy = prefs.getStringList(_readSentKeyLegacy) ?? [];
      if (legacy.isNotEmpty) {
        final r = await LibP2PService.markReadLocally(refs: legacy);
        if (r.containsKey('error')) {
          LogService.log('ChatProvider: migration markReadLocally failed: ${r['error']}');
        } else {
          LogService.log('ChatProvider: migrated ${legacy.length} read_sent → ReadLocally');
        }
        await prefs.remove(_readSentKeyLegacy);
      }
      await prefs.setBool(_readSentMigratedKey, true);
      _readSentMigrated = true;
    } catch (e) {
      LogService.log('ChatProvider: _migrateReadSentIfNeeded ERROR: $e');
    }
  }

  Future<void> _loadSendDelay() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      _sendDelay = prefs.getInt(_sendDelayKey) ?? 0;
      LogService.log('ChatProvider: send_delay=$_sendDelay');
    } catch (e) {
      LogService.log('ChatProvider: _loadSendDelay ERROR: $e');
    }
  }

  Future<void> setSendDelay(int seconds) async {
    _sendDelay = seconds;
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setInt(_sendDelayKey, seconds);
    } catch (e) {
      LogService.log('ChatProvider: setSendDelay ERROR: $e');
    }
    _safeNotify();
  }

  Future<void> _loadDrafts() async {
    try {
      final prefs = await SharedPreferences.getInstance();

      // Одноразовая чистка старого формата (одиночный draft_text).
      if (prefs.containsKey('draft_text')) {
        await prefs.remove('draft_text');
        LogService.log('ChatProvider: удалён старый ключ draft_text');
      }

      final raw = prefs.getString(_draftsKey);
      if (raw == null || raw.isEmpty) return;
      final decoded = jsonDecode(raw);
      if (decoded is Map) {
        _drafts.clear();
        decoded.forEach((k, v) {
          if (k is String && v is String && v.isNotEmpty) {
            _drafts[k] = v;
          }
        });
      }
      LogService.log('ChatProvider: загружено черновиков: ${_drafts.length}');
    } catch (e) {
      LogService.log('ChatProvider: _loadDrafts ERROR: $e');
    }
  }

  /// Сохраняет черновик для указанного чата.
  /// Пустой текст — удаляет черновик (не храним мусор).
  /// Публичный — вызывается из UI при наборе текста.
  Future<void> saveDraft(String peerID, String text) async {
    if (peerID.isEmpty) return;
    if (text.isEmpty) {
      _drafts.remove(peerID);
    } else {
      _drafts[peerID] = text;
    }
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_draftsKey, jsonEncode(_drafts));
    } catch (e) {
      LogService.log('ChatProvider: saveDraft ERROR: $e');
    }
    _safeNotify();
  }

  /// Очищает черновик для указанного чата.
  Future<void> clearDraft(String peerID) async {
    await saveDraft(peerID, '');
  }

  void _subscribeToMessages() {
    _messageSub?.cancel();
    _messageSub = LibP2PService.getMessageStream().listen((messageJSON) {
      try {
        // Специальное событие: тап по уведомлению → открыть чат.
        if (messageJSON.contains('"open_chat"')) {
          final map = jsonDecode(messageJSON) as Map<String, dynamic>;
          final peerID = map['open_chat'] as String? ?? '';
          if (peerID.isNotEmpty) {
            LogService.log('P2P: событие open_chat → $peerID');
            _openChatController.add(peerID);
          }
          return;
        }

        final map = jsonDecode(messageJSON) as Map<String, dynamic>;
        final isOwn = map['isOwn'] ?? false;
        final sender = map['sender'] ?? '';
        final replicatedFrom = map['replicatedFrom'] ?? '';

        final shortSender = sender.length > 8 ? sender.substring(0, 8) : sender;
        final shortReplicatedFrom = replicatedFrom.length > 8 ? replicatedFrom.substring(0, 8) : replicatedFrom;
        final shortMyID = _libp2pPeerId.length > 8 ? _libp2pPeerId.substring(0, 8) : _libp2pPeerId;

        // Признак «своё»: own-флаг, свой PeerID, сеть.
        final isSelf = isOwn
            || sender == '🌐 Сеть'
            || shortSender == shortMyID
            || shortReplicatedFrom == shortMyID;

        // Свои сообщения — не дублировать. НО: если это обновление
        // уже существующего (push ExpiresAt при after_read) —
        // пропустить в _addMessage (обновит expiresAt).
        final msgIdCheck = map['id'] as String? ?? '';
        final isUpdate = msgIdCheck.isNotEmpty && _messagesMap.containsKey(msgIdCheck);
        // Файловые чанки (в т.ч. свои) пропускаем — они нужны UI
        // для группировки и отображения файла.
        final mediaTypeS = map['media_type'] as String? ?? '';
        final chunkTotalS = map['chunk_total'] as int? ?? 0;
        final isFileChunkS = mediaTypeS == 'file' && chunkTotalS > 0;
        if (!isUpdate && isSelf && !isFileChunkS) {
          return;
        }

        // Добавляем пира в список узлов ConnectScreen (для LTE-сетей, где NSD не работает).
        // Только для ЧУЖИХ — свой PeerID в контактах не нужен.
        if (!isSelf && sender.isNotEmpty && p2p != null) {
          p2p!.addDiscoveredPeer(sender);
        }

        // Сервисные (контакт-протокол) — не UI-сообщения.
        final type = map['type'] as int? ?? 0;
        // Type=6 — [CONTACT_HELLO]: техническое, не показываем.
        if (type == 6) {
          LogService.log('P2P: [CONTACT_HELLO] от $sender (не UI)');
          return;
        }
        // Type=7 — [CONTACT_HELLO_ACK]: бизнес-логика (sendContactRequest E2E).
        if (type == 7) {
          _handleContactHelloAck(sender, map['text'] as String? ?? '');
          return;
        }
        // Type=3 — [CONTACT_REQUEST]: push-событие для UI.
        // Go уже сохранил запрос. UI читает при событии (не polling).
        if (type == 3) {
          LogService.log('P2P: [CONTACT_REQUEST] от $sender');
          _contactRequestController.add(sender);
          return;
        }
        // Type=4 — [CONTACT_ACCEPT]: наш запрос принят.
        // Go обновил contact.confirmed = true. UI перечитывает контакты.
        if (type == 4) {
          LogService.log('P2P: [CONTACT_ACCEPT] от $sender');
          _contactAcceptController.add(sender);
          return;
        }
        // Type=5 — [CONTACT_REJECT]: наш запрос отклонён.
        if (type == 5) {
          LogService.log('P2P: [CONTACT_REJECT] от $sender');
          _contactRejectController.add(sender);
          return;
        }
        // Type=1,2 — [DELIVERED]/[READ]: обрабатываются в Go, в UI не нужны.
        if (type >= 1 && type <= 2) {
          return;
        }
        // Type=8 — [TTL_UPDATE]: служебное, обрабатывается в Go.
        if (type == 8) {
          return;
        }

        final previewText = (map['text'] as String? ?? '');
        final preview = previewText.length > 40 ? '${previewText.substring(0, 40)}…' : previewText;
        LogService.log('P2P: входящее от $sender: $preview');

        // Увеличиваем непрочитанные только для НОВЫХ сообщений от ЧУЖИХ,
        // когда чат с ними не открыт. Иначе — дубликат или своё.
        final msgIdU = map['id'] as String? ?? '';
        final isNew = msgIdU.isNotEmpty && !_messagesMap.containsKey(msgIdU);
        final mediaTypeU = map['media_type'] as String? ?? '';
        final chunkTotalU = map['chunk_total'] as int? ?? 0;
        final chunkIndexU = map['chunk_index'] as int? ?? 0;
        final isFileChunkU = mediaTypeU == 'file' && chunkTotalU > 0;
        final isLastChunkU = isFileChunkU && chunkIndexU == chunkTotalU - 1;
        // Файл: бейдж только по последнему чанку. Обычное — всегда.
        final shouldCount = isFileChunkU ? isLastChunkU : true;
        if (isNew && !isSelf && shouldCount && sender != _currentOpenPeerID) {
          _unreadByPeer[sender] = (_unreadByPeer[sender] ?? 0) + 1;
          _safeNotify();
        }

        addExternalMessage(map);
      } catch (e) {
        LogService.log('P2P: ошибка парсинга: $e');
      }
    }, onError: (e) {
      LogService.log('P2P: ошибка стрима: $e');
    });
    LogService.log('P2P: подписка на стрим сообщений (глобально)');
  }

  Future<void> _startLibP2P({String bootstrapPeers = ''}) async {
    if (_libp2pStarted) return;

    LogService.log('libp2p: попытка запуска...');

    try {
      final ethHash = EthicsService.ethicsHash;
      if (ethHash.isEmpty) {
        LogService.log('libp2p: ethHash пустой, откладываю запуск на 3 сек...');
        await Future.delayed(const Duration(seconds: 3));
        return _startLibP2P(bootstrapPeers: bootstrapPeers);
      }

      final prefs = await SharedPreferences.getInstance();
      final savedBootstrap = bootstrapPeers.isNotEmpty
          ? bootstrapPeers
          : prefs.getString('bootstrap_peers') ?? '';
      LogService.log('libp2p: bootstrapPeers=${savedBootstrap.isNotEmpty ? savedBootstrap : "нет"}');

      final result = await LibP2PService.start(
        ethHash: ethHash,
        bootstrapPeers: savedBootstrap,
        enableMDNS: false,
      );

      if (result.containsKey('error')) {
        LogService.log('libp2p: ОШИБКА запуска: ${result['error']}');
        _libp2pAvailable = false;
        _safeNotify();
        return;
      }

      _libp2pStarted = true;
      _libp2pAvailable = true;
      LogService.log('libp2p: запущен успешно');

      final status = await LibP2PService.getStatus();
      _libp2pPeerId = status['id'] ?? '';
      LogService.log('libp2p: PeerID=$_libp2pPeerId');

      _subscribeToMessages();
      await loadMessages();
      await _migrateReadSentIfNeeded();
      // После миграции — пересчитать имена (и, возможно, бейджи).
      await loadPeerNames();

      // Проверяем, не был ли тап по уведомлению до старта UI.
      await _checkPendingOpenChat();

      _startStatusPolling();
      _startPurgeTimer();
      await loadTtl();

      _safeNotify();
    } catch (e) {
      LogService.log('libp2p: ИСКЛЮЧЕНИЕ при старте: $e');
      _libp2pAvailable = false;
      _safeNotify();
    }
  }

  /// Запускает периодический опрос статусов сообщений из Go-ядра.
  /// Каждые 3 секунды тянет map {msg_id: 1|2|3} и обновляет UI.
  void _startStatusPolling() {
    _statusTimer?.cancel();
    _statusTimer = Timer.periodic(const Duration(seconds: 3), (_) async {
      await refreshMessageStatuses();
    });
  }

  Timer? _purgeTimer;

  void _startPurgeTimer() {
    _purgeTimer?.cancel();
    _purgeTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      purgeExpired();
    });
  }

  /// Обновляет статусы сообщений из Go-ядра.
  Future<void> refreshMessageStatuses() async {
    if (!_libp2pStarted) return;
    try {
      final statuses = await LibP2PService.getMessageStatuses();
      if (statuses.isEmpty) return;

      bool changed = false;
      for (final entry in statuses.entries) {
        final prev = _messageStatuses[entry.key];
        if (prev == null || entry.value > prev) {
          _messageStatuses[entry.key] = entry.value;
          changed = true;
        }
      }
      if (changed) {
        _safeNotify();
      }
    } catch (_) {
      // Тихий fail — статусы не критичны.
    }
  }

  /// Запускает bootstrap-handshake с разовым представлением.
  /// 1. Сохраняет displayName для peerID (передадим после ACK).
  /// 2. Шлёт [CONTACT_HELLO].
  /// Публичный — вызывается из UI после диалога.
  Future<Map<String, dynamic>> startContactHandshake(
    String peerID,
    String displayName,
  ) async {
    if (peerID.isEmpty) {
      return {'error': 'peerID is required'};
    }
    _pendingContactNames[peerID] = displayName;
    LogService.log('ContactHandshake: $peerID, displayName="$displayName"');
    final result = await LibP2PService.sendContactHello(peerID: peerID);
    if (result.containsKey('error')) {
      // Не удалось отправить — сбрасываем имя.
      _pendingContactNames.remove(peerID);
      LogService.log('ContactHandshake: sendContactHello failed: ${result['error']}');
    }
    return result;
  }

  /// Обрабатывает [CONTACT_HELLO_ACK] (Type=7) от B.
  /// 1. Парсит payload (peerID B, ed25519_pub, x25519_pub, signature).
  /// 2. AddContact(B) — теперь A может шифровать E2E к B.
  /// 3. sendContactRequest(B) — E2E.
  /// 4. Эмитит в helloAckStream — UI покажет «Запрос отправлен».
  void _handleContactHelloAck(String sender, String payloadText) async {
    if (sender.isEmpty || payloadText.isEmpty) {
      LogService.log('HelloAck: пустой sender/payload');
      return;
    }
    Map<String, dynamic> payload;
    try {
      payload = jsonDecode(payloadText) as Map<String, dynamic>;
    } catch (e) {
      LogService.log('HelloAck: ошибка парсинга payload: $e');
      return;
    }
    final peerID = payload['peerID'] as String? ?? '';
    final ed25519Pub = payload['ed25519_pub'] as String? ?? '';
    final x25519Pub = payload['x25519_pub'] as String? ?? '';
    final signature = payload['signature'] as String? ?? '';

    if (peerID != sender) {
      LogService.log('HelloAck: peerID mismatch ($peerID != $sender)');
      return;
    }
    if (ed25519Pub.isEmpty || x25519Pub.isEmpty) {
      LogService.log('HelloAck: пустые ключи от $sender');
      return;
    }

    // Разовое представление из диалога (или пусто — Go возьмёт MyDisplayName).
    final displayName = _pendingContactNames.remove(peerID) ?? '';

    LogService.log('HelloAck: от $sender — отправка [CONTACT_REQUEST] E2E, name="$displayName"');

    // У A уже есть B (из QR) — ключи B уже сохранены.
    // AddContact не нужен. Сразу шлём [CONTACT_REQUEST] E2E.
    final reqResult = await LibP2PService.sendContactRequest(
      peerID: peerID,
      name: displayName,
    );
    if (reqResult.containsKey('error')) {
      LogService.log('HelloAck: sendContactRequest failed: ${reqResult['error']}');
      return;
    }

    _helloAckController.add(HelloAck(
      peerID: peerID,
      ed25519Pub: ed25519Pub,
      x25519Pub: x25519Pub,
      signature: signature,
    ));
    LogService.log('HelloAck: [CONTACT_REQUEST] отправлен $peerID');
  }

  /// Проверяет, не был ли тап по уведомлению до старта UI.
  /// Если да — эмитит событие openChatStream.
  Future<void> _checkPendingOpenChat() async {
    try {
      final peerID = await LibP2PService.getPendingOpenChat();
      if (peerID.isNotEmpty) {
        LogService.log('P2P: pendingOpenChat при старте → $peerID');
        _openChatController.add(peerID);
      }
    } catch (e) {
      LogService.log('P2P: _checkPendingOpenChat ERROR: $e');
    }
  }

  void _safeNotify() {
    notifyListeners();
  }

  Future<Map<String, dynamic>> sendViaLibP2P(String text) async {
    if (!_libp2pStarted) {
      return {'error': 'libp2p not started'};
    }
    try {
      return await LibP2PService.send(text: text, period: _ttlPeriod, mode: _ttlMode);
    } catch (e) {
      return {'error': e.toString()};
    }
  }

  /// Отправляет голосовое сообщение текущему контакту (E2E).
  /// mediaData — base64 Opus/Ogg. duration — секунды.
  /// Требует открытый чат (_currentNodeIp — PeerID получателя).
  Future<bool> sendVoice({
    required String mediaData,
    required int duration,
  }) async {
    if (_currentNodeIp.isEmpty || !_currentNodeIp.startsWith('Qm')) {
      _error = 'Нет получателя — откройте чат';
      _safeNotify();
      return false;
    }
    if (mediaData.isEmpty) {
      _error = 'Пустое голосовое';
      _safeNotify();
      return false;
    }

    final response = await LibP2PService.sendVoice(
      peerID: _currentNodeIp,
      mediaData: mediaData,
      duration: duration,
      period: _ttlPeriod,
      mode: _ttlMode,
    );

    if (response.containsKey('error')) {
      _error = 'Ошибка отправки голосового: ${response['error']}';
      _safeNotify();
      return false;
    }

    final msgId = response['id'];
    if (msgId == null || msgId.toString().isEmpty) {
      _error = 'Ошибка: Go-ядро не вернуло ID голосового';
      _safeNotify();
      return false;
    }

    _ownMessageIds.add(msgId.toString());
    await _saveOwnMessageIds();

    final now = DateTime.now().toUtc().toIso8601String();
    final ttlSec = _ttlPeriodSeconds(_ttlPeriod);

    final msg = Message(
      id: msgId.toString(),
      text: mediaData,
      plainText: mediaData,
      sender: 'Вы',
      time: now,
      isOwn: true,
      score: 0,
      weight: 0.5,
      archived: false,
      channel: _activeChannel,
      ttlPeriodSeconds: ttlSec,
      ttlMode: _ttlMode,
      expiresAt: (ttlSec > 0 && _ttlMode == 'hard')
          ? DateTime.now().add(Duration(seconds: ttlSec))
          : null,
      recipient: _currentNodeIp,
      mediaType: 'voice',
      duration: duration,
    );

    _addMessage(msg);
    _messageStatuses[msg.id] = 1;

    p2p?.saveOwnMessage(_currentNodeIp, msg);

    if (_currentNodeIp.isNotEmpty) {
      _peerSeenController.add(_currentNodeIp);
    }

    return true;
  }

  /// Отправляет фото текущему контакту (E2E). Один чанк.
  /// Файл читается Go с диска по пути — base64 через MethodChannel не идёт.
  Future<bool> sendPhotoByPath({
    required String filePath,
    required String fileName,
  }) async {
    if (_currentNodeIp.isEmpty || !_currentNodeIp.startsWith('Qm')) {
      _error = 'Нет получателя — откройте чат';
      _safeNotify();
      return false;
    }
    if (filePath.isEmpty) {
      _error = 'Пустой путь к фото';
      _safeNotify();
      return false;
    }

    final response = await LibP2PService.sendPhotoByPath(
      peerID: _currentNodeIp,
      filePath: filePath,
      fileName: fileName,
      period: _ttlPeriod,
      mode: _ttlMode,
    );

    if (response.containsKey('error')) {
      _error = 'Ошибка отправки фото: ${response['error']}';
      _safeNotify();
      return false;
    }

    final msgId = response['id'];
    if (msgId == null || msgId.toString().isEmpty) {
      _error = 'Ошибка: Go-ядро не вернуло ID фото';
      _safeNotify();
      return false;
    }

    _ownMessageIds.add(msgId.toString());
    await _saveOwnMessageIds();

    final now = DateTime.now().toUtc().toIso8601String();
    final ttlSec = _ttlPeriodSeconds(_ttlPeriod);

    // Локальное сообщение — с base64 (для отображения у отправителя).
    // Фото < 1 МБ, base64 приемлемо. Читаем файл локально,
    // НЕ через MethodChannel. У получателя — свой base64 из расшифровки.
    String photoB64 = '';
    try {
      final bytes = await File(filePath).readAsBytes();
      photoB64 = base64Encode(bytes);
    } catch (e) {
      LogService.log('PHOTO: read local failed: $e');
    }

    final msg = Message(
      id: msgId.toString(),
      text: photoB64,
      plainText: photoB64,
      sender: 'Вы',
      time: now,
      isOwn: true,
      score: 0,
      weight: 0.5,
      archived: false,
      channel: _activeChannel,
      ttlPeriodSeconds: ttlSec,
      ttlMode: _ttlMode,
      expiresAt: (ttlSec > 0 && _ttlMode == 'hard')
          ? DateTime.now().add(Duration(seconds: ttlSec))
          : null,
      recipient: _currentNodeIp,
      mediaType: 'photo',
    );

    _addMessage(msg);
    _messageStatuses[msg.id] = 1;

    p2p?.saveOwnMessage(_currentNodeIp, msg);

    if (_currentNodeIp.isNotEmpty) {
      _peerSeenController.add(_currentNodeIp);
    }

    return true;
  }

  /// Отправляет файл текущему контакту (E2E).
  /// Go читает файл с диска по пути, режет на чанки по 64 КБ,
  /// шифрует каждый, отправляет.
  /// Base64 через MethodChannel не проходит — OOM устранён.
  Future<bool> sendFileByPath({
    required String filePath,
    required String fileName,
  }) async {
    if (_currentNodeIp.isEmpty || !_currentNodeIp.startsWith('Qm')) {
      _error = 'Нет получателя — откройте чат';
      _safeNotify();
      return false;
    }
    if (filePath.isEmpty) {
      _error = 'Пустой путь к файлу';
      _safeNotify();
      return false;
    }

    final response = await LibP2PService.sendFileByPath(
      peerID: _currentNodeIp,
      filePath: filePath,
      fileName: fileName,
      period: _ttlPeriod,
      mode: _ttlMode,
    );

    if (response.containsKey('error')) {
      _error = 'Ошибка отправки файла: ${response['error']}';
      _safeNotify();
      return false;
    }

    final mediaId = response['id'];
    if (mediaId == null || mediaId.toString().isEmpty) {
      _error = 'Ошибка: Go-ядро не вернуло MediaID';
      _safeNotify();
      return false;
    }

    // Локальное сообщение НЕ создаём — UI покажет сгруппированные чанки
    // (mediaType=file, chunkTotal>0) как одно сообщение «📎 file.pdf».

    if (_currentNodeIp.isNotEmpty) {
      _peerSeenController.add(_currentNodeIp);
    }

    return true;
  }

  Future<List<Message>> getMessagesViaLibP2P() async {
    if (!_libp2pStarted) return [];
    try {
      final data = await LibP2PService.getMessages();
      final shortMyID = _libp2pPeerId.length > 8 ? _libp2pPeerId.substring(0, 8) : _libp2pPeerId;
      final result = <Message>[];
      for (final json in data) {
        final map = json as Map<String, dynamic>;
        final sender = map['sender'] ?? '';
        final shortSender = sender.length > 8 ? sender.substring(0, 8) : sender;
        final mediaType = map['media_type'] ?? '';
        final mediaId = map['media_id'] ?? '';
        final chunkTotal = map['chunk_total'] ?? 0;

        // Восстановление пути для собранного файла на диске.
        String localPath = '';
        if (mediaType == 'file' && chunkTotal > 0 && mediaId.toString().isNotEmpty) {
          final p = await MediaStorage.getFilePath(mediaId.toString());
          if (p != null) localPath = p;
        }

        result.add(Message(
          id: map['id'] ?? '',
          text: map['text'] ?? '',
          plainText: map['plainText'] ?? '',
          version: map['version'] ?? 0,
          sender: sender,
          time: map['time'] ?? DateTime.now().toUtc().toIso8601String(),
          isOwn: shortSender == shortMyID || map['isOwn'] == true,
          score: map['score'] ?? 0,
          weight: (map['weight'] as num?)?.toDouble() ?? 0.5,
          archived: map['archived'] ?? false,
          channel: map['channel'] ?? _activeChannel,
          ttlPeriodSeconds: map['ttl_period_s'] ?? 0,
          ttlMode: map['ttl_mode'] ?? '',
          expiresAt: Message.parseExpiresAt(map['expiresAt']),
          recipient: map['recipient'] ?? '',
          readLocally: map['read_locally'] ?? false,
          mediaType: mediaType,
          duration: map['duration'] ?? 0,
          mediaData: map['media_data'] ?? '',
          fileName: map['file_name'] ?? '',
          fileSize: map['file_size'] ?? 0,
          mediaId: mediaId,
          chunkIndex: map['chunk_index'] ?? 0,
          chunkTotal: chunkTotal,
          localFilePath: localPath,
        ));
      }
      return result;
    } catch (_) {
      return [];
    }
  }

  void setCurrentNode(String nodeIp) {
    _currentNodeIp = _extractPeerID(nodeIp);
    _safeNotify();
  }

  void setTtl(String period, String mode) {
    _ttlPeriod = period;
    _ttlMode = mode;
    _safeNotify();
  }

  /// Сбрасывает непрочитанные для указанного чата.
  void resetUnreadFor(String peerID) {
    _unreadByPeer[peerID] = 0;
    _safeNotify();
  }

  /// Загружает имена контактов из Go и заполняет _peerNames.
  /// Приоритет: Name → RemoteName → (не заполняем, fallback — PeerID).
  Future<void> loadPeerNames() async {
    if (!_libp2pStarted) return;
    try {
      final contacts = await LibP2PService.getContacts();
      for (final c in contacts) {
        final map = c as Map<String, dynamic>;
        final peerID = map['peerID'] as String? ?? '';
        if (peerID.isEmpty) continue;
        final name = (map['name'] as String? ?? '').trim();
        final remoteName = (map['remote_name'] as String? ?? '').trim();
        if (name.isNotEmpty) {
          _peerNames[peerID] = name;
        } else if (remoteName.isNotEmpty) {
          _peerNames[peerID] = remoteName;
        }
      }
      LogService.log('ChatProvider: loadPeerNames — ${_peerNames.length} имён');
      _safeNotify();
    } catch (e) {
      LogService.log('ChatProvider: loadPeerNames ERROR: $e');
    }
  }

  void initWs() {
    ws.onMessage = (msg) => _addMessage(msg);
    ws.onDisconnected = () {
      _wsConnected = false;
      _safeNotify();
    };
    ws.connect();
    _wsConnected = ws.isConnected;
    _safeNotify();
  }

  void addExternalMessage(Map<String, dynamic> data) {
    final id = data['id'] ?? '';
    if (id.isEmpty) {
      LogService.log('addExternalMessage: пустой id, text="${data['text']}"');
      return;
    }

    final msg = Message(
      id: id,
      text: data['text'] ?? '',
      plainText: data['plainText'] ?? '',
      version: data['version'] ?? 0,
      sender: data['sender'] ?? 'P2P',
      time: data['time'] ?? DateTime.now().toUtc().toIso8601String(),
      isOwn: data['isOwn'] ?? false,
      score: data['score'] ?? 0,
      weight: (data['weight'] as num?)?.toDouble() ?? 0.5,
      archived: data['archived'] ?? false,
      channel: data['channel'] ?? _activeChannel,
      ttlPeriodSeconds: data['ttl_period_s'] ?? 0,
      ttlMode: data['ttl_mode'] ?? '',
      expiresAt: Message.parseExpiresAt(data['expiresAt']),
      recipient: data['recipient'] ?? '',
      readLocally: data['read_locally'] ?? false,
      mediaType: data['media_type'] ?? '',
      duration: data['duration'] ?? 0,
      mediaData: data['media_data'] ?? '',
      fileName: data['file_name'] ?? '',
      fileSize: data['file_size'] ?? 0,
      mediaId: data['media_id'] ?? '',
      chunkIndex: data['chunk_index'] ?? 0,
      chunkTotal: data['chunk_total'] ?? 0,
    );
    _addMessage(msg);
  }

  void _addMessage(Message msg) {
    if (msg.id.isEmpty) {
      LogService.log('ADD SKIP: empty id');
      return;
    }
    final existing = _messagesMap[msg.id];
    if (existing != null) {
      // Сообщение уже есть. Если пришло обновление ExpiresAt (push из Go
      // при after_read → [READ]) — обновляем.
      if (msg.expiresAt != null && existing.expiresAt != msg.expiresAt) {
        _messagesMap[msg.id] = existing.copyWith(expiresAt: msg.expiresAt);
        LogService.log('ADD UPDATE id=${msg.id} expiresAt=${msg.expiresAt}');
        _safeNotify();
      } else {
        LogService.log('ADD SKIP: id exists id=${msg.id}');
      }
      return;
    }
    _messagesMap[msg.id] = msg;

    // Файловый чанк — сохраняем на диск, при готовности собираем.
    // НЕ обновляем lastMessage, НЕ отправляем [READ], НЕ пушим в unread.
    final isFileChunk = msg.isFile && msg.chunkTotal > 0;
    if (isFileChunk) {
      LogService.log('ADD chunk ${msg.chunkIndex + 1}/${msg.chunkTotal} mediaId=${msg.mediaId}');
      // Обновляем lastMessageByPeer по последнему чанку (файл = одно сообщение).
      // Ключ — peerID собеседника: свой recipient или sender.
      if (msg.chunkIndex == msg.chunkTotal - 1) {
        final chatPeer = msg.isOwn ? msg.recipient : msg.sender;
        if (chatPeer.isNotEmpty && chatPeer != '🌐 Сеть') {
          final existing = _lastMessageByPeer[chatPeer];
          if (existing == null || msg.time.compareTo(existing.time) > 0) {
            _lastMessageByPeer[chatPeer] = msg;
          }
        }
      }
      _enqueueFileChunk(msg);
      _safeNotify();
      return;
    }

    // Обновляем кэш последнего сообщения для чата.
    // Ключ — peerID собеседника: свой recipient или sender.
    final chatPeer = msg.isOwn ? msg.recipient : msg.sender;
    if (chatPeer.isNotEmpty && chatPeer != '🌐 Сеть') {
      final existing = _lastMessageByPeer[chatPeer];
      if (existing == null || msg.time.compareTo(existing.time) > 0) {
        _lastMessageByPeer[chatPeer] = msg;
      }
    }

    final preview = msg.text.length > 40 ? '${msg.text.substring(0, 40)}…' : msg.text;
    LogService.log('ADD id=${msg.id} text="$preview" sender=${msg.sender}');

    // [READ] отправляется при открытии чата (setChatOpen).
    // Если чат с этим sender открыт — отправим сразу.
    _sendReadFor(msg);

    _safeNotify();
  }

  // Защита от race condition: пока один чанк собирает файл,
  // остальные ждут. Иначе 5 горутин вызывают tryAssemble одновременно.
  final Set<String> _assembling = {};

  // Очередь запросов getChunk. Ключ — mediaId, значение — Set<chunkIndex>,
  // ожидающих получения base64 из Go.
  // Обрабатывается таймером (100 мс): по одному чанку за раз.
  // Защита от всплеска 440 запросов — OOM/перегрузка MethodChannel.
  final Map<String, Set<int>> _chunkQueue = {};
  Timer? _chunkQueueTimer;

  /// Ставит чанк в очередь на получение base64 из Go.
  /// Реальный вызов getChunk — в _flushChunkQueue (таймер 100 мс).
  /// Свои чанки (isOwn) — не сохраняем: файл у отправителя на диске (sent/).
  /// Иначе — getChunk × 10718 впустую (возвращает пусто), шум в логе.
  void _enqueueFileChunk(Message msg) {
    if (msg.mediaId.isEmpty) return;
    // Свои чанки — не сохраняем как входящие.
    if (msg.isOwn) return;
    // Если файл уже собран — не тянем чанки.
    if (msg.localFilePath.isNotEmpty) return;
    // Если base64 уже в памяти (голосовые, фото) — не нужно.
    if (msg.chunkTotal <= 0) return;

    _chunkQueue.putIfAbsent(msg.mediaId, () => <int>{}).add(msg.chunkIndex);
    _chunkQueueTimer ??= Timer(const Duration(milliseconds: 100), () {
      _chunkQueueTimer = null;
      _flushChunkQueue();
    });
  }

  /// Обрабатывает очередь: берёт первые до 10 чанков, запрашивает base64
  /// через getChunk, сохраняет на диск. Потом — снова таймер (если ещё есть).
  Future<void> _flushChunkQueue() async {
    // Собираем до 10 задач.
    final batch = <MapEntry<String, int>>[];
    for (final entry in _chunkQueue.entries) {
      final mediaId = entry.key;
      final indexes = entry.value.toList()..sort();
      for (final idx in indexes) {
        batch.add(MapEntry(mediaId, idx));
        if (batch.length >= 10) break;
      }
      if (batch.length >= 10) break;
    }
    if (batch.isEmpty) return;

    // Удаляем взятые из очереди.
    for (final e in batch) {
      _chunkQueue[e.key]?.remove(e.value);
      if (_chunkQueue[e.key]?.isEmpty ?? false) {
        _chunkQueue.remove(e.key);
      }
    }

    // Параллельно запрашиваем до 10 чанков.
    await Future.wait(batch.map((e) async {
      await _fetchAndSaveChunk(e.key, e.value);
    }));

    // Если очередь не пуста — следующий таймер.
    if (_chunkQueue.isNotEmpty) {
      _chunkQueueTimer ??= Timer(const Duration(milliseconds: 100), () {
        _chunkQueueTimer = null;
        _flushChunkQueue();
      });
    }
  }

  /// Запрашивает base64 одного чанка через getChunk и сохраняет на диск.
  Future<void> _fetchAndSaveChunk(String mediaId, int chunkIndex) async {
    // Метаданные — из _messagesMap.
    Message? meta;
    for (final m in _messagesMap.values) {
      if (m.mediaId == mediaId && m.chunkIndex == chunkIndex) {
        meta = m;
        break;
      }
    }
    if (meta == null) return;
    // Свои чанки — не сохраняем (двойная защита).
    if (meta.isOwn) return;
    if (meta.localFilePath.isNotEmpty) return; // уже собран

    final base64Data = await LibP2PService.getChunk(
      mediaID: mediaId,
      chunkIndex: chunkIndex,
    );
    if (base64Data == null || base64Data.isEmpty) {
      LogService.log('MEDIA: getChunk failed $mediaId[$chunkIndex]');
      return;
    }

    final ok = await MediaStorage.saveChunk(
      mediaId: mediaId,
      chunkIndex: chunkIndex,
      chunkTotal: meta.chunkTotal,
      fileName: meta.fileName,
      fileSize: meta.fileSize,
      base64Data: base64Data,
    );
    if (!ok) return;

    // Чанк сохранён на диск — base64 в Go-памяти больше не нужен.
    // Очищаем, чтобы memory у получателя не раздувалась на больших файлах.
    LibP2PService.clearChunkData(mediaID: mediaId, chunkIndex: chunkIndex);

    // Защита: только один поток собирает файл для данного mediaId.
    if (_assembling.contains(mediaId)) return;
    _assembling.add(mediaId);
    String? path;
    try {
      path = await MediaStorage.tryAssemble(mediaId);
    } finally {
      _assembling.remove(mediaId);
    }
    if (path == null) return;

    // Все чанки собраны — обновляем ВСЕ Message этого mediaId (localFilePath).
    final updates = <String, Message>{};
    for (final m in _messagesMap.values) {
      if (m.mediaId == mediaId && m.localFilePath != path) {
        updates[m.id] = m.withLocalFilePath(path);
      }
    }
    if (updates.isEmpty) return;
    for (final e in updates.entries) {
      _messagesMap[e.key] = e.value;
    }
    LogService.log('MEDIA: file ready $path');
    _safeNotify();

    // Файл собран — если чат открыт, отправляем [READ] по последнему чанку.
    if (_currentOpenPeerID != null && _currentOpenPeerID!.isNotEmpty) {
      final all = _messagesMap.values
          .where((m) => m.mediaId == mediaId && m.chunkTotal > 0)
          .toList();
      if (all.isNotEmpty) {
        all.sort((a, b) => a.chunkIndex.compareTo(b.chunkIndex));
        final last = all.last;
        if (last.sender == _currentOpenPeerID && !last.readLocally) {
          _sendReadFor(last);
        }
      }
    }
  }

  /// Прогресс передачи файла (0-100). Целое.
  /// Отправитель: по доставленным чанкам (status >= 2).
  /// Получатель: по числу сохранённых чанков на диске.
  int fileProgress(String mediaId) {
    if (mediaId.isEmpty) return 0;
    final chunks = _messagesMap.values
        .where((m) => m.mediaId == mediaId && m.chunkTotal > 0)
        .toList();
    if (chunks.isEmpty) return 0;
    final total = chunks.first.chunkTotal;
    if (total <= 0) return 0;

    final isOwn = chunks.first.isOwn;
    int ready = 0;
    if (isOwn) {
      for (final c in chunks) {
        final s = _messageStatuses[c.id] ?? c.messageStatus ?? 0;
        if (s >= 2) ready++;
      }
    } else {
      ready = MediaStorage.chunksCount(mediaId);
      if (ready < 0) return 100; // маркер «собран»
    }
    return ((ready / total) * 100).clamp(0, 100).round();
  }

  /// Отправляет [READ] для одного сообщения (если ещё не отправляли).
  /// Вызывается: при открытии чата (для всех непрочитанных) и при
  /// получении нового сообщения, если чат открыт.
  void _sendReadFor(Message msg) {
    if (msg.isOwn) return;                              // свои — не читаем
    if (msg.sender == 'Вы') return;
    if (msg.sender == '🌐 Сеть') return;
    if (msg.sender.isEmpty) return;
    if (msg.sender != _currentOpenPeerID) return;       // не текущий чат
    if (msg.readLocally) return;
    // Файловый чанк — [READ] отправляется только для ПОСЛЕДНЕГО чанка
    // файла и только когда файл собран (localFilePath установлен).
    if (msg.isFile && msg.chunkTotal > 0) {
      if (msg.chunkIndex != msg.chunkTotal - 1) return;
      if (msg.localFilePath.isEmpty) return;
    }
    // Локально — пометить прочитанным (источник истины — Go).
    // Без этого readLocally не выставляется при новом сообщении в
    // открытом чате → при перезапуске появляется бейдж.
    LibP2PService.markReadLocally(refs: [msg.id]).then((r) {
      if (r.containsKey('error')) {
        LogService.log('P2P: markReadLocally failed for ${msg.id}: ${r['error']}');
      }
    }).catchError((e) {
      LogService.log('P2P: markReadLocally exception for ${msg.id}: $e');
    });
    // Fire-and-forget: не блокируем UI.
    LibP2PService.sendRead(ref: msg.id, recipient: msg.sender).then((r) {
      if (r.containsKey('error')) {
        LogService.log('P2P: sendRead failed for ${msg.id}: ${r['error']}');
      }
    }).catchError((e) {
      LogService.log('P2P: sendRead exception for ${msg.id}: $e');
    });
  }

  /// Удаляет сообщение по ID. Для файловых чанков — удаляет ВСЕ чанки
  /// этого mediaId (один файл = много чанков, они группируются в UI).
  /// Синхронизирует с Go: FileDeleteAll в Go-ядре.
  void deleteMessage(String id) {
    final msg = _messagesMap[id];
    if (msg == null) return;

    if (msg.isFile && msg.mediaId.isNotEmpty) {
      // Удаляем все чанки этого mediaId.
      final toRemove = <String>[];
      for (final entry in _messagesMap.entries) {
        if (entry.value.mediaId == msg.mediaId) {
          toRemove.add(entry.key);
          _messageStatuses.remove(entry.key);
        }
      }
      for (final k in toRemove) {
        _messagesMap.remove(k);
      }
      // Кэш последнего сообщения — пересобрать.
      final chatPeer = msg.isOwn ? msg.recipient : msg.sender;
      if (chatPeer.isNotEmpty) {
        _lastMessageByPeer.remove(chatPeer);
      }
      // Go: удалить все чанки файла из memory + sent/ (если есть).
      LibP2PService.deleteFile(msg.mediaId).catchError((_) => <String, dynamic>{});
      LogService.log('DELETE file ${msg.mediaId} — удалено чанков: ${toRemove.length}');
    } else {
      _messagesMap.remove(id);
      _messageStatuses.remove(id);
    }
    _safeNotify();
  }

  /// Полное удаление переписки с контактом.
  /// Удаляет сообщения, статусы, кэш последнего сообщения,
  /// черновик, непрочитанные. Вызывается при удалении контакта.
  Future<void> removePeerMessages(String peerID) async {
    if (peerID.isEmpty) return;

    // 1. Сообщения: sender == peerID (входящие) или recipient == peerID (свои).
    final toRemove = <String>[];
    for (final entry in _messagesMap.entries) {
      final m = entry.value;
      if (m.sender == peerID || m.recipient == peerID) {
        toRemove.add(entry.key);
      }
    }
    for (final id in toRemove) {
      _messagesMap.remove(id);
      _messageStatuses.remove(id);
    }

    // 2. Кэш последнего сообщения.
    _lastMessageByPeer.remove(peerID);

    // 2b. Имя контакта.
    _peerNames.remove(peerID);

    // 3. Черновик чата.
    _drafts.remove(peerID);

    // 4. Непрочитанные.
    _unreadByPeer.remove(peerID);
    _unreadSnapshotByPeer.remove(peerID);

    // 5. Если этот чат открыт — закрыть.
    if (_currentOpenPeerID == peerID) {
      _currentOpenPeerID = null;
    }

    // 6. Сохранить изменения.
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_draftsKey, jsonEncode(_drafts));
    } catch (e) {
      LogService.log('ChatProvider: removePeerMessages saveDrafts ERROR: $e');
    }

    LogService.log('ChatProvider: removePeerMessages $peerID — удалено сообщений: ${toRemove.length}');
    _safeNotify();
  }

  void clearAllMessages() {
    _messagesMap.clear();
    _lastMessageByPeer.clear();
    _safeNotify();
  }

  void purgeExpired() {
    final before = _messagesMap.length;
    final expiredPeers = <String>{};
    _messagesMap.removeWhere((key, msg) {
      final e = msg.isExpired;
      if (e) {
        if (msg.sender.isNotEmpty && msg.sender != 'Вы' && msg.sender != '🌐 Сеть') {
          expiredPeers.add(msg.sender);
        }
        if (msg.recipient.isNotEmpty) {
          expiredPeers.add(msg.recipient);
        }
      }
      return e;
    });
    if (_messagesMap.length == before) return;

    for (final peerID in expiredPeers) {
      Message? latest;
      for (final m in _messagesMap.values) {
        final fromPeer = m.sender == peerID;
        final toPeer = m.recipient == peerID;
        if (!fromPeer && !toPeer) continue;
        if (latest == null || m.time.compareTo(latest.time) > 0) {
          latest = m;
        }
      }
      if (latest == null) {
        _lastMessageByPeer.remove(peerID);
      } else {
        _lastMessageByPeer[peerID] = latest;
      }

      if (_currentOpenPeerID == peerID) {
        _unreadByPeer[peerID] = 0;
      } else {
        int alive = 0;
        for (final m in _messagesMap.values) {
          if (m.sender == peerID && !m.readLocally) {
            alive++;
          }
        }
        _unreadByPeer[peerID] = alive;
      }
    }

    _safeNotify();
  }

  Future<void> loadMessages() async {
    _loading = true;
    _safeNotify();

    try {
      if (_libp2pStarted) {
        final fromLibP2P = await getMessagesViaLibP2P();
        int added = 0;
        for (final msg in fromLibP2P) {
          if (msg.sender != '🌐 Сеть' && msg.id.isNotEmpty) {
            final before = _messagesMap.length;
            _addMessage(msg);
            final isNew = _messagesMap.length > before;
            if (isNew) added++;
            // Восстановление счётчика непрочитанных: входящее,
            // ещё не прочитано локально, чат не открыт.
            // Файловые чанки — не считаем unread per-chunk.
            // Один бейдж на файл: считаем только по последнему чанку.
            final isFileChunk = msg.isFile && msg.chunkTotal > 0;
            final isLastChunk = msg.isFile && msg.chunkTotal > 0 && msg.chunkIndex == msg.chunkTotal - 1;
            final shouldCount = isFileChunk ? isLastChunk : true;

            if (isNew &&
                !msg.isOwn &&
                shouldCount &&
                !msg.readLocally &&
                msg.sender.isNotEmpty &&
                msg.sender != '🌐 Сеть' &&
                msg.sender != _currentOpenPeerID) {
              _unreadByPeer[msg.sender] = (_unreadByPeer[msg.sender] ?? 0) + 1;
            }
          }
        }
        LogService.log('Загружено из libp2p: ${fromLibP2P.length}, добавлено новых: $added');
        _safeNotify();
      }
    } catch (e) {
      LogService.log('Загрузка сообщений: ОШИБКА: $e');
    }

    _loading = false;
    _safeNotify();
  }

  Future<void> loadChannels() async {
    try {
      _channels = await api.getChannels();
      _safeNotify();
    } catch (_) {}
  }

  Future<bool> sendMessage(String text) async {
    if (text.trim().isEmpty) return false;

    final ethics = EthicsService.evaluate(text);
    if (!ethics.allowed) {
      _error = 'Сообщение заблокировано этическим фильтром';
      _safeNotify();
      return false;
    }

    // Если задержка > 0 — ставим в pending, не отправляем сразу.
    if (_sendDelay > 0) {
      _enqueuePending(text, ethics.weight);
      return true;
    }

    // Задержка = 0 — отправляем сразу.
    return await _sendNow(text, ethics.weight);
  }

  /// Ставит сообщение в очередь на отправку через _sendDelay секунд.
  void _enqueuePending(String text, double weight) {
    // Локальный ID (не msg_id из Go — его ещё нет).
    final pendingId = 'pending_${DateTime.now().microsecondsSinceEpoch}';
    final now = DateTime.now().toUtc().toIso8601String();

    final ttlSec = _ttlPeriodSeconds(_ttlPeriod);
    final msg = Message(
      id: pendingId,
      text: text,
      sender: 'Вы',
      time: now,
      isOwn: true,
      score: 0,
      weight: weight,
      archived: false,
      channel: _activeChannel,
      ttlPeriodSeconds: ttlSec,
      ttlMode: _ttlMode,
      expiresAt: null,
      pendingState: 'pending',
      recipient: _currentNodeIp,
    );

    _pendingMessages[pendingId] = msg;
    _pendingSeconds[pendingId] = _sendDelay;

    // Тикер каждую секунду: обновляет счётчик, по истечении — отправляет.
    final timer = Timer.periodic(const Duration(seconds: 1), (t) {
      final left = (_pendingSeconds[pendingId] ?? 0) - 1;
      if (left > 0) {
        _pendingSeconds[pendingId] = left;
        _safeNotify();
      } else {
        t.cancel();
        _pendingTimers.remove(pendingId);
        _pendingSeconds.remove(pendingId);
        _pendingMessages.remove(pendingId);
        // Отправляем.
        _sendNow(msg.text, msg.weight);
        _safeNotify();
      }
    });

    _pendingTimers[pendingId] = timer;
    _safeNotify();
    LogService.log('ChatProvider: pending $pendingId, delay=$_sendDelay сек');
  }

  /// Отменяет pending: возвращает текст для редактирования.
  /// Возвращает текст (или null).
  String? cancelPending(String pendingId) {
    final msg = _pendingMessages.remove(pendingId);
    _pendingTimers.remove(pendingId)?.cancel();
    _pendingSeconds.remove(pendingId);
    _safeNotify();
    if (msg == null) return null;
    LogService.log('ChatProvider: cancelPending $pendingId');
    return msg.text;
  }

  /// Удаляет pending без возврата текста (свайп).
  void deletePending(String pendingId) {
    _pendingMessages.remove(pendingId);
    _pendingTimers.remove(pendingId)?.cancel();
    _pendingSeconds.remove(pendingId);
    _safeNotify();
    LogService.log('ChatProvider: deletePending $pendingId');
  }

  /// Превращает все pending в черновик (для back).
  /// Берёт первое (единственное) pending.
  Future<void> pendingToDraft() async {
    if (_pendingMessages.isEmpty) return;
    final first = _pendingMessages.values.first;
    _pendingTimers[first.id]?.cancel();
    _pendingTimers.remove(first.id);
    _pendingSeconds.remove(first.id);
    _pendingMessages.remove(first.id);
    await saveDraft(_currentNodeIp, first.text);
    LogService.log('ChatProvider: pending → draft (${first.text.length} симв.)');
    _safeNotify();
  }

  /// Отправляет сообщение сейчас (сразу или после таймера).
  Future<bool> _sendNow(String text, double weight) async {
    // Если есть PeerID получателя — адресная E2E-отправка.
    // Иначе — broadcast (обратная совместимость).
    Map<String, dynamic> response;
    if (_currentNodeIp.isNotEmpty && _currentNodeIp.startsWith('Qm')) {
      response = await LibP2PService.sendToPeer(
        peerID: _currentNodeIp,
        text: text,
        period: _ttlPeriod,
        mode: _ttlMode,
      );
    } else {
      response = await sendViaLibP2P(text);
    }

    if (response.containsKey('error')) {
      _error = 'Ошибка отправки: ${response['error']}';
      _safeNotify();
      return false;
    }

    final msgId = response['id'];
    if (msgId == null || msgId.toString().isEmpty) {
      _error = 'Ошибка: Go-ядро не вернуло ID сообщения';
      _safeNotify();
      return false;
    }

    _ownMessageIds.add(msgId.toString());
    await _saveOwnMessageIds();

    final now = DateTime.now().toUtc().toIso8601String();

    final ttlSec = _ttlPeriodSeconds(_ttlPeriod);
    final msg = Message(
      id: msgId.toString(),
      text: text,
      sender: 'Вы',
      time: now,
      isOwn: true,
      score: 0,
      weight: weight,
      archived: false,
      channel: _activeChannel,
      ttlPeriodSeconds: ttlSec,
      ttlMode: _ttlMode,
      expiresAt: (ttlSec > 0 && _ttlMode == 'hard')
          ? DateTime.now().add(Duration(seconds: ttlSec))
          : null,
      recipient: _currentNodeIp,
    );

    _addMessage(msg);

    // Локально сразу ставим статус "отправлено" (1).
    _messageStatuses[msg.id] = 1;

    p2p?.saveOwnMessage(_currentNodeIp, msg);

    // Отправлено — черновик больше не нужен.
    await clearDraft(_currentNodeIp);

    // Пир точно в сети — уведомляем UI (обновит status → alive).
    if (_currentNodeIp.isNotEmpty) {
      _peerSeenController.add(_currentNodeIp);
    }

    return true;
  }

  void switchChannel(String channel) {
    if (channel == _activeChannel) return;
    _activeChannel = channel;
    _safeNotify();
  }

  Future<void> sendFeedback(String id, int score) async {
    Message? target;
    for (final entry in _messagesMap.entries) {
      if (entry.value.id == id) {
        target = entry.value;
        break;
      }
    }

    if (target == null) return;

    final newWeight = (score == 1)
        ? (target.weight + 0.15).clamp(0.0, 1.0)
        : (target.weight - 0.15).clamp(0.0, 1.0);

    final updated = Message(
      id: target.id,
      text: target.text,
      plainText: target.plainText,
      version: target.version,
      sender: target.sender,
      time: target.time,
      isOwn: target.isOwn,
      score: score,
      weight: newWeight,
      archived: target.archived,
      channel: target.channel,
      ttlPeriodSeconds: target.ttlPeriodSeconds,
      ttlMode: target.ttlMode,
      expiresAt: target.expiresAt,
    );

    _messagesMap[target.id] = updated;

    try {
      await api.sendFeedback(id, score);
    } catch (_) {}

    _safeNotify();
  }

  Future<void> stopLibP2P() async {
    if (_libp2pStarted) {
      await LibP2PService.stop();
      _libp2pStarted = false;
      _libp2pAvailable = false;
      _safeNotify();
    }
  }

  @override
  void dispose() {
    _statusTimer?.cancel();
    _purgeTimer?.cancel();
    _chunkQueueTimer?.cancel();
    for (final t in _pendingTimers.values) {
      t.cancel();
    }
    _pendingTimers.clear();
    _messageSub?.cancel();

    _helloAckController.close();
    _contactRequestController.close();
    _contactAcceptController.close();
    _contactRejectController.close();
    _peerSeenController.close();
    _openChatController.close();
    stopLibP2P();
    super.dispose();
  }
}
// mobile/lib/providers/chat_provider.dart