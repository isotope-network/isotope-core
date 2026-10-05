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

## Структура ядра (v1.29.0)

Ядро ISOTOPE — **библиотека** (пакет `core`).

node/
├── *.go              # package core (ядро)
├── main/main.go      # package main (точка входа для десктопа)
└── mobile/mobile.go  # package mobile (обёртка gomobile)

### Экспортированный API ядра

package core

type Config struct {
    EthHash    string
    Transports []string
    Bootstrap  []string
    Port       int
    ListenIP   string
}

func NewNode(config Config) *Node
func InitP2P(node *Node) error
func StartHTTP(node *Node) error
func StartMobile(node *Node) error
func Stop(node *Node) error
func HashText(text string) string
func GetMultiaddrs(node *Node) []string
func ConnectToPeer(node *Node, addr string) error

### Методы Node

func (n *Node) SendMessage(text string, ttl int) (string, error)
func (n *Node) SendToPeer(recipient, text string, ttl int) (string, error)
func (n *Node) SendReadBatch(refs []string, recipient string) error
func (n *Node) MarkReadLocally(refs []string) error
func (n *Node) AddContact(...) error
func (n *Node) RemoveContact(peerID string) error
func (n *Node) RenameContact(peerID, localName string) error
func (n *Node) GetMessages() []Message
func (n *Node) GetPeers() []string
func (n *Node) GetWeight() float64
func (n *Node) GetStatus() string

### Структура Message

type Message struct {
    ID               string
    Recipient        string
    Sender           string
    Text             string
    PlainText        string
    TTL              int
    Priority         int
    Timestamp        int64
    Version          int          // 0 = история, 2 = E2E
    Type             MessageType  // 0 = обычное, 1-9 = служебные
    Ref              string       // одиночная ссылка
    Refs             []string     // массив ссылок (батч)
    Status           MessageStatus // sent / delivered / read / hidden
    ReadLocally      bool         // я прочитал входящее
    TtlPeriodSeconds int          // 0 = never
    TtlMode          string       // "after_read" | "hard" | ""
    ExpiresAt        time.Time
    ReadEnabled      *bool
}

---

## Мобильная обёртка (mobile.go)

**Статус:** работает (v1.29.0)

### Методы (возвращают JSON-строки)

| Метод | Описание |
|-------|----------|
| Start(stateFile) | Запуск узла |
| Send(text, ttl) | Отправка (broadcast) |
| SendToPeer(recipient, text, ttl) | Отправка контакту |
| SendReadBatch(refs, recipient) | Батч [READ] |
| MarkReadLocally(refs) | Пометить прочитанным локально |
| GetMessages() | Список сообщений |
| GetPeers() | Список пиров |
| GetWeight() | Вес узла |
| GetStatus() | JSON-статус |
| GetMultiaddrs() | Список адресов |
| ConnectToPeer(addr) | Подключение |
| ConnectToPeerWithFallback(addrs) | С перебором |
| Announce(json) | Публикация адресов |
| FindPeerByID(peerID) | Поиск пира |
| AddContact(...) | Добавить контакт |
| RemoveContact(peerID) | Удалить контакт |
| RenameContact(peerID, name) | Переименовать |
| SendContactRequest(peerID, name) | Запрос |
| AcceptRequestByID(id) | Принять |
| RejectRequestByID(id) | Отклонить |
| GetRequests() | Список запросов |
| SetMyDisplayName(name) | Представление |
| SetTtl(period, mode) | TTL |
| GetMyQRData() | QR-данные |

---

## E2E-шифрование (v1.24)

### Ключи

| Ключ | Назначение | Тип | Файл |
|------|------------|-----|------|
| PeerID | Идентификация | RSA 2048 | isotope_state.json.key |
| Ed25519 | Подпись | Ed25519 | isotope_state.json.ed25519.key |
| X25519 | Шифрование E2E | X25519 | isotope_state.json.x25519.key |

### QR-формат (v1)

{
  "v": 1,
  "peerID": "QmX...",
  "ed25519_pub": "base64...",
  "x25519_pub": "base64...",
  "signature": "",
  "display_name": "Пётр",
  "read_enabled": true
}

### Шифрование

- box.Seal (X25519 + XSalsa20-Poly1305)
- Формат: nonce(24) || ciphertext
- Version = 2 — E2E
- Version = 0 — история

### Подпись

- Ed25519 над peerID || x25519_pub
- verified — сигнал UI, не пропуск

### Контакты

Файл: isotope_contacts.json

{
  "v": 1,
  "contacts": [
    {
      "peerID": "QmX...",
      "ed25519_pub": "...",
      "x25519_pub": "...",
      "signature": "...",
      "name": "Пётр (ремонт)",
      "remote_name": "Пётр, о ремонте в 17:30",
      "verified": false,
      "confirmed": false,
      "read_enabled": true,
      "added_at": "..."
    }
  ]
}

---

## Единый источник истины (v1.29)

**Принцип:** если поле выводимо из `Message` — оно в `Message`.

### Message.Status (v1.29)

**Было:** `MessageStatus` — отдельный map в Node и State.

**Проблема:** второй источник истины. Четыре пути удаления сообщений не чистили map.

**Стало:** `Message.Status`.

- Удаляется сообщение → уходит статус.
- Автоматически, во всех путях.
- `Memory.SetStatus(id, status)` — по образцу `SetExpiresAt`.
- Миграция: при `loadState` старый `messageStatus` → `Message.Status`.
- Лог: `[STATUS] migrated N statuses to Message.Status`.

### Message.ReadLocally (v1.29)

**Было:** `_readSent` в Dart (Set<String> в SharedPreferences).

**Проблема:** второй источник истины. При перезапуске — теряется.

**Стало:** `Message.ReadLocally bool`.

- `Memory.MarkReadLocally(refs)`.
- Dart: `_sendReadBatchFor` → `markReadLocally` перед `sendReadBatch`.
- При `loadMessages` — пересчёт `_unreadByPeer` для `readLocally == false`.
- Миграция: `_readSent` → `markReadLocally`, флаг `read_sent_migrated`.

### Message.Refs (v1.29)

**Было:** `[READ]` по одному на сообщение.

**Стало:** `Message.Refs []string` — массив.

- `Ref` — одиночная ссылка.
- `Refs` — батч.
- Обратная совместимость: если `Refs` пуст — читаем `Ref`.
- Version 0. Обратная совместимость через содержимое поля.
- `SendReadBatch(refs, recipient)`.
- `handleServiceMessage` case `TypeRead` — цикл по `Refs`.

---

## Offline-очередь (v1.29)

**Проблема:** оффлайн-получатель. VPS не буферизует.

**Решение:** pending до `[DELIVERED]`.

- `replicateMessage` — при адресном всегда `enqueuePending`.
- `flushPending` — не удаляет из очереди, только переотправляет.
- `removePendingByRef(ref)` — удаление по `[DELIVERED]`.
- `case TypeDelivered` → `removePendingByRef(m.Ref)`.
- Триггеры: `ConnectedF` (Notifiee) + `announceLoop` (4 мин).

**Проверено:** 10 сообщений подряд, получатель оффлайн → при появлении приходят мгновенно пачкой.

**VPS — курьер, не хранилище.**

---

## Миграции (v1.29)

**Обязательны. Старые данные не теряются.**

- `messageStatus` → `Message.Status`.
- `_readSent` → `ReadLocally`.
- Каждая миграция логируется.

---

## Контакт-протокол (v1.27)

Контакт — это не добавление в список. Это взаимное обещание слышать.
Ты предлагаешь. Я принимаю. Или отклоняю.
Без принуждения. Без навязывания.

### Bootstrap-handshake

1. `[CONTACT_HELLO]` — открытый. PeerID A + публичные ключи A.
2. `[CONTACT_HELLO_ACK]` — открытый. PeerID B + публичные ключи B.
3. `[CONTACT_REQUEST]` — E2E. Полный payload.
4. `[CONTACT_ACCEPT]` — E2E. Подтверждение + представление B.

### Разделение транспортов

| Тип сообщения | Транспорт |
|---------------|-----------|
| Открытые сервисные (HELLO, ACK) | Bootstrap (всегда) |
| E2E (REQUEST, ACCEPT, сообщения) | Circuit → bootstrap fallback |

### tempContacts

- Создаются при `[CONTACT_HELLO]`, если B не знает A.
- Нужны только для расшифровки `[CONTACT_REQUEST]`.
- Удаляются после `requests.Add`.
- Не сохраняются на диск, не в UI.

### Симметрия

- Обе стороны `confirmed: true`.
- Через `[CONTACT_ACCEPT]`.

### Удаление контакта — тихий отказ

Удаление — это не блокировка. Это тихий отказ.
Блокировка — принуждение. Тишина — свобода.

- `RemoveContact` — полная чистка.
- `isotope_deleted.json` — удалённые не возвращаются.
- B не знает о факте удаления.

---

## Система имён (v1.27)

PeerID — моя техническая суть.
RemoteName — моё представление для других.
Name — моё имя для себя.

Три уровня — три свободы: быть собой, быть понятым, быть узнанным.

### Два поля у контакта

| Поле | Кто задаёт | Передаётся | Приоритет |
|------|------------|------------|-----------|
| Name | Я | Нет | Высший |
| RemoteName | Контакт | Да | Средний |
| — | — | — | Fallback: PeerID |

UI: `Name` → `RemoteName` → PeerID.

### Single source of names (v1.29)

- Убран `_contactNames` из `connect_screen`.
- `_displayName` делегирует в `chatProvider.nameFor(peerID)`.
- `_loadContactsFromCore` вызывает `loadPeerNames()`.

---

## Статусы сообщений (v1.26)

Пять состояний:

| Иконка | Значение |
|--------|----------|
| ⏳ | Отправляется |
| ✓ | Отправлено |
| ✓✓ | Доставлено. Прочтения не будет |
| ✓✓ (цвет) | Прочитано |

### read_enabled

- В контакте.
- В Settings (`my_read_enabled`).
- Передаётся с каждым сообщением.
- В QR.

### hidden — терминальное

- Замок = «прочтения не будет».
- Не откатывается.

---

## TTL — право на забвение (v1.28)

Сообщение не «удаляется». Оно отпускается.
Как отпускают прошлое — без сожаления.
Забвение — это не потеря. Это освобождение.

### Периоды

10s / 30s / 1m / 5m / 15m / 30m / 1h / 4h / 24h / never

Дефолт: `never` («Не удаляются»).

### Режимы

«После прочтения» — уважение к получателю.
«Жёсткий» — контроль отправителя.
Два режима — два выбора. Не навязано — предложено.

**hard:**
- Таймер от получения.
- Отправитель: от `[DELIVERED]`.
- Получатель: от получения.

**after_read:**
- Таймер от прочтения.
- Оба делятся → синхронно.
- Получатель не делится → авто-hard + `[TTL_UPDATE]`.
- Отправитель не делится → авто-hard + `[TTL_UPDATE]`.

### Fallback 48 часов

Fallback — честность. Если получатель не делится — сеть решает сама.

### [TTL_UPDATE] (Type=8)

- Version = 0.
- Payload: `expires_in_seconds`.
- При авто-hard.

### FLAG_SECURE — право на тишину

Секрет — защита.
Длинное — не секрет.

Короткие TTL (10с – 1 мин) — запрет скриншотов.
Это право на тишину.
Тишина — это тоже свобода.

### Бейдж + превью

Живут ровно столько, сколько сообщение.
Всё вместе.

---

## Таймер отправки (v1.26)

- Задержка 0/3/5/10 сек.
- Сообщение сразу с круговым прогрессом.
- Кнопка «Отмена».
- Back — черновик.
- Home — таймер продолжается.

---

## Multiline input (v1.29)

- `TextField`: `maxLines: null`, `minLines: 1`.
- `keyboardType: multiline`, `textInputAction: newline`.
- `ConstrainedBox(maxHeight: 140)` — рост до ~5 строк, потом скролл.

---

## Relay-circuit (v1.23+)

**Проблема:** узлы за NAT не могут принимать входящие.

**Решение:** relay через VPS.

- Узел резервирует слот (`client.Reserve`).
- VPS форвардит, не хранит.
- ANNOUNCE автоматически добавляет relay-адрес.
- FIND fallback — relay-адрес.

**Обновление резервации (v1.25):**
- Notifiee `ConnectedF` — при reconnect.
- relayLoop 30 секунд.
- Exponential backoff.

**Условие отключения VPS:**

1. DHT покрывает 15+ узлов.
2. Hole punching работает для большинства NAT.
3. 2-3 независимых relay-узла.

---

## Иммунитет

Иммунитет — способность сети сохранять себя
без центрального управления.

### Технический иммунитет

- Три уровня защиты.
- Репликация: на 2+ живых узла.
- Селф-хилинг: heartbeat.
- Fallback: локальный IP → relay.
- Exponential backoff.

### Социальный иммунитет

- Вес от этического хеша.
- Лайки/дизлайки.
- Время размывает.
- Архив → удаление.

### Единство

Технический держит сеть живой.
Социальный — чистой.

### Масштаб

| Узлов | Технический | Социальный |
|-------|-------------|------------|
| 2-3 | Защищённый канал | Личное доверие |
| 5-10 | Ранний консенсус | Первые оценки |
| 15-20 | Иммунитет включается | Коллективная очистка |
| 50+ | Стабильный иммунитет | Устойчивые нормы |
| 100+ | Неубиваемая сеть | Саморегуляция |

---

## Ключевые компоненты

### 1. P2P-сеть (libp2p)

- Транспорт: TCP + WebSocket + TLS.
- Обнаружение: mDNS + DHT + NSD.
- Синхронизация: Gossip.
- Маршрутизация: Onion Routing v2.
- Relay-circuit.
- Маскировка.
- Reconnect, Flush.

### 2. Этический движок

- Вектор: 100 измерений.
- Слои: каждые 20 сообщений.
- Хеш: 7 заповедей.

### 3. Память

- Взвешенная, FIFO с архивом.
- Вес, старение, архив.
- TTL.
- Шифрование AES-256-GCM.
- PlainText.
- Status (v1.29).
- ReadLocally (v1.29).

### 4. Каналы

- Пороги: full=0.3, comment=0.5, vote=0.7.

### 5. Безопасность

- Onion Routing v2.
- Маскировка.
- Стеганография.
- Селф-хилинг.
- E2E: X25519 + box.Seal.
- Подпись: Ed25519.
- FLAG_SECURE.

### 6. Весовая модель

| Вес | Права |
|-----|-------|
| 0–0.3 | Читать |
| 0.3–0.5 | Полный доступ |
| 0.5–0.7 | Комментировать |
| 0.7–1.0 | Голосовать, relay |

### 7. Самоадаптация

- node/adapt.go.
- avgWeight, lowWeightRatio, highWeightRatio.
- Фоновая адаптация.

### 8. ИИ-слой

- Статус: v3.0+.

### 9. API

- REST: /send, /messages, /status, /health, /feedback.
- WebSocket: /ws.
- Фильтры.
- Стего.

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

## Инфраструктура

### VPS bootstrap/relay

- IP: 186.246.31.176.
- PeerID: QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi.
- Bootstrap: /ip4/186.246.31.176/tcp/9001/ws/p2p/QmR8u5YFdcKpM2onQvk7KV5qioai87aysi9JWLdV1LX1bi.
- Порты: 9000 (TCP), 9001 (WS), 8081 (HTTP API).

НЕ удалять /root/isotope/state/.

### Обновление VPS

pkill -f isotope-node
sleep 1
cd /root/isotope-core && git pull
cd /root/isotope-core/node && go build -o isotope-node ./main
nohup env NODE_ID=bootstrap ISOTOPE_PORT=9000 ISOTOPE_HTTP_PORT=8081 ISOTOPE_ENABLE_RELAY=true ./isotope-node > isotope.log 2>&1 &

### Сборка .aar

cd /d D:\isotope\node
del isotope.aar
gomobile bind -target=android -androidapi 21 -ldflags "-checklinkname=0" -o isotope.aar ./mobile
copy /y isotope.aar D:\isotope\mobile\android\app\libs\

### Сборка APK

cd /d D:\isotope\mobile
flutter build apk --debug

---

## Принципы

1. Децентрализация. Нет сервера.
2. Этический иммунитет.
3. Самообучение.
4. Приватность.
5. Неуязвимость.
6. Унификация.
7. Правка в корне.
8. Открытость без наивности.
9. Эмерджентное доверие.
10. Приватность по умолчанию.
11. Пользователь не гадает.
12. Право на забвение (TTL).
13. Право на тишину (FLAG_SECURE).
14. Тихий отказ.
15. Единый источник истины (v1.29).
16. Offline-очередь — ответственность отправителя (v1.29).
17. Миграции обязательны (v1.29).

---

## Масштаб и иммунитет

Иммунная система включается при 15+ узлах.

[Шкала иммунитета →](IMMUNITY_SCALE.md)

---

## Подробнее

- [P2P-сеть](P2P_NETWORK.md)
- [Этический движок](ETHICS_ENGINE.md)
- [Весовая модель доступа](WEIGHT_ACCESS_MODEL.md)
- [Токеномика](TOKENOMICS.md)
- [B2B-обмен данными](B2B_DATA_EXCHANGE.md)
- [Самоадаптация](AUTONOMOUS_ADAPTATION.md)
- [Шкала иммунитета](IMMUNITY_SCALE.md)