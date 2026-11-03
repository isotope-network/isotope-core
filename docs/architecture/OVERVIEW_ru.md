# Архитектура ISOTOPE

## Обзор

ISOTOPE — инфраструктура для этичного, неуязвимого,
самообучающегося обмена данными и ИИ.

    [Пользователь] ←→ [Мессенджер / API] ←→ [Узел ISOTOPE] ←→ P2P-сеть ←→ [Другие узлы]
                            │                        │
                      [Нейросеть]              [ИИ-модель]
                      [Память]                 [Этический паспорт]
                      [Этический движок]       [Инференс]

---

## Структура ядра (v2.0.6)

Ядро ISOTOPE — **библиотека** (пакет `core`).
node/
├── *.go # package core (ядро)
├── main/main.go # package main (точка входа для десктопа)
└── mobile/mobile.go # package mobile (обёртка gomobile)

text

**Разбиение v2.0.6:** `node.go` (198 КБ → 35 КБ + 8 файлов):

- `node.go` — ядро: Node struct, NewNode, InitP2P, Stop, StartMobile, StartHTTP, GetStatus, sendToRecipient, fallbackSend, offline queue, loadState.
- `node_p2p.go` — handleStream, ping, reconnect, announceLoop, cleanup, processMessageInternal, Sphinx-получатель.
- `node_e2e.go` — E2E-ключи, шифрование, QR, SetSeedPhrase, GetSeedPhrase.
- `node_files.go` — chunkWorker, retryUnacked, pushFileStatus, GetChunk, FileAck, DeleteFile, GetFileStatuses.
- `node_service.go` — сервисные, статусы, батчи, TTL.
- `node_contacts.go` — контакты, temp, requests.
- `node_announce.go` — ANNOUNCE, FIND, relay, Multiaddrs, Connect.
- `node_settings.go` — настройки, deleted, SendMessage/SendToPeer.
- `node_batch.go` — batch, dummy, обфускация, randomDelay.
- `node/seed.go` — BIP39 → Ed25519 + X25519.
- `node/sphinx.go` — Sphinx (build/process/extract), sleepExponential, flags, SphinxPayloadSize.
- `node/memory.go` — Message, Memory, byID, fileChunks.
- `node/state.go` — saveState, loadStateData, gzip.
- `node/contacts.go`, `node/settings.go`, `node/requests.go`, `node/deleted.go`, `node/dht.go`, `node/media.go`, `node/adapt.go`.

### Структура Message (v2.0.6)

```go
type Message struct {
    ID               string
    Recipient        string
    Sender           string
    SenderName       string        // v1.30 — для уведомлений
    Text             string
    PlainText        string
    TTL              int
    Priority         int
    Timestamp        int64
    Version          int           // 0 = история, 2 = E2E
    Type             MessageType   // 0 = обычное, 1-9 = служебные
    Ref              string        // одиночная ссылка
    Refs             []string      // массив ссылок (батч)
    Status           MessageStatus // sent / delivered / read / hidden
    ReadLocally      bool          // я прочитал входящее
    TtlPeriodSeconds int           // 0 = never
    TtlMode          string        // "after_read" | "hard" | ""
    ExpiresAt        time.Time
    ReadEnabled      *bool
    MediaType        string        // "" | "text" | "voice" | "file" | "photo" | "batch" | "dummy"
    MediaData        string        // base64 (голосовые)
    FileName         string        // для файлов
    FileSize         int64         // для файлов
    Duration         int           // секунды, для голосовых
    MediaID          string        // для чанков файла
    ChunkIndex       int           // 0, 1, 2, ...
    ChunkTotal       int           // общее число чанков
    MixingEnabled    bool          // v2.0.2 — флаг mixing в Sphinx-пакете
}
Методы Node (основные, v2.0.6)
go
func (n *Node) SendMessage(text string, ttl int) (string, error)
func (n *Node) SendToPeer(recipient, text string, ttl int) (string, error)
func (n *Node) SendReadBatch(refs []string, recipient string) error
func (n *Node) MarkReadLocally(refs []string) error
func (n *Node) AddContact(...) error
func (n *Node) RemoveContact(peerID string) error
func (n *Node) RenameContact(peerID, localName string) error
func (n *Node) GetMessages() []Message
func (n *Node) GetPeers() []string
func (n *Node) nameForPeer(peerID string) string
func (n *Node) SetSeedPhrase(phrase string) error
func (n *Node) GetSeedPhrase() string
func (n *Node) SendVoiceByPath(...) error
func (n *Node) SendFileByPath(...) error
func (n *Node) SendPhotoByPath(...) error
func (n *Node) SendFileComplete(mediaID, sender string) error
func (n *Node) GetFileStatuses() string
func (n *Node) FileAck(mediaID string, indices []int) error
func (n *Node) SetMixingEnabled(enabled bool) error
func (n *Node) GetMixingEnabled() bool
func (n *Node) SetMaskingEnabled(enabled bool) error
func (n *Node) GetMaskingEnabled() bool
func (n *Node) SetDummyScreenOn(screenOn bool) error
Sphinx routing (v2.0.1)
Слоистое шифрование. Каждый relay снимает свой слой. Relay знает только следующий hop (полный peer.ID). Не знает отправителя и получателя.

SphinxHopLayerSize = 249, SphinxHeaderSize = 747.

SphinxNextPeerIDLenSize = 1, SphinxNextPeerIDSize = 64.

SphinxPayloadSize = 65536 (64 КБ).

SphinxPayloadLenSize = 4 — [length 4 BE][JSON][padding].

sendViaSphinx — только прямой (Wi-Fi), фильтр /p2p-circuit, AddAddrs, NewStream. Fallback — sendToRecipient.

handleStream (relay) — FindPeerByID + direct (симметрично).

processSphinxPacket — peer.Decode (не peer.ID).

VPS не в цепочке (когда 3 доверенных контакта).

Mixing (v2.0.2, v2.0.3)
v2.0.2 — Mixing (текст)
Флаг в Sphinx-пакете — свойство сообщения, не настройка relay.

Message.MixingEnabled.

SphinxFlagsSize = 1, SphinxFlagMixing.

buildSphinxPacket(..., mixing bool) — flags в каждом relay-слое. MAC покрывает flags.

processSphinxPacket → (peer.ID, bool, []byte, error).

sleepExponential — relay спит при mixing (mean 5 сек, max 30 сек).

v2.0.3 — Mixing для файлов
Чанки файлов через Sphinx + mixing.

SphinxPayloadFileSize = 131072 (128 КБ, double base64).

sphinxFileSem = 2.

SphinxTimeout = 15s, write deadline 30s.

Retry через Sphinx.

Маскировка (v2.0.4)
Все пакеты 64 КБ. Текст, файл, фото, голос — не отличить.

Батч текста (10 текстов = 1 пакет). Трафик ×1.4, не ×14.

Dummy-трафик:

SphinxFlagDummy (бит 1). От отправителя, не relay.

sendDummy — случайный интервал 10–120 сек (×5 в фоне), случайный контакт, случайный текст (50–200 символов), случайная задержка (1–60 сек).

Получатель — drop (не memory, не hook, не [DELIVERED]).

maskingEnabled — тумблер в Privacy.

Чанк 47 КБ — влезает в 64 КБ payload.

Double base64 убран для файлов. encryptBytesForRecipient, decryptBytesFromSender.

Голос, файлы, фото (v1.31–v2.0.5)
MediaType (string): "" | "text" | "voice" | "file" | "photo" | "batch" | "dummy".

Голосовые: Opus 24 kbps, 5 мин ≈ 900 КБ. Один чанк (MediaType = "voice"). Плеер: play/pause, прогресс-бар, seek.

Файлы: чанки 64 КБ (v1.31–v2.0.3), 47 КБ (v2.0.4+). MediaID, ChunkIndex, ChunkTotal. Сборка на диске.

Фото: сжатие 1600px JPEG 80. Миниатюра, fullscreen с зумом.

isChunkedMediaType (file/voice/photo) — вместо == "file".

retryInfo.mediaType.

_VoicePlayer / _PhotoView с localFilePath.

Seed-фраза + Android Keystore + i18n (v2.0.6)
BIP39 seed (12/24 слова) → Ed25519 (PeerID) + X25519 (шифрование). Детерминированно.

node/seed.go — GenerateSeedPhrase, ValidateSeedPhrase, SeedToEd25519, SeedToX25519.

Android Keystore — AES-256, CBC, NoPadding, alias isotope_seed_key. Seed переживает перезапуск.

InitP2P: если seedPhrase != "" — Ed25519 identity (из seed), X25519 из seed, не loadOrGenerateE2EKeys, не loadPrivateKey (RSA). Иначе — RSA.

i18n: LocaleProvider, l10n.yaml, app_en.arb, app_ru.arb (~250 ключей). Переключатель языка в Настройках.

Мобильная обёртка (mobile.go)
Статус: работает (v2.0.6)

Методы (возвращают JSON-строки)
Метод	Описание
Start(stateFile)	Запуск узла
Send(text, ttl)	Отправка (broadcast)
SendToPeer(recipient, text, ttl)	Отправка контакту
SendReadBatch(refs, recipient)	Батч [READ]
MarkReadLocally(refs)	Пометить прочитанным локально
GetMessages()	Список сообщений
GetPeers()	Список пиров
GetWeight()	Вес узла
GetStatus()	JSON-статус
GetMultiaddrs()	Список адресов
ConnectToPeer(addr)	Подключение
ConnectToPeerWithFallback(addrs)	С перебором
Announce(json)	Публикация адресов
FindPeerByID(peerID)	Поиск пира
AddContact(...)	Добавить контакт
RemoveContact(peerID)	Удалить контакт
RenameContact(peerID, name)	Переименовать
SendContactRequest(peerID, name)	Запрос
AcceptRequestByID(id)	Принять
RejectRequestByID(id)	Отклонить
GetRequests()	Список запросов
SetMyDisplayName(name)	Представление
GetMyDisplayName()	Получить представление
SetTtl(period, mode)	TTL
GetTtl()	Получить TTL
GetShowNotificationContent()	Настройка уведомлений
SetShowNotificationContent(bool)	Настройка уведомлений
GetMyQRData()	QR-данные
GenerateSeedPhrase(words)	Создать seed-фразу
SetSeedPhraseBeforeStart(phrase)	Seed до Start
SetSeedPhrase(phrase)	Seed после Start
GetSeedPhrase()	Получить seed
SendVoiceByPath(path, ttl)	Голосовое
SendFileByPath(path, ttl)	Файл
SendPhotoByPath(path, ttl)	Фото
SendFileComplete(mediaID, sender)	[FILE_COMPLETE]
GetFileStatuses()	Статусы файлов
FileAck(mediaID, indicesJSON)	Внутренний ack
GetMixingEnabled() / SetMixingEnabled(bool)	Mixing
GetMaskingEnabled() / SetMaskingEnabled(bool)	Маскировка
SetDummyScreenOn(screenOn)	Dummy (screenOn)
GetNotificationSettings() / SetNotificationSettings(json)	Шторка
Е2Е-шифрование (v1.24)
Ключи
Ключ	Назначение	Тип	Файл
PeerID	Идентификация	RSA 2048	isotope_state.json.key
Ed25519	Подпись	Ed25519	isotope_state.json.ed25519.key
X25519	Шифрование E2E	X25519	isotope_state.json.x25519.key
QR-формат (v1)
json
{
  "v": 1,
  "peerID": "QmX...",
  "ed25519_pub": "base64...",
  "x25519_pub": "base64...",
  "signature": "",
  "display_name": "Пётр",
  "read_enabled": true
}
Шифрование
box.Seal (X25519 + XSalsa20-Poly1305).

Формат: nonce(24) || ciphertext.

Version = 2 — E2E.

Version = 0 — история.

Подпись
Ed25519 над peerID || x25519_pub.

verified — сигнал UI, не пропуск.

Единый источник истины (v1.29)
Принцип: если поле выводимо из Message — оно в Message.

Message.Status (v1.29)
Было: MessageStatus — отдельный map в Node и State.

Стало: Message.Status.

Удаляется сообщение → уходит статус.

Memory.SetStatus(id, status).

Миграция: при loadState старый messageStatus → Message.Status.

Message.ReadLocally (v1.29)
Было: _readSent в Dart.

Стало: Message.ReadLocally bool.

Memory.MarkReadLocally(refs).

Dart: _sendReadBatchFor → markReadLocally перед sendReadBatch.

Миграция: _readSent → markReadLocally, флаг read_sent_migrated.

Message.Refs (v1.29)
Было: [READ] по одному.

Стало: Message.Refs []string.

Ref — одиночная ссылка.

Refs — батч.

Обратная совместимость: Refs пуст → читаем Ref.

SendReadBatch(refs, recipient).

Offline-очередь (v1.29)
Проблема: оффлайн-получатель. VPS не буферизует.

Решение: pending до [DELIVERED].

replicateMessage — при адресном всегда enqueuePending.

flushPending — не удаляет, только переотправляет.

removePendingByRef(ref) — по [DELIVERED].

Триггеры: ConnectedF (Notifiee) + announceLoop (4 мин).

Fix (v1.30): enqueuePending — только для не-сервисных (!isServiceType). Причина: сервисные не имеют [DELIVERED] → бесконечный цикл handshake.

Уведомления системы (v1.30)
Message.SenderName
Поле SenderName в Message.

Заполняется в 4 местах создания входящего.

nameForPeer(peerID) — приоритет: свой → MyDisplayName; контакт → Name → RemoteName; иначе пусто.

Kotlin onMessage
Если !isForeground → showMessageNotification.

Канал isotope_messages, importance HIGH.

Тап → Intent EXTRA_PEER_ID → onNewIntent → MethodChannel → Dart openChatStream.

ShowNotificationContent
Settings.ShowNotificationContent bool.

Дефолт true.

Privacy toggle в настройках.

Разрешения per-action (v1.30)
permission_service.dart — новый файл.

ActivityCompat.requestPermissions в Kotlin, onRequestPermissionsResult → результат в Dart.

Запросы:

POST_NOTIFICATIONS — при старте.

CAMERA — при QR-скане.

BLUETOOTH_*, FINE_LOCATION, NEARBY_WIFI_DEVICES — при «Найти рядом».

Батарея — диалог с инструкцией + openAppSettings.

При отказе: SnackBar с кнопкой «Настройки».

AndroidManifest.xml: CAMERA добавлен явно.

WakeLock (v1.30)
IsotopeService.kt — PARTIAL_WAKE_LOCK (isotope:network).

acquireWakeLock в onCreate, releaseWakeLock в onDestroy.

Зачем: Foreground Service сам по себе не предотвращает сон CPU. Без WakeLock Go-рутины libp2p замораживаются в фоне.

Multiline input (v1.30)
chat_screen.dart — TextField в ConstrainedBox(maxHeight: 140).

maxLines: null, minLines: 1.

keyboardType: TextInputType.multiline.

textInputAction: TextInputAction.newline.

onSubmitted убран. Отправка — только через кнопку.

Контакт-протокол (v1.27)
Контакт — это не добавление в список. Это взаимное обещание слышать. Ты предлагаешь. Я принимаю. Или отклоняю. Без принуждения. Без навязывания.

Bootstrap-handshake
[CONTACT_HELLO] — открытый. PeerID A + публичные ключи A.

[CONTACT_HELLO_ACK] — открытый. PeerID B + публичные ключи B.

[CONTACT_REQUEST] — E2E. Полный payload.

[CONTACT_ACCEPT] — E2E. Подтверждение + представление B.

Разделение транспортов
Тип сообщения	Транспорт
Открытые сервисные (HELLO, ACK)	Bootstrap (всегда)
E2E (REQUEST, ACCEPT, сообщения)	Circuit → bootstrap fallback
tempContacts
Создаются при [CONTACT_HELLO], если B не знает A.

Нужны только для расшифровки [CONTACT_REQUEST].

Удаляются после requests.Add.

Удаление контакта — тихий отказ
Удаление — это не блокировка. Это тихий отказ. Блокировка — принуждение. Тишина — свобода.

RemoveContact — полная чистка.

isotope_deleted.json — удалённые не возвращаются.

B не знает о факте удаления.

Система имён (v1.27)
PeerID — моя техническая суть. RemoteName — моё представление для других. Name — моё имя для себя.

Три уровня — три свободы: быть собой, быть понятым, быть узнанным.

Два поля у контакта
Поле	Кто задаёт	Передаётся	Приоритет
Name	Я	Нет	Высший
RemoteName	Контакт	Да	Средний
—	—	—	Fallback: PeerID
UI: Name → RemoteName → PeerID.

Single source of names (v1.29)
Убран _contactNames из connect_screen.

_displayName делегирует в chatProvider.nameFor(peerID).

Статусы сообщений (v1.26, v1.34)
Пять состояний, четыре иконки:

Иконка	Значение
⏳	Отправляется
✓	Отправлено
✓✓	Доставлено. Прочтения не будет
✓🔒	Доставлено. Получатель не делится статусом
✓✓ (цвет)	Прочитано
read_enabled
В контакте.

В Settings (my_read_enabled).

Передаётся с каждым сообщением.

В QR.

hidden — терминальное
Замок = «прочтения не будет».

Не откатывается.

Галочки файлов (v1.34)
Те же 5 состояний, 4 иконки, что у текстовых.

⏳ — в очереди.

✓ (серая) — отправлено.

✓✓ (две серые) — доставлено + у получателя «Открыть» (файл собран) — только после [FILE_COMPLETE].

✓🔒 (одна серая + замочек) — доставлено, получатель не делится read_enabled.

✓✓ (две зелёные) — прочитано ([READ] получен).

GetFileStatuses — status = max по чанкам (read > hidden > delivered > sent).

[READ] для файлов — через mediaID.

TTL — право на забвение (v1.28)
Сообщение не «удаляется». Оно отпускается. Как отпускают прошлое — без сожаления. Забвение — это не потеря. Это освобождение.

Периоды
10s / 30s / 1m / 5m / 15m / 30m / 1h / 4h / 24h / never.

Дефолт: never («Не удаляются»).

Режимы
«После прочтения» — уважение к получателю. «Жёсткий» — контроль отправителя.

hard:

Таймер от получения.

Отправитель: от [DELIVERED].

Получатель: от получения.

after_read:

Таймер от прочтения.

Оба делятся → синхронно.

Получатель не делится → авто-hard + [TTL_UPDATE].

Отправитель не делится → авто-hard + [TTL_UPDATE].

Fallback 48 часов
Fallback — честность. Если получатель не делится — сеть решает сама.

[TTL_UPDATE] (Type=8)
Version = 0.

Payload: expires_in_seconds.

При авто-hard.

FLAG_SECURE — право на тишину
Секрет — защита. Длинное — не секрет.

Короткие TTL (10с – 1 мин) — запрет скриншотов. Это право на тишину. Тишина — это тоже свобода.

Бейдж + превью
Живут ровно столько, сколько сообщение. Всё вместе.

Таймер отправки (v1.26)
Задержка 0/3/5/10 сек.

Сообщение сразу с круговым прогрессом.

Кнопка «Отмена».

Back — черновик.

Home — таймер продолжается.

Миграции (v1.29)
Обязательны. Старые данные не теряются.

messageStatus → Message.Status.

_readSent → ReadLocally.

Каждая миграция логируется.

Передача файлов (v1.32–v1.33)
Чанки 64 КБ (v1.31–v2.0.3), 47 КБ (v2.0.4+).

Backpressure: chunkQueue (буфер 1000) + 10 chunkWorker.

sent/<mediaID>.bin — копия на диске, источник для retry.

Батч [DELIVERED]: queueDelivered (2 сек / 10 refs). Идемпотентность: [DELIVERED] шлётся всегда.

Retry unackedChunks: retryInfo — метаданные (~200 байт). Message восстанавливается из sent/ через fillFileChunkFromSent. Backoff до 1 часа.

Прогресс — push, не pull: file_status (throttle 1 сек), file_complete (один раз, completedFiles).

Восстановление после перезапуска: resumeIncompleteFiles, GetFileStatuses.

O(1) индексы (v1.33): byID, fileChunks. rebuildIndexes после DeleteExpired/PurgeDead.

Внутренний ack (v1.33): hookAckWindow, FileAck, cleanupPendingAcksLoop (30 сек тик, 120 сек таймаут).

Fallback WS↔TCP (v1.33): expandTransportAlternatives — порт вычисляется динамически (TCP = port, WS = port+1).

gzip state (v1.33): JSON → gzip → (encrypt) → файл. 24,4 МБ → 1,1 МБ (22x).

Кэш getFilePath (v1.33): MediaStorage._pathCache, invalidatePathCache.

TypeFileComplete = 9 (v1.33): isServiceType (добавлен Type 9), handleFileComplete, SendFileComplete.

Relay-circuit (v1.23+)
Проблема: узлы за NAT не могут принимать входящие.

Решение: relay через VPS.

Узел резервирует слот (client.Reserve).

VPS форвардит, не хранит.

ANNOUNCE автоматически добавляет relay-адрес.

FIND fallback — relay-адрес.

Обновление резервации (v1.25):

Notifiee ConnectedF — при reconnect.

relayLoop 30 секунд.

Exponential backoff.

Условие отключения VPS:

DHT покрывает 15+ узлов.

Hole punching работает для большинства NAT.

2-3 независимых relay-узла.

Иммунитет
Иммунитет — способность сети сохранять себя без центрального управления.

Технический иммунитет
Три уровня защиты.

Репликация: на 2+ живых узла.

Селф-хилинг: heartbeat.

Fallback: локальный IP → relay.

Exponential backoff.

Социальный иммунитет
Вес от этического хеша.

Лайки/дизлайки.

Время размывает.

Архив → удаление.

Единство
Технический держит сеть живой. Социальный — чистой.

Масштаб
Узлов	Технический	Социальный
2-3	Защищённый канал	Личное доверие
5-10	Ранний консенсус	Первые оценки
15-20	Иммунитет включается	Коллективная очистка
50+	Стабильный иммунитет	Устойчивые нормы
100+	Неубиваемая сеть	Саморегуляция
Ключевые компоненты
1. P2P-сеть (libp2p)
Транспорт: TCP + WebSocket + TLS.

Обнаружение: mDNS + DHT + NSD.

Синхронизация: Gossip.

Маршрутизация: Sphinx (v2.0.1).

Relay-circuit.

Маскировка (v2.0.4).

Reconnect, Flush.

2. Этический движок
Вектор: 100 измерений.

Слои: каждые 20 сообщений.

Хеш: 7 заповедей.

3. Память
Взвешенная, FIFO с архивом.

Вес, старение, архив.

TTL.

Шифрование AES-256-GCM.

PlainText.

Status, ReadLocally.

O(1) индексы (v1.33).

4. Каналы
Пороги: full=0.3, comment=0.5, vote=0.7.

5. Безопасность
Sphinx (v2.0.1).

Mixing (v2.0.2, v2.0.3).

Маскировка (v2.0.4).

Стеганография.

Селф-хилинг.

E2E: X25519 + box.Seal.

Подпись: Ed25519.

FLAG_SECURE.

6. Весовая модель
Вес	Права
0–0.3	Читать
0.3–0.5	Полный доступ
0.5–0.7	Комментировать
0.7–1.0	Голосовать, relay
7. Самоадаптация
node/adapt.go.

avgWeight, lowWeightRatio, highWeightRatio.

Фоновая адаптация.

Статус: архитектура (v2.5+).

8. ИИ-слой
Статус: v3.0+.

9. API
REST: /send, /messages, /status, /health, /feedback.

WebSocket: /ws.

Фильтры.

Стего.

Уровни защиты
Уровень	Что защищено	Статус
Обфускация	Маскировка трафика	✅ v1.13
E2E-шифрование	Содержимое	✅ v1.24
Подпись	Подлинность	✅ v1.24
Sphinx	Маршрут скрыт от relay	✅ v2.0.1
Mixing	Тайминги скрыты	✅ v2.0.2
Маскировка	Тип, объём, паттерн скрыты	✅ v2.0.4
Метаданные (полные)	Кто с кем	🔜 v2.0.6+ (адаптивная маскировка)
Временны́е паттерны (полные)	Тайминг, объём	🔜 Padding, mixing
Инфраструктура
VPS bootstrap/relay
IP: 186.246.31.176.

PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi.

Bootstrap: /ip4/186.246.31.176/tcp/9000/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi (TCP; WS 9001 — fallback, блокируется провайдером Городок; fallback автоматический — 3d82934). Временно, до маскировки трафика (v2.0+).

Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API).

НЕ удалять /root/isotope/state/.

Телефоны (тестовые)
Актуальные PeerID — в хендоффе (меняются при сносе приложения).

Xiaomi, Huawei, Samsung — 4 телефона.

Обновление VPS
bash
cd /root/isotope-core
git pull
git --no-pager log --oneline -3
cd /root/isotope-core/node
go build -o isotope-node ./main
kill $(pgrep -f isotope-node)
sleep 1
pgrep -f isotope-node          # пусто
nohup env NODE_ID=bootstrap ISOTOPE_PORT=9000 ISOTOPE_HTTP_PORT=8081 ISOTOPE_ENABLE_RELAY=true ./isotope-node > isotope.log 2>&1 &
sleep 3
pgrep -f isotope-node          # один новый PID
curl -s http://127.0.0.1:8081/status
⚠️ pkill -f isotope-node на VPS НЕ работает. Только kill $(pgrep -f isotope-node).

Сборка .aar
bash
cd /d D:\isotope\node
del isotope.aar
gomobile bind -target=android -androidapi 21 -ldflags "-checklinkname=0" -o isotope.aar ./mobile
copy /y isotope.aar D:\isotope\mobile\android\app
