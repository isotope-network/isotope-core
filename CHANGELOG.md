# История изменений ISOTOPE

## v2.0.6 (2026-11-02)

«Seed-фраза, Keystore, локализация»

### Добавлено
- Seed-фраза:
  - BIP39 seed (12/24 слова) → Ed25519 (PeerID) + X25519 (шифрование). Детерминированно.
  - `node/seed.go` — `GenerateSeedPhrase`, `ValidateSeedPhrase`, `SeedToEd25519`, `SeedToX25519`.
  - Библиотека: `github.com/kslamph/bip39-hdwallet/bip39` (drop-in замена tyler-smith).
- Android Keystore:
  - AES-256, CBC, NoPadding, alias `isotope_seed_key`.
  - `saveSeedToKeystore`, `loadSeedFromKeystore` в `MainActivity.kt`.
  - `getOrCreateKeystoreKey` не удаляет существующий ключ.
  - Seed переживает перезапуск.
- i18n (en/ru):
  - `LocaleProvider` (ChangeNotifier, SharedPreferences `app_locale`).
  - `l10n.yaml`, `app_en.arb`, `app_ru.arb` (~250 ключей).
  - Переведены: onboarding, settings, privacy, connect, profile, messages, log, QR.
  - Переключатель языка в Настройках (Системный / English / Русский).
- Разбиение `node.go`:
  - 198 КБ → 35 КБ + 8 файлов: `node_e2e.go`, `node_files.go`, `node_service.go`, `node_p2p.go`, `node_contacts.go`, `node_announce.go`, `node_settings.go`, `node_batch.go`.

### Изменено
- `InitP2P`: если `seedPhrase != ""` — Ed25519 identity (из seed), X25519 из seed, не `loadOrGenerateE2EKeys`, не `loadPrivateKey` (RSA). Иначе — RSA (как раньше).
- `mobile.go`: `GenerateSeedPhrase`, `SetSeedPhraseBeforeStart` (глобальная `pendingSeedPhrase`), `SetSeedPhrase`, `GetSeedPhrase`. `Start` читает `pendingSeedPhrase` до `StartMobile`.
- `chat_provider.dart`: `loadSeedFromKeystore` до `Start` (в `_startLibP2P`).
- `privacy_settings_screen.dart`: секция Seed-фраза — Создать / Показать / Восстановить.
- `settings_screen.dart`: пункт Язык (RadioGroup), подпись Приватности (privacySubtitle).

### Проверено
- Создал фразу → снёс → установил → восстановил → PeerID тот же.
- Keystore: force-stop → запуск → PeerID тот же (Ed25519, из Keystore).
- i18n: English → все экраны на английском; Русский → обратно; перезапуск — сохраняется.

### Технические данные
- HEAD: 0e12f4d
- VPS: bf87124 (Go не менялся с bf87124)

### Коммиты
- 65ca178, bf87124, dc1026a, 4a8359c, 187dfe9, ebd5ca8, 498b729, 0e12f4d

### Известные проблемы
- Keystore проверен только на Huawei. Samsung, Xiaomi — не проверяли.
- i18n: chat_screen, message_bubble, requests_section — не переведены (backlog).
- Seed-фраза: контакты не восстанавливаются (отдельно — бэкап контактов).

### Следующий шаг
- Бэкап контактов (A) — экспорт/импорт `isotope_contacts.json`.
- i18n backlog (chat, message_bubble, requests, time_format).

---

## v2.0.5 (2026-10-28)

«Голос и фото через чанки»

### Добавлено
- `isChunkedMediaType` (file/voice/photo) — вместо `== "file"`.
- `SendVoiceByPath`, `SendPhotoByPath` (чанки).
- `retryInfo.mediaType`.
- `_VoicePlayer` / `_PhotoView` с `localFilePath`.
- `getMessagesViaLibP2P` — `isChunkedMediaType` (агрегат).
- `duration.toLong` (gomobile).

### Исправлено
- `_VoicePlayer` mediaBase64 пуст — `localFilePath`.
- `_PhotoView` mediaBase64 пуст — `localFilePath`.
- Дубль `_messagesMap` (ADD id=) — `sendPhotoByPath` без `_addMessage` + `getMessagesViaLibP2P` — `isChunkedMediaType`.

### Проверено
- Голос 5 мин / 1 сек — работает.
- Фото — работает.

### Технические данные
- HEAD: a9ed773
- VPS: ba0fdbc

### Известные проблемы
- `TestBatchFlush10` — не сделан.
- `sendReplicaToPeer` лог — только для текстовых.
- `Mobile.stop` при закрытии — saveState через событие.
- Heap 114 МБ при старте — ленивая загрузка.
- `pendingDelivered` — не ограничен.
- Samsung Android 10 — краш.

---

## v2.0.4 (2026-10-28)

«Маскировка»

### Добавлено
- Все пакеты 64 КБ. `SphinxPayloadSize = 65536`. Текст, файл, фото, голос — не отличить.
- Батч текста (10 текстов = 1 пакет). Трафик ×1.4, не ×14.
- Dummy-трафик:
  - `SphinxFlagDummy` (бит 1). От отправителя, не relay.
  - `sendDummy` — случайный интервал 10–120 сек (×5 в фоне), случайный контакт, случайный текст (50–200 символов), случайная задержка (1–60 сек).
  - Получатель — drop (не memory, не hook, не `[DELIVERED]`).
  - `maskingEnabled` — тумблер в Privacy.
- Чанк 47 КБ — влезает в 64 КБ payload.
- Double base64 убран для файлов. `encryptBytesForRecipient`, `decryptBytesFromSender`.
- `sendSphinxPacketToRelay1` — вынесено из `sendViaSphinx` (dummy без msg).

### Изменено
- `buildSphinxPacket(route, payload, mixing, dummy)` — 4 аргумента.
- `processSphinxPacket` — 5 возвратов: `(next, mixing, dummy, remainder, err)`.
- `handleSphinxReplicaMsg` — `if MediaType == "dummy"` → drop.
- `SetDummyScreenOn` — три уровня (Dart, Kotlin, Go). Баг: `setDummyScreenOn` case в `MainActivity.kt` не добавили (тихо). Фикс `8417761`.
- Чистка: `_getStateFilePath`, `last_chunk_id`, `_fileLastChunkId` — удалены.

### Проверено
- Dummy: `[DUMMY] sent` ×3 за 5 мин (screenOn, 10–120 сек). ×3 за 10 мин (screenOff, 50–600 сек).
- Текст + батч (masking ON) — работает.
- Файл 10 МБ (Mixing OFF, masking ON) — 5–6 сек.
- Файл 10 МБ (Mixing ON, masking ON) — 48 сек, 209 Sphinx-пакетов 64 КБ.

### Технические данные
- HEAD: 8417761 (d.1.1), faf373c (d)
- VPS: 80c7cd2 (Go не менялся с d.1.1)

### Коммиты
- 10f95df, 2006d01, cb47148, 61864ac, 397db35, aee3084, faf373c
- 4b01909, 80c7cd2, 8417761

### Известные проблемы
- Dummy 4–64 КБ — ошибка (не делать). Dummy должен быть как реальный (64 КБ).
- Обфускация — v2.0.6 (отдельно, DPI, сложно).

---

## v2.0.3 (2026-10-27)

«Mixing для файлов»

### Добавлено
- Файлы через Sphinx + mixing:
  - `sendFileChunksAsync`: `MixingEnabled` для чанков.
  - `chunkWorker`: `if MixingEnabled { sphinxFileSem; sendViaSphinx } else sendToRecipient`.
  - `retryUnacked`: `if MixingEnabled { go func; sphinxFileSem; sendViaSphinx }`.
  - `retryInfo.mixingEnabled` + `restoreChunkMessage` + `resumeFile`.
- `SphinxPayloadFileSize = 131072` (128 КБ, double base64).
- `sphinxFileSem = 2` (hotfix, Huawei нагрузка).
- `SphinxTimeout = 15s`, write deadline 30s.
- `sendServiceViaBootstrap` — NewStream 15 → 30.
- `sendConfirmation` — `[DELIVERED]`/`[READ]` — сразу `fallbackSend` (не `tryDirectSend`).
- `sendReplicaToPeer` — NewStream 10 → 30 (VPS → Huawei).

### Проверено
- 1 МБ с mixing — 15 сек (19 чанков). Samsung открылся.
- 48 МБ с mixing — 3:28. Samsung открылся.
- 107 МБ с mixing — 5:10. Samsung 98% (34 не дошли). Hotfix — 7:46, собрано, галочки.
- 107 МБ без mixing — 9 сек.

### Технические данные
- HEAD: ec46a5a
- VPS: ec46a5a (PID 1003258)

### Коммиты
- 3ffe7a5, ca7839d, fb0b989, ec46a5a

### Известные проблемы
- 114 МБ — 98% (Samsung нагрузка). 15 сек + семафор 3 — 5:10.
- 107 МБ — 98% (Huawei нагрузка, `[DELIVERED]` timeout). Hotfix — 7:46.
- Двойной base64 — корень. Убрать в v2.0.4.

---

## v2.0.2 (2026-10-26)

«Mixing — флаг в пакете»

### Добавлено
- `Message.MixingEnabled bool` — в `memory.go`.
- `SphinxFlagsSize = 1`, `SphinxHopLayerSize = 250`, `SphinxHeaderSize = 750`, `SphinxFlagMixing`.
- `buildSphinxPacket(..., mixing bool)` — flags в каждом relay-слое. MAC покрывает flags.
- `processSphinxPacket` → `(peer.ID, bool, []byte, error)` — читает flags до MAC.
- `sleepExponential` — relay спит при mixing.
- `handleStream` (relay) — mixing из пакета, `if mixing { sleepExponential }`.
- `processMessageInternal`: `MixingEnabled: settings && useOnion`.
- `privacy_settings_screen.dart` — тумблер Mixing, подпись.

### Проверено
- Huawei Mixing вкл, Xiaomi Mixing выкл. Relay спят (mixing=true, delay= 1.88 + 6.99). Samsung ~15 сек.
- Без mixing — mixing=false, быстро.
- Доказано: mixing — свойство сообщения. Relay спят по флагу, не по своей настройке.

### Технические данные
- HEAD: 22d53e0
- VPS: 22d53e0 (PID 978219)

### Коммиты
- 2cf9160, 22d53e0

---

## v2.0.1 (2026-10-26)

«Sphinx routing»

### Добавлено
- Sphinx routing:
  - Слоистое шифрование. Каждый relay снимает свой слой.
  - Relay знает только следующий hop (полный `peer.ID`). Не знает отправителя и получателя.
  - VPS не в цепочке (когда 3 доверенных контакта).
  - Payload с длиной (4 байта). `[length 4 BE][JSON][padding]`.
  - `SphinxHopLayerSize = 249`, `SphinxHeaderSize = 747`.
  - `SphinxNextPeerIDLenSize = 1`, `SphinxNextPeerIDSize = 64`.
- `sendViaSphinx` — только прямой (Wi-Fi), фильтр `/p2p-circuit`, `AddAddrs`, `NewStream`. Fallback — `sendToRecipient`.
- `handleStream` (relay) — `FindPeerByID` + direct (симметрично).
- `processSphinxPacket`: `peer.Decode` (не `peer.ID`).

### Изменено
- Удалены: `sphinxCompactID`, `sphinxCacheID`, `sphinxLookupID`, `SphinxCompactIDSize`, `sphinxIDCache`, `sphinxIDCacheMu`.

### Проверено
- Huawei → relay1 → relay2 → Samsung. VPS не видел (`memory: 74`).
- `[SPHINX] sent ... size=4830`. `process ... MAC ok, decrypt ok`. Три текстовых сообщения — дошли.

### Технические данные
- HEAD: 51852c5
- VPS: 51852c5 (PID 975329)

### Коммиты
- 15fde92, 76d03ac, e7279e3, 8953007, 51852c5

### Известные проблемы
- Все 4 — в контактах у всех (компромисс v2.0.1). DHT (v2.0.5) — relay'и не из контактов.

---

## v1.35.0 (2026-10-19)

«Превью и шторка»

### Добавлено
- Превью для изображений и видео:
  - Определение по расширению: .jpg/.jpeg/.png/.gif/.webp/.bmp → image; .mp4/.mov/.mkv/.avi/.webm/.3gp → video.
  - В чате: image → `Image.file` + fullscreen (`InteractiveViewer`); video → первый кадр (`zero_cached_thumbnail`) + fullscreen-плеер (`video_player`).
  - В контактах: 🎨 для image, 🎬 для video, 📷 для photo, 📎 для other.
  - В шторке: те же иконки.
- Прогресс-бар в контактах:
  - `LinearProgressIndicator` под превью контакта, если последнее сообщение — файл и `fileProgress(mediaID)` в 0..99.
- Настройки шторки (3 тумблера):
  - `Settings.ShowNotificationCount` / `ShowNotificationNames` / `ShowNotificationProgress`.
  - Go: `SettingsStore` + `GetNotificationSettings` / `SetNotificationSettings`.
  - Dart: `privacy_settings_screen.dart` — три тумблера.
  - Kotlin: `IsotopeService.updateNotification(title, text, progress)` — `NotificationCompat.setProgress`.
  - Применяются сразу (`reloadNotificationSettings`, без перезапуска).
- Прогресс получателя:
  - `acceptedCount` (Go) — отдельный счётчик принятых чанков по `mediaID`.
  - `pushFileStatus(mediaID, accepted, total)`.
  - `_lastMessageByPeer` обновляется при первом чанке файла — превью в контактах сразу.
- Свои файлы в `files/`:
  - Go: `SendFileByPath` копирует в `files/<mediaID>.<ext>` (постоянная копия).
  - Dart: `MediaStorage.init()` использует `LibP2PService.getFilesDir()` — тот же `filesDir`, что Go.
  - `_restoreOwnFilePathAsync(mediaID)` — восстановление пути своего файла при первом чанке.
- Превью + прогресс одновременно:
  - `_buildFileView` / `_FileRow` — `Column(превью, бар)`. Не «или-или».
  - «Открыть» + прогресс — одновременно (файл на диске есть, но передача идёт).

### Изменено
- Чистка логов: убраны диагностические `NOTIF:`, `updateForegroundNotification:`, `updateNotification:`.
- `_fileMessages[mediaID] = msg.withLocalFilePath(agg.localFilePath)` — сохраняет `localFilePath` на последнем чанке.

### Проверено
- Файлы всех размеров — доходят, собираются, открываются.
- Прогресс-бар у отправителя и получателя — в чате, контактах, шторке.
- Превью изображений и видео — у отправителя и получателя.
- Превью + прогресс-бар одновременно во время передачи.
- .pdf — «Открыть» + прогресс-бар одновременно.
- Свои файлы сохраняются в `files/` — после перезапуска на месте.
- Видео — первый кадр сразу (не серый квадрат).
- Полноэкранный просмотр: изображения — zoom/pan, видео — play/pause/seek.
- Три тумблера шторки — независимые, сохраняются, применяются сразу.
- Иконки по типу: 📷, 🎨, 🎬, 📎.
- Текстовые, голосовые, фото с камеры, галочки, TTL, контакты, QR, имена — работают.

### Технические данные
- HEAD: 7028099
- VPS: b047c50 (Go не менялся)

### Известные проблемы
- «Сохранить» для файлов — backlog.
- «Переслать» — backlog.
- Бэкап приложения — backlog.
- Согласие на приём файла (право отказа) — архитектура согласована, не начат.
- Порядок файлов в очереди — последовательный — backlog.
- `sendReplicaToPeer` — лог только для текстовых — backlog.
- `Mobile.stop` при закрытии — saveState через событие — backlog.
- Heap 114 МБ при старте — ленивая загрузка — backlog.
- `_fileLastChunkId`, `last_chunk_id` — не используются — удалить.
- DHT Provide — падает (мало пиров).
- `pendingDelivered` на Xiaomi — не ограничен.
- Артефакты (2 пустых с весом 0.3) — наблюдать.

### Следующий шаг
- «Сохранить» для файлов (MediaStore/SAF).
- «Переслать» — отправить полученный/сохранённый файл другому контакту.
- Бэкап приложения (`isotope_media/` + state + contacts + settings).
- Согласие на приём файла (`TypeFileOffer`/`Accept`/`Decline`).
- Чистка `_fileLastChunkId`, `last_chunk_id`.

---

## v1.34.0 (2026-10-18)

«Галочки файлов»

### Добавлено
- Галочки статусов файлов — принцип:
  - Те же 5 состояний, 4 иконки, что у текстовых.
  - ⏳ — в очереди (pending).
  - ✓ (серая) — отправлено (Go вернул ID, `[DELIVERED]` нет).
  - ✓✓ (две серые) — доставлено + у получателя «Открыть» (файл собран) — только после `[FILE_COMPLETE]`, не после `[DELIVERED]`.
  - ✓🔒 (одна серая + замочек) — доставлено, получатель не делится `read_enabled`.
  - ✓✓ (две зелёные) — прочитано (`[READ]` получен).
- `GetFileStatuses` — status = max по чанкам (read > hidden > delivered > sent).
- Dart: `_fileStatuses[mediaID]`, `refreshFileStatuses` (polling 3 сек).
- `[READ]` для файлов — через `mediaID` (общий для отправителя и получателя), не через ID чанка.
- `Mobile.stop` при `onDestroy` — saveState/savePendingQueue.

### Изменено
- `handleDeliveredRef`: для файловых чанков не ставить `StatusDelivered`/`StatusHidden`.
- `deliveredCount++` + `pushFileStatus` — вне if/else (работает при любом `read_enabled`).
- `_sendReadBatchFor`: `[READ]` для файла — только если `localFilePath` есть (собран).

### Проверено
- ✓ — только когда файл отправлен полностью (не во время).
- ✓✓ / ✓🔒 — только после `[FILE_COMPLETE]` (после «Открыть» у получателя).
- Зелёные ✓✓ — после `[READ]`, даже если собрал не в чате.
- Прогресс при hidden (`read_enabled=false`) — работает.

### Технические данные
- HEAD: c9c04fd
- VPS: c9c04fd (PID 826279)

### Коммиты
- 0387890, 477f9d4, c9c04fd, 4eb9df0, a63119f, 31c77ad

### Известные проблемы
- Дубль `[FILE_COMPLETE]` — `tryAssemble` дважды в `_fetchAndSaveChunk` (закрыто в v1.33.0).
- Остальные — см. v1.35.0.

### Следующий шаг
- Прогресс в контактах.
- Настройки шторки.
- Прогресс получателя.
- Превью image/video.
- Свои файлы в `files/`.

---

## v1.33.0 (2026-10-16)

«Надёжность и транспорт»

### Добавлено
- O(1) индексы (Этап 1A):
  - `byID` (msgID → позиция) + `fileChunks` (mediaID → chunkIndex → позиция).
  - Ленивая инициализация в `Add`.
  - `GetByID`/`SetStatus`/`SetExpiresAt`/`ClearChunkData`/`Remove` — O(1) через `byID`.
  - `Remove` — swap с последним + переиндексация moved.
  - `rebuildIndexes` после `DeleteExpired`/`PurgeDead`.
  - `ClearFileChunk(mediaID, idx)` / `GetFileChunk(mediaID, idx)` — O(1) через `fileChunks`.
- Внутренний ack для `messageHook` (Этап 1B):
  - `hookAckWindow` (`map[int]time.Time`), `pendingHookAcks` + `pendingHookAcksMu`.
  - `Node.FileAck(mediaID, indices)` — `ClearFileChunk` + удаление из окна. Идемпотентно.
  - `cleanupPendingAcksLoop` — 30 сек тик, 120 сек таймаут, `[ACK_TIMEOUT]`.
  - Dart: `_ackBuffer` + `_flushAcks` + `_flushAllAcks`. Убран `clearChunkData` из `_fetchAndSaveChunk`.
- Fallback WS↔TCP:
  - `expandTransportAlternatives` — порт вычисляется динамически (TCP = port, WS = port+1).
  - `loadBootstrapPeers` — каждый адрес расширяется до двух (WS + TCP).
  - Решает проблему блокировки WS 9001 провайдером Городок.
  - TODO: remove when obfuscation is in place.
- gzip state:
  - `saveState` — JSON → gzip → (encrypt) → файл.
  - `loadStateData` — magic-check (1F 8B) → распаковка.
  - Обратная совместимость: plain JSON читается как есть.
  - 24,4 МБ → 1,1 МБ (22x сжатие).
- Кэш `getFilePath`:
  - `MediaStorage._pathCache` — кэш `mediaId → path` (включая `null`).
  - `invalidatePathCache(mediaId)` — публичный метод.
  - Убран `else if (existing.localFilePath.isEmpty)` из `getMessagesViaLibP2P`.
  - «Подготовка...» на файле 1,7 ГБ — незаметна (вместо 40-60 сек).
- `TypeFileComplete = 9` (Этап 2A):
  - `isServiceType` (добавлен Type 9), `handleFileComplete`, `SendFileComplete`.
  - `[FILE_COMPLETE]` после `tryAssemble` (только для входящих, `!aggAfter.isOwn`).
- Семафор для файловых чанков на relay:
  - `n.sendSem` (10) — файловые чанки не блокируют сервисные.
  - Прогресс отправителя — разрыв 1% (было 45%).
- `[FILE_ASSEMBLED]` — hold `sent/` until receiver assembles.

### Исправлено
- Баг 3: `[FILE_COMPLETE]` не доходил до Huawei — лавина `[RETRY]` на VPS блокировала `sendServiceViaBootstrap`.
- Баг 4: лавина `[RETRY] restore ... failed` на VPS — `chunkWorker` на relay добавлял в `unackedChunks`.
  - `chunkWorker` — `if n.isRelay { return }`.
  - `replicateMessage` — `isFileChunk && !n.isRelay`.
  - `InitP2P` — `if !n.isRelay` вокруг воркеров + `retryLoop`.
- Баг 5: дубль `[FILE_COMPLETE]` — `_fileCompleteSent` (`Set<String>`).
- Баг 6: трафик 2× — `PlainText` в JSON. Убран из `chunkWorker`, `replicateMessage`, `sendFileChunksAsync`.
- Баг 7: прогресс отправителя отставал нелинейно — семафор на relay.
- `attempts` в `retryUnacked` — мёртвый код. `attempts++` при fail, после 3 — giving up + delete.
- `read_enabled updated` — только при изменении.

### Проверено
- Тест 91,4 МБ — 7 мин 9 сек. Файл дошёл, открылся.
- VPS `grep -c "RETRY restore"` = 0 (было — сотни каждые 15 сек).
- Huawei `[FILE_COMPLETE]` — есть.
- `[ACK]` идёт батчами (count=2/4/5). `[ACK_TIMEOUT]` — 0.
- `getChunk failed` — 0. Падений сервиса — 0.
- «Сборка...» — 1:34. Разрыв прогресса — 17-24 сек, стабильный.
- gzip: 24,4 МБ → 1,1 МБ. memory: 26872 после перезапуска.
- Кэш `getFilePath`: «Подготовка...» незаметна.

### Технические данные
- HEAD: 29ed3b7 (потом 6a03bea)
- VPS: 29ed3b7 (PID 783810 → 6a03bea, PID 803943)
- Bootstrap: TCP 9000 (актуально с 16.10; WS 9001 — fallback, блокируется провайдером Городок; fallback автоматический — 3d82934). Временно, до маскировки трафика (v2.0+)

### Коммиты
- 5275acc, 31b2855, 3d82934, 8c9ea30, 6a50321, 29ed3b7
- cf5ffe1, 7625b27, 412d5cc, 6531a81, fb38d35, 3bea5cd, 6a03bea

### Известные проблемы
- `Mobile.stop` не вызывается при закрытии — saveState только через событие (закрыто в v1.34.0).
- memory: 26870 — чанки остаются после сборки (закрыто в v1.34.0 через `[FILE_COMPLETE]`).
- Heap 114 МБ при старте — загрузка 26 870 Message.
- lowmemorykiller на Xiaomi — наблюдение.
- DHT стартует при 2 узлах. Уточнить при 15+.
- netlinkrib: permission denied — libp2p на Android.

### Следующий шаг
- Галочки файлов.
- Прогресс в контактах.
- Настройки шторки.
- Превью image/video.

---

## v1.32.0 (2026-10-14)

«Передача файлов»

### Добавлено
- Передача файлов — полностью:
  - Чанки 64 КБ. `MediaType="file"`, `MediaID`, `ChunkIndex`, `ChunkTotal`, `FileName`, `FileSize`.
  - `sendFileChunksAsync` — один поток подачи, читает `sent/<mediaID>.bin` последовательно.
  - `chunkQueue` (буфер 1000) + 10 `chunkWorker` — backpressure, не drop.
  - `sent/<mediaID>.bin` — копия на диске, источник для retry.
  - Получатель: `saveChunk` на диск, `clearChunkData` (base64 в Go), `tryAssemble` (склейка).
  - Папка `chunks/<mediaID>/` удаляется после сборки.
- Батч `[DELIVERED]`:
  - `queueDelivered(sender, ref)` — копит refs.
  - Один таймер на sender (2 сек). 10 refs — flush немедленно.
  - Идемпотентность: `[DELIVERED]` шлётся всегда (даже если `memory.Add == false`).
- Retry `unackedChunks`:
  - `retryInfo` — метаданные (mediaID, chunkIndex, msgID, fileName, fileSize, chunkTotal, TTL, attempts, lastSent) ~200 байт.
  - Message восстанавливается из `sent/` через `fillFileChunkFromSent`.
  - Через 30 сек — `retryUnacked`. `attempts >= 3` — giving up.
  - Backpressure: `time.After(2s)` — отложить, не дропать.
- Прогресс — push, не pull:
  - `file_status` — `{mediaID, delivered, total}` — throttle 1 сек.
  - `file_complete` — `{mediaID}` — один раз (`completedFiles`).
  - Отправитель — `deliveredCount[mediaID]`. Получатель — `chunksCount[mediaID]`.
- Восстановление после перезапуска:
  - `resumeIncompleteFiles` — обход `sent/<mediaID>.bin`, восстановление `unackedChunks`.
  - `GetFileStatuses` — Go → Dart, заполнение `_fileProgress`.
- WAKE_LOCK permission для `IsotopeService` (e706d8a).
- Диагностика и подготовка:
  - `io.MultiWriter` для Go-логов в stdout (тег `I/GoLog` в adb logcat).
  - `randomDelay`/лог только для текстовых (B3).
  - `isPeerDead` только для `Peers()` (D).
  - очистка `deadPeers` в `Disconnected` (F″).
  - параллелизм 5→2 (G).
- Прочее:
  - `saveState` reads seen via `GetSeenCopy` (9be2a42).
  - `GetByID` без `GetAll` copy (225f3e2).
  - clear base64 in memory after `saveChunk` (4f98121).
  - delete all chunks by `mediaID` + local photo preview (a7bc93b).
  - `sent/` copy + streaming read + `flushPending` from disk (3913354).
  - throttle `savePendingQueue` + metadata-only for file chunks (cf83dab).
  - `sendFileByPath` + `sendPhotoByPath` + memory without base64 (79359cf).
  - OOM fix: `getChunk`, no base64 in hook/`GetMessages` (5f0d892).
  - `tryDirectSend` with 3s timeout (9d47745).

### Проверено
- Файлы всех размеров (9,6 МБ, 10 МБ, 78 МБ, 87 МБ, 107 МБ, 280 МБ):
  - Доходят до получателя.
  - Собираются на диске (кнопка «Открыть»).
  - Открываются (видео, файлы).
  - Прогресс-бар на обоих концах.
  - «Подготовка… → Отправка/Получение N% → Сборка… → Открыть/Отправлено».
  - Восстановление отправки после перезапуска.
  - Восстановление прогресса после перезапуска.
  - Очередь из 4 файлов — все собираются.
  - Устойчивость: принудительная остановка приложения — не теряет данные.
- Тест 1,68 ГБ (26 869 чанков, Huawei LTE → Xiaomi Wi-Fi, через VPS relay):
  - Прогон 2: запуск 13:02, «Открыть» 14:40. Время 1:38.
  - Файл собран, открывается.
  - Найдены 2 бага (Android выгрузка, `messageHook`) — backlog.
- Текстовые сообщения, контакты, QR, имена, статусы прочтения — работают.

### Технические данные
- HEAD: 62d73aa
- VPS: 62d73aa, PID 719635
- Bootstrap: TCP 9000 (WS 9001 — fallback)

### Коммиты
- 34d8824, 62791bb, 2ea643b, 49aad2f, 89ec480, e706d8a, 9be2a42, 225f3e2
- 4f98121, a7bc93b, 3913354, cf83dab, 79359cf, 5f0d892, 9d47745
- cb2fe99, f589a06, a6bb28c, 8ee5aec, d8a97ab, 27c5956, d795c91, 62d73aa

### Известные проблемы
- Android выгрузка приложения в фоне (MIUI) — прогресс теряется.
- `messageHook` теряет события при перегрузке (1,68 ГБ — 1 190 чанков потеряны).
- Галочки файлов — backlog.
- Порядок файлов в очереди — последовательный — backlog.
- `read_enabled updated` — только при изменении — backlog.
- `sendReplicaToPeer` — лог только для текстовых — backlog.
- Memory-aware лимиты (`availMem`) — backlog.
- Future-очередь `tryAssemble` — backlog.
- Артефакты (2 пустых с весом 0.3) — наблюдать.

### Следующий шаг
- O(1) индексы.
- Внутренний ack.
- Fallback WS↔TCP.
- gzip state.
- `TypeFileComplete`.
- Галочки файлов.

---

## v1.31.0 (2026-10-08)

«Голос, файлы, фото»

### Добавлено
- Голосовые сообщения — полностью:
  - Запись (Flutter record + разрешение RECORD_AUDIO per-action).
  - E2E (`box.Seal`) — как для текста.
  - Отправка (`SendToPeer` + `MediaType: "voice"`).
  - Плеер в `MessageBubble`: play/pause, прогресс-бар, длительность, seek.
  - Сохранение после перезапуска.
  - Превью «🎤 Голосовое сообщение».
  - `MediaType = "voice"`, один чанк.
- Файлы — работают (баг доставки — см. «Известные проблемы»):
  - Чанки 64 КБ, параллельно 5 потоков.
  - Сборка на диске (`isotope_media/chunks/` → `files/`).
  - Превью «📎 файл.txt», «Открыть».
  - Прогресс: «Отправка N%» / «Получение N%».
  - Батч `[DELIVERED]` (500 мс / 10 refs).
  - `[READ]` по последнему чанку — зелёные галочки.
  - Бейдж один на файл.
  - Уведомление одно на файл (не на чанк).
- Фото — работают:
  - Сжатие (1600px, JPEG 80).
  - Миниатюра в бабле, fullscreen с зумом (`InteractiveViewer`).
  - Превью «📷 Фото».
  - `MediaType = "photo"`, один чанк.
- Общие фиксы:
  - Задвоение файла у отправителя — убрано.
  - Превью для своих сообщений (текст, голос, файл, фото) — работает.
  - `_lastMessageByPeer` — ключ `chatPeer = isOwn ? recipient : sender`.
  - OOM-fix: `.queue` на VPS не растёт (relay не копит pending).
  - Глобальный семафор (10) на `sendReplicaToPeer` — защита libp2p.
  - Логи облегчены (убраны base64 чанков).

### Проверено
- Голосовые: запись, отправка, приём, плеер (пауза, seek), сохранение после перезапуска.
- Файлы: чанки, сборка, прогресс, батч `[DELIVERED]`, `[READ]` по последнему чанку, бейдж один, уведомление одно.
- Фото: сжатие, миниатюра, fullscreen с зумом.
- Превью для своих сообщений.

### Технические данные
- HEAD: 530c262
- VPS: 530c262, PID 605466

### Коммиты
- d8ae953, faa2f0c, 24340f2, f19fd43, 4224eae, 8ac05b4, c155d88, c7f0934, f2ea570
- 30503b8, e800e11, 21e9628, b8a0fe2, 41de31a, 4f8c259, 95e13d8, ada4c43, 899dd66, 2852db8, 530c262

### Известные проблемы
- Регрессия доставки Huawei→Xiaomi после `530c262` — сессия пересекла аккаунты, корень не зафиксирован, закрыта в v1.32.0.

### Следующий шаг
- Диагностика OOM 670 МБ.
- WAKE_LOCK.
- Файлы: backpressure, retry, resume.
- O(1) индексы.

---

## v1.30.0 (2026-10-06)

«Уведомления и пробуждение»

### Добавлено
- Уведомления системы:
  - `Message.SenderName` (`json:"sender_name,omitempty"`).
  - `nameForPeer(peerID)` — приоритет: свой → `MyDisplayName`; контакт → `Name` → `RemoteName`; иначе пусто.
  - Заполнение в 4 местах создания входящего.
  - Kotlin `onMessage` — если `!isForeground` → `showMessageNotification`.
  - Канал `isotope_messages`, importance HIGH.
  - Тап → Intent `EXTRA_PEER_ID` → `onNewIntent` → MethodChannel → Dart `openChatStream`.
- Разрешения per-action:
  - `permission_service.dart` — новый файл.
  - `ActivityCompat.requestPermissions` в Kotlin, `onRequestPermissionsResult` → результат в Dart.
  - `POST_NOTIFICATIONS` — при старте.
  - `CAMERA` — при QR-скане.
  - `BLUETOOTH_*`, `FINE_LOCATION`, `NEARBY_WIFI_DEVICES` — при «Найти рядом».
  - Батарея — диалог с инструкцией + `openAppSettings`.
  - SnackBar при отказе — кнопка «Настройки».
  - `AndroidManifest.xml` — `CAMERA` добавлен явно.
- Настройка `ShowNotificationContent`:
  - `Settings.ShowNotificationContent bool`.
  - Дефолт `true`.
  - Методы: Go `GetShowNotificationContent` / `SetShowNotificationContent`.
  - `mobile.go`, `libp2p_service.dart`, `MainActivity.kt`, `privacy_settings_screen.dart`.
- WakeLock:
  - `IsotopeService.kt` — `PARTIAL_WAKE_LOCK` (`isotope:network`).
  - `acquireWakeLock` в `onCreate`, `releaseWakeLock` в `onDestroy`.
  - Go-рутины libp2p не замерзают в фоне.

### Изменено
- Единый источник истины — три рефакторинга:
  - `MessageStatus` → `Message.Status`.
  - `_readSent` (Dart) → `Message.ReadLocally` (Go).
  - `_contactNames` (Dart) → `ChatProvider.nameFor(peerID)`.
- Batch `[READ]` — новый формат:
  - `Message.Refs []string`.
  - `SendReadBatch(refs, recipient)`.
  - `handleServiceMessage` case `TypeRead` — цикл по `Refs`.
  - Version 0. Обратная совместимость через содержимое.
  - `MarkReadLocally(refs)`.
- Offline queue — pending до `[DELIVERED]`:
  - `replicateMessage` — при адресном всегда `enqueuePending`.
  - `flushPending` — только переотправка, не удаление.
  - `removePendingByRef(ref)` — удаление по `[DELIVERED]`.
  - Триггеры: `ConnectedF` + `announceLoop` (4 мин).
- Offline queue fix — сервисные не в pending:
  - `enqueuePending` — только для не-сервисных (`!isServiceType`).
  - Причина: сервисные не имеют `[DELIVERED]` → бесконечный цикл handshake.
- Multiline input:
  - `TextField` в `ConstrainedBox(maxHeight: 140)`.
  - `maxLines: null`, `minLines: 1`, `keyboardType: multiline`.
  - `textInputAction: newline`, `onSubmitted` убран.

### Проверено
- Уведомления (имя + превью, тап → чат).
- Разрешения (камера, BLE/nearby, батарея).
- Privacy toggle.
- Offline queue (pending до `[DELIVERED]`).
- Batch `[READ]`.
- `ReadLocally` (бейдж после перезапуска).
- Multiline input.
- WakeLock (фоновое подключение).

### Технические данные
- HEAD: 60e537e
- VPS: 4ef7fb5

### Следующий шаг
- Голосовые сообщения (E2E, стеганография в WAV).
- Файлы (E2E, через relay или напрямую).
- BOTTOM OVERFLOWED — UI-баг в `connect_screen` (исправлено).
- Samsung Android 10 — диагностика (отложено).
- 6-10 минут подключения Xiaomi — диагностика (решено fallback WS↔TCP, временно).

---

## v1.29.0 (2026-10-05)

«Единый источник истины»

### Добавлено
- Single source of names:
  - Убран `_contactNames` из `connect_screen.dart`.
  - `_displayName` делегирует в `chatProvider.nameFor(peerID)`.
  - `_loadContactsFromCore` вызывает `loadPeerNames()`.
- Batch `[READ]`:
  - `Message.Refs []string` + fallback на `Message.Ref`.
  - Version 0. Обратная совместимость через содержимое поля.
  - `SendReadBatch(refs, recipient)`.
  - `handleServiceMessage` case `TypeRead` — цикл по `Refs`.
- `Message.Status` (перенос из `MessageStatus`):
  - Поле `Status MessageStatus` в `Message`.
  - Удалены `n.messageStatus`, `n.messageStatusMu`, `State.MessageStatus`.
  - `Memory.SetStatus(id, status)`.
  - Миграция: при `loadState` старый `messageStatus` → `Message.Status`.
- `Message.ReadLocally`:
  - Поле `ReadLocally bool` в Go + Dart.
  - `Memory.MarkReadLocally(refs)`, `Node.MarkReadLocally(refs)`.
  - `LibP2PService.markReadLocally`, `MainActivity.kt` case.
  - `_sendReadBatchFor` → `markReadLocally` перед `sendReadBatch`.
  - `loadMessages` — пересчёт `_unreadByPeer` для `readLocally == false`.
  - Удалён `_readSent`. Миграция: `_readSent` → `markReadLocally`, флаг `read_sent_migrated`.
- Offline queue: pending до `[DELIVERED]`:
  - `replicateMessage` — при адресном всегда `enqueuePending`.
  - `flushPending` — не удаляет из очереди, только переотправляет.
  - `removePendingByRef(ref)` — удаление по `[DELIVERED]`.
  - case `TypeDelivered` → `removePendingByRef(m.Ref)`.
  - Триггеры: `ConnectedF` (Notifiee) + `announceLoop` (4 мин).
- Multiline input:
  - `TextField` в `chat_screen.dart`: `maxLines: null`, `minLines: 1`.
  - `keyboardType: multiline`, `textInputAction: newline`.
  - `ConstrainedBox(maxHeight: 140)`.

### Проверено
- 10 сообщений подряд, получатель оффлайн → при появлении приходят мгновенно пачкой.
- Бейдж переживает перезапуск. Не теряется, не двоится.

### Коммиты
- 011d719, a36c617, 9a19879, 212f8f3, 6a9fac6, 64e1d8b, 4353842

---

## v1.28.0 (2026-10-04)

«Право на забвение»

### Добавлено
- TTL — полностью:
  - `Settings.TtlPeriod` / `Settings.TtlMode`.
  - `Message.TtlPeriodSeconds` / `TtlMode` / `ExpiresAt`.
  - Периоды: 10s / 30s / 1m / 5m / 15m / 30m / 1h / 4h / 24h / never.
  - «forever» → «never».
  - Режимы: hard / after_read.
  - `[TTL_UPDATE]` (Type=8).
  - Fallback 48 часов.
  - `DeleteExpired` в `cleanupLoop` (тикер 1 мин).
- FLAG_SECURE:
  - TTL от 10 сек до 1 мин — скриншоты запрещены.
  - Оптимизация `_lastSecureFlag`.
  - Снятие при dispose.
- Удаление контакта:
  - `RemoveContact` — полная чистка.
  - `isotope_deleted.json`.
  - Тихий отказ.
- Per-chat / per-peer:
  - `messagesFor(peerID)`.
  - Черновики — `Map<peerID, String>`.
  - Непрочитанные — `Map<peerID, int>`.
  - `[READ]` — только для текущего чата.
- UI:
  - Настройки → Сообщения (TTL: периоды + режимы).
  - Настройки → Приватность (`read_enabled`).
  - Профиль (`MyDisplayName`).
  - Оранжевые периоды 10s / 30s / 1m.
  - Умный формат времени.
  - Таймер отправки.
  - AppBar — чисто, только имя контакта.
  - Диалог «Как вас представить?».

### Коммиты
- d482dfc, 27ca05c

---

## v1.27.0 (2026-10-01)

«Контакты и идентификация»

### Добавлено
- Контакт-протокол (bootstrap-handshake):
  - `[CONTACT_HELLO]` → `[CONTACT_HELLO_ACK]` → `[CONTACT_REQUEST]` → `[CONTACT_ACCEPT]`.
  - Открытые сервисные — через bootstrap.
  - E2E — circuit → bootstrap fallback.
  - `tempContacts`.
  - Push через `messageHook`.
  - Симметрия: `confirmed: true`.
- Система имён:
  - `Name` / `RemoteName` / `PeerID`.
  - `MyDisplayName` в Settings.
  - QR содержит `display_name`.
  - Диалог «Как вас представить?».
  - Профиль → «Ваше имя».
  - Долгий тап → Открыть / Переименовать / Удалить.
  - Предупреждение о безопасности.

### Коммиты
- 6896800, bead650, 692418d, 67778cd, a2004db, d083997
- 849c388, 36a6af6, 0cb9de2, 2a9083a, 196af37, 156fe0a, 8e6280c

---

## v1.26.0 (2026-09-27)

«Стабильность и статусы»

### Добавлено
- PlainText в `Message`.
- Автоочистка своих E2E без `PlainText`.
- Relay: refresh reservation on reconnect + 30s loop.
- Relay: exponential backoff.
- UI: отображение `plainText`.
- Контакт-протокол (этап 5, 27.09):
  - `MessageType` (enum), `Type`, `Ref` в `Message`.
  - `requests.go` — `isotope_requests.json`.
  - `SendDelivered`, `SendRead`, `SendContactRequest`, `Accept/Reject`.
  - `handleServiceMessage` — обработка типов.
  - `messageStatus map[string]MessageStatus` — статусы.
  - `GetMessages` фильтрует `Type==0`.
  - UI: `requests_section.dart`.
  - `chat_provider.dart` — `sendRead` при показе.
- Onboarding (1.4, 27.09):
  - 3 экрана, флаг `first_launch_seen`.
- Подпись Ed25519 над `peerID||x25519_pub` (4.4, 27.09).
- Статусы сообщений (1.5, 28.09):
  - Пять состояний, четыре иконки:
    - ⏳ — «Отправляется…»
    - ✓ — «Отправлено»
    - ✓✓ (серый) — «Доставлено. Не прочитано.»
    - ✓✓🔒 (серый) — «Доставлено. Получатель не делится статусом.»
    - ✓✓ (зелёный) — «Прочитано.»
  - `[DELIVERED]` — всегда.
  - `[READ]` — только если `myReadEnabled == true` у получателя.
  - Свой `read_enabled == false` → не вижу чужих `[READ]`.
  - Симметрия.
  - Замок — факт «скрыто», не «не прочитано».
  - `[READ]` — только при открытии чата (`setChatOpen(true)`).
  - `StatusHidden` (4) — в константах `MessageStatus`.
  - `myReadEnabled bool` — поле в Node (дефолт `true`).
  - `qrDataV1.ReadEnabled *bool` — в QR (опциональное, дефолт `true`).
  - `Contact.ReadEnabled bool` — поле в контакте.
  - `GetMessageStatuses` — отдаёт map для UI.
  - `messageStatus` сохранение:
    - `scheduleSaveState` (throttle 5 сек) — при `setMessageStatus`.
    - `InitP2P` — обернуть `if n.messageStatus == nil` (не сбрасывать после `loadState`).
    - `loadState` — лог `[STATUS] restored N`.
- Таймер отправки (29.09):
  - `_pendingMessages`, `_pendingTimers`, `_pendingSeconds`.
  - Полупрозрачный бабл (60%), круговой прогресс, секунды, кнопка «Отмена».
  - Блокировка ввода (`TextField.enabled: false`).
  - `PopScope` — back → черновик + SnackBar.
  - Свайп — удаление.
  - `_sendDelay` в Provider (SharedPreferences `send_delay`). По умолчанию 0.
  - `_draftText` (SharedPreferences `draft_text`). Один на чат.
  - UI задержки: `messages_settings_screen.dart` — `RadioListTile` 0/3/5/10 сек.
- UI Настройки (29.09):
  - `settings_screen.dart`, `privacy_settings_screen.dart`, `messages_settings_screen.dart`.
  - Структура: ЛИЧНЫЕ (Приватность, Сообщения) + СИСТЕМНЫЕ (Подключение, Ввести код контакта).
  - Шестерёнка в `connect_screen` → `SettingsScreen`.
- Рефакторинг D (28.09):
  - `isServiceType(t MessageType) bool` — только классификация.
  - `handleServiceMessage(m Message)` — только обработка (без bool, без return).
  - В `handleStream`: три ветки — мне → обработать, relay → форвард, иначе → drop.
- VPS skip DHT и reconnectLoop (28.09):
  - `if !n.isRelay` вокруг `NewDHT` и `n.reconnectLoop()`.

### Исправлено
- Цикл DELIVERED/READ — разорван. Настоящая причина: VPS работал на старом коде (запущен Sep25, до `a92376a`). В том коде `handleServiceMessage` не было — `DELIVERED`/`READ` форвардились как обычные сообщения.
- Debug-логи убраны (`[CONFIRM] marshaled`, `[SERVICE] got Type`).
- VPS skip DHT/reconnectLoop — потеряны в `24ef2fd`, восстановлены в `7769609`.
- Потеря правок в коммите `24ef2fd` — прикладной вернул старую версию `node.go`. Урок: после прикладного — `git diff` + `findstr`.
- `messageStatus` — терялись при перезапуске. Причины: `setMessageStatus` не вызывал `saveState`; `InitP2P` сбрасывал `n.messageStatus = make(...)` после `loadState`.
- Self-QR блокируется.
- Retry loading contacts.
- Upsert nodes on alive-event.

### Проверено
- Цикл DELIVERED/READ — закрыт.
- Статусы: ✓ → ✓✓ серый → ✓✓ зелёный. Перезагрузка — сохраняет статусы.
- Таймер: задержка 5 сек → сообщение сразу в чате, круговой прогресс, секунды, кнопка «Отмена». Ввод блокируется. «Отмена» → текст в поле. Свайп → удаление. Back → черновик + SnackBar. Через 5 сек — отправка.
- Системные настройки — активированы (Подключение, Ввести код контакта).
- Рефакторинг D — relay не съедает чужие сервисные.
- VPS — нет `[DHT]`, нет `[RECONNECT]`.

### Технические данные
- HEAD: d3ef7e9 (29.09), 598c145 (27.09)
- VPS: d3ef7e9 (PID 388296), 598c145 (PID 340880)
- Bootstrap: TCP 9000 (WS 9001 — fallback)

### Коммиты
- 598c145, 112f87c, 2665b68, bbd5ccf, 24ef2fd, 7769609, 62b2ee5, ad4bd12
- 02aa15c, 6fa4707, e7eda18, 3a025f2, 25902f4, 7625298, d3ef7e9
- c3f55f8 (27.09, сессия 1 — fix handleStream фрагментация)
- fad8d26 (1.4 onboarding), 57f9591, a92376a (этап 5 protocol), e3db7c7

### Известные проблемы
- UI-настройка `read_enabled` в разделе «Приватность» — backlog (закрыто в v1.28.0).
- Тултипы к иконкам статусов — backlog.
- `[CONTACT_ACCEPT]` с payload — отложено.
- `[PROFILE_UPDATE]` — backlog.
- Батч `[READ]` — backlog (закрыто в v1.29.0).
- Архитектура контактов — единый источник истины — backlog (закрыто в v1.29.0).
- Periodic FIND — backlog.
- Foreground service — Xiaomi/Huawei — backlog.
- DHT Provide — падает (мало пиров).
- Samsung Android 10 — краш — отложено до неопределённого времени.

### Следующий шаг
- Контакт-протокол (bootstrap-handshake).
- Система имён.
- Профиль.
- Per-chat.
- TTL.

---

## v1.25.0 (2026-09-25)

### Добавлено
- UI 1.1 — терминология.
- UI 1.2 — единый вход «Добавить контакт».
- UI 1.3 — пустое состояние.

### Коммиты
- 5ebe222, 4940998, 988fe17

---

## v1.24.0 (2026-09-23)

«E2E-шифрование»

### Добавлено
- E2E 4.1–4.4:
  - 4.1 (`e54d056`): E2E-ключи (Ed25519 + X25519), QR v:1. Файлы `.ed25519.key`, `.x25519.key`. Публичные — вычисляются.
  - 4.2 (`9c2fc8b`): `Message.Recipient` (string, omitempty). `sendToRecipient` — direct или через bootstrap. Relay — пересылает по `Recipient`. Broadcast — как раньше.
  - 4.3.1 (`dc82e81`): два ключа в файлах. `curve25519.X25519` (не `ScalarBaseMult`). `GetMyQRData` расширен.
  - 4.3.2 (`f3aaeab`): `isotope_contacts.json` (отдельно от state). `verified: false`.
  - 4.3.3 (`3a3b859`): E2E-шифрование Text через `box.Seal`, `Version=2`. `nonce(24) || ciphertext`. Relay видит только шифротекст.
  - 4.4: подпись Ed25519 над `peerID || x25519_pub`. При QR — проверка подписи. `signature` заполняется. `verified: true`.
- Backoff reconnect (`1d2e3d3`): 1→2→4→8→16→30 сек.
- Параллельный dial (`57cfeaf`): `context.WithCancel`. Первый успех — победа.
- ANNOUNCE TTL 5 → 8 мин (`87524bd`).
- Логи: обрезка строк в Go и Flutter (4 КБ) (`1a44715`).

### Коммиты
- e54d056, 9c2fc8b, dc82e81, f3aaeab, 3a3b859, 39c97c5
- 1d2e3d3, 57cfeaf, 87524bd, 1a44715, b225a22

---

## v1.23.0 (2026-09-17)

«Relay-circuit и reconnect»

### Добавлено
- Multi-address ANNOUNCE.
- Relay-circuit.
- Flush on reconnect (13.09, три уровня):
  - `Notifiee ConnectedF` — основной триггер.
  - `markPeerAlive` → dead → alive — страховка (возвращает bool, flush в `pingPeers`).
  - `reconnectLoop` — третий рубеж.
  - Результат: flush за 1 секунду после восстановления связи.
- Backoff reconnect (18.09):
  - 1с → 2с → 4с → 8с → 16с → 30с (потолок).
  - Сброс при успешном reconnect — 1с.
  - Верхняя граница 30с.
- Параллельный dial (18.09):
  - `context.WithCancel`.
  - Первый успех — победа, остальные отменяются.
  - Не 5-сек ожидание, а 1-2 сек.
- TTL 5→8 мин (18.09):
  - `announceLoop` = 4 мин. Зазор при TTL 5 мин = 1 мин.
  - TTL 8 мин = 2 цикла. Зазор 4 мин.
  - При успешном ANNOUNCE — немедленно обновить TTL.
  - На VPS: если `connected: false` — удалять announced сразу.
- QR-проверка (13.09):
  - QR = `isotope:<peerID>`.
  - `[FIND]` → VPS → multiaddr.
  - `[CONNECT]` failed на локальный IP → `[NOTIFY]` connected → `[CONNECT]` success на relay.
  - Время: 5-7 секунд.

### Проверено
- Flush on reconnect — 1 сек.
- Backoff — VPS лог.
- Параллельный dial — 1 сек.
- TTL 5→8 — VPS.
- QR — полная цепочка отработала.

### Технические данные
- HEAD: 87524bd (18.09)
- VPS: 87524bd
- Bootstrap: WS 9001 (тогда)

### Коммиты
- 6b223c0, 1d2e3d3, 57cfeaf, 87524bd, 7301bc9

### Следующий шаг
- E2E (4.1–4.4)
- Контакт-протокол

---

## v1.22.0 (2026-09-13)

«Foreground Service и статусы узлов»

### Добавлено
- Foreground Service.
- Battery Optimization Whitelist.
- Отображение пиров через libp2p.
- Три статуса узла (`unknown` / `alive` / `dead`):
  - `unknown` (серый) — честность. Не `alive` (врём), не `dead` (врём).
  - `alive` (зелёный).
  - `dead` (перечёркнут).
- Время исправлено — `formattedTime` парсит UTC + `.toLocal()`.
- Кнопка «обновить» (`_scanNetwork`) больше не теряет узлы — убран `clearAllNodes()`.
- Кнопка «Отозвать» убрана из UI чата.
- Heartbeat больше не убивает libp2p-пиров — пропускаются (`libp2p://`).
- Синхронизация узлов при старте — `_syncDiscoveredNodesFromP2P` + `_loadNodes` эмитит в UI.
- Bootstrap/VPS фильтруется в 3 местах (`_isBootstrapPeer`, `_listenToP2P`, `_addNode`).
- `addDiscoveredPeer` — обновляет + эмитит (для `alive` из `dead`/`unknown`).
- Flush on reconnect (13.09, три уровня) — см. v1.23.0.
- QR-проверка (13.09) — см. v1.23.0.

### Проверено
- Три статуса — `unknown` (серый), `alive` (зелёный), `dead` (перечёркнут).
- Время — локальное, совпадает с ПК.
- Кнопка «обновить» — узлы не теряются.
- Кнопка «Отозвать» — отсутствует.
- Двусторонняя связь Xiaomi (Wi-Fi) ↔ Huawei (LTE) через VPS.
- Сообщения, бейдж, подзаголовок, время — работают.
- Глобус — работает (нашёл узел). Проверен на 1 узле, не проверен на нескольких.

### Технические данные
- HEAD: f8931df
- VPS: (старый PeerID QmNmr3Yq...)

### Коммиты
- b706f97, 716a48f, ccec4be, ff3d552, e1f6895, f8931df

### Отложено
- Глобус на нескольких узлах — не проверен.
- Старые дубликаты в state.
- BLE — отложен.

### Следующий шаг
- Relay-circuit
- Multi-address ANNOUNCE
- E2E

---

## v1.21.0 (2026-09-11)

### Исправлено
- Дубликаты сообщений.
- Бейдж непрочитанных.
- ANR.
- Reconnect loop.

### Коммиты
- ec0c59e, 919967d, c06d8a8

---

## v1.19.0 (2026-09-01)

### Добавлено
- Рефакторинг ядра.
- libp2p через FFI.
- Стабильный PeerID.
- NSD-обнаружение.

---

## v1.18.1 (2026-08-28)

### Добавлено
- `package main` → `package core`.
- Экспорт API.

---

## v1.18 (2026-08-16)

### Добавлено
- Каналы с весовыми уровнями.

---

## v1.17 (2026-08-16)

### Добавлено
- Самоадаптация.

---

## v1.16 (2026-08-16)

### Добавлено
- Onion Routing v2.

---

## v1.15 (2026-08-16)

### Добавлено
- Репликация.

---

## v1.14 (2026-08-16)

### Добавлено
- Голосовая стеганография.

---

## v1.13 (2026-08-16)

### Добавлено
- Обфускация AES-GCM.

---

## v1.12 (2026-08-16)

### Добавлено
- Селф-хилинг.

---

## v1.11 (2026-08-16)

### Добавлено
- TTL.
- Локальное шифрование.

---

## v1.10 (2026-08-15)

### Добавлено
- Onion Routing v1.

---

## v1.9 (2026-08-14)

### Добавлено
- 5 узлов в docker-compose.
- Документация.

---

## v1.8 (2026-08-14)

### Добавлено
- Ассоциативная память.
- Децентрализованный bootstrap.

---

## v1.7 (2026-08-14)

### Добавлено
- WebSocket + TLS.
- Этический хеш.

---

## v1.6 (2026-08-11)

### Добавлено
- Priority Gossip.

---

## v1.5 (2026-07-27)

### Добавлено
- REST API.
- WebSocket.
- Мобильное приложение.

---

## v1.4 (2026-07-19)

### Добавлено
- 100-мерные векторы.
- Биграммы.
- Мониторинг сети.

---

## v1.3 (2026-07-19)

### Добавлено
- preHash / antiHash.

---

## v1.2 (2026-07-19)

### Добавлено
- Этическое обучение.

---

## v1.1 (2026-07-19)

### Добавлено
- Этический иммунитет.

---

## v1.0 (2026-07-16)

### Первый стабильный релиз

---

**ISOTOPE эволюционирует.**
**Каждая версия — шаг к неуязвимой связи.**