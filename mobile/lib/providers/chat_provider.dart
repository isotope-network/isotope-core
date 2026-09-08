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
  List<String> get logs => LogService.logs;

  void initialize({String bootstrapPeers = ''}) {
    LogService.log('ChatProvider: initialize() CALLED');
    try {
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
    _currentNodeIp = nodeIp.split(':')[0];

    LogService.log('ChatProvider.configure: nodeIp=$nodeIp');
  }

  void _subscribeToMessages() {
    _messageSub?.cancel();
    _messageSub = LibP2PService.getMessageStream().listen((messageJSON) {
      try {
        final map = jsonDecode(messageJSON) as Map<String, dynamic>;
        final isOwn = map['isOwn'] ?? false;
        final sender = map['sender'] ?? '';
        final replicatedFrom = map['replicatedFrom'] ?? '';

        if (isOwn || sender == '🌐 Сеть' || sender == _libp2pPeerId || replicatedFrom == _libp2pPeerId) {
          return;
        }

        LogService.log('P2P: входящее от $sender: ${map['text']}');
        _unreadCount++;
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

      _safeNotify();
    } catch (e) {
      LogService.log('libp2p: ИСКЛЮЧЕНИЕ при старте: $e');
      _libp2pAvailable = false;
      _safeNotify();
    }
  }

  void _safeNotify() {
    scheduleMicrotask(() {
      notifyListeners();
    });
  }

  Future<bool> sendViaLibP2P(String text) async {
    if (!_libp2pStarted) return false;
    try {
      final result = await LibP2PService.send(text: text, ttl: _currentTtl);
      return !result.containsKey('error');
    } catch (_) {
      return false;
    }
  }

  Future<List<Message>> getMessagesViaLibP2P() async {
    if (!_libp2pStarted) return [];
    try {
      final data = await LibP2PService.getMessages();
      return data.map((json) {
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
          expiresAt: null,
        );
      }).toList();
    } catch (_) {
      return [];
    }
  }

  void setCurrentNode(String nodeIp) {
    _currentNodeIp = nodeIp.split(':')[0];
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
    final msg = Message(
      id: data['id'] ?? DateTime.now().millisecondsSinceEpoch.toString(),
      text: data['text'] ?? '',
      sender: data['sender'] ?? 'P2P',
      time: data['time'] ?? DateTime.now().toIso8601String(),
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
    if (!_messagesMap.containsKey(msg.id)) {
      _messagesMap[msg.id] = msg;
      _safeNotify();
    }
  }

  void deleteMessage(String id) {
    _messagesMap.remove(id);
    _safeNotify();
  }

  void clearAllMessages() {
    _messagesMap.clear();
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
        for (final msg in fromLibP2P) {
          if (msg.sender != '🌐 Сеть' && msg.sender != _libp2pPeerId) {
            if (!_messagesMap.containsKey(msg.id)) {
              _messagesMap[msg.id] = msg;
            }
          }
        }
        LogService.log('Загружено из libp2p: ${fromLibP2P.length}');
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

    final msgId = DateTime.now().millisecondsSinceEpoch.toString();
    final now = DateTime.now().toUtc().toIso8601String();

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
      expiresAt: _currentTtl > 0 ? DateTime.now().add(Duration(seconds: _currentTtl)) : null,
    );

    _addMessage(msg);

    p2p?.saveOwnMessage(_currentNodeIp, msg);

    if (_libp2pStarted) {
      await sendViaLibP2P(text);
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
    _messageSub?.cancel();
    stopLibP2P();
    super.dispose();
  }
}