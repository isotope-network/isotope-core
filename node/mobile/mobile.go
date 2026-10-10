// node/mobile/mobile.go
package mobile

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	sbimain "sbimain"
)

// MessageCallback — интерфейс для уведомления Flutter о новых сообщениях
type MessageCallback interface {
	OnMessage(message string)
}

var (
	node            *sbimain.Node
	nodeMu          sync.Mutex
	logs            []string
	logsMu          sync.Mutex
	logFile         *os.File
	filesDir        string
	messageCallback MessageCallback
)

// logWriter — перехватывает логи ядра
type logWriter struct{}

func (w *logWriter) Write(p []byte) (int, error) {
	addLog("%s", string(p))
	return len(p), nil
}

// addLog — добавляет запись в журнал.
// Защита от огромных строк: обрезаем до maxMsgLen.
// Источник огромных строк — libp2p и другие библиотеки, которые пишут
// через log.SetOutput; маркер [BIG N chars] помогает найти источник.
func addLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	const maxMsgLen = 4096
	if len(msg) > maxMsgLen {
		removed := len(msg) - maxMsgLen
		if len(msg) > 65536 {
			// Особо большие — помечаем и оставляем префикс.
			msg = fmt.Sprintf("[BIG %d chars] %s...[truncated %d chars]",
				len(msg), msg[:256], removed)
		} else {
			msg = msg[:maxMsgLen] + fmt.Sprintf("...[truncated %d chars]", removed)
		}
	}

	line := time.Now().Format("2006-01-02 15:04:05") + " " + msg

	logsMu.Lock()
	logs = append(logs, line)
	if len(logs) > 500 {
		logs = logs[len(logs)-500:]
	}
	logsMu.Unlock()

	if logFile != nil {
		logFile.WriteString(line + "\n")
	}
}

// errorJSON — безопасная сериализация ошибки в JSON.
func errorJSON(msg string) string {
	data, _ := json.Marshal(map[string]string{
		"status": "error",
		"error":  msg,
	})
	return string(data)
}

// GetLogs — возвращает последние логи
func GetLogs() string {
	logsMu.Lock()
	defer logsMu.Unlock()
	return strings.Join(logs, "\n")
}

// SaveLog — сохраняет логи в файл
func SaveLog() string {
	if logFile != nil {
		logFile.Sync()
	}
	path := filesDir + "/isotope.log"
	return path
}

// SetFilesDir — устанавливает директорию для файлов
func SetFilesDir(dir string) {
	filesDir = dir

	if dir != "" {
		var err error
		logFile, err = os.OpenFile(dir+"/isotope.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			logFile = nil
		}
	}
}

// Start — запускает узел
func Start(ethHash string, bootstrapPeers string, enableMDNS bool) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node != nil {
		return `{"status":"already_started"}`
	}

	log.SetOutput(&logWriter{})
	addLog("[MOBILE] Starting node...")

	if filesDir == "" {
		filesDir = "."
	}

	cfg := sbimain.Config{
		EthHash:    ethHash,
		Transports: []string{"ws", "tcp"},
		Bootstrap:  parseBootstrapPeers(bootstrapPeers),
		Port:       0,
		EnableMDNS: enableMDNS,
		ListenIP:   "0.0.0.0",
	}

	n := sbimain.NewNode(cfg)

	n.SetMessageHook(func(msg string) {
		if messageCallback != nil {
			messageCallback.OnMessage(msg)
		}
	})

	stateFile := filesDir + "/isotope_state.json"
	if err := n.StartMobile(stateFile); err != nil {
		addLog("[MOBILE] Failed to start node: %v", err)
		return errorJSON(err.Error())
	}

	node = n
	addLog("[MOBILE] Node started successfully")
	addLog("[MOBILE] PeerID: %s", n.GetStatus())

	return `{"status":"started"}`
}

// Stop — останавливает узел
func Stop() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{"status":"not_started"}`
	}

	if err := node.Stop(); err != nil {
		addLog("[MOBILE] Failed to stop node: %v", err)
		return errorJSON(err.Error())
	}

	node = nil
	addLog("[MOBILE] Node stopped")
	return `{"status":"stopped"}`
}

// GetStatus — возвращает статус узла
func GetStatus() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{"id":"","peers":0,"memory":0,"layers":0,"requests":0}`
	}
	return node.GetStatus()
}

// GetMessages — возвращает все ОБЫЧНЫЕ сообщения (Type == 0).
// Служебные (TypeDelivered, TypeRead, TypeContact* ) не попадают в UI.
// Для статусов — отдельный метод GetMessageStatuses (позже, 1.5).
// Для файловых чанков Text/PlainText обнуляются — base64 идёт через GetChunk.
func GetMessages() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `[]`
	}

	all := node.GetMessages()
	filtered := make([]sbimain.Message, 0, len(all))
	for _, m := range all {
		if m.Type == 0 {
			filtered = append(filtered, m)
		}
	}
	data, _ := json.Marshal(filtered)
	return string(data)
}

// GetChunk — возвращает base64 одного чанка файла по mediaID + chunkIndex.
// Используется Dart-ом, чтобы получить base64 без пересылки всего списка
// сообщений через MethodChannel (OOM при больших файлах).
// Формат ответа: {"status":"ok","data":"<base64>"} или errorJSON.
func GetChunk(mediaID string, chunkIndex int) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if mediaID == "" {
		return errorJSON("mediaID is required")
	}

	data, err := node.GetMessageChunkBase64(mediaID, chunkIndex)
	if err != nil {
		return errorJSON(err.Error())
	}

	result := map[string]string{
		"status": "ok",
		"data":   data,
	}
	out, _ := json.Marshal(result)
	return string(out)
}

// DeleteFile — удаляет все чанки файла из memory по mediaID.
// Также удаляет sent/<MediaID>.bin (если есть) и счётчик.
// Возвращает {"status":"ok","removed":N} или errorJSON.
func DeleteFile(mediaID string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if mediaID == "" {
		return errorJSON("mediaID is required")
	}

	removed := node.DeleteFile(mediaID)
	addLog("[FILE] deleted mediaID=%s removed=%d chunks", mediaID, removed)
	return fmt.Sprintf(`{"status":"ok","removed":%d}`, removed)
}

// SendMessage — отправляет сообщение всем пирам (broadcast).
// period: "10s" | "30s" | "1m" | ... | "never".
// mode: "" (при never) | "after_read" | "hard".
func SendMessage(text string, period string, mode string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	id, err := node.SendMessage(text, period, mode)
	if err != nil {
		return errorJSON(err.Error())
	}

	return fmt.Sprintf(`{"status":"ok","id":"%s"}`, id)
}

// SendToPeer — отправляет сообщение конкретному пиру по PeerID.
// period: "10s" | "30s" | "1m" | ... | "never".
// mode: "" (при never) | "after_read" | "hard".
func SendToPeer(peerID string, text string, period string, mode string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	id, err := node.SendToPeer(peerID, text, period, mode)
	if err != nil {
		return errorJSON(err.Error())
	}

	return fmt.Sprintf(`{"status":"ok","id":"%s"}`, id)
}

// SendVoice — отправляет голосовое сообщение.
// mediaData — base64 Opus/Ogg. duration — секунды.
// period: "10s" | "30s" | "1m" | ... | "never".
// mode: "" (при never) | "after_read" | "hard".
func SendVoice(peerID string, mediaData string, duration int, period string, mode string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if peerID == "" {
		return errorJSON("peerID is required")
	}
	if mediaData == "" {
		return errorJSON("mediaData is required")
	}

	id, err := node.SendVoice(peerID, mediaData, duration, period, mode)
	if err != nil {
		return errorJSON(err.Error())
	}

	addLog("[MEDIA] voice sent to %s (id=%s, %ds)", peerID, id, duration)
	return fmt.Sprintf(`{"status":"ok","id":"%s"}`, id)
}

// SendFile — отправляет файл конкретному пиру (E2E).
// fileBase64 — base64 исходного файла. fileSize — размер в байтах.
// period: "10s" | "30s" | "1m" | ... | "never".
// mode: "" (при never) | "after_read" | "hard".
// Возвращает {"status":"ok","id":"<MediaID>"}.
func SendFile(peerID string, fileBase64 string, fileName string, fileSize int64, period string, mode string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if peerID == "" {
		return errorJSON("peerID is required")
	}
	if fileBase64 == "" {
		return errorJSON("fileBase64 is required")
	}

	mediaID, err := node.SendFile(peerID, fileBase64, fileName, fileSize, period, mode)
	if err != nil {
		return errorJSON(err.Error())
	}

	addLog("[FILE] sent to %s (mediaID=%s, %s, %d bytes)", peerID, mediaID, fileName, fileSize)
	return fmt.Sprintf(`{"status":"ok","id":"%s"}`, mediaID)
}

// SendFileByPath — отправляет файл, читая его с диска по пути.
// filePath — внутри app dir. Go сам режет на чанки по 64 КБ.
// Base64 через MethodChannel не проходит — OOM устранён.
// Возвращает {"status":"ok","id":"<MediaID>"}.
func SendFileByPath(peerID, filePath, fileName, period, mode string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if peerID == "" {
		return errorJSON("peerID is required")
	}
	if filePath == "" {
		return errorJSON("filePath is required")
	}

	mediaID, err := node.SendFileByPath(peerID, filePath, fileName, period, mode)
	if err != nil {
		return errorJSON(err.Error())
	}

	addLog("[FILE] sent (by path) to %s (mediaID=%s, %s)", peerID, mediaID, fileName)
	return fmt.Sprintf(`{"status":"ok","id":"%s"}`, mediaID)
}

// SendPhoto — отправляет фото конкретному пиру (E2E).
// photoBase64 — base64 JPEG (сжатое).
// period: "10s" | "30s" | "1m" | ... | "never".
// mode: "" (при never) | "after_read" | "hard".
func SendPhoto(peerID string, photoBase64 string, period string, mode string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if peerID == "" {
		return errorJSON("peerID is required")
	}
	if photoBase64 == "" {
		return errorJSON("photoBase64 is required")
	}

	id, err := node.SendPhoto(peerID, photoBase64, period, mode)
	if err != nil {
		return errorJSON(err.Error())
	}

	addLog("[MEDIA] photo sent to %s (id=%s)", peerID, id)
	return fmt.Sprintf(`{"status":"ok","id":"%s"}`, id)
}

// SendPhotoByPath — отправляет фото, читая файл с диска по пути.
// filePath — внутри app dir (Dart передаёт путь из image_picker).
// Base64 через MethodChannel не проходит — OOM устранён.
func SendPhotoByPath(peerID, filePath, fileName, period, mode string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if peerID == "" {
		return errorJSON("peerID is required")
	}
	if filePath == "" {
		return errorJSON("filePath is required")
	}

	id, err := node.SendPhotoByPath(peerID, filePath, fileName, period, mode)
	if err != nil {
		return errorJSON(err.Error())
	}

	addLog("[MEDIA] photo sent (by path) to %s (id=%s)", peerID, id)
	return fmt.Sprintf(`{"status":"ok","id":"%s"}`, id)
}

// GetPeers — возвращает список пиров
func GetPeers() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `[]`
	}

	peers := node.GetPeers()
	data, _ := json.Marshal(peers)
	return string(data)
}

// GetMultiaddrs — возвращает multiaddr узла
func GetMultiaddrs() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `[]`
	}

	addrs := node.GetMultiaddrs()
	data, _ := json.Marshal(addrs)
	return string(data)
}

// ConnectToPeer — подключается к пиру
func ConnectToPeer(multiaddr string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	if err := node.ConnectToPeer(multiaddr); err != nil {
		return errorJSON(err.Error())
	}

	if dhtNode := node.GetDHT(); dhtNode != nil {
		dhtNode.RefreshOnDemand()
	}

	return `{"status":"connected"}`
}

// Announce — отправляет список наших multiaddr на bootstrap (справочник).
// Принимает JSON-массив строк из Dart.
func Announce(multiaddrsJSON string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	var multiaddrs []string
	if err := json.Unmarshal([]byte(multiaddrsJSON), &multiaddrs); err != nil {
		return errorJSON("invalid JSON: " + err.Error())
	}

	if len(multiaddrs) == 0 {
		return errorJSON("empty multiaddrs")
	}

	node.SendAnnounce(multiaddrs)
	addLog("[ANNOUNCE] sent: %d addrs", len(multiaddrs))

	return `{"status":"announced"}`
}

// FindPeerByID — ищет список multiaddr по PeerID через bootstrap-справочник.
func FindPeerByID(peerID string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	addrs, err := node.FindPeerByID(peerID)
	if err != nil {
		return errorJSON(err.Error())
	}

	result := map[string]interface{}{
		"status":     "found",
		"multiaddrs": addrs,
	}
	data, _ := json.Marshal(result)
	return string(data)
}

// GetEd25519PublicKey — возвращает Ed25519-публичный ключ (base64).
// Используется для отображения/диагностики. Для QR — GetMyQRData.
func GetEd25519PublicKey() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return ""
	}
	return node.GetEd25519PublicKey()
}

// GetX25519PublicKey — возвращает X25519-публичный ключ (base64).
func GetX25519PublicKey() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return ""
	}
	return node.GetX25519PublicKey()
}

// GetMyQRData — возвращает JSON для QR-кода версии 1.
// Формат: {"v":1,"peerID":"Qm...","ed25519_pub":"base64...","x25519_pub":"base64...","signature":"base64..."}
// Signature — подпись peerID || x25519_pub (4.4).
func GetMyQRData() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	return node.GetMyQRData()
}

// AddContact — добавляет или обновляет контакт.
// localName — как я называю контакт (не передаётся в сеть).
// remoteName — представление контакта о себе (пришло из QR или payload).
// readEnabled — сообщил ли контакт, что делится статусом прочтения.
// Возвращает {"status":"ok","verified":true|false} — verified читается после записи.
func AddContact(peerID, ed25519Pub, x25519Pub, signature, localName, remoteName string, readEnabled bool) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	if peerID == "" {
		return errorJSON("peerID is required")
	}

	if err := node.AddContact(peerID, ed25519Pub, x25519Pub, signature, localName, remoteName, readEnabled); err != nil {
		return errorJSON(err.Error())
	}

	verified := false
	if c, ok := node.GetContact(peerID); ok {
		verified = c.Verified
	}
	addLog("[CONTACTS] added: %s (verified=%v)", peerID, verified)

	if verified {
		return `{"status":"ok","verified":true}`
	}
	return `{"status":"ok","verified":false}`
}

// SetContactReadEnabled — устанавливает read_enabled для контакта.
func SetContactReadEnabled(peerID string, enabled bool) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if peerID == "" {
		return errorJSON("peerID is required")
	}
	if err := node.SetContactReadEnabled(peerID, enabled); err != nil {
		return errorJSON(err.Error())
	}
	return `{"status":"ok"}`
}

// SetMyReadEnabled — устанавливает мою настройку "делюсь ли статусом прочтения".
func SetMyReadEnabled(enabled bool) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	node.SetMyReadEnabled(enabled)
	return `{"status":"ok"}`
}

// GetMyReadEnabled — возвращает мою настройку.
func GetMyReadEnabled() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{"read_enabled":true}`
	}
	enabled := node.GetMyReadEnabled()
	if enabled {
		return `{"read_enabled":true}`
	}
	return `{"read_enabled":false}`
}

// SetMyDisplayName — устанавливает представление по умолчанию.
// Используется в QR и [CONTACT_REQUEST], если не переопределено.
func SetMyDisplayName(name string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if err := node.SetMyDisplayName(name); err != nil {
		return errorJSON(err.Error())
	}
	addLog("[SETTINGS] my_display_name set: %q", name)
	return `{"status":"ok"}`
}

// GetShowNotificationContent — возвращает настройку показа содержимого уведомлений.
func GetShowNotificationContent() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{"show_notification_content":true}`
	}
	enabled := node.GetShowNotificationContent()
	if enabled {
		return `{"show_notification_content":true}`
	}
	return `{"show_notification_content":false}`
}

// SetShowNotificationContent — устанавливает настройку показа содержимого.
func SetShowNotificationContent(enabled bool) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if err := node.SetShowNotificationContent(enabled); err != nil {
		return errorJSON(err.Error())
	}
	addLog("[SETTINGS] show_notification_content set: %v", enabled)
	return `{"status":"ok"}`
}

// GetMyDisplayName — возвращает представление по умолчанию.
func GetMyDisplayName() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{"my_display_name":""}`
	}
	name := node.GetMyDisplayName()
	data, _ := json.Marshal(map[string]string{"my_display_name": name})
	return string(data)
}

// SetTtl — устанавливает период и режим удаления сообщений.
// period: "10s" | "30s" | "1m" | "5m" | "15m" | "30m" | "1h" | "4h" | "24h" | "never".
// mode: "" (при never) | "after_read" | "hard".
func SetTtl(period string, mode string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if err := node.SetTtl(period, mode); err != nil {
		return errorJSON(err.Error())
	}
	addLog("[SETTINGS] ttl set: period=%q mode=%q", period, mode)
	return `{"status":"ok"}`
}

// GetTtl — возвращает период и режим удаления сообщений.
// Формат: {"ttl_period":"...","ttl_mode":"..."}.
// ttl_mode — "" при never.
func GetTtl() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{"ttl_period":"never","ttl_mode":""}`
	}
	period, mode := node.GetTtl()
	data, _ := json.Marshal(map[string]string{
		"ttl_period": period,
		"ttl_mode":   mode,
	})
	return string(data)
}

// RenameContact — устанавливает локальное имя контакта.
// Локальное имя — как я называю контакт. Не передаётся в сеть.
func RenameContact(peerID, localName string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if peerID == "" {
		return errorJSON("peerID is required")
	}
	if err := node.RenameContact(peerID, localName); err != nil {
		return errorJSON(err.Error())
	}
	addLog("[CONTACTS] renamed %s → %q", peerID, localName)
	return `{"status":"ok"}`
}

// GetDeletedPeers — возвращает JSON-массив удалённых peerID.
// Dart фильтрует _discoveredNodes по этому списку.
func GetDeletedPeers() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `[]`
	}
	list := node.GetDeletedPeers()
	data, _ := json.Marshal(list)
	return string(data)
}

// RemoveFromDeleted — убирает peerID из списка удалённых.
// Вызывается при QR-возврате контакта (явное действие пользователя).
func RemoveFromDeleted(peerID string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if peerID == "" {
		return errorJSON("peerID is required")
	}
	if err := node.RemoveFromDeleted(peerID); err != nil {
		return errorJSON(err.Error())
	}
	addLog("[DELETED] removed %s", peerID)
	return `{"status":"ok"}`
}

// RemoveContact — удаляет контакт у меня. У собеседника остаётся.
func RemoveContact(peerID string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if peerID == "" {
		return errorJSON("peerID is required")
	}
	if err := node.RemoveContact(peerID); err != nil {
		return errorJSON(err.Error())
	}
	addLog("[CONTACTS] removed %s", peerID)
	return `{"status":"ok"}`
}

// SendContactHello — отправляет [CONTACT_HELLO] получателю.
// Открытое (Version=0). Запускает bootstrap-handshake:
// получатель ответит [CONTACT_HELLO_ACK] с публичными ключами,
// после чего можно слать [CONTACT_REQUEST] (E2E).
func SendContactHello(peerID string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if peerID == "" {
		return errorJSON("peerID is required")
	}

	if err := node.SendContactHello(peerID); err != nil {
		return errorJSON(err.Error())
	}
	addLog("[REQUESTS] contact_hello sent to %s", peerID)
	return `{"status":"ok"}`
}

// SendContactRequest — отправляет запрос на контакт по PeerID.
func SendContactRequest(peerID, name string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if peerID == "" {
		return errorJSON("peerID is required")
	}

	id, err := node.SendContactRequest(peerID, name)
	if err != nil {
		return errorJSON(err.Error())
	}
	addLog("[REQUESTS] sent to %s (id=%s)", peerID, id)
	return fmt.Sprintf(`{"status":"ok","id":"%s"}`, id)
}

// GetRequests — возвращает JSON со всеми pending-запросами.
func GetRequests() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `[]`
	}
	requests := node.GetRequests()
	data, _ := json.Marshal(requests)
	return string(data)
}

// AcceptRequestByID — принимает входящий запрос по ID.
func AcceptRequestByID(id string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if id == "" {
		return errorJSON("id is required")
	}
	if err := node.AcceptRequestByID(id); err != nil {
		return errorJSON(err.Error())
	}
	addLog("[REQUESTS] accepted %s", id)
	return `{"status":"ok"}`
}

// RejectRequestByID — отклоняет входящий запрос по ID.
func RejectRequestByID(id string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if id == "" {
		return errorJSON("id is required")
	}
	if err := node.RejectRequestByID(id); err != nil {
		return errorJSON(err.Error())
	}
	addLog("[REQUESTS] rejected %s", id)
	return `{"status":"ok"}`
}

// SendRead — отправляет подтверждение прочтения по msg_id.
// Вызывается из UI при открытии чата.
func SendRead(ref, recipient string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if ref == "" || recipient == "" {
		return errorJSON("ref and recipient are required")
	}
	if err := node.SendRead(ref, recipient); err != nil {
		return errorJSON(err.Error())
	}
	return `{"status":"ok"}`
}

// SendReadBatch — отправляет батч подтверждений прочтения.
// Принимает JSON-массив msg_id и peerID получателя.
// Одно сообщение [READ] вместо N отдельных.
func SendReadBatch(refsJSON, recipient string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}
	if recipient == "" {
		return errorJSON("recipient is required")
	}

	var refs []string
	if err := json.Unmarshal([]byte(refsJSON), &refs); err != nil {
		return errorJSON("invalid JSON: " + err.Error())
	}
	if len(refs) == 0 {
		return `{"status":"ok"}`
	}

	if err := node.SendReadBatch(refs, recipient); err != nil {
		return errorJSON(err.Error())
	}
	addLog("[READ] batch sent: %d refs to %s", len(refs), recipient)
	return `{"status":"ok"}`
}

// MarkReadLocally — помечает входящие сообщения как прочитанные локально.
// Принимает JSON-массив msg_id. Вызывается из Dart при открытии чата
// (перед sendReadBatch). Возвращает количество изменённых.
func MarkReadLocally(refsJSON string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	var refs []string
	if err := json.Unmarshal([]byte(refsJSON), &refs); err != nil {
		return errorJSON("invalid JSON: " + err.Error())
	}
	if len(refs) == 0 {
		return `{"status":"ok","changed":0}`
	}

	changed := node.MarkReadLocally(refs)
	return fmt.Sprintf(`{"status":"ok","changed":%d}`, changed)
}

// GetMessageStatuses — возвращает JSON со статусами всех сообщений.
// Формат: {"<msg_id>": 1|2|3, ...}. 0 — неизвестен (не включается).
// 1 — отправлено, 2 — доставлено, 3 — прочитано.
// Вызывается из UI для отрисовки галочек.
func GetMessageStatuses() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{}`
	}
	statuses := node.GetMessageStatuses()
	data, _ := json.Marshal(statuses)
	return string(data)
}

// GetContacts — возвращает JSON со всеми контактами.
func GetContacts() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `[]`
	}

	contacts := node.GetContacts()
	data, _ := json.Marshal(contacts)
	return string(data)
}

// GetContact — возвращает JSON контакта по PeerID.
func GetContact(peerID string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	c, ok := node.GetContact(peerID)
	if !ok {
		return errorJSON("contact not found")
	}
	data, _ := json.Marshal(c)
	return string(data)
}

// ConnectToPeerWithFallback — подключается к пиру, пробуя параллельно все multiaddr.
// Принимает JSON-массив строк.
func ConnectToPeerWithFallback(multiaddrsJSON string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	var multiaddrs []string
	if err := json.Unmarshal([]byte(multiaddrsJSON), &multiaddrs); err != nil {
		return errorJSON("invalid JSON: " + err.Error())
	}

	if len(multiaddrs) == 0 {
		return errorJSON("empty multiaddrs")
	}

	usedAddr, err := node.ConnectToPeerWithFallback(multiaddrs)
	if err != nil {
		return errorJSON(err.Error())
	}

	if dhtNode := node.GetDHT(); dhtNode != nil {
		dhtNode.RefreshOnDemand()
	}

	result := map[string]string{
		"status": "connected",
		"used":   usedAddr,
	}
	data, _ := json.Marshal(result)
	return string(data)
}

// ============================================================
// DHT ОБЁРТКИ
// ============================================================

// JoinDHT — вход в DHT сеть
func JoinDHT(bootstrapPeers string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	peers := parseBootstrapPeers(bootstrapPeers)
	if len(peers) == 0 {
		return errorJSON("no bootstrap peers")
	}

	host := node.GetHost()
	if host == nil {
		return errorJSON("host not available")
	}

	dhtNode, err := sbimain.NewDHT(host)
	if err != nil {
		addLog("[DHT] Failed to create DHT: %v", err)
		return errorJSON(err.Error())
	}

	if err := dhtNode.JoinDHT(peers); err != nil {
		addLog("[DHT] Failed to join DHT: %v", err)
		return errorJSON(err.Error())
	}

	node.SetDHT(dhtNode)
	addLog("[DHT] Joined DHT network")
	return `{"status":"joined"}`
}

// FindPeer — поиск пира по PeerID через DHT (старый метод).
func FindPeer(peerID string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	knownPeers := node.GetKnownPeers()
	for _, addr := range knownPeers {
		if strings.Contains(addr, peerID) {
			addLog("[PEERS] Найден локально: %s", addr)
			result := map[string]interface{}{
				"status": "found",
				"addrs":  []string{addr},
			}
			data, _ := json.Marshal(result)
			return string(data)
		}
	}

	dhtNode := node.GetDHT()
	if dhtNode != nil && dhtNode.IsDHTActive() {
		addLog("[DHT] DHT активен, ищу %s...", peerID)
		dhtNode.RefreshOnDemand()

		addrInfos, err := dhtNode.FindPeer(peerID)
		if err == nil && len(addrInfos) > 0 {
			var addrs []string
			for _, ai := range addrInfos {
				for _, addr := range ai.Addrs {
					addrs = append(addrs, addr.String()+"/p2p/"+ai.ID.String())
				}
			}
			result := map[string]interface{}{
				"status": "found",
				"addrs":  addrs,
			}
			data, _ := json.Marshal(result)
			return string(data)
		}
		addLog("[DHT] FindPeer failed: %v", err)
	} else {
		addLog("[PEERS] DHT не активен (малая сеть), только локальный поиск")
	}

	return `{"status":"not_found","addrs":[]}`
}

// FindPeersViaNetwork — запрашивает список известных узлов у подключённых пиров
func FindPeersViaNetwork() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	peers := node.GetPeers()
	if len(peers) == 0 {
		return errorJSON("no connected peers")
	}

	addLog("[PEERS] Обмениваюсь списками с %d пирами...", len(peers))

	for _, peerID := range peers {
		if err := node.ExchangePeers(peerID); err != nil {
			addLog("[PEERS] Exchange with %s failed: %v", peerID, err)
			continue
		}
	}

	var allAddrs []string
	knownPeers := node.GetKnownPeers()
	myID := node.GetHost().ID().String()
	for _, addr := range knownPeers {
		if !strings.Contains(addr, myID) {
			allAddrs = append(allAddrs, addr)
		}
	}

	addLog("[PEERS] Найдено узлов: %d", len(allAddrs))

	result := map[string]interface{}{
		"status": "found",
		"addrs":  allAddrs,
	}
	data, _ := json.Marshal(result)
	return string(data)
}

// Provide — анонсирует себя в DHT
func Provide() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return errorJSON("node not started")
	}

	dhtNode := node.GetDHT()
	if dhtNode == nil {
		return errorJSON("DHT not initialized")
	}

	if err := dhtNode.Provide(); err != nil {
		addLog("[DHT] Provide failed: %v", err)
		return errorJSON(err.Error())
	}

	return `{"status":"provided"}`
}

// GetDHTInfo — информация о DHT
func GetDHTInfo() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{"started":false}`
	}

	dhtNode := node.GetDHT()
	if dhtNode == nil {
		return `{"started":false}`
	}

	return dhtNode.GetDHTInfo()
}

// SetMessageCallback — устанавливает колбэк для новых сообщений
func SetMessageCallback(callback MessageCallback) {
	messageCallback = callback
}

// ============================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ============================================================

// parseBootstrapPeers — разбирает строку bootstrap-пиров
func parseBootstrapPeers(bootstrapPeers string) []string {
	if bootstrapPeers == "" {
		return []string{}
	}

	var peers []string
	for _, p := range strings.Split(bootstrapPeers, ",") {
		if p = strings.TrimSpace(p); p != "" {
			peers = append(peers, p)
		}
	}
	return peers
}

// node/mobile/mobile.go
