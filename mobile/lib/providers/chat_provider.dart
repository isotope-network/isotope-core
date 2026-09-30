// mobile/lib/providers/chat_provider.dart
import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/message.dart';
import '../services/api_service.dart';
import '../services/ws_service.dart';
import '../services/p2p_service.dart';
import '../services/ethics_service.dart';
import '../services/libp2p_service.dart';
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
  int _currentTtl = 86400;

  bool _libp2pStarted = false;
  bool _libp2pAvailable = false;
  String _libp2pPeerId = '';
  StreamSubscription? _messageSub;
  int _unreadCount = 0;
  int _unreadSnapshot = 0;
  bool _chatOpen = false;

  // Кэш: PeerID → последнее сообщение от этого пира
  final Map<String, Message> _lastMessageByPeer = {};

  // ID входящих сообщений, для которых уже отправили [READ].
  // Сохраняется в SharedPreferences (последние _readSentMax IDs).
  // При перезапуске — загружается, не сбрасывается.
  final Set<String> _readSent = {};
  static const String _readSentKey = 'read_sent_ids';
  static const int _readSentMax = 1000;
  bool _readSentLoaded = false;

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

  // Черновик (один на текущий чат). Текст + время.
  String? _draftText;
  static const String _draftKey = 'draft_text';

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
            ttl: m.ttl,
            expiresAt: m.expiresAt,
          );
        })
        .toList();
    list.sort((a, b) => a.time.compareTo(b.time));
    return list;
  }

  List<Message> get messages => allMessages;

  String get activeChannel => _activeChannel;
  String get currentNodeIp => _currentNodeIp;
  int get currentTtl => _currentTtl;
  List<Map<String, dynamic>> get channels => _channels;
  bool get wsConnected => _wsConnected;
  bool get loading => _loading;
  String? get error => _error;
  bool get libp2pStarted => _libp2pStarted;
  bool get libp2pAvailable => _libp2pAvailable;
  String get libp2pPeerId => _libp2pPeerId;
  int get unreadCount => _unreadCount;
  int get unreadSnapshot => _unreadSnapshot;
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

  /// Есть ли черновик.
  bool get hasDraft => _draftText != null && _draftText!.isNotEmpty;

  /// Текст черновика (или null).
  String? get draftText => _draftText;

  /// Возвращает последнее сообщение от указанного пира (O(1))
  Message? getLastMessageForPeer(String peerID) => _lastMessageByPeer[peerID];

  /// Возвращает количество известных пиров с историей (для отладки)
  int get knownPeersCount => _lastMessageByPeer.length;

  /// Устанавливает P2PService (вызывается из ConnectScreen до входа в чат).
  /// Нужно для добавления libp2p-пиров в список узлов, когда приходит сообщение
  /// от нового пира (например, в LTE-сети, где NSD не работает).
  void setP2P(P2PService p2pService) {
    p2p = p2pService;
    LogService.log('ChatProvider: setP2P вызван');
  }

  void setChatOpen(bool open, {bool preserveUnread = false}) {
    if (open) {
      _unreadSnapshot = _unreadCount;
      _chatOpen = true;
      _unreadCount = 0;
      // Отправить [READ] для всех непрочитанных входящих.
      for (final msg in _messagesMap.values) {
        _sendReadFor(msg);
      }
    } else {
      _chatOpen = false;
    }
    _safeNotify();
  }

  void initialize({String bootstrapPeers = ''}) {
    LogService.log('ChatProvider: initialize() CALLED');
    try {
      _loadOwnMessageIds();
      _loadReadSent();
      _loadSendDelay();
      _loadDraft();
      _startLibP2P(bootstrapPeers: bootstrapPeers);
    } catch (e) {
      LogService.log('ChatProvider: initialize() ERROR: $e');
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

  Future<void> _loadReadSent() async {
    if (_readSentLoaded) return;
    try {
      final prefs = await SharedPreferences.getInstance();
      final ids = prefs.getStringList(_readSentKey) ?? [];
      _readSent.addAll(ids);
      _readSentLoaded = true;
      LogService.log('ChatProvider: _readSent загружено: ${ids.length}');
    } catch (e) {
      LogService.log('ChatProvider: _loadReadSent ERROR: $e');
    }
  }

  Future<void> _saveReadSent() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      // Ограничиваем — последние _readSentMax ID.
      // Set не сохраняет порядок — берём .toList().sublist.
      final list = _readSent.toList();
      final trimmed = list.length > _readSentMax
          ? list.sublist(list.length - _readSentMax)
          : list;
      await prefs.setStringList(_readSentKey, trimmed);
    } catch (e) {
      LogService.log('ChatProvider: _saveReadSent ERROR: $e');
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

  Future<void> _loadDraft() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final txt = prefs.getString(_draftKey);
      if (txt != null && txt.isNotEmpty) {
        _draftText = txt;
        LogService.log('ChatProvider: загружен черновик (${txt.length} симв.)');
      }
    } catch (e) {
      LogService.log('ChatProvider: _loadDraft ERROR: $e');
    }
  }

  Future<void> _saveDraft(String? text) async {
    _draftText = text;
    try {
      final prefs = await SharedPreferences.getInstance();
      if (text == null || text.isEmpty) {
        await prefs.remove(_draftKey);
      } else {
        await prefs.setString(_draftKey, text);
      }
    } catch (e) {
      LogService.log('ChatProvider: _saveDraft ERROR: $e');
    }
    _safeNotify();
  }

  /// Очистить черновик.
  Future<void> clearDraft() async {
    await _saveDraft(null);
  }

  void _subscribeToMessages() {
    _messageSub?.cancel();
    _messageSub = LibP2PService.getMessageStream().listen((messageJSON) {
      try {
        final map = jsonDecode(messageJSON) as Map<String, dynamic>;
        final isOwn = map['isOwn'] ?? false;
        final sender = map['sender'] ?? '';
        final replicatedFrom = map['replicatedFrom'] ?? '';

        final shortSender = sender.length > 8 ? sender.substring(0, 8) : sender;
        final shortReplicatedFrom = replicatedFrom.length > 8 ? replicatedFrom.substring(0, 8) : replicatedFrom;
        final shortMyID = _libp2pPeerId.length > 8 ? _libp2pPeerId.substring(0, 8) : _libp2pPeerId;

        if (isOwn || sender == '🌐 Сеть' || shortSender == shortMyID || shortReplicatedFrom == shortMyID) {
          return;
        }

        // Добавляем пира в список узлов ConnectScreen (для LTE-сетей, где NSD не работает)
        if (sender.isNotEmpty && p2p != null) {
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
        // Type=1,2,4,5 — [DELIVERED]/[READ]/[ACCEPT]/[REJECT]:
        // обрабатываются в Go, в UI не нужны. Отсекаем.
        if (type >= 1 && type <= 5) {
          return;
        }

        LogService.log('P2P: входящее от $sender: ${map['text']}');
        if (!_chatOpen) {
          _unreadCount++;
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

      _startStatusPolling();

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

    LogService.log('HelloAck: от $sender — отправка [CONTACT_REQUEST] E2E');

    // У A уже есть B (из QR) — ключи B уже сохранены.
    // AddContact не нужен. Сразу шлём [CONTACT_REQUEST] E2E.
    final reqResult = await LibP2PService.sendContactRequest(peerID: peerID, name: '');
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

  void _safeNotify() {
    notifyListeners();
  }

  Future<Map<String, dynamic>> sendViaLibP2P(String text) async {
    if (!_libp2pStarted) {
      return {'error': 'libp2p not started'};
    }
    try {
      return await LibP2PService.send(text: text, ttl: _currentTtl);
    } catch (e) {
      return {'error': e.toString()};
    }
  }

  Future<List<Message>> getMessagesViaLibP2P() async {
    if (!_libp2pStarted) return [];
    try {
      final data = await LibP2PService.getMessages();
      final shortMyID = _libp2pPeerId.length > 8 ? _libp2pPeerId.substring(0, 8) : _libp2pPeerId;
      return data.map((json) {
        final map = json as Map<String, dynamic>;
        final sender = map['sender'] ?? '';
        final shortSender = sender.length > 8 ? sender.substring(0, 8) : sender;
        return Message(
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
          ttl: map['ttl'] ?? 0,
          expiresAt: null,
        );
      }).toList();
    } catch (_) {
      return [];
    }
  }

  void setCurrentNode(String nodeIp) {
    _currentNodeIp = _extractPeerID(nodeIp);
    _safeNotify();
  }

  void setTtl(int ttl) {
    _currentTtl = ttl;
    _safeNotify();
  }

  void resetUnread() {
    _unreadCount = 0;
    _safeNotify();
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
      ttl: data['ttl'] ?? 0,
      expiresAt: null,
    );
    _addMessage(msg);
  }

  void _addMessage(Message msg) {
    if (msg.id.isEmpty) {
      LogService.log('ADD SKIP: empty id, text="${msg.text}"');
      return;
    }
    if (_messagesMap.containsKey(msg.id)) {
      LogService.log('ADD SKIP: id exists id=${msg.id} len=${msg.id.length} text="${msg.text}"');
      return;
    }
    _messagesMap[msg.id] = msg;

    // Обновляем кэш последнего сообщения от пира
    if (msg.sender != 'Вы' && msg.sender != '🌐 Сеть' && msg.sender.isNotEmpty) {
      final existing = _lastMessageByPeer[msg.sender];
      if (existing == null || msg.time.compareTo(existing.time) > 0) {
        _lastMessageByPeer[msg.sender] = msg;
      }
    }

    LogService.log('ADD id=${msg.id} len=${msg.id.length} text="${msg.text}" sender=${msg.sender}');

    // [READ] отправляется не здесь, а при открытии чата (setChatOpen).
    // Если чат уже открыт — отправим сразу (пользователь видит сообщение).
    if (_chatOpen) {
      _sendReadFor(msg);
    }

    _safeNotify();
  }

  /// Отправляет [READ] для одного сообщения (если ещё не отправляли).
  /// Вызывается: при открытии чата (для всех непрочитанных) и при
  /// получении нового сообщения, если чат открыт.
  void _sendReadFor(Message msg) {
    if (msg.isOwn
        || msg.sender == 'Вы'
        || msg.sender == '🌐 Сеть'
        || msg.sender.isEmpty
        || _readSent.contains(msg.id)) {
      return;
    }
    _readSent.add(msg.id);
    // Fire-and-forget: не блокируем UI.
    LibP2PService.sendRead(ref: msg.id, recipient: msg.sender).then((r) {
      if (r.containsKey('error')) {
        LogService.log('P2P: sendRead failed for ${msg.id}: ${r['error']}');
      }
    }).catchError((e) {
      LogService.log('P2P: sendRead exception for ${msg.id}: $e');
    });
    // Сохраняем _readSent (последние 1000).
    _saveReadSent();
  }

  void deleteMessage(String id) {
    _messagesMap.remove(id);
    _safeNotify();
  }

  void clearAllMessages() {
    _messagesMap.clear();
    _lastMessageByPeer.clear();
    _safeNotify();
  }

  void purgeExpired() {
    final before = _messagesMap.length;
    _messagesMap.removeWhere((key, msg) => msg.isExpired);
    if (_messagesMap.length != before) {
      _safeNotify();
    }
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
            if (_messagesMap.length > before) added++;
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
      ttl: _currentTtl,
      expiresAt: null,
      pendingState: 'pending',
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
    await _saveDraft(first.text);
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
        ttl: _currentTtl,
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
      ttl: _currentTtl,
      expiresAt: _currentTtl > 0 ? DateTime.now().add(Duration(seconds: _currentTtl)) : null,
    );

    _addMessage(msg);

    // Локально сразу ставим статус "отправлено" (1).
    _messageStatuses[msg.id] = 1;

    p2p?.saveOwnMessage(_currentNodeIp, msg);

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
      ttl: target.ttl,
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
    for (final t in _pendingTimers.values) {
      t.cancel();
    }
    _pendingTimers.clear();
    _messageSub?.cancel();
    _helloAckController.close();
    _contactRequestController.close();
    stopLibP2P();
    super.dispose();
  }
}
// mobile/lib/providers/chat_provider.dart