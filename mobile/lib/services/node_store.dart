import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/node_info.dart';

/// Хранилище списка известных узлов (ключ — PeerID)
class NodeStore {
  static const String _nodesKey = 'isotope_nodes';

  /// Загружает список узлов
  static Future<List<NodeInfo>> loadNodes() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final jsonStr = prefs.getString(_nodesKey);
      if (jsonStr == null || jsonStr.isEmpty) return [];

      final data = json.decode(jsonStr);
      if (data is List) {
        return data
            .map((item) => NodeInfo.fromJson(item as Map<String, dynamic>))
            .toList();
      }
      return [];
    } catch (_) {
      return [];
    }
  }

  /// Сохраняет список узлов
  static Future<void> saveNodes(List<NodeInfo> nodes) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final jsonStr = json.encode(nodes.map((n) => n.toJson()).toList());
      await prefs.setString(_nodesKey, jsonStr);
    } catch (_) {}
  }

  /// Добавляет или обновляет узел (ключ — PeerID)
  static Future<void> upsertNode(NodeInfo node) async {
    final nodes = await loadNodes();
    final index = nodes.indexWhere((n) => n.key == node.key);

    if (index >= 0) {
      nodes[index] = node;
    } else {
      nodes.add(node);
    }

    await saveNodes(nodes);
  }

  /// Обновляет lastSeen для узла
  static Future<void> updateLastSeen(String key) async {
    final nodes = await loadNodes();
    final index = nodes.indexWhere((n) => n.key == key);

    if (index >= 0) {
      nodes[index] = nodes[index].copyWith(lastSeen: DateTime.now());
      await saveNodes(nodes);
    }
  }

  /// Помечает узел как dead
  static Future<void> markDead(String key) async {
    final nodes = await loadNodes();
    final index = nodes.indexWhere((n) => n.key == key);

    if (index >= 0) {
      nodes[index] = nodes[index].copyWith(status: NodeStatus.dead);
      await saveNodes(nodes);
    }
  }

  /// Помечает узел как alive
  static Future<void> markAlive(String key) async {
    final nodes = await loadNodes();
    final index = nodes.indexWhere((n) => n.key == key);

    if (index >= 0) {
      nodes[index] = nodes[index].copyWith(
        status: NodeStatus.alive,
        lastSeen: DateTime.now(),
      );
      await saveNodes(nodes);
    }
  }

  /// Удаляет узел
  static Future<void> removeNode(String key) async {
    final nodes = await loadNodes();
    nodes.removeWhere((n) => n.key == key);
    await saveNodes(nodes);
  }

  /// Удаляет dead-узлы старше 24 часов
  static Future<int> cleanupDeadNodes() async {
    final nodes = await loadNodes();
    final before = nodes.length;

    nodes.removeWhere((n) {
      if (n.status == NodeStatus.dead) {
        final age = DateTime.now().difference(n.lastSeen);
        return age.inHours >= 24;
      }
      return false;
    });

    if (nodes.length != before) {
      await saveNodes(nodes);
    }

    return before - nodes.length;
  }

  /// Очищает все узлы
  static Future<void> clearAll() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_nodesKey);
  }
}