package mobile

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	core "sbimain"
)

var node *core.Node
var logMu sync.Mutex
var logs []string

// logWriter — перехватывает логи из ядра
type logWriter struct{}

func (w *logWriter) Write(p []byte) (int, error) {
	addLog("%s", string(p))
	return len(p), nil
}

// addLog — добавляет запись в журнал
func addLog(format string, args ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()
	entry := fmt.Sprintf(format, args...)
	logs = append(logs, entry)
	if len(logs) > 300 {
		logs = logs[1:]
	}
	fmt.Printf("[MOBILE] %s\n", entry)
}

// GetLogs — возвращает все логи в JSON
func GetLogs() string {
	logMu.Lock()
	defer logMu.Unlock()
	return toJSON(logs)
}

// Start — запускает узел
func Start(ethHash string, stateFile string, bootstrapPeers string, port int64, listenIP string) string {
	addLog("Start: начало, stateFile=%s", stateFile)

	// Перехватываем логи из ядра
	log.SetOutput(&logWriter{})
	addLog("Start: перехват логов ядра установлен")

	if stateFile == "" {
		addLog("Start: ОШИБКА — stateFile пуст")
		return errorJSON("start", fmt.Errorf("stateFile is required"))
	}

	var bootstrap []string
	if bootstrapPeers != "" {
		for _, p := range splitComma(bootstrapPeers) {
			if p != "" {
				bootstrap = append(bootstrap, p)
			}
		}
	}
	addLog("Start: bootstrap узлов: %d", len(bootstrap))

	cfg := core.Config{
		EthHash:    ethHash,
		Transports: []string{"ws"},
		Bootstrap:  bootstrap,
		Port:       int(port),
		EnableMDNS: false,
		ListenIP:   listenIP,
	}

	n := core.NewNode(cfg)
	addLog("Start: Node создан")

	if err := n.StartMobile(stateFile); err != nil {
		addLog("Start: ОШИБКА StartMobile: %v", err)
		return errorJSON("start", err)
	}
	addLog("Start: StartMobile успешен")

	node = n
	addLog("Start: завершён, stateFile=%s", stateFile)
	return fmt.Sprintf(`{"status":"started","stateFile":"%s"}`, stateFile)
}

// Send — отправляет сообщение
func Send(text string, ttl int64) string {
	if node == nil {
		addLog("Send: ОШИБКА — узел не запущен")
		return errorJSON("send", fmt.Errorf("node not started"))
	}
	addLog("Send: text=%s, ttl=%d", text, ttl)
	msgID, err := node.SendMessage(text, int(ttl))
	if err != nil {
		addLog("Send: ОШИБКА: %v", err)
		return errorJSON("send", err)
	}
	addLog("Send: успешно, msgID=%s", msgID)
	return fmt.Sprintf(`{"message_id":"%s","status":"sent"}`, msgID)
}

// GetMessages — возвращает все сообщения в JSON
func GetMessages() string {
	if node == nil {
		addLog("GetMessages: ОШИБКА — узел не запущен")
		return errorJSON("get_messages", fmt.Errorf("node not started"))
	}
	msgs := node.GetMessages()
	addLog("GetMessages: %d сообщений", len(msgs))
	return toJSON(msgs)
}

// GetPeers — возвращает список пиров в JSON
func GetPeers() string {
	if node == nil {
		addLog("GetPeers: ОШИБКА — узел не запущен")
		return errorJSON("get_peers", fmt.Errorf("node not started"))
	}
	peers := node.GetPeers()
	addLog("GetPeers: %d пиров", len(peers))
	return toJSON(peers)
}

// GetWeight — возвращает вес узла в JSON
func GetWeight() string {
	if node == nil {
		addLog("GetWeight: ОШИБКА — узел не запущен")
		return errorJSON("get_weight", fmt.Errorf("node not started"))
	}
	weight := node.GetWeight()
	addLog("GetWeight: %.4f", weight)
	return fmt.Sprintf(`{"weight":%f}`, weight)
}

// GetStatus — возвращает статус узла
func GetStatus() string {
	if node == nil {
		addLog("GetStatus: ОШИБКА — узел не запущен")
		return errorJSON("get_status", fmt.Errorf("node not started"))
	}
	status := node.GetStatus()
	addLog("GetStatus: %s", status)
	return status
}

// GetMultiaddrs — возвращает адреса узла
func GetMultiaddrs() string {
	if node == nil {
		addLog("GetMultiaddrs: ОШИБКА — узел не запущен")
		return errorJSON("get_multiaddrs", fmt.Errorf("node not started"))
	}
	addrs := node.GetMultiaddrs()
	addLog("GetMultiaddrs: %d адресов", len(addrs))
	return toJSON(addrs)
}

// ConnectToPeer — подключение к пиру
func ConnectToPeer(multiaddr string) string {
	if node == nil {
		addLog("ConnectToPeer: ОШИБКА — узел не запущен")
		return errorJSON("connect_to_peer", fmt.Errorf("node not started"))
	}
	addLog("ConnectToPeer: %s", multiaddr)
	if err := node.ConnectToPeer(multiaddr); err != nil {
		addLog("ConnectToPeer: ОШИБКА: %v", err)
		return errorJSON("connect_to_peer", err)
	}
	addLog("ConnectToPeer: успешно")
	return fmt.Sprintf(`{"status":"connected","multiaddr":"%s"}`, multiaddr)
}

// Stop — останавливает узел
func Stop() string {
	if node == nil {
		addLog("Stop: ОШИБКА — узел не запущен")
		return errorJSON("stop", fmt.Errorf("node not started"))
	}
	addLog("Stop: остановка...")
	if err := node.Stop(); err != nil {
		addLog("Stop: ОШИБКА: %v", err)
		return errorJSON("stop", err)
	}
	node = nil
	addLog("Stop: остановлен")
	return `{"status":"stopped"}`
}

// toJSON — сериализует в JSON
func toJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return `{"error":"json_marshal_failed"}`
	}
	return string(data)
}

// errorJSON — создаёт JSON с ошибкой
func errorJSON(operation string, err error) string {
	result := map[string]string{
		"error":     err.Error(),
		"operation": operation,
	}
	data, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return `{"error":"json_marshal_failed","operation":"` + operation + `"}`
	}
	return string(data)
}

// splitComma — разделяет строку по запятой
func splitComma(s string) []string {
	var result []string
	current := ""
	for _, ch := range s {
		if ch == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else if ch != ' ' {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}