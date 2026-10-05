# Дорожная карта ISOTOPE

## Текущий статус

**Версия:** v1.29.0 (единый источник истины)

Ядро — библиотека. Контакт-протокол. Система имён. TTL. Batch [READ]. Message.Status. Message.ReadLocally. Offline-очередь до [DELIVERED]. Multiline input.

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
- Точка входа: node/main/main.go

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
- UI: настройки TTL

### Единый источник истины (v1.29.0)
- Single source of names
- Batch [READ] — Message.Refs
- Message.Status (перенос из MessageStatus)
- Message.ReadLocally
- Offline-очередь: pending до [DELIVERED]
- Multiline input

---

## В работе / Ближайшие задачи

### Приоритет 1 (сейчас)
- 🔜 **[PROFILE_UPDATE]** (Type=9) — смена read_enabled / display_name без QR
- 🔜 **TTL для tempContacts** (5 минут)
- 🔜 **VPS reconnectLoop** — отключить на relay-сервере
- 🔜 **Уведомления системы**

### Приоритет 2
- 🔜 Circuit direct при 15+ узлах
- 🔜 Foreground service — разное поведение Xiaomi / Huawei
- 🔜 Полупрозрачность pending-сообщения
- 🔜 TTL в настройках — перенести из chat_screen
- 🔜 Черновик UI — тап по 📝 → возврат в поле

### Приоритет 3
- 🔜 DHT Provide — падает при малом числе пиров
- 🔜 Samsung Android 10 — краш
- 🔜 ANNOUNCE TTL expired — эпизодически
- 🔜 VPS memory:66 — мусор

### Приоритет 4 (v2.0+)
- 🔜 **Onion-маршрутизация** — защита метаданных
- 🔜 **Обфускация трафика** — после onion
- 🔜 **Padding, mixing** — временны́е паттерны
- 🔜 **BLE** — возрождение
- 🔜 **Hole punching** через /p2p-circuit/
- 🔜 **DHT на мобильном** (при 15+ узлах)

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
- Уведомления системы
- Оптимизация ANNOUNCE

---

## Инфраструктура

### VPS bootstrap/relay
- IP: 186.246.31.176
- PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Bootstrap: /ip4/186.246.31.176/tcp/9001/ws/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi
- Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API)
- Роль: временная инфраструктура (relay для узлов за NAT)

### Телефоны (тестовые)
- Xiaomi: QmTtUkwXmx9vnUSTLCfdPZn6MNaPkFbxTMHjofFzHMfUz5
- Huawei: QmWyqjGzRQS2T2j4vGtet2j1M4hqDhJD99BPoG6QvKZBv6

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
6. **Мелкие UX + инфраструктура** — сейчас
7. **Onion, padding, mixing** — v2.0+
8. **AI Mesh** — v3.0
9. **Полная автономия** — v4.0

Каждый этап — новый уровень децентрализации.
VPS отключается, когда DHT и hole punching закроют его роль.