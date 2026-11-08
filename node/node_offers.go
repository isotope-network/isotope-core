// node/node_offers.go
package core

// node_offers.go — право отказа от файла (TypeFileOffer/Accept/Decline).
// Отправитель → [FILE_OFFER] с метаданными. Получатель → [FILE_ACCEPT] или
// [FILE_DECLINE]. При accept — sendFileChunksAsync. При decline — sent/
// удаляется, отправитель видит «Файл не принят».
// Таймаут оффера — 24 часа (cleanupLoop).

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// OFFER_TIMEOUT — сколько ждём ответа на [FILE_OFFER]. 24 часа.
const OFFER_TIMEOUT = 24 * time.Hour

// FileOffer — входящий оффер на приём файла (получатель).
// Хранится в isotope_offers.json. Не путать с sent/<mediaID>.bin (отправитель).
type FileOffer struct {
	MediaID        string    `json:"media_id"`
	OfferMsgID     string    `json:"offer_msg_id"`
	Sender         string    `json:"sender"`
	FileName       string    `json:"file_name"`
	FileSize       int64     `json:"file_size"`
	ChunkTotal     int       `json:"chunk_total"`
	TtlSeconds     int       `json:"ttl_seconds"`
	TtlMode        string    `json:"ttl_mode"`
	ExpiresAt      time.Time `json:"expires_at"` // TTL файла (если задан)
	CreatedAt      time.Time `json:"created_at"`
	OfferExpiresAt time.Time `json:"offer_expires_at"`    // 24 ч
	Cancelled      bool      `json:"cancelled,omitempty"` // отправитель отменил оффер
}

// OffersFile — формат isotope_offers.json.
type OffersFile struct {
	V      int         `json:"v"`
	Offers []FileOffer `json:"offers"`
}

// OFFERS_VERSION — текущая версия формата.
const OFFERS_VERSION = 1

// OffersStore — потокобезопасное хранилище входящих офферов.
type OffersStore struct {
	mu     sync.Mutex
	path   string
	offers []FileOffer
}

// NewOffersStore — создаёт store и загружает файл (если есть).
func NewOffersStore(path string) *OffersStore {
	store := &OffersStore{
		path:   path,
		offers: []FileOffer{},
	}
	if err := store.Load(); err != nil {
		log.Printf("[OFFERS] load failed: %v", err)
	}
	return store
}

// Load — читает файл. Если файла нет — пустой список.
func (store *OffersStore) Load() error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if store.path == "" {
		return nil
	}

	data, err := os.ReadFile(store.path)
	if err != nil {
		if os.IsNotExist(err) {
			store.offers = []FileOffer{}
			return nil
		}
		return fmt.Errorf("read failed: %w", err)
	}

	var file OffersFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	if file.V != OFFERS_VERSION {
		log.Printf("[OFFERS] unexpected version %d (expected %d)", file.V, OFFERS_VERSION)
	}

	if file.Offers == nil {
		file.Offers = []FileOffer{}
	}
	store.offers = file.Offers
	log.Printf("[OFFERS] loaded %d offers", len(store.offers))
	return nil
}

// saveLocked — сохраняет без лока (вызывается из методов с уже взятым локом).
func (store *OffersStore) saveLocked() error {
	if store.path == "" {
		return fmt.Errorf("offers path not set")
	}

	file := OffersFile{
		V:      OFFERS_VERSION,
		Offers: store.offers,
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	if err := os.WriteFile(store.path, data, 0600); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	return nil
}

// Add — добавляет оффер. Дедупликация по MediaID.
func (store *OffersStore) Add(offer FileOffer) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if offer.MediaID == "" {
		return fmt.Errorf("mediaID is required")
	}

	for i := range store.offers {
		if store.offers[i].MediaID == offer.MediaID {
			store.offers[i] = offer
			return store.saveLocked()
		}
	}

	store.offers = append(store.offers, offer)
	return store.saveLocked()
}

// Get — возвращает оффер по MediaID.
func (store *OffersStore) Get(mediaID string) (FileOffer, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	for _, o := range store.offers {
		if o.MediaID == mediaID {
			return o, true
		}
	}
	return FileOffer{}, false
}

// Remove — удаляет оффер по MediaID.
func (store *OffersStore) Remove(mediaID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	var kept []FileOffer
	for _, o := range store.offers {
		if o.MediaID != mediaID {
			kept = append(kept, o)
		}
	}
	if len(kept) == len(store.offers) {
		return fmt.Errorf("offer %s not found", mediaID)
	}
	store.offers = kept
	return store.saveLocked()
}

// GetAll — возвращает копию списка офферов.
func (store *OffersStore) GetAll() []FileOffer {
	store.mu.Lock()
	defer store.mu.Unlock()

	result := make([]FileOffer, len(store.offers))
	copy(result, store.offers)
	return result
}

// CleanupExpired — удаляет офферы старше OFFER_TIMEOUT.
// Возвращает список истёкших (mediaID + sender) — для push'а в UI.
func (store *OffersStore) CleanupExpired() []FileOffer {
	store.mu.Lock()
	defer store.mu.Unlock()

	now := time.Now()
	var alive []FileOffer
	var expired []FileOffer
	for _, o := range store.offers {
		if !o.OfferExpiresAt.IsZero() && now.After(o.OfferExpiresAt) {
			log.Printf("[OFFERS] expired %s (created %s)", o.MediaID, o.CreatedAt.Format(time.RFC3339))
			expired = append(expired, o)
		} else {
			alive = append(alive, o)
		}
	}
	if len(expired) > 0 {
		store.offers = alive
		_ = store.saveLocked()
	}
	return expired
}

// ============================================================
// NODE — методы для офферов
// ============================================================

// SendFileOffer — отправитель → получатель: [FILE_OFFER].
// Version=0. Ref = offerMsgID. Text = JSON {mediaID, fileName, fileSize, chunkTotal}.
// Через sendServiceViaBootstrap (синхронно, надёжно).
func (n *Node) SendFileOffer(peerID, mediaID, fileName string, fileSize int64, chunkTotal int, ttlSeconds int, ttlMode string, expiresAt time.Time) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if peerID == "" || mediaID == "" {
		return fmt.Errorf("peerID and mediaID are required")
	}

	payload := map[string]interface{}{
		"mediaID":    mediaID,
		"fileName":   fileName,
		"fileSize":   fileSize,
		"chunkTotal": chunkTotal,
		"ttlSeconds": ttlSeconds,
		"ttlMode":    ttlMode,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	offerMsgID := generateMsgID(fmt.Sprintf("offer:%s:%d", mediaID, time.Now().UnixNano()))
	msg := Message{
		ID:        offerMsgID,
		Text:      string(payloadJSON),
		Sender:    n.host.ID().String(),
		Recipient: peerID,
		Type:      TypeFileOffer,
		Version:   0,
		Ref:       offerMsgID,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		return fmt.Errorf("send offer failed: %w", err)
	}
	log.Printf("[FILE_OFFER] sent to %s (mediaID=%s, offer=%s, %d bytes, %d chunks)",
		peerID, mediaID, offerMsgID, fileSize, chunkTotal)
	return nil
}

// handleFileOffer — получатель: обработать [FILE_OFFER].
// Сохранить в pendingOffers, push в UI через messageHook.
func (n *Node) handleFileOffer(m Message) {
	if m.Sender == "" || m.Text == "" {
		log.Printf("[SERVICE] file_offer: empty sender/text, dropped")
		return
	}
	var payload struct {
		MediaID    string `json:"mediaID"`
		FileName   string `json:"fileName"`
		FileSize   int64  `json:"fileSize"`
		ChunkTotal int    `json:"chunkTotal"`
		TtlSeconds int    `json:"ttlSeconds"`
		TtlMode    string `json:"ttlMode"`
	}
	if err := json.Unmarshal([]byte(m.Text), &payload); err != nil {
		log.Printf("[SERVICE] file_offer: parse failed from %s: %v", m.Sender, err)
		return
	}
	if payload.MediaID == "" {
		log.Printf("[SERVICE] file_offer: empty mediaID from %s", m.Sender)
		return
	}
	if n.offers == nil {
		log.Printf("[SERVICE] file_offer: offers store not initialized")
		return
	}

	offer := FileOffer{
		MediaID:        payload.MediaID,
		OfferMsgID:     m.Ref,
		Sender:         m.Sender,
		FileName:       payload.FileName,
		FileSize:       payload.FileSize,
		ChunkTotal:     payload.ChunkTotal,
		TtlSeconds:     payload.TtlSeconds,
		TtlMode:        payload.TtlMode,
		CreatedAt:      time.Now(),
		OfferExpiresAt: time.Now().Add(OFFER_TIMEOUT),
	}
	if err := n.offers.Add(offer); err != nil {
		log.Printf("[SERVICE] file_offer: add failed: %v", err)
		return
	}
	log.Printf("[FILE_OFFER] received from %s (mediaID=%s, %s, %d bytes)",
		m.Sender, payload.MediaID, payload.FileName, payload.FileSize)

	// v2.0.7: порог — у получателя. Решение «запрашивать или нет» — его право.
	// -1 — всегда запрашивать (push в UI, ждать). 0 — никогда (авто-принять).
	// N > 0 — запрашивать, если fileSize >= N*МБ. Иначе — авто-принять.
	thresholdMB := -1
	if n.settingsStore != nil {
		thresholdMB = n.settingsStore.GetFileRequestThreshold()
	}

	needAsk := false
	switch {
	case thresholdMB < 0:
		needAsk = true
	case thresholdMB == 0:
		needAsk = false
	default:
		thresholdBytes := int64(thresholdMB) * 1024 * 1024
		needAsk = payload.FileSize >= thresholdBytes
	}

	if !needAsk {
		// Авто-принять: сразу [FILE_ACCEPT]. Без UI (пользователь выбрал «не спрашивать»).
		log.Printf("[FILE_OFFER] auto-accept (threshold=%d MB, size=%d): %s",
			thresholdMB, payload.FileSize, payload.MediaID)
		if err := n.SendFileAccept(m.Ref, payload.MediaID, m.Sender); err != nil {
			log.Printf("[FILE_OFFER] auto-accept failed: %v", err)
		}
		return
	}

	// Push в UI: предложение принять.
	if n.messageHook != nil {
		push := map[string]interface{}{
			"file_offer": map[string]interface{}{
				"media_id":     payload.MediaID,
				"offer_msg_id": m.Ref,
				"sender":       m.Sender,
				"sender_name":  n.nameForPeer(m.Sender),
				"file_name":    payload.FileName,
				"file_size":    payload.FileSize,
				"chunk_total":  payload.ChunkTotal,
			},
		}
		if data, err := json.Marshal(push); err == nil {
			n.messageHook(string(data))
		}
	}
}

// SendFileAccept — получатель → отправитель: [FILE_ACCEPT].
// Ref = offerMsgID, Text = mediaID.
func (n *Node) SendFileAccept(offerMsgID, mediaID, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if offerMsgID == "" || mediaID == "" || recipient == "" {
		return fmt.Errorf("offerMsgID, mediaID, recipient are required")
	}

	msg := Message{
		ID:        generateMsgID(fmt.Sprintf("accept:%s:%d", mediaID, time.Now().UnixNano())),
		Text:      mediaID,
		Sender:    n.host.ID().String(),
		Recipient: recipient,
		Type:      TypeFileAccept,
		Version:   0,
		Ref:       offerMsgID,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		return fmt.Errorf("send accept failed: %w", err)
	}
	log.Printf("[FILE_ACCEPT] sent to %s (mediaID=%s)", recipient, mediaID)
	return nil
}

// SendFileDecline — получатель → отправитель: [FILE_DECLINE].
// Ref = offerMsgID, Text = mediaID.
func (n *Node) SendFileDecline(offerMsgID, mediaID, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if offerMsgID == "" || mediaID == "" || recipient == "" {
		return fmt.Errorf("offerMsgID, mediaID, recipient are required")
	}

	msg := Message{
		ID:        generateMsgID(fmt.Sprintf("decline:%s:%d", mediaID, time.Now().UnixNano())),
		Text:      mediaID,
		Sender:    n.host.ID().String(),
		Recipient: recipient,
		Type:      TypeFileDecline,
		Version:   0,
		Ref:       offerMsgID,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		return fmt.Errorf("send decline failed: %w", err)
	}
	log.Printf("[FILE_DECLINE] sent to %s (mediaID=%s)", recipient, mediaID)
	return nil
}

// SendFileCancel — отправитель → получатель: [FILE_CANCEL] (v2.0.7+).
// Отправитель передумал ждать [FILE_ACCEPT]. Получатель удаляет оффер.
func (n *Node) SendFileCancel(offerMsgID, mediaID, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if offerMsgID == "" || mediaID == "" || recipient == "" {
		return fmt.Errorf("offerMsgID, mediaID, recipient are required")
	}

	msg := Message{
		ID:        generateMsgID(fmt.Sprintf("cancel:%s:%d", mediaID, time.Now().UnixNano())),
		Text:      mediaID,
		Sender:    n.host.ID().String(),
		Recipient: recipient,
		Type:      TypeFileCancel,
		Version:   0,
		Ref:       offerMsgID,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		return fmt.Errorf("send cancel failed: %w", err)
	}
	log.Printf("[FILE_CANCEL] sent to %s (mediaID=%s)", recipient, mediaID)
	return nil
}

// SendFileAbort — отправитель → получатель: [FILE_ABORT] (v2.0.7+).
// Оффер уже принят, чанки идут. Отправитель передумал.
// Получатель удаляет chunks/ и files/, видит «Отправитель прервал отправку».
// Ставит aborted[mediaID] — sendFileChunksAsync остановится перед следующим чанком.
func (n *Node) SendFileAbort(peerID, mediaID string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if peerID == "" || mediaID == "" {
		return fmt.Errorf("peerID and mediaID are required")
	}

	n.abortedMu.Lock()
	n.aborted[mediaID] = true
	n.abortedMu.Unlock()

	msg := Message{
		ID:        generateMsgID(fmt.Sprintf("abort:%s:%d", mediaID, time.Now().UnixNano())),
		Text:      mediaID,
		Sender:    n.host.ID().String(),
		Recipient: peerID,
		Type:      TypeFileAbort,
		Version:   0,
		Ref:       mediaID,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		log.Printf("[FILE_ABORT] send failed (will retry): %v", err)
	}
	log.Printf("[FILE_ABORT] sent to %s (mediaID=%s)", peerID, mediaID)
	n.scheduleAbortRetry(mediaID, "abort", peerID)
	return nil
}

// handleFileAbort — получатель: отправитель прервал отправку.
// Удалить все чанки из memory, files/<mediaID>.* (DeleteFile), push file_abort.
func (n *Node) handleFileAbort(m Message) {
	mediaID := m.Text
	if mediaID == "" {
		log.Printf("[SERVICE] file_abort: empty mediaID from %s", m.Sender)
		return
	}
	log.Printf("[FILE_ABORT] received from %s (mediaID=%s)", m.Sender, mediaID)

	// DeleteFile: memory (все чанки) + files/<mediaID>.* + sent/ + .meta + chunks/.
	// getChunk → null (memory пуст) → _fetchAndSaveChunk (Dart) остановится.
	if removed := n.DeleteFile(mediaID); removed >= 0 {
		log.Printf("[FILE_ABORT] deleted %d chunks for %s", removed, mediaID)
	}

	// v2.0.7+: сохранить aborted на диск — переживёт перезапуск.
	// getMessagesViaLibP2P (Dart) / resumeFile (Go) — не догружать.
	if n.abortedStore != nil {
		if err := n.abortedStore.Add(mediaID); err != nil {
			log.Printf("[FILE_ABORT] abortedStore.Add(%s) failed: %v", mediaID, err)
		}
	}

	// Push в UI: отправитель прервал отправку.
	if n.messageHook != nil {
		push := map[string]interface{}{
			"file_abort": map[string]interface{}{
				"media_id": mediaID,
				"sender":   m.Sender,
			},
		}
		if data, err := json.Marshal(push); err == nil {
			n.messageHook(string(data))
		}
	}

	if err := n.SendFileAbortAck(mediaID, "abort", m.Sender); err != nil {
		log.Printf("[FILE_ABORT] ack send failed: %v", err)
	}
}

// handleFileAccept — отправитель: получатель принял файл.
// Запустить sendFileChunksAsync.
func (n *Node) handleFileAccept(m Message) {
	mediaID := m.Text
	if mediaID == "" {
		log.Printf("[SERVICE] file_accept: empty mediaID from %s", m.Sender)
		return
	}
	log.Printf("[FILE_ACCEPT] received from %s (mediaID=%s)", m.Sender, mediaID)

	// v2.0.7: метаданные — из sent/<mediaID>.bin.meta.
	// Чанков в memory ещё нет (оффер — до sendFileChunksAsync).
	meta, err := n.readFileMeta(mediaID)
	if err != nil {
		log.Printf("[FILE_ACCEPT] read meta failed for %s: %v, ignored", mediaID, err)
		return
	}
	if meta.FileName == "" || meta.FileSize <= 0 || meta.ChunkTotal <= 0 {
		log.Printf("[FILE_ACCEPT] invalid meta for %s (name=%q size=%d chunks=%d), ignored",
			mediaID, meta.FileName, meta.FileSize, meta.ChunkTotal)
		return
	}

	// Push в UI: получатель принял, отправка начинается.
	if n.messageHook != nil {
		push := map[string]interface{}{
			"file_accept": map[string]interface{}{
				"media_id": mediaID,
				"sender":   m.Sender,
			},
		}
		if data, err := json.Marshal(push); err == nil {
			n.messageHook(string(data))
		}
	}

	mediaType := meta.MediaType
	if mediaType == "" {
		mediaType = "file"
	}
	log.Printf("[FILE_ACCEPT] starting chunks for %s (%s, %d bytes, %d chunks)",
		mediaID, meta.FileName, meta.FileSize, meta.ChunkTotal)

	go n.sendFileChunksAsync(m.Sender, mediaID, meta.FileName, meta.FileSize,
		mediaType, meta.TtlSeconds, meta.TtlMode, meta.ExpiresAt)
}

// handleFileDecline — отправитель: получатель отказался.
// Удалить sent/<mediaID>.bin, files/<mediaID>.*, пометить «не принят».
func (n *Node) handleFileDecline(m Message) {
	mediaID := m.Text
	if mediaID == "" {
		log.Printf("[SERVICE] file_decline: empty mediaID from %s", m.Sender)
		return
	}
	log.Printf("[FILE_DECLINE] received from %s (mediaID=%s)", m.Sender, mediaID)

	// Удалить sent/, sent/.meta и files/.
	_ = os.Remove(n.sentPath(mediaID))
	_ = os.Remove(n.metaPath(mediaID))
	if matches, err := filepath.Glob(filepath.Join(n.filesDirPath(), mediaID+".*")); err == nil {
		for _, p := range matches {
			_ = os.Remove(p)
		}
	}
	// Очистить счётчики.
	n.deliveredCountMu.Lock()
	delete(n.deliveredCount, mediaID)
	n.deliveredCountMu.Unlock()
	n.acceptedCountMu.Lock()
	delete(n.acceptedCount, mediaID)
	n.acceptedCountMu.Unlock()
	n.fileStatusMu.Lock()
	delete(n.completedFiles, mediaID)
	delete(n.fileStatusLastPush, mediaID)
	n.fileStatusMu.Unlock()
	n.unackedChunksMu.Lock()
	for id, info := range n.unackedChunks {
		if info.mediaID == mediaID {
			delete(n.unackedChunks, id)
		}
	}
	n.unackedChunksMu.Unlock()

	// Push в UI: «Файл не принят».
	if n.messageHook != nil {
		push := map[string]interface{}{
			"file_decline": map[string]interface{}{
				"media_id": mediaID,
				"sender":   m.Sender,
			},
		}
		if data, err := json.Marshal(push); err == nil {
			n.messageHook(string(data))
		}
	}
}

// handleFileCancel — получатель: отправитель отменил оффер.
// Удалить оффер из OffersStore, push file_cancel в UI.
func (n *Node) handleFileCancel(m Message) {
	mediaID := m.Text
	if mediaID == "" {
		log.Printf("[SERVICE] file_cancel: empty mediaID from %s", m.Sender)
		return
	}
	log.Printf("[FILE_CANCEL] received from %s (mediaID=%s)", m.Sender, mediaID)

	// v2.0.7: не удалять оффер — пометить cancelled.
	// Получатель видит «Отправитель отменил отправку» (кнопки неактивны).
	if n.offers != nil {
		if offer, ok := n.offers.Get(mediaID); ok {
			offer.Cancelled = true
			if err := n.offers.Add(offer); err != nil {
				log.Printf("[FILE_CANCEL] offers.Add(%s) failed: %v", mediaID, err)
			}
		}
	}

	// Push в UI: отправитель отменил оффер.
	if n.messageHook != nil {
		push := map[string]interface{}{
			"file_cancel": map[string]interface{}{
				"media_id":  mediaID,
				"sender":    m.Sender,
				"cancelled": true,
			},
		}
		if data, err := json.Marshal(push); err == nil {
			n.messageHook(string(data))
		}
	}

	if err := n.SendFileAbortAck(mediaID, "cancel", m.Sender); err != nil {
		log.Printf("[FILE_CANCEL] ack send failed: %v", err)
	}
}

// ============================================================
// ACK для [FILE_ABORT]/[FILE_CANCEL] (v2.0.7+)
// Отправитель шлёт до ACK. Без giving up.
// ============================================================

// abortRetryBackoff — интервалы переотправки abort/cancel.
// После последнего — потолок (последний элемент).
var abortRetryBackoff = []time.Duration{
	5 * time.Second,
	15 * time.Second,
	30 * time.Second,
	1 * time.Minute,
	2 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
}

// minInt — минимум из двух int.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SendFileAbortAck — получатель → отправитель: ACK на [FILE_ABORT]/[FILE_CANCEL].
// Ref = mediaID. Text = "abort" | "cancel".
func (n *Node) SendFileAbortAck(mediaID, kind, recipient string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	if mediaID == "" || recipient == "" {
		return fmt.Errorf("mediaID and recipient are required")
	}
	msg := Message{
		ID:        generateMsgID(fmt.Sprintf("abort-ack:%s:%d", mediaID, time.Now().UnixNano())),
		Text:      kind,
		Sender:    n.host.ID().String(),
		Recipient: recipient,
		Type:      TypeFileAbortAck,
		Version:   0,
		Ref:       mediaID,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
	}
	if err := n.sendServiceViaBootstrap(msg); err != nil {
		return fmt.Errorf("send abort-ack failed: %w", err)
	}
	log.Printf("[FILE_ABORT_ACK] sent to %s (mediaID=%s, kind=%s)", recipient, mediaID, kind)
	return nil
}

// handleFileAbortAck — отправитель: получатель подтвердил abort/cancel.
// Остановить retry.
func (n *Node) handleFileAbortAck(m Message) {
	mediaID := m.Ref
	if mediaID == "" {
		log.Printf("[FILE_ABORT_ACK] empty ref from %s", m.Sender)
		return
	}
	n.pendingAbortsMu.Lock()
	if t, ok := n.pendingAbortsTimers[mediaID]; ok {
		t.Stop()
		delete(n.pendingAbortsTimers, mediaID)
	}
	delete(n.pendingAborts, mediaID)
	n.pendingAbortsMu.Unlock()
	log.Printf("[FILE_ABORT_ACK] received for %s (kind=%s) from %s", mediaID, m.Text, m.Sender)
}

// scheduleAbortRetry — запускает retry [FILE_ABORT]/[FILE_CANCEL] до ACK.
func (n *Node) scheduleAbortRetry(mediaID, kind, recipient string) {
	if mediaID == "" || recipient == "" {
		return
	}
	n.pendingAbortsMu.Lock()
	if _, exists := n.pendingAborts[mediaID]; exists {
		n.pendingAbortsMu.Unlock()
		return
	}
	n.pendingAborts[mediaID] = time.Now()
	n.pendingAbortsMu.Unlock()
	go n.abortRetryLoop(mediaID, kind, recipient)
}

// abortRetryLoop — переотправка abort/cancel с backoff до ACK.
// Без giving up — до ACK.
func (n *Node) abortRetryLoop(mediaID, kind, recipient string) {
	attempt := 0
	for {
		backoff := abortRetryBackoff[minInt(attempt, len(abortRetryBackoff)-1)]
		time.Sleep(backoff)

		n.pendingAbortsMu.Lock()
		_, still := n.pendingAborts[mediaID]
		n.pendingAbortsMu.Unlock()
		if !still {
			return
		}

		var err error
		switch kind {
		case "cancel":
			offerMsgID := mediaID
			if n.offers != nil {
				if off, ok := n.offers.Get(mediaID); ok {
					offerMsgID = off.OfferMsgID
				}
			}
			err = n.SendFileCancel(offerMsgID, mediaID, recipient)
		default:
			err = n.SendFileAbort(recipient, mediaID)
		}
		if err != nil {
			log.Printf("[FILE_ABORT] retry %s (attempt %d) failed: %v", mediaID, attempt+1, err)
		} else {
			log.Printf("[FILE_ABORT] retry %s (attempt %d) sent", mediaID, attempt+1)
		}
		attempt++
	}
}

// node/node_offers.go
