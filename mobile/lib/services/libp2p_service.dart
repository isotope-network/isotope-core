import 'dart:convert';
import 'package:flutter/services.dart';

/// Сервис для работы с libp2p через gomobile (.aar)
class LibP2PService {
  static const MethodChannel _channel = MethodChannel('isotope/libp2p');

  /// Глобальный флаг — запущен ли узел
  static bool _started = false;

  /// Глобальный флаг — запущен ли DHT
  static bool _dhtStarted = false;

  /// Возвращает путь к файлу состояния (через нативный filesDir)
  static Future<String> _getStateFilePath() async {
    final filesDir = await _channel.invokeMethod<String>('getFilesDir');
    return '$filesDir/isotope_state.json';
  }

  /// Получает логи из ядра
  static Future<List<String>> getCoreLogs() async {
    try {
      final response = await _channel.invokeMethod<String>('getLogs');
      if (response == null || response.isEmpty) return [];
      return response.split('\n');
    } on PlatformException {
      return [];
    }
  }

  /// Запускает узел
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
      final decoded = jsonDecode(response ?? '{"error":"empty_response"}');
      if (!decoded.containsKey('error')) {
        _started = true;
      }
      return decoded;
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'start'};
    }
  }

  /// Отправляет сообщение
  static Future<Map<String, dynamic>> send({
    required String text,
    int ttl = 0,
  }) async {
    try {
      final response = await _channel.invokeMethod<String>('send', {
        'text': text,
        'ttl': ttl,
      });
      return jsonDecode(response ?? '{"error":"empty_response"}');
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'send'};
    }
  }

  /// Получает все сообщения
  static Future<List<dynamic>> getMessages() async {
    try {
      final response = await _channel.invokeMethod<String>('getMessages');
      final decoded = jsonDecode(response ?? '[]');
      return decoded is List ? decoded : [];
    } on PlatformException {
      return [];
    }
  }

  /// Получает список пиров
  static Future<List<dynamic>> getPeers() async {
    try {
      final response = await _channel.invokeMethod<String>('getPeers');
      final decoded = jsonDecode(response ?? '[]');
      return decoded is List ? decoded : [];
    } on PlatformException {
      return [];
    }
  }

  /// Получает статус узла
  static Future<Map<String, dynamic>> getStatus() async {
    try {
      final response = await _channel.invokeMethod<String>('getStatus');
      return jsonDecode(response ?? '{"id":"","peers":0,"memory":0,"layers":0}');
    } on PlatformException {
      return {'id': '', 'peers': 0, 'memory': 0, 'layers': 0};
    }
  }

  /// Получает свои multiaddr
  static Future<List<String>> getMultiaddrs() async {
    try {
      final response = await _channel.invokeMethod<String>('getMultiaddrs');
      final decoded = jsonDecode(response ?? '[]');
      if (decoded is List) {
        return decoded.cast<String>();
      }
      return [];
    } on PlatformException {
      return [];
    }
  }

  /// Подключается к пиру
  static Future<Map<String, dynamic>> connectToPeer(String multiaddr) async {
    try {
      final response = await _channel.invokeMethod<String>('connectToPeer', {
        'multiaddr': multiaddr,
      });
      return jsonDecode(response ?? '{"error":"empty_response"}');
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'connect_to_peer'};
    }
  }

  /// Входит в DHT сеть
  static Future<Map<String, dynamic>> joinDHT(String bootstrapPeers) async {
    try {
      final response = await _channel.invokeMethod<String>('joinDHT', {
        'bootstrapPeers': bootstrapPeers,
      });
      final decoded = jsonDecode(response ?? '{"error":"empty_response"}');
      if (decoded['status'] == 'joined') {
        _dhtStarted = true;
      }
      return decoded;
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'join_dht'};
    }
  }

  /// Ищет пира по PeerID через DHT
  static Future<Map<String, dynamic>> findPeer(String peerID) async {
    try {
      final response = await _channel.invokeMethod<String>('findPeer', {
        'peerID': peerID,
      });
      return jsonDecode(response ?? '{"error":"empty_response"}');
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'find_peer'};
    }
  }

  /// Анонсирует себя в DHT
  static Future<Map<String, dynamic>> provide() async {
    try {
      final response = await _channel.invokeMethod<String>('provide');
      return jsonDecode(response ?? '{"error":"empty_response"}');
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'provide'};
    }
  }

  /// Получает информацию о DHT
  static Future<Map<String, dynamic>> getDHTInfo() async {
    try {
      final response = await _channel.invokeMethod<String>('getDHTInfo');
      return jsonDecode(response ?? '{"started":false}');
    } on PlatformException {
      return {'started': false};
    }
  }

  /// Останавливает узел
  static Future<Map<String, dynamic>> stop() async {
    if (!_started) {
      return {'status': 'not_started'};
    }

    try {
      final response = await _channel.invokeMethod<String>('stop');
      final decoded = jsonDecode(response ?? '{"error":"empty_response"}');
      if (!decoded.containsKey('error')) {
        _started = false;
        _dhtStarted = false;
      }
      return decoded;
    } on PlatformException catch (e) {
      return {'error': e.message ?? 'platform_error', 'operation': 'stop'};
    }
  }

  /// Проверяет, запущен ли узел
  static bool get isStarted => _started;

  /// Проверяет, запущен ли DHT
  static bool get isDHTStarted => _dhtStarted;
}