import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/node_info.dart';
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

const String DEFAULT_BOOTSTRAP_ADDR = '/ip4/186.246.31.176/tcp/9001/ws/p2p/QmNmr3YqGD9uKpPCx7W86t7Tc3vrBJF1GbmTAzDQ25Sskx';

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
  String _localPeerId = '';
  String _localMultiaddr = '';
  String _bootstrapPeers = DEFAULT_BOOTSTRAP_ADDR;
  StreamSubscription? _nodeSub;
  StreamSubscription? _ipSub;
  VoidCallback? _chatListener;

  @override
  void initState() {
    super.initState();
    LogService.log('=== ConnectScreen initState ===');

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        final chatProvider = context.read<ChatProvider>();
        chatProvider.initialize(bootstrapPeers: _bootstrapPeers);
      }
    });

    _loadBootstrapPeers();
    _checkSavedNode();
    _listenToP2P();
    _startServer();
    _startNetworkMonitoring();
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

  Future<void> _findPeersViaNetwork() async {
    setState(() {
      _findingNetwork = true;
      _status = 'Поиск узлов через сеть...';
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
        final parts = addrStr.split('/');
        final ipIndex = parts.indexOf('ip4');
        if (ipIndex >= 0 && ipIndex + 1 < parts.length) {
          final ip = parts[ipIndex + 1];
          if (ip != _localIp && !ip.startsWith('127.')) {
            final node = NodeInfo(
              peerID: addrStr.contains('/p2p/') ? addrStr.split('/p2p/').last : '',
              knownMultiaddrs: [addrStr],
              lastSeen: DateTime.now(),
            );
            _addNode(node);
          }
        }
      }

      setState(() {
        _status = 'Найдено узлов: ${_discoveredNodes.length}';
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
        _status = 'Найдено узлов: ${_discoveredNodes.length}';
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

  void _scanNetwork() {
    final p2p = context.read<P2PService>();
    setState(() {
      _scanning = true;
      _discoveredNodes.clear();
      _status = 'Поиск узлов...';
    });

    p2p.clearAllNodes();
    p2p.stopNsdDiscovery();
    p2p.startNsdDiscovery();

    final shortId = _localIp?.replaceAll('.', '') ?? 'node';
    p2p.announceNative('ISOTOPE-$shortId', _localIp ?? '');

    Future.delayed(const Duration(seconds: 3), () {
      if (mounted) {
        setState(() {
          _scanning = false;
          _status = _discoveredNodes.isEmpty ? 'Узлы не найдены' : 'Найдено: ${_discoveredNodes.length}';
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

  Future<void> _connectManual() async {
    final address = _manualController.text.trim();
    if (address.isEmpty) return;
    if (!address.contains(':')) {
      setState(() => _error = 'Формат: IP:порт');
      return;
    }
    await _connectToNode(NodeInfo(
      peerID: '',
      knownMultiaddrs: [address],
      lastSeen: DateTime.now(),
    ));
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
              child: Text('Адреса узла', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
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

  void _showMyQR() {
    final multiaddr = '/ip4/${_localIp ?? '0.0.0.0'}/tcp/9001/ws/p2p/$_localPeerId';
    showDialog(
      context: context,
      builder: (ctx) {
        return AlertDialog(
          title: const Text('Мой адрес (QR)'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 200,
                height: 200,
                color: Colors.white,
                padding: const EdgeInsets.all(8),
                child: CustomPaint(
                  painter: QrPainter(
                    data: multiaddr,
                    version: QrVersions.auto,
                    errorCorrectionLevel: QrErrorCorrectLevel.L,
                    emptyColor: Colors.white,
                  ),
                ),
              ),
              const SizedBox(height: 12),
              Text(multiaddr, style: const TextStyle(fontSize: 10), textAlign: TextAlign.center),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () {
                Clipboard.setData(ClipboardData(text: multiaddr));
                Navigator.pop(ctx);
                ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Multiaddr скопирован')));
              },
              child: const Text('Копировать'),
            ),
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Закрыть')),
          ],
        );
      },
    );
  }

  Future<void> _scanQR() async {
    try {
      final result = await Navigator.push(
        context,
        MaterialPageRoute(builder: (_) => const QRScanScreen()),
      );
      if (result != null && result is String && result.isNotEmpty) {
        final clean = result.trim();
        _multiaddrController.text = clean;
        await _connectViaMultiaddr(clean);
      }
    } catch (e) {
      LogService.log('QRScan: ERROR: $e');
    }
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
        _status = 'Подключено к узлу';
        _error = null;
      });
    } catch (e) {
      setState(() => _error = 'Ошибка: $e');
    }
  }

  void _showManualMultiaddrDialog() {
    showDialog(
      context: context,
      builder: (ctx) {
        return AlertDialog(
          title: const Text('Ввести multiaddr'),
          content: TextField(
            controller: _multiaddrController,
            decoration: const InputDecoration(
              hintText: '/ip4/.../tcp/9001/ws/p2p/Qm...',
              labelText: 'Multiaddr узла',
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
                if (addr.isNotEmpty) _connectViaMultiaddr(addr);
              },
              child: const Text('Подключиться'),
            ),
          ],
        );
      },
    );
  }

  void _openLogs() {
    Navigator.push(context, MaterialPageRoute(builder: (_) => const LogScreen()));
  }

  @override
  void dispose() {
    final chatProvider = context.read<ChatProvider>();
    if (_chatListener != null) {
      chatProvider.removeListener(_chatListener!);
    }
    _nodeSub?.cancel();
    _ipSub?.cancel();
    _networkService.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('ISOTOPE — Подключение'),
        actions: [
          IconButton(icon: const Icon(Icons.qr_code), onPressed: _showMyQR, tooltip: 'Показать мой QR'),
          IconButton(icon: const Icon(Icons.qr_code_scanner), onPressed: _scanQR, tooltip: 'Сканировать QR'),
          IconButton(icon: const Icon(Icons.input), onPressed: _showManualMultiaddrDialog, tooltip: 'Ввести multiaddr'),
          IconButton(icon: const Icon(Icons.settings), onPressed: _showBootstrapDialog, tooltip: 'Bootstrap-адрес'),
          IconButton(icon: const Icon(Icons.article_outlined), onPressed: _openLogs, tooltip: 'Журнал'),
          IconButton(
            icon: _findingNetwork ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)) : const Icon(Icons.public),
            onPressed: (_findingNetwork || _connecting) ? null : _findPeersViaNetwork,
            tooltip: 'Найти через сеть',
          ),
          IconButton(
            icon: _scanning ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)) : const Icon(Icons.refresh),
            onPressed: (_scanning || _connecting) ? null : _scanNetwork,
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
            const Text('Найденные узлы:', style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
            const SizedBox(height: 8),
            Expanded(
              flex: 2,
              child: Consumer<ChatProvider>(
                builder: (_, chatProvider, __) {
                  return ListView.builder(
                    itemCount: _discoveredNodes.length,
                    itemBuilder: (context, index) {
                      final node = _discoveredNodes[index];
                      final unread = chatProvider.unreadCount;
                      final isDead = node.status == NodeStatus.dead;
                      return ListTile(
                        leading: Icon(isDead ? Icons.wifi_off : Icons.router, color: isDead ? Colors.grey : null),
                        title: Text(node.currentAddress, style: TextStyle(color: isDead ? Colors.grey : null)),
                        subtitle: Text(isDead ? 'Недоступен' : 'Узел ISOTOPE'),
                        trailing: unread > 0
                            ? Container(
                                padding: const EdgeInsets.all(6),
                                decoration: BoxDecoration(color: Colors.red, borderRadius: BorderRadius.circular(12)),
                                child: Text('$unread', style: const TextStyle(color: Colors.white, fontSize: 12)),
                              )
                            : null,
                        onTap: _connecting || isDead ? null : () => _connectToNode(node),
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