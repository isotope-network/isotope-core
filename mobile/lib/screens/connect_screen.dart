// mobile/lib/screens/connect_screen.dart
import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/node_info.dart';
import '../models/message.dart';
import '../services/mdns_service.dart';
import '../services/p2p_service.dart';
import '../services/libp2p_service.dart';
import '../services/log_service.dart';
import '../services/ethics_service.dart';
import '../services/identity_service.dart';
import '../services/network_service.dart';
import '../providers/chat_provider.dart';
import 'chat_screen.dart';
import 'log_screen.dart';
import 'qr_scan_screen.dart';

const String DEFAULT_BOOTSTRAP_ADDR = '/ip4/186.246.31.176/tcp/9001/ws/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi';
const String BOOTSTRAP_PEER_ID = 'QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi';

/// Префикс для QR-кодов ISOTOPE — чтобы отличать от чужих QR.
const String ISOTOPE_QR_PREFIX = 'isotope:';

class ConnectScreen extends StatefulWidget {
  const ConnectScreen({super.key});

  @override
  State<ConnectScreen> createState() => _ConnectScreenState();
}

class _ConnectScreenState extends State<ConnectScreen> {
  final TextEditingController _manualController = TextEditingController(text: 'http://192.168.31.203:8081');
  final TextEditingController _bootstrapController = TextEditingController(text: DEFAULT_BOOTSTRAP_ADDR);
  final TextEditingController _multiaddrController = TextEditingController();
  final NetworkService _networkService = NetworkService();

  List<NodeInfo> _discoveredNodes = [];
  List<NodeInfo> _pendingNodes = [];
  List<String> _logs = [];
  bool _scanning = false;
  bool _connecting = false;
  bool _findingNetwork = false;
  String? _error;
  String? _status;
  String? _localIp;
  bool _serverStarted = false;
  String _localMultiaddr = '';
  String _bootstrapPeers = DEFAULT_BOOTSTRAP_ADDR;
  StreamSubscription? _nodeSub;
  StreamSubscription? _ipSub;
  VoidCallback? _chatListener;
  Timer? _coreLogsTimer;
  final Set<String> _coreLogsSeen = {};
  static const int _maxCoreLogsSeen = 5000;

  String _myPeerId = '';
  bool _announced = false;

  @override
  void initState() {
    super.initState();
    LogService.log('=== ConnectScreen initState ===');

    _initAsync();

    _checkSavedNode();
    _listenToP2P();
    _startServer();
    _startNetworkMonitoring();

    _coreLogsTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      _pullCoreLogs();
    });
    Future.delayed(const Duration(seconds: 3), () => _pullCoreLogs());
  }

  Future<void> _initAsync() async {
    await _loadBootstrapPeers();

    if (!mounted) return;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        final chatProvider = context.read<ChatProvider>();
        final p2p = context.read<P2PService>();
        chatProvider.setP2P(p2p);
        chatProvider.initialize(bootstrapPeers: _bootstrapPeers);
        _checkBatteryOptimization();

        _scheduleAnnounce();
      }
    });
  }

  void _scheduleAnnounce() {
    Future.delayed(const Duration(seconds: 5), () async {
      if (!mounted) return;

      for (int i = 0; i < 6; i++) {
        if (_localIp != null && _localIp!.isNotEmpty && _myPeerId.isNotEmpty) {
          break;
        }
        if (_myPeerId.isEmpty) {
          try {
            final status = await LibP2PService.getStatus();
            _myPeerId = status['id'] as String? ?? '';
          } catch (_) {}
        }
        await Future.delayed(const Duration(seconds: 5));
      }

      if (_localIp != null && _localIp!.isNotEmpty && _myPeerId.isNotEmpty) {
        _sendAnnounce();
      } else {
        LogService.log('ANNOUNCE: не дождались _localIp или PeerID (ip=$_localIp, peer=$_myPeerId)');
      }
    });
  }

  /// Отправить ANNOUNCE на bootstrap (список multiaddr).
  Future<void> _sendAnnounce() async {
    if (_announced) return;
    if (_localIp == null || _localIp!.isEmpty) return;
    if (_myPeerId.isEmpty) return;

    final multiaddrs = <String>[
      '/ip4/$_localIp/tcp/9001/ws/p2p/$_myPeerId',
    ];

    try {
      final result = await LibP2PService.announce(multiaddrs);
      if (result.containsKey('error')) {
        LogService.log('ANNOUNCE: ошибка: ${result['error']}');
        return;
      }
      _announced = true;
      LogService.log('ANNOUNCE: отправлено ${multiaddrs.length} адресов');
    } catch (e) {
      LogService.log('ANNOUNCE: исключение: $e');
    }
  }

  Future<void> _pullCoreLogs() async {
    try {
      final coreLogs = await LibP2PService.getCoreLogs();
      for (final line in coreLogs) {
        if (line.isEmpty) continue;
        if (_coreLogsSeen.contains(line)) continue;
        _coreLogsSeen.add(line);
        LogService.log('CORE: $line');
      }
      // Ограничиваем Set — защита от роста
      if (_coreLogsSeen.length > _maxCoreLogsSeen) {
        _coreLogsSeen.clear();
      }
    } catch (_) {}
  }

  void _syncDiscoveredNodesFromP2P() {
    try {
      final p2p = context.read<P2PService>();
      int added = 0;
      for (final node in p2p.discoveredNodes) {
        if (node.peerID == BOOTSTRAP_PEER_ID) continue;
        if (_discoveredNodes.any((n) => n.key == node.key)) continue;
        _discoveredNodes.add(node);
        added++;
      }
      if (added > 0 && mounted) {
        LogService.log('ConnectScreen: синхронизировано контактов: $added');
        setState(() {
          _status = 'Контактов: ${_discoveredNodes.length}';
        });
      }
    } catch (e) {
      LogService.log('ConnectScreen: sync error: $e');
    }
  }

  Future<void> _checkBatteryOptimization() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final alreadyAsked = prefs.getBool('battery_opt_asked') ?? false;
      if (alreadyAsked) return;

      final ignoring = await LibP2PService.isIgnoringBatteryOptimizations();
      if (ignoring) {
        await prefs.setBool('battery_opt_asked', true);
        return;
      }

      if (!mounted) return;

      await showDialog(
        context: context,
        barrierDismissible: false,
        builder: (ctx) => AlertDialog(
          title: const Text('Фоновая работа'),
          content: const Text(
            'ISOTOPE — это P2P-сеть. Чтобы принимать сообщения, когда приложение свёрнуто, '
            'разрешите работу в фоне.\n\n'
            'Это откроет системные настройки — выберите «Разрешить» или «Не оптимизировать».',
          ),
          actions: [
            TextButton(
              onPressed: () async {
                await prefs.setBool('battery_opt_asked', true);
                if (ctx.mounted) Navigator.pop(ctx);
              },
              child: const Text('Позже'),
            ),
            TextButton(
              onPressed: () async {
                await prefs.setBool('battery_opt_asked', true);
                if (ctx.mounted) Navigator.pop(ctx);
                await LibP2PService.requestIgnoreBatteryOptimizations();
              },
              child: const Text('Разрешить'),
            ),
          ],
        ),
      );
    } catch (e) {
      LogService.log('Battery opt check: $e');
    }
  }

  Future<void> _loadBootstrapPeers() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final saved = prefs.getString('bootstrap_peers') ?? DEFAULT_BOOTSTRAP_ADDR;
      _bootstrapPeers = saved;
      _bootstrapController.text = saved;
      if (mounted) setState(() {});
    } catch (e) {
      LogService.log('Bootstrap: ERROR: $e');
    }
  }

  Future<void> _saveBootstrapPeers(String value) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString('bootstrap_peers', value.trim());
      _bootstrapPeers = value.trim();
    } catch (e) {
      LogService.log('Bootstrap: ERROR: $e');
    }
  }

  bool _isBootstrapPeer(String addr) {
    return addr.contains(BOOTSTRAP_PEER_ID);
  }

  Future<void> _findPeersViaNetwork() async {
    setState(() {
      _findingNetwork = true;
      _status = 'Поиск контактов через сеть...';
      _error = null;
    });

    try {
      final response = await LibP2PService.findPeersViaNetwork();
      if (response.containsKey('error')) {
        setState(() {
          _error = 'Ошибка поиска: ${response['error']}';
          _status = null;
        });
        return;
      }

      final addrs = response['addrs'] as List<dynamic>? ?? [];
      for (final addr in addrs) {
        final addrStr = addr.toString();

        if (_isBootstrapPeer(addrStr)) {
          continue;
        }

        final parts = addrStr.split('/');
        final ipIndex = parts.indexOf('ip4');
        if (ipIndex >= 0 && ipIndex + 1 < parts.length) {
          final ip = parts[ipIndex + 1];
          if (ip != _localIp && !ip.startsWith('127.')) {
            final node = NodeInfo(
              peerID: addrStr.contains('/p2p/') ? addrStr.split('/p2p/').last : '',
              knownMultiaddrs: [addrStr],
              lastSeen: DateTime.now(),
              status: NodeStatus.alive,
            );
            _addNode(node);
          }
        }
      }

      setState(() {
        _status = 'Контактов: ${_discoveredNodes.length}';
        _error = null;
      });
    } catch (e) {
      setState(() {
        _error = 'Ошибка: $e';
        _status = null;
      });
    } finally {
      if (mounted) {
        setState(() => _findingNetwork = false);
      }
    }
  }

  void _startNetworkMonitoring() {
    _networkService.startMonitoring();
    _ipSub = _networkService.onIpChanged.listen((newIp) {
      if (newIp != null && newIp != _localIp) {
        final p2p = context.read<P2PService>();
        if (!p2p.canRestart) return;
        p2p.markRestart();
        _handleNetworkChange(newIp);
      }
    });
  }

  Future<void> _handleNetworkChange(String newIp) async {
    setState(() {
      _localIp = newIp;
      _status = 'Смена сети: $newIp';
    });

    final p2p = context.read<P2PService>();
    await p2p.stopAnnounce();
    p2p.stopNsdDiscovery();

    final shortId = newIp.replaceAll('.', '');
    await p2p.announceNative('ISOTOPE-$shortId', newIp);
    p2p.startNsdDiscovery();

    _announced = false;
    _sendAnnounce();

    setState(() {
      _status = 'Телефон-узел: $newIp:8081';
      _error = null;
    });
  }

  Future<void> _startServer() async {
    final p2p = context.read<P2PService>();

    if (!_serverStarted) {
      final ip = await p2p.startHttpServer();
      _serverStarted = true;
      if (mounted) {
        setState(() {
          _localIp = ip;
          if (ip != null) _status = 'Телефон-узел: $ip:8081';
        });
      }
      await Future.delayed(const Duration(seconds: 5));
    }

    for (final node in _pendingNodes) {
      _addNode(node);
    }
    _pendingNodes.clear();

    final shortId = _localIp?.replaceAll('.', '') ?? 'node';
    await p2p.announceNative('ISOTOPE-$shortId', _localIp ?? '');
    p2p.startNsdDiscovery();

    _syncDiscoveredNodesFromP2P();

    Future.delayed(const Duration(seconds: 10), () {
      if (mounted) p2p.startNsdDiscovery();
    });
  }

  void _listenToP2P() {
    final p2p = context.read<P2PService>();
    final chatProvider = context.read<ChatProvider>();

    _chatListener = () {
      if (mounted) setState(() {});
    };
    chatProvider.addListener(_chatListener!);

    _nodeSub = p2p.onNodeFound.listen((node) {
      if (node.peerID == BOOTSTRAP_PEER_ID) return;

      if (_localIp == null) {
        if (!_pendingNodes.any((n) => n.key == node.key)) {
          _pendingNodes.add(node);
        }
      } else {
        _addNode(node);
      }
    });

    p2p.onLog.listen((log) {
      if (mounted) {
        setState(() => _logs.add(log));
      }
    });

    p2p.onUnread.listen((senderIp) {
      if (mounted) setState(() {});
    });
  }

  void _addNode(NodeInfo node) {
    if (_localIp == null) return;
    if (node.peerID == BOOTSTRAP_PEER_ID) return;
    if (node.currentAddress.startsWith('BLE:')) return;
    if (node.currentAddress.contains(_localIp!)) return;
    if (node.currentAddress.startsWith('127.') || node.currentAddress.startsWith('localhost')) return;

    if (mounted) {
      final existingIndex = _discoveredNodes.indexWhere((n) => n.key == node.key);
      if (existingIndex >= 0) {
        _discoveredNodes[existingIndex] = node;
      } else {
        _discoveredNodes.add(node);
      }
      setState(() {
        _connecting = false;
        _scanning = false;
        _error = null;
        _status = 'Контактов: ${_discoveredNodes.length}';
      });
    }
  }

  Future<void> _checkSavedNode() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final saved = prefs.getString('node_address');
      LogService.log('Saved node address: ${saved ?? "нет"}');
    } catch (e) {
      LogService.log('Saved: ERROR: $e');
    }
  }

  void _scanNearby() {
    final p2p = context.read<P2PService>();
    setState(() {
      _scanning = true;
      _discoveredNodes.clear();
      _status = 'Поиск контактов рядом...';
    });

    p2p.stopNsdDiscovery();
    p2p.startNsdDiscovery();

    final shortId = _localIp?.replaceAll('.', '') ?? 'node';
    p2p.announceNative('ISOTOPE-$shortId', _localIp ?? '');

    _syncDiscoveredNodesFromP2P();

    Future.delayed(const Duration(seconds: 3), () {
      if (mounted) {
        setState(() {
          _scanning = false;
          _status = _discoveredNodes.isEmpty ? 'Нет контактов' : 'Контактов: ${_discoveredNodes.length}';
        });
      }
    });
  }

  Future<void> _connectToNode(NodeInfo node) async {
    final p2p = context.read<P2PService>();
    setState(() => _connecting = true);

    var clean = node.currentAddress.trim();
    clean = clean.replaceFirst('http://', '');
    clean = clean.replaceFirst('https://', '');
    clean = clean.replaceFirst('ws://', '');
    clean = clean.replaceFirst('wss://', '');

    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('node_address', clean);

    if (mounted) {
      setState(() => _connecting = false);

      await Navigator.push(
        context,
        MaterialPageRoute(
          builder: (_) => ChatScreen(
            nodeAddress: clean,
            p2pService: p2p,
          ),
        ),
      );

      if (mounted) {
        final chatProvider = context.read<ChatProvider>();
        chatProvider.setChatOpen(false);
        LogService.log('ConnectScreen: возврат из чата → setChatOpen(false)');
        setState(() {});
      }
    }
  }

  void _showNodeAddresses(NodeInfo node) {
    showModalBottomSheet(
      context: context,
      builder: (ctx) {
        return ListView(
          shrinkWrap: true,
          children: [
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text('Адреса контакта', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
            ),
            ...node.addresses.map((addr) {
              final isCurrent = addr == node.currentAddress;
              return ListTile(
                leading: Icon(isCurrent ? Icons.check_circle : Icons.circle_outlined, color: isCurrent ? Colors.green : Colors.grey),
                title: Text(addr),
                subtitle: isCurrent ? Text('Текущий') : null,
                onTap: () {
                  Navigator.pop(ctx);
                  _connectToNode(NodeInfo(peerID: node.peerID, knownMultiaddrs: [addr], lastSeen: DateTime.now()));
                },
              );
            }),
          ],
        );
      },
    );
  }

  void _showBootstrapDialog() {
    showDialog(
      context: context,
      builder: (ctx) {
        return AlertDialog(
          title: const Text('Bootstrap-адрес'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: _bootstrapController,
                decoration: const InputDecoration(
                  hintText: '/ip4/.../tcp/9001/ws/p2p/Qm...',
                  labelText: 'Multiaddr bootstrap-узла',
                ),
                maxLines: 3,
                minLines: 1,
              ),
              const SizedBox(height: 8),
              const Text('Оставьте пустым, если не нужен.', style: TextStyle(fontSize: 12, color: Colors.grey)),
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Отмена')),
            TextButton(
              onPressed: () {
                _saveBootstrapPeers(_bootstrapController.text);
                Navigator.pop(ctx);
                if (mounted) setState(() {});
              },
              child: const Text('Сохранить'),
            ),
          ],
        );
      },
    );
  }

  Future<void> _showMyQR() async {
    try {
      // Этап 4.1: пробуем получить новый формат QR (JSON с E2E-ключом).
      // Fallback на старый формат — если что-то не так.
      String qrData = '';

      final jsonData = await LibP2PService.getMyQRData();
      if (jsonData.isNotEmpty && !jsonData.contains('"error"') && jsonData.startsWith('{')) {
        qrData = '$ISOTOPE_QR_PREFIX$jsonData';
        LogService.log('QR: используется формат v:1 (JSON с E2E)');
      } else {
        // Fallback: старый формат — только PeerID.
        if (_myPeerId.isEmpty) {
          final status = await LibP2PService.getStatus();
          _myPeerId = status['id'] as String? ?? '';
        }
        if (_myPeerId.isEmpty) {
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('PeerID ещё не готов, подождите')),
            );
          }
          return;
        }
        qrData = '$ISOTOPE_QR_PREFIX$_myPeerId';
        LogService.log('QR: используется старый формат v:0 (только PeerID)');
      }

      if (!mounted) return;
      showDialog(
        context: context,
        builder: (ctx) {
          return AlertDialog(
            title: const Text('Мой QR (контакт)'),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 220,
                  height: 220,
                  color: Colors.white,
                  padding: const EdgeInsets.all(8),
                  child: CustomPaint(
                    painter: QrPainter(
                      data: qrData,
                      version: QrVersions.auto,
                      errorCorrectionLevel: QrErrorCorrectLevel.M,
                      emptyColor: Colors.white,
                    ),
                  ),
                ),
                const SizedBox(height: 12),
                const Text(
                  'Покажите этот QR другу',
                  style: TextStyle(fontSize: 13, fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 6),
                Text(
                  _myPeerId,
                  style: const TextStyle(fontSize: 10, fontFamily: 'monospace'),
                  textAlign: TextAlign.center,
                ),
              ],
            ),
            actions: [
              TextButton(
                onPressed: () {
                  Clipboard.setData(ClipboardData(text: qrData));
                  Navigator.pop(ctx);
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Код скопирован')),
                  );
                },
                child: const Text('Копировать'),
              ),
              TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Закрыть')),
            ],
          );
        },
      );
    } catch (e) {
      LogService.log('QR show error: $e');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Ошибка: $e')),
        );
      }
    }
  }

  Future<void> _scanQR() async {
    try {
      final result = await Navigator.push(
        context,
        MaterialPageRoute(builder: (_) => const QRScanScreen()),
      );
      if (result == null || result is! String || result.isEmpty) return;

      var code = result.trim();

      if (code.startsWith(ISOTOPE_QR_PREFIX)) {
        code = code.substring(ISOTOPE_QR_PREFIX.length);
      }

      String peerId = '';
      String e2ePub = '';

      // Этап 4.1: новый формат — JSON с версией.
      if (code.startsWith('{')) {
        try {
          final json = jsonDecode(code) as Map<String, dynamic>;
          final v = json['v'] as int? ?? 0;
          if (v >= 1) {
            peerId = (json['peerID'] as String?) ?? '';
            e2ePub = (json['e2e_pub'] as String?) ?? '';
            LogService.log('QR: распознан формат v:$v, peerID=$peerId, e2e_pub=${e2ePub.isNotEmpty ? "есть" : "нет"}');
            // TODO 4.3: сохранить e2ePub для будущего E2E-шифрования.
          }
        } catch (e) {
          LogService.log('QR: ошибка парсинга JSON: $e');
        }
      }

      // Обратная совместимость: старый формат — только PeerID.
      if (peerId.isEmpty) {
        if (code.contains('/p2p/')) {
          code = code.split('/p2p/').last;
        }
        code = code.trim();
        final peerIdMatch = RegExp(r'[A-Za-z0-9]+').firstMatch(code);
        if (peerIdMatch != null) {
          peerId = peerIdMatch.group(0)!;
          LogService.log('QR: распознан старый формат, peerID=$peerId');
        }
      }

      if (peerId.isEmpty) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Не удалось распознать PeerID')),
          );
        }
        return;
      }

      if (peerId == _myPeerId) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Это ваш собственный код')),
          );
        }
        return;
      }

      await _findAndConnectByPeerId(peerId);
    } catch (e) {
      LogService.log('QRScan: ERROR: $e');
      setState(() {
        _error = 'Ошибка: $e';
        _connecting = false;
      });
    }
  }

  /// Универсальный поиск по PeerID + подключение через список multiaddr.
  Future<void> _findAndConnectByPeerId(String peerId) async {
    setState(() {
      _connecting = true;
      _status = 'Поиск контакта...';
      _error = null;
    });

    try {
      final findResult = await LibP2PService.findPeerByID(peerId);
      if (findResult.containsKey('error')) {
        setState(() {
          _error = 'Не удалось найти: ${findResult['error']}';
          _status = null;
          _connecting = false;
        });
        return;
      }

      final rawAddrs = findResult['multiaddrs'];
      final multiaddrs = <String>[];
      if (rawAddrs is List) {
        for (final a in rawAddrs) {
          final s = a.toString().trim();
          if (s.isNotEmpty) multiaddrs.add(s);
        }
      }

      if (multiaddrs.isEmpty) {
        setState(() {
          _error = 'Адрес не найден';
          _status = null;
          _connecting = false;
        });
        return;
      }

      final connectResult = await LibP2PService.connectToPeerWithFallback(multiaddrs);
      if (connectResult.containsKey('error')) {
        setState(() {
          _error = 'Ошибка подключения: ${connectResult['error']}';
          _status = null;
          _connecting = false;
        });
        return;
      }

      final usedAddr = connectResult['used'] as String? ?? multiaddrs.first;
      final shortId = peerId.length > 12 ? peerId.substring(0, 12) : peerId;

      final node = NodeInfo(
        peerID: peerId,
        knownMultiaddrs: [usedAddr],
        lastSeen: DateTime.now(),
        status: NodeStatus.alive,
      );
      _addNode(node);

      setState(() {
        _status = 'Подключено к контакту $shortId';
        _error = null;
        _connecting = false;
      });
    } catch (e) {
      LogService.log('FindAndConnect: ERROR: $e');
      setState(() {
        _error = 'Ошибка: $e';
        _connecting = false;
      });
    }
  }

  void _showManualMultiaddrDialog() {
    showDialog(
      context: context,
      builder: (ctx) {
        return AlertDialog(
          title: const Text('Ввести PeerID или multiaddr'),
          content: TextField(
            controller: _multiaddrController,
            decoration: const InputDecoration(
              hintText: 'Qm... или /ip4/.../p2p/Qm...',
              labelText: 'PeerID или multiaddr контакта',
            ),
            maxLines: 3,
            minLines: 1,
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Отмена')),
            TextButton(
              onPressed: () {
                final addr = _multiaddrController.text.trim();
                Navigator.pop(ctx);
                if (addr.isNotEmpty) {
                  if (addr.startsWith('/')) {
                    _connectViaMultiaddr(addr);
                  } else {
                    _findAndConnectByPeerId(addr);
                  }
                }
              },
              child: const Text('Подключиться'),
            ),
          ],
        );
      },
    );
  }

  Future<void> _connectViaMultiaddr(String multiaddr) async {
    final clean = multiaddr.trim();
    try {
      final response = await LibP2PService.connectToPeer(clean);
      if (response.containsKey('error')) {
        setState(() => _error = 'Ошибка подключения: ${response['error']}');
        return;
      }
      setState(() {
        _status = 'Подключено к контакту';
        _error = null;
      });
    } catch (e) {
      setState(() => _error = 'Ошибка: $e');
    }
  }

  void _showAddContactDialog() {
    showModalBottomSheet(
      context: context,
      builder: (ctx) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Padding(
                padding: EdgeInsets.all(16),
                child: Text(
                  'Добавить контакт',
                  style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
                ),
              ),
              ListTile(
                leading: const Icon(Icons.qr_code_scanner),
                title: const Text('Сканировать QR'),
                subtitle: const Text('Наведите камеру или выберите изображение из файла'),
                onTap: () {
                  Navigator.pop(ctx);
                  _scanQR();
                },
              ),
              ListTile(
                leading: const Icon(Icons.wifi_find),
                title: const Text('Найти рядом'),
                subtitle: const Text('Поиск контактов в той же сети (NSD)'),
                onTap: () {
                  Navigator.pop(ctx);
                  _scanNearby();
                },
              ),
              ListTile(
                leading: const Icon(Icons.input),
                title: const Text('Ввести вручную'),
                subtitle: const Text('PeerID или multiaddr контакта'),
                onTap: () {
                  Navigator.pop(ctx);
                  _showManualMultiaddrDialog();
                },
              ),
              const SizedBox(height: 8),
            ],
          ),
        );
      },
    );
  }

  void _openLogs() {
    Navigator.push(context, MaterialPageRoute(builder: (_) => const LogScreen()));
  }

  String _displayName(NodeInfo node, ChatProvider chatProvider) {
    try {
      if (node.peerID.isNotEmpty) {
        final short = node.peerID.length > 12 ? node.peerID.substring(0, 12) : node.peerID;
        return 'Контакт $short';
      }
      return node.currentAddress;
    } catch (_) {
      return 'Контакт';
    }
  }

  String _lastMessagePreview(NodeInfo node, ChatProvider chatProvider) {
    try {
      final peerID = node.peerID;
      if (peerID.isEmpty) return 'Контакт ISOTOPE';
      final last = chatProvider.getLastMessageForPeer(peerID);
      if (last == null) {
        switch (node.status) {
          case NodeStatus.unknown:
            return 'Не проверен';
          case NodeStatus.alive:
            return 'Контакт ISOTOPE';
          case NodeStatus.dead:
            return 'Недоступен';
        }
      }
      final text = last.text.length > 40 ? '${last.text.substring(0, 40)}...' : last.text;
      return text;
    } catch (_) {
      return 'Ошибка';
    }
  }

  String _lastMessageTime(NodeInfo node, ChatProvider chatProvider) {
    try {
      final peerID = node.peerID;
      if (peerID.isEmpty) return '';

      final last = chatProvider.getLastMessageForPeer(peerID);
      if (last == null) return '';

      return last.formattedTime;
    } catch (_) {
      return '';
    }
  }

  IconData _getNodeIcon(NodeStatus status) {
    switch (status) {
      case NodeStatus.unknown:
        return Icons.help_outline;
      case NodeStatus.alive:
        return Icons.router;
      case NodeStatus.dead:
        return Icons.wifi_off;
    }
  }

  Color? _getNodeIconColor(NodeStatus status) {
    switch (status) {
      case NodeStatus.unknown:
        return Colors.grey.shade500;
      case NodeStatus.alive:
        return null;
      case NodeStatus.dead:
        return Colors.grey;
    }
  }

  Color? _getNodeTextColor(NodeStatus status) {
    switch (status) {
      case NodeStatus.unknown:
        return Colors.grey.shade700;
      case NodeStatus.alive:
        return null;
      case NodeStatus.dead:
        return Colors.grey;
    }
  }

  @override
  void dispose() {
    final chatProvider = context.read<ChatProvider>();
    if (_chatListener != null) {
      chatProvider.removeListener(_chatListener!);
    }
    _nodeSub?.cancel();
    _ipSub?.cancel();
    _coreLogsTimer?.cancel();
    _networkService.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('ISOTOPE'),
        actions: [
          IconButton(
            icon: const Icon(Icons.qr_code),
            onPressed: _showMyQR,
            tooltip: 'Показать мой QR',
          ),
          IconButton(
            icon: const Icon(Icons.person_add),
            onPressed: _showAddContactDialog,
            tooltip: 'Добавить контакт',
          ),
          IconButton(
            icon: const Icon(Icons.settings),
            onPressed: _showBootstrapDialog,
            tooltip: 'Bootstrap-адрес',
          ),
          IconButton(
            icon: const Icon(Icons.article_outlined),
            onPressed: _openLogs,
            tooltip: 'Журнал',
          ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (_localIp != null)
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(color: Colors.green.shade50, borderRadius: BorderRadius.circular(8)),
                child: Text('📍 Ваш адрес: $_localIp:8081', style: const TextStyle(fontWeight: FontWeight.bold), textAlign: TextAlign.center),
              ),
            if (_bootstrapPeers.isNotEmpty)
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(color: Colors.orange.shade50, borderRadius: BorderRadius.circular(8)),
                child: Text('🔗 Bootstrap: ${_bootstrapPeers.length > 50 ? _bootstrapPeers.substring(0, 50) : _bootstrapPeers}...', style: const TextStyle(fontSize: 10), textAlign: TextAlign.center),
              ),
            if (_status != null)
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(color: Colors.blue.shade50, borderRadius: BorderRadius.circular(8)),
                child: Row(
                  children: [
                    if (_connecting || _scanning || _findingNetwork) const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2)),
                    const SizedBox(width: 8),
                    Expanded(child: Text(_status!)),
                  ],
                ),
              ),
            if (_error != null)
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(color: Colors.orange.shade50, borderRadius: BorderRadius.circular(8)),
                child: Text(_error!, style: const TextStyle(color: Colors.orange)),
              ),
            const SizedBox(height: 16),
            const Text('Контакты:', style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
            const SizedBox(height: 8),
            Expanded(
              flex: 2,
              child: Consumer<ChatProvider>(
                builder: (_, chatProvider, __) {
                  if (_discoveredNodes.isEmpty) {
                    return Center(
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 24),
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            Icon(Icons.people_outline, size: 64, color: Colors.grey.shade400),
                            const SizedBox(height: 16),
                            const Text(
                              'Нет контактов',
                              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                            ),
                            const SizedBox(height: 8),
                            Text(
                              'Нажмите «Добавить контакт» и покажите QR-код, '
                              'отправьте ссылку или найдите рядом.',
                              style: TextStyle(fontSize: 13, color: Colors.grey.shade700),
                              textAlign: TextAlign.center,
                            ),
                          ],
                        ),
                      ),
                    );
                  }

                  return ListView.builder(
                    itemCount: _discoveredNodes.length,
                    itemBuilder: (context, index) {
                      final node = _discoveredNodes[index];
                      final unread = chatProvider.unreadCount;
                      final displayName = _displayName(node, chatProvider);
                      final lastMessage = _lastMessagePreview(node, chatProvider);
                      final lastTime = _lastMessageTime(node, chatProvider);

                      return ListTile(
                        leading: Icon(
                          _getNodeIcon(node.status),
                          color: _getNodeIconColor(node.status),
                        ),
                        title: Text(
                          displayName,
                          style: TextStyle(color: _getNodeTextColor(node.status)),
                        ),
                        subtitle: Text(
                          lastMessage,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(fontSize: 13),
                        ),
                        trailing: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          crossAxisAlignment: CrossAxisAlignment.end,
                          children: [
                            if (lastTime.isNotEmpty)
                              Text(
                                lastTime,
                                style: const TextStyle(fontSize: 11, color: Colors.grey),
                              ),
                            if (unread > 0)
                              Container(
                                margin: const EdgeInsets.only(top: 4),
                                padding: const EdgeInsets.all(6),
                                decoration: BoxDecoration(color: Colors.red, borderRadius: BorderRadius.circular(12)),
                                child: Text('$unread', style: const TextStyle(color: Colors.white, fontSize: 12)),
                              ),
                          ],
                        ),
                        onTap: node.status == NodeStatus.dead || _connecting
                            ? null
                            : () => _connectToNode(node),
                      );
                    },
                  );
                },
              ),
            ),
            const SizedBox(height: 8),
            const Text('Журнал:', style: TextStyle(fontSize: 14, fontWeight: FontWeight.bold)),
            const SizedBox(height: 4),
            Expanded(
              flex: 2,
              child: Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(color: Colors.black87, borderRadius: BorderRadius.circular(8)),
                child: ListView.builder(
                  itemCount: _logs.length,
                  itemBuilder: (context, index) {
                    return Text(_logs[index], style: const TextStyle(color: Colors.greenAccent, fontSize: 11));
                  },
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}