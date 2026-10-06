# Дорожная карта ISOTOPE

## Текущий статус

**Версия:** v1.30.0 (уведомления и пробуждение)

Ядро — библиотека. Контакт-протокол. Система имён. TTL. Batch [READ]. Message.Status. Message.ReadLocally. Offline-очередь до [DELIVERED]. Уведомления системы. Разрешения per-action. WakeLock.

---

## Завершено

### Ядро (v1.0–v1.18)
- P2P: libp2p + mDNS + DHT + Gossip
- Priority Gossip
- Ассоциативная память
- WebSocket + TLS
- Обфускация AES-GCM
- Голосовая стеганография
- Onion Routing v2
- TTL
- Локальное шифрование
- Селф-хилинг
- Репликация
- Самоадаптация
- Каналы
- Весовая модель
- Эмерджентное доверие
- Этический хеш
- Нейросеть

### Рефакторинг (v1.18.1)
- package main → package core

### Мобильная версия (v1.19.0)
- libp2p через FFI
- Стабильный PeerID
- Обработка смены сети
- NSD-обнаружение

### Мобильная стабилизация (v1.21.0)
- Единый ID сообщений
- Non-blocking Mobile calls
- Reconnect loop
- Бейдж непрочитанных

### Foreground Service (v1.22.0)
- Foreground Service
- Battery Optimization Whitelist
- Отображение пиров через libp2p

### Relay-circuit (v1.23.0)
- Multi-address ANNOUNCE
- Relay-circuit
- Flush on reconnect

### E2E-шифрование (v1.24.0)
- X25519 + Ed25519 ключи
- box.Seal, Version=2
- Ed25519 подпись
- Backoff reconnect
- Параллельный dial

### UI 1.1–1.3 (v1.25.0)
- Терминология
- Единый вход «Добавить контакт»
- Пустое состояние

### Стабильность (v1.26.0)
- PlainText
- Автоочистка
- Self-QR блокируется
- Relay: refresh + backoff
- Retry loading contacts

### Контакт-протокол и имена (v1.27.0)
- Bootstrap-handshake
- tempContacts
- Push через messageHook
- Симметрия: confirmed
- Система имён: Name / RemoteName / PeerID
- MyDisplayName
- Диалог «Как вас представить?»

### TTL (v1.28.0)
- Периоды: 10s / 30s / 1m / 5m / 15m / 30m / 1h / 4h / 24h / never
- Режимы: hard / after_read
- [TTL_UPDATE] (Type=8)
- Fallback 48 ч
- FLAG_SECURE
- Удаление контакта: тихий отказ
- Per-chat / per-peer

### Единый источник истины (v1.29.0)
- Single source of names
- Batch [READ] — Message.Refs
- Message.Status (перенос из MessageStatus)
- Message.ReadLocally
- Offline-очередь: pending до [DELIVERED]
- Multiline input

### Уведомления и пробуждение (v1.30.0)
- Уведомления системы (Message.SenderName, nameForPeer)
- Канал isotope_messages, importance HIGH
- Тап → Intent → MethodChannel → Dart
- Разрешения per-action
- permission_service.dart
- CAMERA в манифесте явно
- Батарея — диалог с инструкцией
- SnackBar при отказе
- ShowNotificationContent (Settings)
- WakeLock (PARTIAL_WAKE_LOCK)
- Offline-очередь fix (только не-сервисные)

---

## СЛЕДУЮЩИЙ ШАГ — ГОЛОСОВЫЕ И ФАЙЛЫ

**Голосовые сообщения:**

- Запись голоса (Flutter / Kotlin).
- E2E-шифрование аудио.
- Отправка через SendToPeer.
- Отображение в UI (плеер).
- Опционально — стеганография в WAV (v1.14 — фундамент есть).

**Файлы:**

- Выбор файла (image_picker / file_picker).
- E2E-шифрование.
- Отправка через SendToPeer.
- Чанки для больших файлов.
- Прогресс загрузки.

---

## В работе / Ближайшие задачи

### Приоритет 1 (следующий шаг)
- 🔜 **Голосовые сообщения**
- 🔜 **Файлы**
- 🔜 BOTTOM OVERFLOWED — UI-баг в connect_screen
- 🔜 Samsung Android 10 — диагностика
- 🔜 6-10 минут подключения Xiaomi — диагностика (GOLOG_LOG_LEVEL=debug)

### Приоритет 2
- 🔜 Удаление сообщений вручную (долгий тап)
- 🔜 Настройки → «Данные»: очистить, экспорт / импорт ключей
- 🔜 О программе
- 🔜 Поиск по сообщениям
- 🔜 Закреплённые контакты
- 🔜 Архив контактов
- 🔜 Группы контактов

### Приоритет 3
- 🔜 Группировка уведомлений
- 🔜 Звук / вибрация (настройки)
- 🔜 Circuit direct при 15+ узлах
- 🔜 Foreground service — стабильность Xiaomi / Huawei

### Приоритет 4 (v2.0+)
- 🔜 Onion-маршрутизация — защита метаданных
- 🔜 Обфускация трафика — финальная
- 🔜 Padding, mixing — временны́е паттерны
- 🔜 BLE — возрождение
- 🔜 Hole punching через /p2p-circuit/
- 🔜 DHT на мобильном (при 15+ узлах)

---

## Будущие версии

### v2.0 — Телефон как полноценный узел
- libp2p FFI полностью стабилен
- BLE работает
- Foreground Service + Battery Whitelist
- Hole punching через /p2p-circuit/
- DHT на мобильном (при 15+ узлах)
- Мультиплексирование транспортов
- **Условие отключения VPS:** DHT 15+ узлов + hole punching + 2-3 relay-узла

### v3.0 — ISOTOPE AI Mesh
- Распределённый инференс ИИ
- Swarm Inference
- Семантическая маршрутизация
- Федеративное обучение
- Анонимный сбор данных
- Этический паспорт моделей
- **Интеллект без хозяина**

### v3.1 — Развитие AI Mesh
- (внутренняя разработка)

### v4.0 — Полная автономия
- Сеть без интернета
- Mesh-сообщества (isotope.zone)
- Самоорганизация без bootstrap
- Полный иммунитет (100+ узлов)

---

## Уровни защиты

| Уровень | Что защищено | Статус |
|---------|--------------|--------|
| Обфускация | Маскировка трафика | ✅ v1.13 |
| E2E-шифрование | Содержимое | ✅ v1.24 |
| Подпись | Подлинность | ✅ v1.24 |
| Метаданные | Кто с кем | 🔜 v2.0+ |
| Временны́е паттерны | Тайминг | 🔜 |

---

## Отложено

- BLE — нестабилен
- IPFS для сайта — заблокирован
- Samsung Android 10 — краш
- DHT Provide — при малом числе пиров
- TTL для tempContacts (5 мин) — отклонено Хранителем
- [PROFILE_UPDATE] — отклонено (фишинг)
- 6-10 минут подключения Xiaomi — отложено

---

## Инфраструктура

### VPS bootstrap/relay
- IP: 186.246.31.176
- PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Bootstrap: /ip4/186.246.31.176/tcp/9001/ws/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API)
- Роль: временная инфраструктура

### Телефоны (тестовые)
- Xiaomi: QmT4HDmccPSnFw6qngNsAZQ9KGvhbHWR6CmSEHW4HfYeKH (Wi-Fi)
- Huawei: QmYdhBT4wmXcbJYADd3z7aGka3Xcc8k1m2eagmSg876yJ (Wi-Fi / LTE)

### Обновление VPS

pkill -f isotope-node
sleep 1
pgrep -f isotope-node          # пусто
cd /root/isotope-core && git pull
cd /root/isotope-core/node && go build -o isotope-node ./main
nohup env NODE_ID=bootstrap ISOTOPE_PORT=9000 ISOTOPE_HTTP_PORT=8081 ISOTOPE_ENABLE_RELAY=true ./isotope-node > isotope.log 2>&1 &
sleep 3
pgrep -f isotope-node          # один, новый
curl -s http://127.0.0.1:8081/status

НЕ удалять /root/isotope/state/.

### Сборка .aar

cd /d D:\isotope\node
del isotope.aar
gomobile bind -target=android -androidapi 21 -ldflags "-checklinkname=0" -o isotope.aar ./mobile
copy /y isotope.aar D:\isotope\mobile\android\app\libs\

### Сборка APK

cd /d D:\isotope\mobile
flutter build apk --debug

**Порядок:** правка → проверка → коммит → CI → .aar (если Go) → APK → установка → VPS → тест.

---

## Стратегия

1. **E2E и подписи** — v1.24 (закрыто)
2. **UI и стабильность** — v1.25–v1.26 (закрыто)
3. **Контакты и идентификация** — v1.27 (закрыто)
4. **Право на забвение** — v1.28 (закрыто)
5. **Единый источник истины** — v1.29 (закрыто)
6. **Уведомления и пробуждение** — v1.30 (закрыто)
7. **Голосовые и файлы** — сейчас
8. **UX-долг** — после
9. **Инфраструктура** — после
10. **Onion, padding, mixing** — v2.0+
11. **AI Mesh** — v3.0
12. **Полная автономия** — v4.0

Каждый этап — новый уровень децентрализации.
VPS отключается, когда DHT и hole punching закроют его роль.