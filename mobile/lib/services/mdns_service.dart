import 'dart:async';
import 'package:multicast_dns/multicast_dns.dart';

class MDNSService {
  static Future<List<String>> discoverNodes() async {
    final client = MDnsClient();
    await client.start();

    final nodes = <String>{};

    try {
      await for (final PtrResourceRecord ptr in client
          .lookup<PtrResourceRecord>(
            ResourceRecordQuery.serverPointer('_sbicore._tcp.local'),
          )
          .timeout(const Duration(seconds: 5), onTimeout: (sink) => sink.close())) {
        try {
          await for (final SrvResourceRecord srv in client
              .lookup<SrvResourceRecord>(
                ResourceRecordQuery.service(ptr.domainName),
              )
              .timeout(const Duration(seconds: 3), onTimeout: (sink) => sink.close())) {
            nodes.add('${srv.target}:${srv.port}');
          }
        } catch (e) {
          // Пропускаем
        }
      }
    } catch (e) {
      // mDNS не нашёл узлов
    }

    client.stop();
    return nodes.toList();
  }
}