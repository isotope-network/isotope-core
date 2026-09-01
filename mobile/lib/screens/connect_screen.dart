import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../models/node_info.dart';
import '../services/mdns_service.dart';
import '../services/p2p_service.dart';
import '../services/libp2p_service.dart';
import '../services/log_service.dart';
import '../services/ethics_service.dart';
import '../services/identity_service.dart';
import '../services/network_service.dart';
import 'chat_screen.dart';
import 'log_screen.dart';

class ConnectScreen extends StatefulWidget {
  const ConnectScreen({super.key});

  @override
  State<ConnectScreen> createState() => _ConnectScreenState();
}

class _ConnectScreenState extends State<ConnectScreen> {
  final TextEditingController _manualController = TextEditingController(text: 'http://192.168.31.203:8081');
  final NetworkService _networkService = NetworkService();

  List<NodeInfo> _discoveredNodes = [];
  List<NodeInfo> _pendingNodes = [];
  List<String> _logs = [];
  bool _scanning = false;
  bool _connecting = false;
  String? _error;
  String? _status;
  String? _localIp;
  bool _serverStarted = false;
  bool _libp2pStarted = false;
  String _localPeerId = '';
  String _localMultiaddr = '';
  StreamSubscription? _nodeSub;
  StreamSubscription? _ipSub;

  @override
  void initState() {
    super.initState();
    LogService.log('=== ConnectScreen initState ===');
    _checkSavedNode();
    _listenToP2P();
    _startServer();
    _startNetworkMonitoring();
  }

  void _startNetworkMonitoring() {
    LogService.log('Network monitoring: starting...');
    _networkService.startMonitoring();
    _ipSub = _networkService.onIpChanged.listen((newIp) {
      if (newIp != null && newIp != _localIp) {
        LogService.log('Network: IP changed: $_localIp -> $newIp');

        final p2p = context.read<P2PService>();
        if (!p2p.canRestart) {
          LogService.log('Network: restart skipped (debounce)');
          return;
        }

        p2p.markRestart();
        _handleNetworkChange(newIp);
      }
    });
  }

  Future<void> _handleNetworkChange(String newIp) async {
    LogService.log('=== Network change handler: $newIp ===');

    setState(() {
      _localIp = newIp;
      _status = 'Смена сети: $newIp';
    });

    final p2p = context.read<P2PService>();

    LogService.log('Network: stopping announce...');
    await p2p.stopAnnounce();
    LogService.log('Network: stopping NSD discovery...');
    p2p.stopNsdDiscovery();

    if (_libp2pStarted) {
      LogService.log('Network: stopping libp2p...');
      await LibP2PService.stop();
      _libp2pStarted = false;
      LogService.log('Network: libp2p stopped');
    }

    LogService.log('Network: restarting libp2p...');
    await _startLibP2P();

    final shortId = newIp.replaceAll('.', '');
    LogService.log('Network: announcing with new IP: $newIp');
    await p2p.announceNative('ISOTOPE-$shortId', newIp);
    LogService.log('Network: starting NSD discovery...');
    p2p.startNsdDiscovery();

    setState(() {
      _status = 'Телефон-узел: $newIp:8081';
      _error = null;
    });

    LogService.log('=== Network change complete ===');
  }

  Future<void> _startLibP2P() async {
    if (_libp2pStarted) {
      LogService.log('libp2p: already started, skipping');
      return;
    }

    LogService.log('=== libp2p start ===');

    try {
      final ethHash = EthicsService.ethicsHash;
      LogService.log('libp2p: ethHash length=${ethHash.length}');

      final savedPeerId = await IdentityService.getPeerId();
      final savedStr = savedPeerId != null && savedPeerId.length > 16
          ? savedPeerId.substring(0, 16)
          : savedPeerId ?? 'нет';
      LogService.log('libp2p: saved PeerID=$savedStr');

      final listenIP = _localIp ?? '0.0.0.0';
      LogService.log('libp2p: listenIP=$listenIP');

      final result = await LibP2PService.start(
        ethHash: ethHash,
        bootstrapPeers: '',
        port: 0,
        listenIP: listenIP,
      );
      final resultStr = result.toString();
      LogService.log('libp2p: start result=${resultStr.length > 100 ? resultStr.substring(0, 100) : resultStr}');

      if (result.containsKey('error')) {
        LogService.log('libp2p: ERROR: ${result['error']}');
        if (mounted) setState(() {});
        return;
      }

      _libp2pStarted = true;
      LogService.log('libp2p: started successfully');

      final status = await LibP2PService.getStatus();
      _localPeerId = status['id'] ?? '';
      final peerIdStr = _localPeerId.isNotEmpty && _localPeerId.length > 16
          ? _localPeerId.substring(0, 16)
          : _localPeerId.isEmpty ? 'не получен' : _localPeerId;
      LogService.log('libp2p: PeerID=$peerIdStr');
      LogService.log('libp2p: peers=${status['peers']}');
      LogService.log('libp2p: memory=${status['memory']}');

      if (_localPeerId.isNotEmpty && savedPeerId != _localPeerId) {
        await IdentityService.savePeerId(_localPeerId);
        LogService.log('libp2p: PeerID saved to secure storage');
      } else if (_localPeerId.isNotEmpty && savedPeerId == _localPeerId) {
        LogService.log('libp2p: PeerID matches saved (restored)');
      }

      final multiaddrs = await LibP2PService.getMultiaddrs();
      LogService.log('libp2p: multiaddrs count=${multiaddrs.length}');
      if (multiaddrs.isNotEmpty) {
        for (final addr in multiaddrs) {
          if (addr.contains('/ws/')) {
            _localMultiaddr = addr;
            break;
          }
        }
        if (_localMultiaddr.isEmpty) {
          _localMultiaddr = multiaddrs.first;
        }
        final mStr = _localMultiaddr;
        LogService.log('libp2p: multiaddr=${mStr.length > 60 ? mStr.substring(0, 60) : mStr}');
      }

      final p2p = context.read<P2PService>();
      p2p.setLocalPeerId(_localPeerId);
      p2p.setLocalMultiaddr(_localMultiaddr);
      LogService.log('libp2p: PeerID and multiaddr set in P2PService');

      // Получаем логи из ядра
      final coreLogs = await LibP2PService.getCoreLogs();
      for (final log in coreLogs) {
        LogService.log('CORE: $log');
      }

      if (mounted) setState(() {});
    } catch (e) {
      LogService.log('libp2p: EXCEPTION: $e');
      if (mounted) setState(() {});
    }
    LogService.log('=== libp2p start complete ===');
  }

  Future<void> _startServer() async {
    LogService.log('=== Start server ===');
    final p2p = context.read<P2PService>();

    if (!_serverStarted) {
      LogService.log('Server: starting HTTP...');
      final ip = await p2p.startHttpServer();
      _serverStarted = true;
      LogService.log('Server: HTTP started, IP=$ip');
      if (mounted) {
        setState(() {
          _localIp = ip;
          if (ip != null) {
            _status = 'Телефон-узел: $ip:8081';
          }
        });
      }

      LogService.log('Server: starting libp2p...');
      await _startLibP2P();

      LogService.log('Server: waiting 5 sec for network stabilization...');
      await Future.delayed(const Duration(seconds: 5));
      LogService.log('Server: network stabilized');
    } else {
      LogService.log('Server: already started, skipping');
    }

    for (final node in _pendingNodes) {
      LogService.log('Server: adding pending node: ${node.currentAddress}');
      _addNode(node);
    }
    _pendingNodes.clear();

    final shortId = _localIp?.replaceAll('.', '') ?? 'node';
    LogService.log('Server: announcing: ISOTOPE-$shortId, IP=${_localIp}');
    await p2p.announceNative('ISOTOPE-$shortId', _localIp ?? '');
    LogService.log('Server: starting NSD discovery...');
    p2p.startNsdDiscovery();
    LogService.log('=== Start server complete ===');
  }

  void _listenToP2P() {
    LogService.log('=== Listen to P2P ===');
    final p2p = context.read<P2PService>();

    _nodeSub = p2p.onNodeFound.listen((node) {
      final peerIdStr = node.peerID.isNotEmpty && node.peerID.length > 12
          ? node.peerID.substring(0, 12)
          : node.peerID.isEmpty ? 'нет' : node.peerID;
      LogService.log('P2P event: node found: ${node.currentAddress} (PeerID: $peerIdStr)');
      if (_localIp == null) {
        if (!_pendingNodes.any((n) => n.key == node.key)) {
          _pendingNodes.add(node);
          LogService.log('P2P: added to pending');
        }
      } else {
        _addNode(node);
      }
    });

    p2p.onLog.listen((log) {
      if (mounted) {
        setState(() {
          _logs.add(log);
        });
      }
    });

    p2p.onUnread.listen((senderIp) {
      LogService.log('P2P event: unread from $senderIp');
      if (mounted) {
        setState(() {});
      }
    });
    LogService.log('=== Listen to P2P complete ===');
  }

  void _addNode(NodeInfo node) {
    if (_localIp == null) return;

    if (node.currentAddress.startsWith('BLE:')) return;
    if (node.currentAddress.contains(_localIp!)) return;
    if (node.currentAddress.startsWith('127.') || node.currentAddress.startsWith('localhost')) return;

    if (mounted) {
      final existingIndex = _discoveredNodes.indexWhere((n) => n.key == node.key);
      if (existingIndex >= 0) {
        LogService.log('UI: updating node: ${node.currentAddress}');
        _discoveredNodes[existingIndex] = node;
      } else {
        LogService.log('UI: adding node: ${node.currentAddress}');
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
    final prefs = await SharedPreferences.getInstance();
    final saved = prefs.getString('node_address');
    LogService.log('Saved node address: ${saved ?? "нет"}');
    if (saved != null && saved.isNotEmpty && mounted) {
      setState(() {
        _status = 'Подключение к $saved...';
        _connecting = true;
      });
      _connecting = false;
    }
  }

  void _scanNetwork() {
    LogService.log('=== Manual scan ===');
    final p2p = context.read<P2PService>();

    setState(() {
      _scanning = true;
      _connecting = false;
      _error = null;
      _status = 'Поиск узлов...';
      _discoveredNodes.clear();
      LogService.log('Scan: UI list cleared');
    });

    LogService.log('Scan: clearing all nodes...');
    p2p.clearAllNodes();
    LogService.log('Scan: stopping NSD...');
    p2p.stopNsdDiscovery();
    LogService.log('Scan: starting NSD...');
    p2p.startNsdDiscovery();

    final shortId = _localIp?.replaceAll('.', '') ?? 'node';
    LogService.log('Scan: announcing...');
    p2p.announceNative('ISOTOPE-$shortId', _localIp ?? '');

    Future.delayed(const Duration(seconds: 3), () {
      if (mounted) {
        setState(() {
          _scanning = false;
          if (_discoveredNodes.isEmpty) {
            _status = 'Узлы не найдены';
            _error = 'Подождите или нажмите 🔄';
            LogService.log('Scan: no nodes found');
          } else {
            _status = 'Найдено узлов: ${_discoveredNodes.length}';
            LogService.log('Scan: found ${_discoveredNodes.length} nodes');
          }
        });
      }
    });
    LogService.log('=== Manual scan initiated ===');
  }

  Future<void> _connectToNode(NodeInfo node) async {
    LogService.log('=== Connect to node: ${node.currentAddress} ===');
    final p2p = context.read<P2PService>();

    setState(() {
      _connecting = true;
      _status = 'Подключение к ${node.currentAddress}...';
      _error = null;
    });

    var clean = node.currentAddress.trim();
    clean = clean.replaceFirst('http://', '');
    clean = clean.replaceFirst('https://', '');
    clean = clean.replaceFirst('ws://', '');
    clean = clean.replaceFirst('wss://', '');
    LogService.log('Connect: clean address=$clean');

    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('node_address', clean);
    LogService.log('Connect: saved to prefs');

    if (mounted) {
      setState(() {
        _connecting = false;
        _status = null;
      });

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
        setState(() {});
      }
    }
  }

  Future<void> _connectManual() async {
    final address = _manualController.text.trim();
    if (address.isEmpty) return;

    if (!address.contains(':')) {
      setState(() => _error = 'Формат: IP:порт (например 192.168.1.5:8081)');
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
              child: Text(
                'Адреса узла',
                style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
              ),
            ),
            ...node.addresses.map((addr) {
              final isCurrent = addr == node.currentAddress;
              return ListTile(
                leading: Icon(
                  isCurrent ? Icons.check_circle : Icons.circle_outlined,
                  color: isCurrent ? Colors.green : Colors.grey,
                ),
                title: Text(addr),
                subtitle: isCurrent ? Text('Текущий') : null,
                onTap: () {
                  Navigator.pop(ctx);
                  _connectToNode(NodeInfo(
                    peerID: node.peerID,
                    knownMultiaddrs: [addr],
                    lastSeen: DateTime.now(),
                  ));
                },
              );
            }),
          ],
        );
      },
    );
  }

  void _openLogs() {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => const LogScreen(),
      ),
    );
  }

  @override
  void dispose() {
    LogService.log('=== ConnectScreen dispose ===');
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
          IconButton(
            icon: const Icon(Icons.article_outlined),
            onPressed: _openLogs,
            tooltip: 'Журнал',
          ),
          IconButton(
            icon: _scanning
                ? const SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.refresh),
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
                decoration: BoxDecoration(
                  color: Colors.green.shade50,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  '📍 Ваш адрес: $_localIp:8081',
                  style: const TextStyle(fontWeight: FontWeight.bold),
                  textAlign: TextAlign.center,
                ),
              ),
            if (_libp2pStarted)
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: Colors.purple.shade50,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  '🔗 libp2p узел активен${_localPeerId.isNotEmpty ? " (${_localPeerId.length > 12 ? _localPeerId.substring(0, 12) : _localPeerId}...)" : ""}',
                  style: const TextStyle(fontSize: 12, fontWeight: FontWeight.bold),
                  textAlign: TextAlign.center,
                ),
              ),
            if (_localMultiaddr.isNotEmpty)
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: Colors.teal.shade50,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  '📡 ${_localMultiaddr.length > 50 ? _localMultiaddr.substring(0, 50) : _localMultiaddr}...',
                  style: const TextStyle(fontSize: 10),
                  textAlign: TextAlign.center,
                ),
              ),
            if (_status != null)
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.blue.shade50,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  children: [
                    if (_connecting || _scanning)
                      const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      ),
                    const SizedBox(width: 8),
                    Expanded(child: Text(_status!)),
                  ],
                ),
              ),
            if (_error != null)
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.orange.shade50,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(_error!, style: const TextStyle(color: Colors.orange)),
              ),
            const SizedBox(height: 16),
            const Text(
              'Найденные узлы:',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            Expanded(
              flex: 2,
              child: ListView.builder(
                itemCount: _discoveredNodes.length,
                itemBuilder: (context, index) {
                  final node = _discoveredNodes[index];
                  final unread = context.read<P2PService>().getUnreadCount(node.currentAddress);
                  final isDead = node.status == NodeStatus.dead;
                  final hasExtraAddresses = node.addresses.length > 1;

                  return ListTile(
                    leading: Icon(
                      isDead ? Icons.wifi_off : Icons.router,
                      color: isDead ? Colors.grey : null,
                    ),
                    title: Row(
                      children: [
                        Expanded(
                          child: Text(
                            node.currentAddress,
                            style: TextStyle(
                              color: isDead ? Colors.grey : null,
                            ),
                          ),
                        ),
                        if (hasExtraAddresses)
                          InkWell(
                            onTap: () => _showNodeAddresses(node),
                            child: Padding(
                              padding: const EdgeInsets.symmetric(horizontal: 8),
                              child: Text(
                                '+${node.addresses.length - 1}',
                                style: const TextStyle(
                                  color: Colors.blue,
                                  fontWeight: FontWeight.bold,
                                  fontSize: 12,
                                ),
                              ),
                            ),
                          ),
                      ],
                    ),
                    subtitle: Text(
                      isDead ? 'Недоступен' : 'Узел ISOTOPE',
                      style: TextStyle(
                        color: isDead ? Colors.grey : null,
                        fontSize: 12,
                      ),
                    ),
                    trailing: unread > 0
                        ? Container(
                            padding: const EdgeInsets.all(6),
                            decoration: BoxDecoration(
                              color: Colors.red,
                              borderRadius: BorderRadius.circular(12),
                            ),
                            child: Text(
                              '$unread',
                              style: const TextStyle(color: Colors.white, fontSize: 12),
                            ),
                          )
                        : null,
                    onTap: _connecting || isDead ? null : () => _connectToNode(node),
                  );
                },
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              'Журнал:',
              style: TextStyle(fontSize: 14, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 4),
            Expanded(
              flex: 2,
              child: Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: Colors.black87,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: ListView.builder(
                  itemCount: _logs.length,
                  itemBuilder: (context, index) {
                    return Text(
                      _logs[index],
                      style: const TextStyle(color: Colors.greenAccent, fontSize: 11),
                    );
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