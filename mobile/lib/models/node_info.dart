/// Статус узла
enum NodeStatus {
  alive,
  dead,
}

/// Информация об узле ISOTOPE
class NodeInfo {
  final String peerID;
  final List<String> knownMultiaddrs; // список адресов (до 5)
  final DateTime lastSeen;
  final NodeStatus status;

  static const int maxKnownAddresses = 5;

  NodeInfo({
    required this.peerID,
    List<String>? knownMultiaddrs,
    required this.lastSeen,
    this.status = NodeStatus.alive,
  }) : knownMultiaddrs = knownMultiaddrs ?? [];

  /// Текущий адрес — последний в списке
  String get currentAddress {
    return knownMultiaddrs.isNotEmpty ? knownMultiaddrs.last : '';
  }

  /// Все адреса
  List<String> get addresses => List.unmodifiable(knownMultiaddrs);

  /// Создаёт копию с обновлёнными полями
  NodeInfo copyWith({
    String? peerID,
    List<String>? knownMultiaddrs,
    DateTime? lastSeen,
    NodeStatus? status,
  }) {
    return NodeInfo(
      peerID: peerID ?? this.peerID,
      knownMultiaddrs: knownMultiaddrs ?? this.knownMultiaddrs,
      lastSeen: lastSeen ?? this.lastSeen,
      status: status ?? this.status,
    );
  }

  /// Добавляет новый адрес (без дубликатов, максимум 5)
  NodeInfo addAddress(String address) {
    final newList = List<String>.from(knownMultiaddrs);
    if (!newList.contains(address)) {
      newList.add(address);
      if (newList.length > maxKnownAddresses) {
        newList.removeAt(0); // удаляем самый старый
      }
    }
    return copyWith(knownMultiaddrs: newList);
  }

  /// Сериализация в JSON
  Map<String, dynamic> toJson() {
    return {
      'peerID': peerID,
      'knownMultiaddrs': knownMultiaddrs,
      'lastSeen': lastSeen.millisecondsSinceEpoch,
      'status': status == NodeStatus.alive ? 'alive' : 'dead',
    };
  }

  /// Десериализация из JSON
  factory NodeInfo.fromJson(Map<String, dynamic> json) {
    final knownList = json['knownMultiaddrs'];
    return NodeInfo(
      peerID: json['peerID'] ?? '',
      knownMultiaddrs: knownList is List ? knownList.cast<String>() : null,
      lastSeen: DateTime.fromMillisecondsSinceEpoch(json['lastSeen'] ?? 0),
      status: json['status'] == 'dead' ? NodeStatus.dead : NodeStatus.alive,
    );
  }

  /// Проверяет, мёртв ли узел
  bool get isExpired {
    final age = DateTime.now().difference(lastSeen);
    return age.inHours >= 24;
  }

  /// Ключ — всегда PeerID
  String get key => peerID;

  /// Адрес для отображения — текущий
  String get displayAddress => currentAddress;
}