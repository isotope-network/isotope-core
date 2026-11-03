# Дорожная карта ISOTOPE

## Текущий статус

**Версия:** v2.0.6 (seed-фраза, Keystore, локализация)

Ядро — библиотека. Sphinx-маршрутизация. Mixing. Маскировка. Голос/фото/файлы. Seed-фраза. Android Keystore. i18n (en/ru). Разбиение `node.go`. Единый источник истины. Offline-очередь. Уведомления. Разрешения per-action. WakeLock.

---

## Завершено

### Ядро (v1.0–v1.18)
- P2P: libp2p + mDNS + DHT + Gossip
- Priority Gossip
- Ассоциативная память
- WebSocket + TLS
- Обфускация AES-GCM
- Голосовая стеганография
- Onion Routing v2 (legacy)
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
- Backoff reconnect
- Параллельный dial
- TTL 5→8 мин

### E2E-шифрование (v1.24.0)
- X25519 + Ed25519 ключи
- box.Seal, Version=2
- Ed25519 подпись
- isotope_contacts.json
- Recipient, адресная маршрутизация

### UI 1.1–1.3 (v1.25.0)
- Терминология
- Единый вход «Добавить контакт»
- Пустое состояние

### Стабильность и статусы (v1.26.0)
- PlainText
- Автоочистка
- Self-QR блокируется
- Relay: refresh + backoff
- Контакт-протокол (этап 5)
- Onboarding (1.4)
- Подпись Ed25519 (4.4)
- Статусы (1.5, 1.5a)
- Таймер отправки
- UI Настройки
- Рефакторинг D
- VPS skip DHT/reconnectLoop

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
- Умный формат времени

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

### Голос, файлы, фото (v1.31.0)
- Голосовые: запись, E2E, отправка, плеер, сохранение
- Файлы: чанки 64 КБ, сборка, прогресс, батч [DELIVERED]
- Фото: сжатие, миниатюра, fullscreen

### Передача файлов (v1.32.0)
- Чанки 64 КБ, backpressure, retry, идемпотентность
- Resume после перезапуска
- O(1)-индексы (1A), внутренний ack (1B)
- fallback WS↔TCP, gzip state (22x), кэш getFilePath
- TypeFileComplete = 9 (2A)
- WAKE_LOCK permission

### Надёжность и транспорт (v1.33.0)
- O(1)-индексы, FileAck, fallback WS↔TCP, gzip, кэш
- TypeFileComplete, семафор на relay
- Баги 3–7 закрыты

### Галочки файлов (v1.34.0)
- 5 состояний, 4 иконки
- [READ] через mediaID
- Прогресс при hidden

### Превью и шторка (v1.35.0)
- Прогресс в контактах, шторка (3 тумблера)
- Прогресс получателя
- Превью image/video
- unified filesDir

### Sphinx routing (v2.0.1)
- Слоистое шифрование
- Relay знает только следующий hop (полный peer.ID)
- VPS не в цепочке (3 доверенных контакта)
- Payload с длиной (4 байта)

### Mixing (v2.0.2, v2.0.3)
- Флаг в Sphinx-пакете (свойство сообщения)
- Relay спит exponential (mean 5, max 30)
- Текст и файлы

### Маскировка (v2.0.4)
- Все пакеты 64 КБ
- Батч текста (10 = 1 пакет)
- Dummy (10–120 сек, ×5 в фоне)
- Чанк 47 КБ
- Double base64 убран для файлов

### Голос/фото на чанки (v2.0.5)
- isChunkedMediaType (file/voice/photo)
- SendVoiceByPath, SendPhotoByPath
- retryInfo.mediaType
- _VoicePlayer / _PhotoView с localFilePath

### Seed-фраза, Keystore, i18n (v2.0.6)
- BIP39 seed → Ed25519 + X25519
- Android Keystore (CBC, NoPadding)
- i18n en/ru (ключевые экраны)
- Разбиение node.go (198 КБ → 35 КБ + 8 файлов)

---

## СЛЕДУЮЩИЙ ШАГ — БЭКЛОГ

**Приоритет 1 (критично):**
- Согласие на приём файла (право отказа) — архитектура согласована
- «Сохранить» для файлов (MediaStore/SAF)
- «Переслать» — отправить полученный/сохранённый файл другому контакту
- Бэкап приложения (`isotope_media/` + state + contacts + settings)

**Приоритет 2 (важно):**
- Порядок файлов в очереди — последовательный
- pendingDelivered — лимит
- Heap 114 МБ при старте — ленивая загрузка

**Приоритет 3 (мелочи):**
- `_fileLastChunkId`, `last_chunk_id` — удалить
- sendReplicaToPeer — лог только для текстовых

**Приоритет 4 (инфраструктура):**
- Samsung Android 10 — краш (отложено до неопределённого времени)
- DHT Provide — падает (мало пиров, 15+)
- ANNOUNCE-сигнал после seed recovery + бэкап контактов + репликация

---

## В работе / Ближайшие задачи

### Приоритет 1 (следующий шаг)
- 🔜 Согласие на приём файла (право отказа)
- 🔜 «Сохранить» для файлов
- 🔜 «Переслать»
- 🔜 Бэкап приложения

### Приоритет 2
- 🔜 Порядок файлов в очереди
- 🔜 pendingDelivered — лимит
- 🔜 Heap 114 МБ при старте

### Приоритет 3
- 🔜 `_fileLastChunkId`, `last_chunk_id` — удалить
- 🔜 sendReplicaToPeer — лог только для текстовых

### Приоритет 4 (v2.1+)
- 🔜 Адаптивная маскировка (20+ стратегий, Reality, fallback)
- 🔜 BLE + Wi-Fi Direct (офлайн-связь)
- 🔜 Тесты 15+ узлов
- 🔜 ANNOUNCE-сигнал + бэкап контактов
- 🔜 Keylogger Protection (SecureKeyboard)
- 🔜 DHT на мобильном (при 15+ узлах)

---

## Будущие версии

### v2.1 — Офлайн-связь и адаптивная маскировка
- BLE + Wi-Fi Direct
- Адаптивная маскировка (20+ стратегий, Reality, fallback)
- Тесты 15+ узлов
- ANNOUNCE-сигнал + бэкап контактов

### v2.5 — Этический хеш глубже
- Многоязычный этический хеш
- Локальный вес
- Seed-фраза (восстановление PeerID)
- Эмерджентное доверие глубже

### v3.0 — ISOTOPE AI Mesh
- Распределённый инференс ИИ
- Swarm Inference
- Семантическая маршрутизация
- Федеративное обучение
- Анонимный сбор данных
- Этический паспорт моделей
- **Интеллект без хозяина**

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
| Sphinx | Маршрут скрыт от relay | ✅ v2.0.1 |
| Mixing | Тайминги скрыты | ✅ v2.0.2 |
| Маскировка | Тип, объём, паттерн скрыты | ✅ v2.0.4 |
| Метаданные (полные) | Кто с кем | 🔜 v2.0.6+ (адаптивная маскировка) |
| Временны́е паттерны (полные) | Тайминг, объём | 🔜 Padding, mixing |

---

## Отложено

- BLE — нестабилен (в планах, v2.1)
- IPFS для сайта — заблокирован
- Samsung Android 10 — краш (отложено до неопределённого времени)
- DHT Provide — при малом числе пиров (15+)
- TTL для tempContacts (5 мин) — отклонено Хранителем
- [PROFILE_UPDATE] — отклонено (фишинг)
- 6-10 минут подключения Xiaomi — решено fallback WS↔TCP (временно)

---

## Инфраструктура

### VPS bootstrap/relay
- IP: 186.246.31.176
- PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Bootstrap: /ip4/186.246.31.176/tcp/9000/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi (TCP; WS 9001 — fallback, блокируется провайдером Городок; fallback автоматический)
- Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API)
- Роль: временная инфраструктура

### Телефоны (тестовые)
- Актуальные PeerID — в handoff (меняются при сносе приложения).

### Обновление VPS

```bash
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

Сборка .aar
bash
cd /d D:\isotope\node
del isotope.aar
gomobile bind -target=android -androidapi 21 -ldflags "-checklinkname=0" -o isotope.aar ./mobile
copy /y isotope.aar D:\isotope\mobile\android\app\libs\
Сборка APK
bash
cd /d D:\isotope\mobile
flutter build apk --debug
Порядок: правка → проверка → коммит → CI → .aar (если Go) → APK → установка → VPS → тест.

Стратегия
E2E и подписи — v1.24 (закрыто)

UI и стабильность — v1.25–v1.26 (закрыто)

Контакты и идентификация — v1.27 (закрыто)

Право на забвение — v1.28 (закрыто)

Единый источник истины — v1.29 (закрыто)

Уведомления и пробуждение — v1.30 (закрыто)

Голос, файлы, фото — v1.31 (закрыто)

Передача файлов — v1.32 (закрыто)

Надёжность и транспорт — v1.33 (закрыто)

Галочки файлов — v1.34 (закрыто)

Превью и шторка — v1.35 (закрыто)

Sphinx — v2.0.1 (закрыто)

Mixing — v2.0.2–v2.0.3 (закрыто)

Маскировка — v2.0.4 (закрыто)

Голос/фото на чанки — v2.0.5 (закрыто)

Seed, Keystore, i18n — v2.0.6 (закрыто)

Бэклог (сохранить, переслать, бэкап, согласие на приём) — сейчас

Адаптивная маскировка, BLE, тесты 15+ — v2.1

AI Mesh — v3.0

Полная автономия — v4.0

Каждый этап — новый уровень децентрализации. VPS отключается, когда DHT и hole punching закроют его роль.