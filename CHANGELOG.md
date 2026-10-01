# История изменений ISOTOPE

## v1.27.0 (2026-10-01)

«Контакты и идентификация»

### Добавлено
- Контакт-протокол (bootstrap-handshake):
  - [CONTACT_HELLO] → [CONTACT_HELLO_ACK] → [CONTACT_REQUEST] → [CONTACT_ACCEPT]
  - Все шаги через bootstrap (открытые — всегда через bootstrap)
  - tempContacts: временные контакты в памяти для расшифровки [CONTACT_REQUEST]
  - Push-события в UI через messageHook (не polling)
  - Симметрия: обе стороны confirmed: true
- Система имён:
  - Name (локальное) / RemoteName (представление) / PeerID (fallback)
  - UI: Name → RemoteName → PeerID
  - MyDisplayName в Settings — представление по умолчанию
  - QR содержит display_name
  - Диалог «Как вас представить?» при отправке запроса (предзаполнено + выделено)
  - Профиль в настройках → «Ваше имя»
  - Долгий тап на контакте → bottom sheet: Открыть / Переименовать / Удалить
  - Предупреждение о безопасности при запросе

### Изменено
- Разделение транспортов по назначению:
  - Открытые сервисные (HELLO, ACK) → bootstrap
  - E2E (REQUEST, ACCEPT, сообщения) → circuit → bootstrap fallback
- Type в processMessageInternal — с рождения, не пост-правка
- Логи: убраны STATUS POLL, [MULTIADDR], [DIAG], [DHT] Provide

### Коммиты
- 6896800 — [CONTACT_ACCEPT] payload (ключи B, read_enabled)
- bead650 — bootstrap-handshake
- 692418d — [CONTACT_HELLO] с ключами A, tempContact
- 67778cd — Type в processMessageInternal, фильтр Type=6
- a2004db — push [CONTACT_REQUEST] через hook
- d083997 — relay throttle fix
- 849c388 — [CONTACT_HELLO] / ACK через bootstrap
- 36a6af6, 0cb9de2, 2a9083a — viaBootstrap параметр
- 196af37 — [CONTACT_ACCEPT] push → UI
- 156fe0a — чистка логов
- 8e6280c — HEAD (имена, диалоги, переименование, безопасность)

### Следующий шаг
- [PROFILE_UPDATE] — для смены MyDisplayName
- Батч [READ]
- Circuit direct при 15+ узлах
- TTL для tempContacts (5 минут)
- Чистка messageStatus при удалении

---

## v1.26.0 (2026-09-27)

### Добавлено
- PlainText в Message: свои E2E-сообщения сохраняются открытым текстом для UI
- Автоочистка: удаление своих E2E-сообщений без PlainText при первом запуске
- Флаг .e2e_cleanup рядом со state
- Memory.Remove — с чисткой seen
- Relay: refresh reservation on reconnect + 30s loop check
- Relay: exponential backoff для reservation retry (2 → 30 сек)
- UI: отображение plainText для своих E2E-сообщений

### Исправлено
- Свой QR блокируется (три уровня: UI + ядро + автоочистка)
- UI: retry loading contacts from core (гонка с Go startup)
- Upsert nodes on alive-event — статус меняется с unknown на alive

### Коммиты
- d0e9744 — own QR blocked, PlainText for own E2E messages, auto-cleanup
- 3235060 — relay: refresh reservation on reconnect + 30s loop check
- 882c998 — relay: exponential backoff for reservation retry
- abc5f01 — UI: display plainText for own E2E messages
- 6e1d1b9 — UI: retry loading contacts from core
- 2f35fb1 — UI: upsert nodes on alive-event

---

## v1.25.0 (2026-09-25)

### Добавлено
- UI 1.1 — терминология: технические термины убраны из интерфейса
- UI 1.2 — единый вход «Добавить контакт», настройки в bottom sheet
- UI 1.3 — пустое состояние с кнопкой действия

### Коммиты
- 5ebe222 — UI 1.1: remove technical terms from UI
- 4940998 — UI 1.2: single add-contact entry, settings bottom sheet
- 988fe17 — UI 1.3: empty state with action button

---

## v1.24.0 (2026-09-23)

### Добавлено
- E2E 4.1 — генерация ключей, QR v:1 с e2e_pub
- E2E 4.2 — поле Recipient, адресная маршрутизация
- E2E 4.3.1 — Ed25519 + X25519, раздельные ключи, curve25519.X25519
- E2E 4.3.2 — isotope_contacts.json с флагом verified
- E2E 4.3.3 — шифрование Text через box.Seal, Version=2
- E2E 4.4 — Ed25519 подпись над peerID || x25519_pub, флаг verified
- Backoff reconnect: 1 → 30 сек, сброс при успехе
- Параллельный dial в ConnectToPeerWithFallback
- ANNOUNCE TTL 5 → 8 мин, удаление disconnected сразу

### Исправлено
- Логи: обрезка длинных строк (4KB max)
- Извлечение PeerID из multiaddr в configure

### Коммиты
- e54d056, 9c2fc8b, dc82e81, f3aaeab, 3a3b859, 39c97c5
- 1d2e3d3, 57cfeaf, 87524bd, 1a44715, b225a22

---

## v1.23.0 (2026-09-17)

### Добавлено
- Multi-address ANNOUNCE (Слой A)
- Relay-circuit (Слой B)
- Flush on reconnect (три уровня)

### Изменено
- VPS PeerID: QmNmr3Yq... → QmR8u5YF...
- Условие отключения VPS

### Коммиты
- 9b5c6c2, 6b223c0

---

## v1.22.0 (2026-09-13)

### Добавлено
- Foreground Service (Android)
- Battery Optimization Whitelist
- Отображение пиров через libp2p

### Коммиты
- b706f97, 716a48f, ccec4be

---

## v1.21.0 (2026-09-11)

### Исправлено
- Дубликаты сообщений (единый ID из Go)
- Бейдж непрочитанных + линия «Непрочитанные»
- ANR на медленных телефонах
- Reconnect loop + keepalive

### Коммиты
- ec0c59e, 919967d, c06d8a8

---

## v1.19.0 (2026-09-01)

### Добавлено
- Рефакторинг ядра: package main → package core
- main/main.go, mobile/mobile.go
- libp2p через FFI (.aar 67 МБ)
- Стабильный PeerID
- Обработка смены сети
- NodeInfo, heartbeat, dead-статус
- Логирование Go → Flutter
- NSD-обнаружение

---

## v1.18.1 (2026-08-28)

### Добавлено
- Рефакторинг: package main → package core
- Точка входа: node/main/main.go
- Экспорт API: Config, NewNode, InitP2P, StartHTTP, StartMobile, Stop

---

## v1.18 (2026-08-16)

### Добавлено
- Каналы с весовыми уровнями (G4)
- Пороги доступа: full=0.3, comment=0.5, vote=0.7
- Эндпоинты: POST/GET /channels

---

## v1.17 (2026-08-16)

### Добавлено
- Самоадаптация: node/adapt.go
- Метрики: avgWeight, lowWeightRatio, highWeightRatio
- Фоновая адаптация каждые 5 минут

---

## v1.16 (2026-08-16)

### Добавлено
- Onion Routing v2: цепочка из 4-5 relay
- Выбор relay по весу > 0.7
- getPeerWeight

---

## v1.15 (2026-08-16)

### Добавлено
- Репликация на 2 случайных живых узла
- Поля ReplicatedFrom, ReplicatedAt
- Стабильный PeerID

---

## v1.14 (2026-08-16)

### Добавлено
- Голосовая стеганография: LSB в WAV
- node/stego.go: embedLSB, extractLSB
- Эндпоинт POST /send_stego

---

## v1.13 (2026-08-16)

### Добавлено
- Обфускация: AES-GCM с префиксом [SHUF]
- Случайные задержки 5-50 мс
- Дедупликация

---

## v1.12 (2026-08-16)

### Добавлено
- Селф-хилинг: heartbeat каждые 30 сек
- Обнаружение мёртвых пиров: 5 сек без PONG
- Автоперезапуск

---

## v1.11 (2026-08-16)

### Добавлено
- Исчезающие сообщения (TTL): вечно, 60 сек, 3600 сек
- ExpiresAt в Message
- Локальное шифрование state: AES-256-GCM

---

## v1.10 (2026-08-15)

### Добавлено
- Onion Routing v1: три режима анонимности

---

## v1.9 (2026-08-14)

### Добавлено
- 5 узлов в docker-compose
- Документация: три столпа ISOTOPE
- README (EN + RU)
- docs/FAQ.md: 20 вопросов

---

## v1.8 (2026-08-14)

### Добавлено
- Ассоциативная память
- Децентрализованный bootstrap: mDNS, DHT, вручную через ENV

---

## v1.7 (2026-08-14)

### Добавлено
- WebSocket + TLS: трафик неотличим от HTTPS
- Новый универсальный этический хеш: семь заповедей

---

## v1.6 (2026-08-11)

### Добавлено
- Priority Gossip: поле Priority в Message
- TTL форвардинга зависит от приоритета

---

## v1.5 (2026-07-27)

### Добавлено
- REST API для внешних клиентов
- WebSocket для реального времени
- Пагинация для /messages
- CORS middleware
- Мобильное приложение (Flutter, базовая версия)
- Защита памяти (лимит 10 000 сообщений, архив)

---

## v1.4 (2026-07-19)

### Добавлено
- 100-мерные векторы (VectorDim = 100)
- Биграммы в textToVector
- Марковские цепочки для русских ответов
- Кнопки preHash/antiHash в дашборде
- Мониторинг здоровья сети
- 55 автотестов (позже 67)

---

## v1.3 (2026-07-19)

### Добавлено
- Персональные фильтры: preHash и antiHash
- Эндпоинты /setprehash и /setantihash

---

## v1.2 (2026-07-19)

### Добавлено
- Этическое обучение нейросети через лайки/дизлайки

---

## v1.1 (2026-07-19)

### Добавлено
- Этический иммунитет: начальный вес зависит от близости к хешу
- Семь заповедей как этический хеш

---

## v1.0 (2026-07-16)

### Первый стабильный релиз
- P2P-сеть на libp2p + mDNS (3 узла)
- Нейросеть: 10 нейронов на слой
- Взвешенная память
- Чат с дашбордом
- 37 успешных тестов

---

**ISOTOPE эволюционирует.**
**Каждая версия — шаг к неуязвимой связи.**