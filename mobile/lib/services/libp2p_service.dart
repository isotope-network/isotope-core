// mobile/lib/services/libp2p_service.dart
import 'dart:async';
import 'dart:convert';
import 'package:flutter/services.dart';

/// Сервис для работы с libp2p через gomobile (.aar)
class LibP2PService {
  static const MethodChannel _channel = MethodChannel('isotope/libp2p');
  static const EventChannel _messageChannel = EventChannel('isotope/messages');

  static bool _started = false;
  static bool _dhtStarted = false;

  /// Безопасный парсинг JSON-ответа от Go.
  static Map<String, dynamic> _safeDecode(String? response, {String fallback = '{"error":"empty_response"}'}) {
    final raw = response ?? fallback;
    try {
      final decoded = jsonDecode(raw);
      if (decoded is Map<String, dynamic>) {
        return decoded;
      }
      return {'error': 'invalid_response_type', 'raw': raw};
    } on FormatException catch (e) {
      return {'error': 'invalid_json: ${e.message}', 'raw': raw};
    }
  }

  static Future<String> _getStateFilePath() async {
    final filesDir = await _channel.invokeMethod<String>('getFilesDir');
    return '$filesDir/isotope_state.json';
  }

  static Future<List<String>> getCoreLogs() async {
    try {
      final response = await _channel.invokeMethod<String>('getLogs');
      if (response == null || response.isEmpty) return [];
      return response.split('\n');
    } on PlatformException {
      return [];
    }
  }

  static Stream<String> getMessageStream() {
    return _messageChannel.receiveBroadcastStream().map((event) => event as String);
  }

  static Future<Map<String, dynamic>> start({
    required String ethHash,
    String bootstrapPeers = '',
    bool enableMDNS = false,
  }) async {
    if (_started) {
      return {'status': 'already_started'};
    }

    try {
      final response = await _channel.invokeMethod<String>('start', {
        'ethHash': ethHash,
        'bootstrapPeers': bootstrapPeers,
        'enableMDNS': enableMDNS,
      });
      final decoded = _safeDecode(response);
      if (!decoded.containsKey('error')) {
        _started = true;
      }
      return decoded;
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'start'};
    }
  }

  static Future<Map<String, dynamic>> send({
    required String text,
    int ttl = 0,
  }) async {
    try {
      final response = await _channel.invokeMethod<String>('send', {
        'text': text,
        'ttl': ttl,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'send'};
    }
  }

  /// Отправляет сообщение конкретному пиру (E2E-шифрованное).
  /// Требует контакт с x25519_pub.
  static Future<Map<String, dynamic>> sendToPeer({
    required String peerID,
    required String text,
    int ttl = 0,
  }) async {
    try {
      final response = await _channel.invokeMethod<String>('sendToPeer', {
        'peerID': peerID,
        'text': text,
        'ttl': ttl,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'send_to_peer'};
    }
  }

  static Future<List<dynamic>> getMessages() async {
    try {
      final response = await _channel.invokeMethod<String>('getMessages');
      final raw = response ?? '[]';
      try {
        final decoded = jsonDecode(raw);
        return decoded is List ? decoded : [];
      } on FormatException {
        return [];
      }
    } on PlatformException {
      return [];
    }
  }

  static Future<List<dynamic>> getPeers() async {
    try {
      final response = await _channel.invokeMethod<String>('getPeers');
      final raw = response ?? '[]';
      try {
        final decoded = jsonDecode(raw);
        return decoded is List ? decoded : [];
      } on FormatException {
        return [];
      }
    } on PlatformException {
      return [];
    }
  }

  static Future<Map<String, dynamic>> getStatus() async {
    try {
      final response = await _channel.invokeMethod<String>('getStatus');
      return _safeDecode(response, fallback: '{"id":"","peers":0,"memory":0,"layers":0}');
    } on PlatformException {
      return {'id': '', 'peers': 0, 'memory': 0, 'layers': 0};
    }
  }

  static Future<List<String>> getMultiaddrs() async {
    try {
      final response = await _channel.invokeMethod<String>('getMultiaddrs');
      final raw = response ?? '[]';
      try {
        final decoded = jsonDecode(raw);
        if (decoded is List) {
          return decoded.cast<String>();
        }
        return [];
      } on FormatException {
        return [];
      }
    } on PlatformException {
      return [];
    }
  }

  static Future<Map<String, dynamic>> connectToPeer(String multiaddr) async {
    try {
      final response = await _channel.invokeMethod<String>('connectToPeer', {
        'multiaddr': multiaddr,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'connect_to_peer'};
    }
  }

  /// Подключается к пиру, пробуя параллельно все multiaddr.
  /// Логика попыток — на стороне Go (быстрее, без перехода через Dart).
  static Future<Map<String, dynamic>> connectToPeerWithFallback(List<String> multiaddrs) async {
    if (multiaddrs.isEmpty) {
      return {'error': 'empty multiaddrs'};
    }
    try {
      final jsonStr = jsonEncode(multiaddrs);
      final response = await _channel.invokeMethod<String>('connectToPeerWithFallback', {
        'multiaddrsJSON': jsonStr,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'connect_to_peer_fallback'};
    }
  }

  /// Отправляет список своих multiaddr на bootstrap (ANNOUNCE).
  static Future<Map<String, dynamic>> announce(List<String> multiaddrs) async {
    if (multiaddrs.isEmpty) {
      return {'error': 'empty multiaddrs'};
    }
    try {
      final jsonStr = jsonEncode(multiaddrs);
      final response = await _channel.invokeMethod<String>('announce', {
        'multiaddrsJSON': jsonStr,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'announce'};
    }
  }

  /// Ищет multiaddr по PeerID через bootstrap-справочник.
  /// Возвращает {"status":"found","multiaddrs":[...]} или {"error":"..."}.
  static Future<Map<String, dynamic>> findPeerByID(String peerID) async {
    try {
      final response = await _channel.invokeMethod<String>('findPeerByID', {
        'peerID': peerID,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'find_peer_by_id'};
    }
  }

  /// Возвращает JSON для QR-кода версии 1:
  /// {"v":1,"peerID":"Qm...","ed25519_pub":"base64...","x25519_pub":"base64...","signature":""}
  /// Пустая строка — если узел не запущен.
  static Future<String> getMyQRData() async {
    try {
      final response = await _channel.invokeMethod<String>('getMyQRData');
      return response ?? '';
    } on PlatformException {
      return '';
    }
  }

  /// Возвращает Ed25519-публичный ключ (base64).
  /// Пустая строка — если узел не запущен.
  static Future<String> getEd25519PublicKey() async {
    try {
      final response = await _channel.invokeMethod<String>('getEd25519PublicKey');
      return response ?? '';
    } on PlatformException {
      return '';
    }
  }

  /// Возвращает X25519-публичный ключ (base64).
  /// Пустая строка — если узел не запущен.
  static Future<String> getX25519PublicKey() async {
    try {
      final response = await _channel.invokeMethod<String>('getX25519PublicKey');
      return response ?? '';
    } on PlatformException {
      return '';
    }
  }

/// Добавляет или обновляет контакт.
  /// localName — как я называю контакт (не передаётся в сеть).
  /// remoteName — представление контакта о себе (из QR или payload).
  /// readEnabled — сообщил ли контакт, что делится статусом прочтения.
  /// Возвращает {"status":"ok"} или {"error":"..."}.
  static Future<Map<String, dynamic>> addContact({
    required String peerID,
    required String ed25519Pub,
    required String x25519Pub,
    String signature = '',
    String localName = '',
    String remoteName = '',
    bool readEnabled = true,
  }) async {
    try {
      final response = await _channel.invokeMethod<String>('addContact', {
        'peerID': peerID,
        'ed25519Pub': ed25519Pub,
        'x25519Pub': x25519Pub,
        'signature': signature,
        'localName': localName,
        'remoteName': remoteName,
        'readEnabled': readEnabled,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'add_contact'};
    }
  }

  /// Возвращает статусы всех сообщений: {"<msg_id>": 1|2|3, ...}.
  /// 1 — отправлено, 2 — доставлено, 3 — прочитано.
  /// Пустая map — если узел не запущен или статусов нет.
  static Future<Map<String, int>> getMessageStatuses() async {
    try {
      final response = await _channel.invokeMethod<String>('getMessageStatuses');
      final raw = response ?? '{}';
      try {
        final decoded = jsonDecode(raw);
        if (decoded is Map) {
          return decoded.map((k, v) => MapEntry(k.toString(), (v as num).toInt()));
        }
        return {};
      } on FormatException {
        return {};
      }
    } on PlatformException {
      return {};
    }
  }

  /// Устанавливает мою настройку "делюсь ли статусом прочтения".
  /// Сохраняется в isotope_settings.json (Go-ядро).
  static Future<Map<String, dynamic>> setMyReadEnabled(bool enabled) async {
    try {
      final response = await _channel.invokeMethod<String>('setMyReadEnabled', {
        'enabled': enabled,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'set_my_read_enabled'};
    }
  }

  /// Возвращает мою настройку "делюсь ли статусом прочтения".
  static Future<bool> getMyReadEnabled() async {
    try {
      final response = await _channel.invokeMethod<String>('getMyReadEnabled');
      final decoded = _safeDecode(response, fallback: '{"read_enabled":true}');
      return decoded['read_enabled'] as bool? ?? true;
    } on PlatformException {
      return true;
    }
  }

  /// Устанавливает представление по умолчанию.
  /// Используется в QR и [CONTACT_REQUEST], если не переопределено.
  static Future<Map<String, dynamic>> setMyDisplayName(String name) async {
    try {
      final response = await _channel.invokeMethod<String>('setMyDisplayName', {
        'name': name,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'set_my_display_name'};
    }
  }

  /// Возвращает представление по умолчанию.
  static Future<String> getMyDisplayName() async {
    try {
      final response = await _channel.invokeMethod<String>('getMyDisplayName');
      final decoded = _safeDecode(response, fallback: '{"my_display_name":""}');
      return decoded['my_display_name'] as String? ?? '';
    } on PlatformException {
      return '';
    }
  }

  /// Устанавливает TTL по умолчанию (секунды, строкой).
  /// "0" — Вечно.
  static Future<Map<String, dynamic>> setMyTtl(String ttl) async {
    try {
      final response = await _channel.invokeMethod<String>('setMyTtl', {
        'ttl': ttl,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'set_my_ttl'};
    }
  }

  /// Возвращает TTL по умолчанию (секунды, строкой).
  /// "0" — Вечно.
  static Future<String> getMyTtl() async {
    try {
      final response = await _channel.invokeMethod<String>('getMyTtl');
      final decoded = _safeDecode(response, fallback: '{"my_ttl":"0"}');
      return decoded['my_ttl'] as String? ?? '0';
    } on PlatformException {
      return '0';
    }
  }

  /// Устанавливает локальное имя контакта.
  /// Локальное имя — как я называю контакт. Не передаётся в сеть.
  static Future<Map<String, dynamic>> renameContact({
    required String peerID,
    required String localName,
  }) async {
    try {
      final response = await _channel.invokeMethod<String>('renameContact', {
        'peerID': peerID,
        'localName': localName,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'rename_contact'};
    }
  }

  /// Удаляет контакт у меня. У собеседника остаётся.
  static Future<Map<String, dynamic>> removeContact({
    required String peerID,
  }) async {
    try {
      final response = await _channel.invokeMethod<String>('removeContact', {
        'peerID': peerID,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'remove_contact'};
    }
  }

  /// Возвращает список удалённых peerID.
  /// Используется для фильтра _discoveredNodes — удалённые не показываются.
  static Future<List<String>> getDeletedPeers() async {
    try {
      final response = await _channel.invokeMethod<String>('getDeletedPeers');
      final raw = response ?? '[]';
      try {
        final decoded = jsonDecode(raw);
        if (decoded is List) {
          return decoded.map((e) => e.toString()).toList();
        }
        return [];
      } on FormatException {
        return [];
      }
    } on PlatformException {
      return [];
    }
  }

  /// Убирает peerID из списка удалённых.
  /// Вызывается при QR-возврате контакта.
  static Future<Map<String, dynamic>> removeFromDeleted({
    required String peerID,
  }) async {
    try {
      final response = await _channel.invokeMethod<String>('removeFromDeleted', {
        'peerID': peerID,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'remove_from_deleted'};
    }
  }


  /// Возвращает список контактов (JSON-массив).
  static Future<List<dynamic>> getContacts() async {
    try {
      final response = await _channel.invokeMethod<String>('getContacts');
      final raw = response ?? '[]';
      try {
        final decoded = jsonDecode(raw);
        return decoded is List ? decoded : [];
      } on FormatException {
        return [];
      }
    } on PlatformException {
      return [];
    }
  }

  /// Возвращает контакт по PeerID.
  /// {"peerID":"...","ed25519_pub":"...",...} или {"error":"..."}.
  static Future<Map<String, dynamic>> getContact(String peerID) async {
    try {
      final response = await _channel.invokeMethod<String>('getContact', {
        'peerID': peerID,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'get_contact'};
    }
  }

  /// Отправляет [CONTACT_HELLO] получателю.
  /// Открытое (Version=0). Запускает bootstrap-handshake:
  /// получатель ответит [CONTACT_HELLO_ACK] с публичными ключами,
  /// после чего можно слать [CONTACT_REQUEST] (E2E).
  static Future<Map<String, dynamic>> sendContactHello({
    required String peerID,
  }) async {
    try {
      final response = await _channel.invokeMethod<String>('sendContactHello', {
        'peerID': peerID,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'send_contact_hello'};
    }
  }

  /// Отправляет запрос на добавление в контакты.
  /// Возвращает {"status":"ok","id":"..."} или {"error":"..."}.
  static Future<Map<String, dynamic>> sendContactRequest({
    required String peerID,
    String name = '',
  }) async {
    try {
      final response = await _channel.invokeMethod<String>('sendContactRequest', {
        'peerID': peerID,
        'name': name,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'send_contact_request'};
    }
  }

  /// Возвращает список входящих запросов на контакт (JSON-массив).
  static Future<List<dynamic>> getRequests() async {
    try {
      final response = await _channel.invokeMethod<String>('getRequests');
      final raw = response ?? '[]';
      try {
        final decoded = jsonDecode(raw);
        return decoded is List ? decoded : [];
      } on FormatException {
        return [];
      }
    } on PlatformException {
      return [];
    }
  }

  /// Принимает входящий запрос по ID.
  static Future<Map<String, dynamic>> acceptRequestByID(String id) async {
    try {
      final response = await _channel.invokeMethod<String>('acceptRequestByID', {
        'id': id,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'accept_request'};
    }
  }

  /// Отклоняет входящий запрос по ID.
  static Future<Map<String, dynamic>> rejectRequestByID(String id) async {
    try {
      final response = await _channel.invokeMethod<String>('rejectRequestByID', {
        'id': id,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'reject_request'};
    }
  }

  /// Отправляет подтверждение прочтения по msg_id.
  static Future<Map<String, dynamic>> sendRead({
    required String ref,
    required String recipient,
  }) async {
    try {
      final response = await _channel.invokeMethod<String>('sendRead', {
        'ref': ref,
        'recipient': recipient,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'send_read'};
    }
  }

  static Future<Map<String, dynamic>> joinDHT(String bootstrapPeers) async {
    try {
      final response = await _channel.invokeMethod<String>('joinDHT', {
        'bootstrapPeers': bootstrapPeers,
      });
      final decoded = _safeDecode(response);
      if (decoded['status'] == 'joined') {
        _dhtStarted = true;
      }
      return decoded;
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'join_dht'};
    }
  }

  static Future<Map<String, dynamic>> findPeer(String peerID) async {
    try {
      final response = await _channel.invokeMethod<String>('findPeer', {
        'peerID': peerID,
      });
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'find_peer'};
    }
  }

  static Future<Map<String, dynamic>> findPeersViaNetwork() async {
    try {
      final response = await _channel.invokeMethod<String>('findPeersViaNetwork');
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'find_peers_network'};
    }
  }

  static Future<Map<String, dynamic>> provide() async {
    try {
      final response = await _channel.invokeMethod<String>('provide');
      return _safeDecode(response);
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'provide'};
    }
  }

  static Future<Map<String, dynamic>> getDHTInfo() async {
    try {
      final response = await _channel.invokeMethod<String>('getDHTInfo');
      return _safeDecode(response, fallback: '{"started":false}');
    } on PlatformException {
      return {'started': false};
    }
  }

  static Future<Map<String, dynamic>> stop() async {
    if (!_started) {
      return {'status': 'not_started'};
    }

    try {
      final response = await _channel.invokeMethod<String>('stop');
      final decoded = _safeDecode(response);
      if (!decoded.containsKey('error')) {
        _started = false;
        _dhtStarted = false;
      }
      return decoded;
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'stop'};
    }
  }

  static Future<bool> isIgnoringBatteryOptimizations() async {
    try {
      final response = await _channel.invokeMethod<bool>('isIgnoringBatteryOptimizations');
      return response ?? false;
    } on PlatformException {
      return false;
    }
  }

  static Future<String> requestIgnoreBatteryOptimizations() async {
    try {
      final response = await _channel.invokeMethod<String>('requestIgnoreBatteryOptimizations');
      return response ?? 'unknown';
    } on PlatformException catch (e) {
      return 'error: ${e.message}';
    }
  }

  static bool get isStarted => _started;
  static bool get isDHTStarted => _dhtStarted;
}
// mobile/lib/services/libp2p_service.dart