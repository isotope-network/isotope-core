import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/services.dart';
import 'package:flutter_blue_plus/flutter_blue_plus.dart';
import 'package:multicast_dns/multicast_dns.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/message.dart';
import '../models/node_info.dart';
import 'ethics_service.dart';
import 'libp2p_service.dart';
import 'node_store.dart';
import 'log_service.dart';

/// P2P-сервис: NSD discovery + HTTP-сервер + управление узлами
class P2PService {
  static const MethodChannel _nsdChannel = MethodChannel('isotope/nsd');
  static const EventChannel _nsdEvents = EventChannel('isotope/nsd/events');

  static const String _serviceName = '_isotope._tcp.local';
  MDnsClient? _mdnsClient;

  final Map<String, NodeInfo> _nodesMap = {};
  List<NodeInfo> get discoveredNodes => _nodesMap.values.toList();

  // Счётчик неудачных пингов для каждого узла
  final Map<String, int> _failedPings = {};

  HttpServer? _httpServer;
  String? _localIp;
  int _httpPort = 8081;
  int get httpPort => _httpPort;

  final Map<String, List<Map<String, dynamic>>> _messageHistory = {};
  final Map<String, int> _unreadCounts = {};

  String _localPeerId = '';
  String _localMultiaddr = '';
  final Set<String> _connectedPeers = {};

  DateTime _lastRestart = DateTime.now();

  final _nodeController = StreamController<NodeInfo>.broadcast();
  Stream<NodeInfo> get onNodeFound => _nodeController.stream;

  final _messageController = StreamController<Map<String, dynamic>>.broadcast();
  Stream<Map<String, dynamic>> get onMessage => _messageController.stream;

  final _unreadController = StreamController<String>.broadcast();
  Stream<String> get onUnread => _unreadController.stream;

  final _recallController = StreamController<String>.broadcast();
  Stream<String> get onRecall => _recallController.stream;

  final _logController = StreamController<String>.broadcast();
  Stream<String> get onLog => _logController.stream;

  final _peerIdController = StreamController<Map<String, String>>.broadcast();
  Stream<Map<String, String>> get onPeerIdFound => _peerIdController.stream;

  StreamSubscription? _nsdSubscription;
  Timer? _heartbeatTimer;
  Timer? _cleanupTimer;
  bool _serverStarted = false;

  Future<int> _findAvailablePort() async {
    for (var port = 8081; port < 8180; port++) {
      try {
        final socket = await ServerSocket.bind(InternetAddress.anyIPv4, port);
        await socket.close();
        return port;
      } catch (_) {}
    }
    return 0;
  }

  Future<String?> startHttpServer() async {
    if (_serverStarted) {
      return _localIp;
    }
    _serverStarted = true;

    await _loadHistory();
    await _loadNodes();

    try {
      _httpPort = await _findAvailablePort();
      _httpServer = await HttpServer.bind(InternetAddress.anyIPv4, _httpPort);
      _localIp = await _getLocalIp();

      _httpServer!.listen((request) async {
        if (request.uri.path == '/status') {
          request.response.headers.contentType = ContentType.json;
          request.response.write('{"status":"ok","peer":"$_localIp","port":$_httpPort}');
          request.response.close();
        } else if (request.uri.path == '/messages') {
          request.response.headers.contentType = ContentType.json;
          final senderIp = request.connectionInfo?.remoteAddress.address ?? 'unknown';
          final history = getHistory(senderIp);
          request.response.write(json.encode(history));
          request.response.close();
        } else if (request.uri.path == '/send' && request.method == 'POST') {
          final body = await utf8.decodeStream(request);
          try {
            final data = json.decode(body);
            final text = data['message'] ?? '';
            final messageId = data['id'] ?? '';
            final senderIp = request.connectionInfo?.remoteAddress.address ?? 'unknown';

            if (text.isNotEmpty) {
              receiveMessage(text, senderIp, messageId: messageId);

              request.response.headers.contentType = ContentType.json;
              request.response.write('{"status":"ok","id":"$messageId"}');
            } else {
              request.response.statusCode = 400;
            }
          } catch (e) {
            request.response.statusCode = 400;
          }
          request.response.close();
        } else if (request.uri.path == '/recall' && request.method == 'POST') {
          final body = await utf8.decodeStream(request);
          try {
            final data = json.decode(body);
            final messageId = data['id'] ?? '';
            final fromIp = request.connectionInfo?.remoteAddress.address ?? 'unknown';

            if (messageId.isNotEmpty) {
              for (final key in _messageHistory.keys.toList()) {
                _messageHistory[key]!.removeWhere((m) => m['id'] == messageId);
              }
              await _saveHistory();

              _recallController.add(messageId);
              _logController.add('RECALL от $fromIp: $messageId');

              request.response.headers.contentType = ContentType.json;
              request.response.write('{"status":"recalled"}');
            } else {
              request.response.statusCode = 400;
            }
          } catch (e) {
            request.response.statusCode = 400;
          }
          request.response.close();
        } else {
          request.response.statusCode = 404;
          request.response.close();
        }
      });

      _logController.add('HTTP-сервер запущен на $_localIp:$_httpPort');
      LogService.log('HTTP-сервер: $_localIp:$_httpPort');

      _startHeartbeat();
      _startCleanup();

      return _localIp;
    } catch (e) {
      _serverStarted = false;
      _logController.add('HTTP-сервер ошибка: $e');
      LogService.log('HTTP-сервер ОШИБКА: $e');
      return null;
    }
  }

  Future<void> _loadNodes() async {
    final nodes = await NodeStore.loadNodes();
    for (final node in nodes) {
      if (node.key.isNotEmpty && !_nodesMap.containsKey(node.key)) {
        _nodesMap[node.key] = node;
        LogService.log('Узлы загружены: ${node.currentAddress} (статус: ${node.status == NodeStatus.alive ? "alive" : "dead"})');
      }
    }
    _logController.add('Узлы загружены: ${_nodesMap.length}');
    LogService.log('Узлы загружены: ${_nodesMap.length}');
  }

  void _startHeartbeat() {
    LogService.log('Heartbeat: запущен (каждые 30 сек)');
    _heartbeatTimer = Timer.periodic(const Duration(seconds: 30), (_) {
      _checkNodesAlive();
    });
  }

  void _startCleanup() {
    _cleanupTimer = Timer.periodic(const Duration(hours: 1), (_) async {
      final removed = await NodeStore.cleanupDeadNodes();
      if (removed > 0) {
        LogService.log('Очистка: удалено dead-узлов: $removed');
      }
    });
  }

  Future<void> _checkNodesAlive() async {
    if (_nodesMap.isEmpty) {
      LogService.log('Heartbeat: нет узлов для проверки');
      return;
    }

    for (final node in _nodesMap.values.toList()) {
      var alive = false;

      for (final addr in node.addresses) {
        if (await _pingNode(addr)) {
          alive = true;
          break;
        }
      }

      if (alive) {
        _failedPings[node.key] = 0;
        await NodeStore.markAlive(node.key);
        if (_nodesMap.containsKey(node.key) && _nodesMap[node.key]!.status == NodeStatus.dead) {
          _nodesMap[node.key] = node.copyWith(status: NodeStatus.alive, lastSeen: DateTime.now());
          LogService.log('Heartbeat: ${node.currentAddress} — alive (восстановлен)');
        }
      } else {
        _failedPings[node.key] = (_failedPings[node.key] ?? 0) + 1;
        final failCount = _failedPings[node.key]!;
        LogService.log('Heartbeat: ${node.currentAddress} — пинг $failCount/3 неудачен');

        if (failCount >= 3) {
          await NodeStore.markDead(node.key);
          if (_nodesMap.containsKey(node.key)) {
            _nodesMap[node.key] = node.copyWith(status: NodeStatus.dead);
            LogService.log('Heartbeat: ${node.currentAddress} — DEAD (3 неудачи)');
          }
        }
      }
    }
  }

  Future<bool> _pingNode(String address) async {
    try {
      final clean = _normalizeKey(address);
      final port = _getPortFromAddress(address);
      final client = HttpClient();
      final request = await client.getUrl(Uri.parse('http://$clean:$port/status'));
      final response = await request.close();
      await response.drain();
      client.close();
      return response.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  int _getPortFromAddress(String address) {
    final parts = address.split(':');
    if (parts.length >= 2) {
      return int.tryParse(parts.last) ?? 8081;
    }
    return 8081;
  }

  void receiveMessage(String text, String sender, {String? messageId}) {
    final key = _normalizeKey(sender);

    // Узел жив — он отправил сообщение
    if (_nodesMap.containsKey(key)) {
      _nodesMap[key] = _nodesMap[key]!.copyWith(
        status: NodeStatus.alive,
        lastSeen: DateTime.now(),
      );
    }
    NodeStore.markAlive(key);
    _failedPings[key] = 0;

    final ethics = EthicsService.evaluate(text);

    final msg = {
      'id': messageId ?? DateTime.now().millisecondsSinceEpoch.toString(),
      'text': text,
      'sender': key,
      'time': DateTime.now().toIso8601String(),
      'isOwn': false,
      'weight': ethics.weight,
      'ttl': 0,
      'expiresAt': null,
    };

    if (!_messageHistory.containsKey(key)) {
      _messageHistory[key] = [];
    }
    _messageHistory[key]!.add(msg);

    _unreadCounts[key] = (_unreadCounts[key] ?? 0) + 1;

    _messageController.add(msg);
    _unreadController.add(key);
    _logController.add('Принято от $key: $text (вес: ${ethics.weight.toStringAsFixed(2)})');
    LogService.log('Принято от $key: $text');

    _saveHistory();
  }

  Future<void> saveOwnMessage(String nodeIp, Message msg) async {
    final key = _normalizeKey(nodeIp);
    if (!_messageHistory.containsKey(key)) {
      _messageHistory[key] = [];
    }
    _messageHistory[key]!.add({
      'id': msg.id,
      'text': msg.text,
      'sender': 'Вы',
      'time': msg.time,
      'isOwn': true,
      'weight': msg.weight,
      'ttl': msg.ttl,
      'expiresAt': msg.expiresAt?.toIso8601String(),
    });
    await _saveHistory();
  }

  Future<String?> _getLocalIp() async {
    try {
      final interfaces = await NetworkInterface.list();

      for (final interface in interfaces) {
        if (interface.name.toLowerCase().contains('wlan')) {
          for (final addr in interface.addresses) {
            if (addr.type == InternetAddressType.IPv4 && !addr.isLoopback) {
              return addr.address;
            }
          }
        }
      }

      for (final interface in interfaces) {
        for (final addr in interface.addresses) {
          if (addr.type == InternetAddressType.IPv4 && !addr.isLoopback) {
            return addr.address;
          }
        }
      }
    } catch (e) {
      _logController.add('IP error: $e');
    }
    return null;
  }

  String _normalizeKey(String address) {
    return address.split(':')[0];
  }

  List<Map<String, dynamic>> getHistory(String nodeIp) {
    final key = _normalizeKey(nodeIp);
    final history = _messageHistory[key] ?? [];
    return _filterActive(history);
  }

  List<Map<String, dynamic>> _filterActive(List<Map<String, dynamic>> messages) {
    final now = DateTime.now();
    return messages.where((m) {
      final exp = m['expiresAt'];
      if (exp == null) return true;
      final expTime = DateTime.tryParse(exp);
      if (expTime == null) return true;
      return now.isBefore(expTime);
    }).toList();
  }

  void purgeExpiredMessages() {
    bool changed = false;
    for (final key in _messageHistory.keys.toList()) {
      final before = _messageHistory[key]!.length;
      _messageHistory[key] = _filterActive(_messageHistory[key]!);
      if (_messageHistory[key]!.length != before) {
        changed = true;
      }
    }
    if (changed) {
      _saveHistory();
    }
  }

  Future<void> _saveHistory() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final jsonStr = json.encode(_messageHistory);
      await prefs.setString('message_history', jsonStr);
    } catch (e) {
      _logController.add('Save history error: $e');
    }
  }

  Future<void> _loadHistory() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final jsonStr = prefs.getString('message_history');
      if (jsonStr != null) {
        final data = json.decode(jsonStr);
        if (data is Map) {
          _messageHistory.clear();
          data.forEach((key, value) {
            if (value is List) {
              _messageHistory[key] = value.cast<Map<String, dynamic>>();
            }
          });
          LogService.log('История загружена: ${_messageHistory.length} узлов');
        }
      }
    } catch (e) {
      _logController.add('Load history error: $e');
    }
  }

  int getUnreadCount(String nodeIp) {
    final key = _normalizeKey(nodeIp);
    return _unreadCounts[key] ?? 0;
  }

  void resetUnread(String nodeIp) {
    final key = _normalizeKey(nodeIp);
    _unreadCounts[key] = 0;
  }

  Future<void> deleteLocalMessage(String nodeIp, String messageId) async {
    final key = _normalizeKey(nodeIp);
    if (_messageHistory.containsKey(key)) {
      _messageHistory[key]!.removeWhere((m) => m['id'] == messageId);
      await _saveHistory();
    }
  }

  Future<void> recallMessage(String nodeIp, String messageId) async {
    try {
      final key = _normalizeKey(nodeIp);
      if (_messageHistory.containsKey(key)) {
        _messageHistory[key]!.removeWhere((m) => m['id'] == messageId);
        await _saveHistory();
      }

      final clean = _normalizeKey(nodeIp);
      final port = _getPortFromAddress(nodeIp);
      final uri = Uri.parse('http://$clean:$port/recall');

      final client = HttpClient();
      final request = await client.postUrl(uri);
      request.headers.contentType = ContentType.json;
      request.write(json.encode({'id': messageId}));
      final response = await request.close();
      await response.drain();
      client.close();

      _recallController.add(messageId);
      _logController.add('RECALL: $messageId (локально удалено, узел уведомлён)');
    } catch (e) {
      _logController.add('RECALL error: $e');
    }
  }

  void setLocalPeerId(String peerId) {
    _localPeerId = peerId;
  }

  void setLocalMultiaddr(String multiaddr) {
    _localMultiaddr = multiaddr;
  }

  String get localPeerId => _localPeerId;
  String get localMultiaddr => _localMultiaddr;

  Future<void> announceNative(String serviceName, String ip) async {
    try {
      final result = await _nsdChannel.invokeMethod('announce', {
        'serviceName': serviceName,
        'ip': ip,
        'peerId': _localPeerId,
        'multiaddr': _localMultiaddr,
        'port': _httpPort,
      });
      LogService.log('NSD: $result (peerId=${_localPeerId.isNotEmpty ? "да" : "нет"}, port=$_httpPort)');
    } catch (e) {
      LogService.log('NSD announce error: $e');
    }
  }

  void startNsdDiscovery() {
    _nsdSubscription?.cancel();
    _nsdSubscription = null;

    _nsdSubscription = _nsdEvents.receiveBroadcastStream().listen(
      (event) {
        if (event is Map) {
          final type = event['event'] as String?;
          if (type == 'found') {
            final address = event['address'] as String?;
            final peerId = event['peerId'] as String?;
            final multiaddr = event['multiaddr'] as String?;

            if (address == null) return;
            if (peerId == null || peerId.isEmpty) return;
            if (_localIp != null && address.contains(_localIp!)) return;

            final nodeInfo = NodeInfo(
              peerID: peerId,
              knownMultiaddrs: [address],
              lastSeen: DateTime.now(),
              status: NodeStatus.alive,
            );

            if (_nodesMap.containsKey(peerId)) {
              final existing = _nodesMap[peerId]!;
              _nodesMap[peerId] = existing.addAddress(address);
              NodeStore.upsertNode(_nodesMap[peerId]!);
              _nodeController.add(_nodesMap[peerId]!);
              _logController.add('NSD обновлён: $address');
              LogService.log('NSD обновлён: $address (PeerID: ${peerId.substring(0, 12)})');
            } else {
              _nodesMap[peerId] = nodeInfo;
              NodeStore.upsertNode(nodeInfo);
              _nodeController.add(nodeInfo);
              _logController.add('NSD найден: $address');
              LogService.log('NSD найден: $address (PeerID: ${peerId.substring(0, 12)})');
            }

            if (multiaddr != null && multiaddr.isNotEmpty) {
              if (_localPeerId.isNotEmpty && multiaddr.contains(_localPeerId)) return;
              if (_connectedPeers.contains(peerId)) return;

              _connectedPeers.add(peerId);
              LogService.log('Multiaddr получен: $multiaddr');

              LibP2PService.connectToPeer(multiaddr).then((result) {
                if (result.containsKey('error')) {
                  LogService.log('libp2p connect: ОШИБКА');
                  _connectedPeers.remove(peerId);
                } else {
                  LogService.log('libp2p connect: ДА');
                }
              });
            }
          }
        }
      },
      onError: (e) {
        LogService.log('NSD error: $e');
      },
    );

    LogService.log('NSD-поиск запущен');
  }

  void stopNsdDiscovery() {
    _nsdSubscription?.cancel();
    _nsdSubscription = null;
  }

  Future<void> stopAnnounce() async {
    try {
      await _nsdChannel.invokeMethod('stopAnnounce');
    } catch (_) {}
  }

  Future<void> clearAllNodes() async {
    _nodesMap.clear();
    _failedPings.clear();
    await NodeStore.clearAll();
    LogService.log('Список узлов очищен');
  }

  /// Добавляет libp2p-пира в список узлов.
  /// Вызывается, когда приходит сообщение от нового пира (например, в LTE-сети,
  /// где NSD не работает). Гарантирует, что пир появится в ConnectScreen.
  void addDiscoveredPeer(String peerID) {
    if (peerID.isEmpty) return;
    if (_localPeerId.isNotEmpty && peerID == _localPeerId) return;
    if (_nodesMap.containsKey(peerID)) return;

    final node = NodeInfo(
      peerID: peerID,
      knownMultiaddrs: ['libp2p://$peerID'],
      lastSeen: DateTime.now(),
      status: NodeStatus.alive,
    );

    _nodesMap[peerID] = node;
    NodeStore.upsertNode(node);
    _nodeController.add(node);
    LogService.log('libp2p: добавлен узел ${peerID.length > 12 ? peerID.substring(0, 12) : peerID} в список');
  }

  bool get canRestart {
    final now = DateTime.now();
    return now.difference(_lastRestart).inSeconds >= 10;
  }

  void markRestart() {
    _lastRestart = DateTime.now();
  }

  void dispose() {
    _mdnsClient?.stop();
    stopNsdDiscovery();
    stopAnnounce();
    _heartbeatTimer?.cancel();
    _cleanupTimer?.cancel();
    _httpServer?.close();
    _nodeController.close();
    _messageController.close();
    _unreadController.close();
    _recallController.close();
    _logController.close();
    _peerIdController.close();
  }
}