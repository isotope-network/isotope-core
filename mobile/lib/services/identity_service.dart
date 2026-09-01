import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Сервис для хранения идентичности узла (PeerID)
class IdentityService {
  static const _storage = FlutterSecureStorage();
  static const String _peerIdKey = 'isotope_peer_id';
  static const String _privateKeyKey = 'isotope_private_key';

  /// Сохраняет PeerID
  static Future<void> savePeerId(String peerId) async {
    await _storage.write(key: _peerIdKey, value: peerId);
  }

  /// Загружает PeerID
  static Future<String?> getPeerId() async {
    return await _storage.read(key: _peerIdKey);
  }

  /// Сохраняет приватный ключ (если будет передаваться из ядра)
  static Future<void> savePrivateKey(String privateKey) async {
    await _storage.write(key: _privateKeyKey, value: privateKey);
  }

  /// Загружает приватный ключ
  static Future<String?> getPrivateKey() async {
    return await _storage.read(key: _privateKeyKey);
  }

  /// Проверяет, есть ли уже сохранённая идентичность
  static Future<bool> hasIdentity() async {
    final peerId = await getPeerId();
    return peerId != null && peerId.isNotEmpty;
  }

  /// Удаляет идентичность (для сброса)
  static Future<void> clearIdentity() async {
    await _storage.delete(key: _peerIdKey);
    await _storage.delete(key: _privateKeyKey);
  }
}