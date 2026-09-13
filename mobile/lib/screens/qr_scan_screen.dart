import 'dart:async';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

class QRScanScreen extends StatefulWidget {
  const QRScanScreen({super.key});

  @override
  State<QRScanScreen> createState() => _QRScanScreenState();
}

class _QRScanScreenState extends State<QRScanScreen> {
  final MobileScannerController _controller = MobileScannerController(
    detectionSpeed: DetectionSpeed.noDuplicates,
  );

  bool _scanned = false;
  bool _torchOn = false;
  bool _processingFile = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _handleResult(String? value) {
    if (_scanned) return;
    if (value == null || value.isEmpty) return;
    _scanned = true;
    Navigator.pop(context, value);
  }

  Future<void> _toggleTorch() async {
    try {
      await _controller.toggleTorch();
      setState(() => _torchOn = !_torchOn);
    } catch (_) {
      // Фонарик недоступен на этом устройстве
    }
  }

  Future<void> _pickFromFile() async {
    if (_processingFile) return;
    setState(() => _processingFile = true);

    try {
      final picker = ImagePicker();
      final XFile? picked = await picker.pickImage(source: ImageSource.gallery);
      if (picked == null) {
        setState(() => _processingFile = false);
        return;
      }

      // analyzeImage возвращает bool. Результат приходит через стрим controller.barcodes.
      final completer = Completer<String?>();
      final sub = _controller.barcodes.listen((capture) {
        if (capture.barcodes.isNotEmpty) {
          final value = capture.barcodes.first.rawValue;
          if (value != null && value.isNotEmpty && !completer.isCompleted) {
            completer.complete(value);
          }
        }
      });

      try {
        await _controller.analyzeImage(picked.path);
        // Ждём немного — картинка анализируется асинхронно
        final result = await completer.future.timeout(
          const Duration(seconds: 5),
          onTimeout: () => null,
        );

        if (result != null) {
          _handleResult(result);
        } else {
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('QR-код не найден в изображении')),
            );
          }
        }
      } finally {
        await sub.cancel();
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Ошибка: $e')),
        );
      }
    } finally {
      if (mounted) {
        setState(() => _processingFile = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Сканировать QR'),
        actions: [
          IconButton(
            icon: Icon(_torchOn ? Icons.flash_on : Icons.flash_off),
            onPressed: _toggleTorch,
            tooltip: _torchOn ? 'Выключить фонарик' : 'Включить фонарик',
          ),
        ],
      ),
      body: Stack(
        children: [
          // Камера
          MobileScanner(
            controller: _controller,
            onDetect: (capture) {
              if (capture.barcodes.isNotEmpty) {
                _handleResult(capture.barcodes.first.rawValue);
              }
            },
          ),

          // Подсказка
          const Positioned(
            top: 24,
            left: 0,
            right: 0,
            child: Center(
              child: DecoratedBox(
                decoration: BoxDecoration(
                  color: Colors.black54,
                  borderRadius: BorderRadius.all(Radius.circular(8)),
                ),
                child: Padding(
                  padding: EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                  child: Text(
                    'Наведите камеру на QR-код',
                    style: TextStyle(color: Colors.white, fontSize: 13),
                  ),
                ),
              ),
            ),
          ),

          // Кнопка «Из файла»
          Positioned(
            bottom: 32,
            left: 24,
            right: 24,
            child: SafeArea(
              child: ElevatedButton.icon(
                onPressed: _processingFile ? null : _pickFromFile,
                icon: _processingFile
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.image_outlined),
                label: Text(
                  _processingFile ? 'Обработка...' : 'Из файла',
                  style: const TextStyle(fontSize: 15),
                ),
                style: ElevatedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(vertical: 14),
                  backgroundColor: Colors.white,
                  foregroundColor: Colors.black87,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}