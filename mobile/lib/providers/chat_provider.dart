import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/message.dart';
import '../services/api_service.dart';
import '../services/ws_service.dart';
import '../services/p2p_service.dart';
import '../services/ethics_service.dart';
import '../services/libp2p_service.dart';
import '../services/log_service.dart';

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

  // libp2p состояние
  bool _libp2pStarted = false;
  bool _libp2pAvailable = false;
  String _libp2pPeerId = '';

  List<Message> get allMessages {
    final list = _messagesMap.values
        .where((m) => m.sender != '🌐 Сеть')
        .where((m) => !m.isExpired)
        .map((m) {
          return Message(
            id: m.id,
            text: m.text,
            sender: m.sender,
            time: m.time,
            isOwn: _ownMessageIds.contains(m.id) || m.isOwn,
            score: m.score,
            weight: m.weight,
            archived: m.archived,
            deliveryStatus: m.deliveryStatus,
            channel: m.channel,
            ttl: m.ttl,
            expiresAt: m.expiresAt,
          );
        })
        .toList();
    list.sort((a, b) => a.time.compareTo(b.time));
    return list;
  }

  List<Message> get messages {
    if (_currentNodeIp.isEmpty) return allMessages;

    final currentKey = _currentNodeIp.split(':')[0];
    return allMessages.where((m) {
      final senderKey = m.sender.split(':')[0];
      return senderKey == currentKey || m.isOwn;
    }).toList();
  }

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
  List<String> get logs => LogService.logs;

  void configure({
    required ApiService api,
    required WsService ws,
    P2PService? p2p,
    String nodeIp = '',
  }) {
    this.api = api;
    this.ws = ws;
    this.p2p = p2p;
    _currentNodeIp = nodeIp.split(':')[0];

    LogService.log('ChatProvider.configure: nodeIp=$nodeIp');

    p2p?.onMessage.listen((data) {
      addExternalMessage(data);
    });

    // Автоматический запуск libp2p при старте
    _startLibP2P();
  }

  /// Запускает libp2p узел в фоне
  Future<void> _startLibP2P() async {
    if (_libp2pStarted) return;

    LogService.log('libp2p: попытка запуска...');

    try {
      final ethHash = EthicsService.ethicsHash;
      LogService.log('libp2p: ethHash=${ethHash.length > 0 ? "да" : "нет"}');

      final prefs = await SharedPreferences.getInstance();
      final bootstrapPeers = prefs.getString('bootstrap_peers') ?? '';
      LogService.log('libp2p: bootstrapPeers=${bootstrapPeers.isNotEmpty ? bootstrapPeers : "нет"}');

      final result = await LibP2PService.start(
        ethHash: ethHash,
        bootstrapPeers: bootstrapPeers,
        enableMDNS: false,
      );

      if (result.containsKey('error')) {
        LogService.log('libp2p: ОШИБКА запуска: ${result['error']}');
        _libp2pAvailable = false;
        notifyListeners();
        return;
      }

      _libp2pStarted = true;
      _libp2pAvailable = true;
      LogService.log('libp2p: запущен успешно');

      // Получаем статус для PeerID
      final status = await LibP2PService.getStatus();
      _libp2pPeerId = status['id'] ?? '';
      final peerIdStr = _libp2pPeerId.isNotEmpty && _libp2pPeerId.length > 16
          ? _libp2pPeerId.substring(0, 16)
          : _libp2pPeerId.isEmpty ? 'не получен' : _libp2pPeerId;
      LogService.log('libp2p: PeerID=$peerIdStr');
      LogService.log('libp2p: пиры=${status['peers'] ?? 0}');
      LogService.log('libp2p: память=${status['memory'] ?? 0}');

      notifyListeners();
    } catch (e) {
      LogService.log('libp2p: ИСКЛЮЧЕНИЕ при старте: $e');
      _libp2pAvailable = false;
      notifyListeners();
    }
  }

  /// Отправляет сообщение через libp2p (fallback к HTTP)
  Future<bool> sendViaLibP2P(String text) async {
    if (!_libp2pStarted) {
      LogService.log('libp2p отправка: НЕТ (не запущен)');
      return false;
    }

    try {
      LogService.log('libp2p отправка: пробую...');
      final result = await LibP2PService.send(text: text, ttl: _currentTtl);
      if (result.containsKey('error')) {
        LogService.log('libp2p отправка: ОШИБКА: ${result['error']}');
        return false;
      }
      LogService.log('libp2p отправка: ДА (id=${result['message_id']})');
      return true;
    } catch (e) {
      LogService.log('libp2p отправка: ИСКЛЮЧЕНИЕ: $e');
      return false;
    }
  }

  /// Получает сообщения из libp2p (fallback к HTTP)
  Future<List<Message>> getMessagesViaLibP2P() async {
    if (!_libp2pStarted) {
      LogService.log('libp2p загрузка: НЕТ (не запущен)');
      return [];
    }

    try {
      LogService.log('libp2p загрузка: пробую...');
      final data = await LibP2PService.getMessages();
      LogService.log('libp2p загрузка: получено ${data.length} сообщений');
      final result = data.map((json) {
        final map = json as Map<String, dynamic>;
        return Message(
          id: map['id'] ?? DateTime.now().millisecondsSinceEpoch.toString(),
          text: map['text'] ?? '',
          sender: map['sender'] ?? 'P2P',
          time: map['time'] ?? DateTime.now().toIso8601String(),
          isOwn: map['isOwn'] ?? false,
          score: map['score'] ?? 0,
          weight: (map['weight'] as num?)?.toDouble() ?? 0.5,
          archived: map['archived'] ?? false,
          channel: map['channel'] ?? _activeChannel,
          ttl: map['ttl'] ?? 0,
          expiresAt: map['expiresAt'] != null
              ? DateTime.tryParse(map['expiresAt'])
              : null,
        );
      }).toList();
      return result;
    } catch (e) {
      LogService.log('libp2p загрузка: ОШИБКА: $e');
      return [];
    }
  }

  void setCurrentNode(String nodeIp) {
    _currentNodeIp = nodeIp.split(':')[0];
    notifyListeners();
  }

  void setTtl(int ttl) {
    _currentTtl = ttl;
    notifyListeners();
  }

  void initWs() {
    ws.onMessage = (msg) {
      _addMessage(msg);
    };

    ws.onDisconnected = () {
      _wsConnected = false;
      notifyListeners();
    };

    ws.connect();
    _wsConnected = ws.isConnected;
    notifyListeners();
  }

  void addExternalMessage(Map<String, dynamic> data) {
    final msg = Message(
      id: data['id'] ?? DateTime.now().millisecondsSinceEpoch.toString(),
      text: data['text'] ?? '',
      sender: data['sender'] ?? 'P2P',
      time: data['time'] ?? DateTime.now().toIso8601String(),
      isOwn: data['isOwn'] ?? false,
      score: data['score'] ?? 0,
      weight: data['weight'] ?? 0.5,
      archived: data['archived'] ?? false,
      channel: data['channel'] ?? _activeChannel,
      ttl: data['ttl'] ?? 0,
      expiresAt: data['expiresAt'] != null
          ? DateTime.tryParse(data['expiresAt'])
          : null,
    );
    _addMessage(msg);
  }

  String _messageKey(Message msg) {
    final timeKey = msg.time.length >= 16 ? msg.time.substring(0, 16) : msg.time;
    return '${msg.sender}|$timeKey|${msg.text}';
  }

  void _addMessage(Message msg) {
    final key = _messageKey(msg);
    if (!_messagesMap.containsKey(key)) {
      _messagesMap[key] = msg;
      notifyListeners();
    }
  }

  void deleteMessage(String id) {
    _messagesMap.removeWhere((key, msg) => msg.id == id);
    notifyListeners();
  }

  void clearAllMessages() {
    _messagesMap.clear();
    notifyListeners();
  }

  void purgeExpired() {
    final before = _messagesMap.length;
    _messagesMap.removeWhere((key, msg) => msg.isExpired);
    if (_messagesMap.length != before) {
      notifyListeners();
    }
  }

  Future<void> loadMessages() async {
    _loading = true;
    notifyListeners();

    try {
      LogService.log('HTTP загрузка: пробую...');
      final loaded = await api.getMessages();
      LogService.log('HTTP загрузка: ДА (${loaded.length} сообщений)');
      for (final msg in loaded) {
        _addMessage(msg);
      }
    } catch (_) {
      LogService.log('HTTP загрузка: НЕТ (недоступен)');
      if (_libp2pStarted) {
        final fromLibP2P = await getMessagesViaLibP2P();
        for (final msg in fromLibP2P) {
          _addMessage(msg);
        }
      }
    }

    _loading = false;
    notifyListeners();
  }

  Future<void> loadChannels() async {
    try {
      _channels = await api.getChannels();
      notifyListeners();
    } catch (_) {}
  }

  Future<bool> sendMessage(String text) async {
    if (text.trim().isEmpty) return false;

    final ethics = EthicsService.evaluate(text);
    if (!ethics.allowed) {
      _error = 'Сообщение заблокировано этическим фильтром (вес: ${ethics.weight.toStringAsFixed(2)})';
      LogService.log('Этика: ЗАБЛОКИРОВАНО (вес=${ethics.weight.toStringAsFixed(2)})');
      notifyListeners();
      return false;
    }

    final msgId = DateTime.now().millisecondsSinceEpoch.toString();
    final now = DateTime.now().toIso8601String();
    final expiresAt = _currentTtl > 0
        ? DateTime.now().add(Duration(seconds: _currentTtl))
        : null;

    _ownMessageIds.add(msgId);

    final msg = Message(
      id: msgId,
      text: text,
      sender: 'Вы',
      time: now,
      isOwn: true,
      score: 0,
      weight: ethics.weight,
      archived: false,
      channel: _activeChannel,
      ttl: _currentTtl,
      expiresAt: expiresAt,
    );

    _addMessage(msg);

    // Сохраняем своё сообщение в историю P2PService
    p2p?.saveOwnMessage(_currentNodeIp, msg);

    // Пробуем HTTP
    try {
      await api.sendMessage(text, channel: _activeChannel, ttl: _currentTtl, id: msgId);
      LogService.log('HTTP отправка: ДА');
      return true;
    } catch (_) {
      LogService.log('HTTP отправка: НЕТ (недоступен)');
      if (_libp2pStarted) {
        final sent = await sendViaLibP2P(text);
        if (sent) return true;
      }
    }

    LogService.log('Отправка: только локально');
    return true;
  }

  void switchChannel(String channel) {
    if (channel == _activeChannel) return;
    _activeChannel = channel;
    notifyListeners();
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
      sender: target.sender,
      time: target.time,
      isOwn: target.isOwn,
      score: score,
      weight: newWeight,
      archived: target.archived,
      deliveryStatus: target.deliveryStatus,
      channel: target.channel,
      ttl: target.ttl,
      expiresAt: target.expiresAt,
    );

    final key = _messageKey(target);
    _messagesMap[key] = updated;

    try {
      await api.sendFeedback(id, score);
    } catch (_) {}

    notifyListeners();
  }

  /// Останавливает libp2p при завершении
  Future<void> stopLibP2P() async {
    if (_libp2pStarted) {
      LogService.log('libp2p: остановка...');
      await LibP2PService.stop();
      _libp2pStarted = false;
      _libp2pAvailable = false;
      LogService.log('libp2p: остановлен');
      notifyListeners();
    }
  }

  @override
  void dispose() {
    stopLibP2P();
    super.dispose();
  }
}