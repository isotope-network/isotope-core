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

var (
	node     *sbimain.Node
	nodeMu   sync.Mutex
	logs     []string
	logsMu   sync.Mutex
	logFile  *os.File
	filesDir string
)

// logWriter — перехватывает логи ядра
type logWriter struct{}

func (w *logWriter) Write(p []byte) (int, error) {
	addLog("%s", string(p))
	return len(p), nil
}

// addLog — добавляет запись в журнал
func addLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
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
	stateFile := filesDir + "/isotope_state.json"
	if err := n.StartMobile(stateFile); err != nil {
		addLog("[MOBILE] Failed to start node: %v", err)
		return fmt.Sprintf(`{"status":"error","error":"%s"}`, err.Error())
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
		return fmt.Sprintf(`{"status":"error","error":"%s"}`, err.Error())
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
		return `{"id":"","peers":0,"memory":0,"layers":0}`
	}
	return node.GetStatus()
}

// GetMessages — возвращает все сообщения
func GetMessages() string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `[]`
	}

	messages := node.GetMessages()
	data, _ := json.Marshal(messages)
	return string(data)
}

// SendMessage — отправляет сообщение всем пирам
func SendMessage(text string, ttl int) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{"status":"error","error":"node not started"}`
	}

	id, err := node.SendMessage(text, ttl)
	if err != nil {
		return fmt.Sprintf(`{"status":"error","error":"%s"}`, err.Error())
	}

	return fmt.Sprintf(`{"status":"ok","id":"%s"}`, id)
}

// SendToPeer — отправляет сообщение конкретному пиру по PeerID
func SendToPeer(peerID string, text string, ttl int) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{"status":"error","error":"node not started"}`
	}

	id, err := node.SendToPeer(peerID, text, ttl)
	if err != nil {
		return fmt.Sprintf(`{"status":"error","error":"%s"}`, err.Error())
	}

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
		return `{"status":"error","error":"node not started"}`
	}

	if err := node.ConnectToPeer(multiaddr); err != nil {
		return fmt.Sprintf(`{"status":"error","error":"%s"}`, err.Error())
	}

	if dhtNode := node.GetDHT(); dhtNode != nil {
		dhtNode.RefreshOnDemand()
	}

	return `{"status":"connected"}`
}

// ============================================================
// DHT ОБЁРТКИ
// ============================================================

// JoinDHT — вход в DHT сеть
func JoinDHT(bootstrapPeers string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{"status":"error","error":"node not started"}`
	}

	peers := parseBootstrapPeers(bootstrapPeers)
	if len(peers) == 0 {
		return `{"status":"error","error":"no bootstrap peers"}`
	}

	host := node.GetHost()
	if host == nil {
		return `{"status":"error","error":"host not available"}`
	}

	dhtNode, err := sbimain.NewDHT(host)
	if err != nil {
		addLog("[DHT] Failed to create DHT: %v", err)
		return fmt.Sprintf(`{"status":"error","error":"%s"}`, err.Error())
	}

	if err := dhtNode.JoinDHT(peers); err != nil {
		addLog("[DHT] Failed to join DHT: %v", err)
		return fmt.Sprintf(`{"status":"error","error":"%s"}`, err.Error())
	}

	node.SetDHT(dhtNode)
	addLog("[DHT] Joined DHT network")
	return `{"status":"joined"}`
}

// FindPeer — поиск пира по PeerID (сначала локально, потом DHT)
func FindPeer(peerID string) string {
	nodeMu.Lock()
	defer nodeMu.Unlock()

	if node == nil {
		return `{"status":"error","error":"node not started"}`
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
		addLog("[DHT] DHT активен, ищу %s...", peerID[:16])
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
		return `{"status":"error","error":"node not started"}`
	}

	peers := node.GetPeers()
	if len(peers) == 0 {
		return `{"status":"error","error":"no connected peers"}`
	}

	addLog("[PEERS] Обмениваюсь списками с %d пирами...", len(peers))

	for _, peerID := range peers {
		if err := node.ExchangePeers(peerID); err != nil {
			addLog("[PEERS] Exchange with %s failed: %v", peerID[:16], err)
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
		return `{"status":"error","error":"node not started"}`
	}

	dhtNode := node.GetDHT()
	if dhtNode == nil {
		return `{"status":"error","error":"DHT not initialized"}`
	}

	if err := dhtNode.Provide(); err != nil {
		addLog("[DHT] Provide failed: %v", err)
		return fmt.Sprintf(`{"status":"error","error":"%s"}`, err.Error())
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