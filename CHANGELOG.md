# История изменений ISOTOPE

## v1.29.0 (2026-10-05)

«Единый источник истины»

### Добавлено
- Single source of names:
  - Убран _contactNames из connect_screen.dart
  - _displayName делегирует в chatProvider.nameFor(peerID)
  - _loadContactsFromCore вызывает loadPeerNames()
- Batch [READ]:
  - Message.Refs []string + fallback на Message.Ref
  - Version 0. Обратная совместимость через содержимое поля
  - SendReadBatch(refs, recipient)
  - handleServiceMessage case TypeRead — цикл по Refs
- Message.Status (перенос из MessageStatus):
  - Поле Status MessageStatus в Message
  - Удалены n.messageStatus, n.messageStatusMu, State.MessageStatus
  - Memory.SetStatus(id, status) — по образцу SetExpiresAt
  - Миграция: при loadState старый messageStatus → Message.Status
- Message.ReadLocally:
  - Поле ReadLocally bool в Go + Dart
  - Memory.MarkReadLocally(refs), Node.MarkReadLocally(refs)
  - LibP2PService.markReadLocally, MainActivity.kt case
  - _sendReadBatchFor → markReadLocally перед sendReadBatch
  - loadMessages — пересчёт _unreadByPeer для readLocally == false
  - Удалён _readSent. Миграция: _readSent → markReadLocally, флаг read_sent_migrated
- Offline queue: pending до [DELIVERED]:
  - replicateMessage — при адресном всегда enqueuePending
  - flushPending — не удаляет из очереди, только переотправляет
  - removePendingByRef(ref) — удаление по [DELIVERED]
  - case TypeDelivered → removePendingByRef(m.Ref)
  - Триггеры: ConnectedF (Notifiee) + announceLoop (4 мин)
- Multiline input:
  - TextField в chat_screen.dart: maxLines: null, minLines: 1
  - keyboardType: multiline, textInputAction: newline
  - ConstrainedBox(maxHeight: 140) — рост до ~5 строк

### Проверено
- 10 сообщений подряд, получатель оффлайн → при появлении приходят мгновенно пачкой
- Бейдж переживает перезапуск. Не теряется, не двоится

### Коммиты
- 011d719 — single source of names
- a36c617 — batch [READ]
- 9a19879 — MessageStatus → Message.Status
- 212f8f3 — multiline input
- 6a9fac6 — offline queue: pending до [DELIVERED]
- 64e1d8b, 4353842 — ReadLocally

### Следующий шаг
- [PROFILE_UPDATE] (Type=9)
- TTL для tempContacts (5 минут)
- VPS reconnectLoop — отключить на relay
- Уведомления системы

---

## v1.28.0 (2026-10-04)

«Право на забвение»

### Добавлено
- TTL — полностью:
  - Settings.TtlPeriod / Settings.TtlMode
  - Message.TtlPeriodSeconds / TtlMode / ExpiresAt
  - Периоды: 10s / 30s / 1m / 5m / 15m / 30m / 1h / 4h / 24h / never
  - «forever» → «never» (обратная совместимость)
  - Режимы: hard / after_read
  - [TTL_UPDATE] (Type=8) — новый тип для авто-hard
  - Fallback 48 часов — принудительное удаление
  - DeleteExpired в cleanupLoop (тикер 1 мин)
  - Фильтрация истёкших в saveState / loadState / Memory.Add / GetMessages
- FLAG_SECURE:
  - TTL от 10 сек до 1 мин — скриншоты запрещены
  - Оптимизация _lastSecureFlag
  - Снятие при dispose
- Удаление контакта:
  - RemoveContact — полная чистка
  - isotope_deleted.json — удалённые не возвращаются
  - Тихий отказ — B не знает
- Per-chat / per-peer:
  - messagesFor(peerID)
  - Черновики — Map<peerID, String>
  - Непрочитанные — Map<peerID, int>
  - [READ] — только для текущего чата
- UI:
  - Настройки → Сообщения (TTL: периоды + режимы)
  - Настройки → Приватность (read_enabled)
  - Профиль (MyDisplayName)
  - Оранжевые периоды 10s / 30s / 1m
  - Умный формат времени
  - Таймер отправки
  - AppBar — чисто, только имя контакта
  - Диалог «Как вас представить?»

### Исправлено
- Абракадабра в превью — displayText вместо text
- «Свой контакт» артефакт — isSelf фильтр
- Имя входящего в чате — ChatProvider.nameFor(peerID)
- [CONTACT_REQUEST] — дедупликация по peerID
- Бейдж + превью — синхронно с TTL

### Коммиты
- d482dfc — HEAD (TTL финальный)
- 27ca05c — TTL механизм

---

## v1.27.0 (2026-10-01)

«Контакты и идентификация»

### Добавлено
- Контакт-протокол (bootstrap-handshake):
  - [CONTACT_HELLO] → [CONTACT_HELLO_ACK] → [CONTACT_REQUEST] → [CONTACT_ACCEPT]
  - Открытые сервисные — всегда через bootstrap
  - E2E — circuit → bootstrap fallback
  - tempContacts
  - Push через messageHook
  - Симметрия: confirmed: true
- Система имён:
  - Name / RemoteName / PeerID
  - MyDisplayName в Settings
  - QR содержит display_name
  - Диалог «Как вас представить?»
  - Профиль → «Ваше имя»
  - Долгий тап → Открыть / Переименовать / Удалить
  - Предупреждение о безопасности

### Изменено
- Разделение транспортов по назначению
- Type в processMessageInternal — с рождения
- Логи: убраны STATUS POLL, [MULTIADDR], [DIAG], [DHT] Provide

### Коммиты
- 6896800, bead650, 692418d, 67778cd, a2004db, d083997
- 849c388, 36a6af6, 0cb9de2, 2a9083a, 196af37, 156fe0a
- 8e6280c — HEAD

---

## v1.26.0 (2026-09-27)

### Добавлено
- PlainText в Message
- Автоочистка своих E2E без PlainText
- Relay: refresh reservation on reconnect + 30s loop
- Relay: exponential backoff
- UI: отображение plainText

### Исправлено
- Свой QR блокируется
- Retry loading contacts
- Upsert nodes on alive-event

### Коммиты
- d0e9744, 3235060, 882c998, abc5f01, 6e1d1b9, 2f35fb1

---

## v1.25.0 (2026-09-25)

### Добавлено
- UI 1.1 — терминология
- UI 1.2 — единый вход «Добавить контакт»
- UI 1.3 — пустое состояние

### Коммиты
- 5ebe222, 4940998, 988fe17

---

## v1.24.0 (2026-09-23)

### Добавлено
- E2E 4.1–4.4: ключи, Recipient, Ed25519+X25519, isotope_contacts.json, box.Seal, подпись
- Backoff reconnect
- Параллельный dial
- ANNOUNCE TTL 5 → 8 мин

### Коммиты
- e54d056, 9c2fc8b, dc82e81, f3aaeab, 3a3b859, 39c97c5
- 1d2e3d3, 57cfeaf, 87524bd, 1a44715, b225a22

---

## v1.23.0 (2026-09-17)

### Добавлено
- Multi-address ANNOUNCE
- Relay-circuit
- Flush on reconnect

### Коммиты
- 9b5c6c2, 6b223c0

---

## v1.22.0 (2026-09-13)

### Добавлено
- Foreground Service
- Battery Optimization Whitelist
- Отображение пиров через libp2p

### Коммиты
- b706f97, 716a48f, ccec4be

---

## v1.21.0 (2026-09-11)

### Исправлено
- Дубликаты сообщений
- Бейдж непрочитанных
- ANR
- Reconnect loop

### Коммиты
- ec0c59e, 919967d, c06d8a8

---

## v1.19.0 (2026-09-01)

### Добавлено
- Рефакторинг ядра
- libp2p через FFI
- Стабильный PeerID
- NSD-обнаружение

---

## v1.18.1 (2026-08-28)

### Добавлено
- package main → package core
- Экспорт API

---

## v1.18 (2026-08-16)

### Добавлено
- Каналы с весовыми уровнями

---

## v1.17 (2026-08-16)

### Добавлено
- Самоадаптация

---

## v1.16 (2026-08-16)

### Добавлено
- Onion Routing v2

---

## v1.15 (2026-08-16)

### Добавлено
- Репликация

---

## v1.14 (2026-08-16)

### Добавлено
- Голосовая стеганография

---

## v1.13 (2026-08-16)

### Добавлено
- Обфускация AES-GCM

---

## v1.12 (2026-08-16)

### Добавлено
- Селф-хилинг

---

## v1.11 (2026-08-16)

### Добавлено
- TTL
- Локальное шифрование

---

## v1.10 (2026-08-15)

### Добавлено
- Onion Routing v1

---

## v1.9 (2026-08-14)

### Добавлено
- 5 узлов в docker-compose
- Документация

---

## v1.8 (2026-08-14)

### Добавлено
- Ассоциативная память
- Децентрализованный bootstrap

---

## v1.7 (2026-08-14)

### Добавлено
- WebSocket + TLS
- Этический хеш

---

## v1.6 (2026-08-11)

### Добавлено
- Priority Gossip

---

## v1.5 (2026-07-27)

### Добавлено
- REST API
- WebSocket
- Мобильное приложение

---

## v1.4 (2026-07-19)

### Добавлено
- 100-мерные векторы
- Биграммы
- Мониторинг сети

---

## v1.3 (2026-07-19)

### Добавлено
- preHash / antiHash

---

## v1.2 (2026-07-19)

### Добавлено
- Этическое обучение

---

## v1.1 (2026-07-19)

### Добавлено
- Этический иммунитет

---

## v1.0 (2026-07-16)

### Первый стабильный релиз

---

**ISOTOPE эволюционирует.**
**Каждая версия — шаг к неуязвимой связи.**