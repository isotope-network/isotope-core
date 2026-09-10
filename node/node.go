package core

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"
)

const protocolID = "/sbicore/1.0.0"
const syncProtocolID = "/sbicore/sync/1.0.0"
const pingProtocolID = "/sbicore/ping/1.0.0"

const OBFUSCATION_PREFIX = "[SHUF]"
const STEGO_PREFIX = "[STEGO]"
const REPLICA_PREFIX = "[REPLICA]"
const RESTORE_PREFIX = "[RESTORE]"

// Config — конфигурация узла
type Config struct {
	EthHash           string
	Transports        []string
	Bootstrap         []string
	Port              int
	EnableMDNS        bool
	ListenIP          string
	EnableRelayServer bool
}

// Node — основной узел сети
type Node struct {
	host                   host.Host
	dhtNode                *DHTNode
	ethHash                string
	preHash                string
	antiHash               string
	lastSyncSent           time.Time
	lastSyncedLayers       [][]float64
	layersDirty            bool
	memory                 Memory
	assoc                  AssocMemory
	layers                 [][]float64
	msgCount               int
	nodeID                 int
	stateFile              string
	configTransports       []string
	configPort             int
	configBootstrap        []string
	configEnableMDNS       bool
	configListenIP         string
	configEnableRelayServer bool
	messageHook            func(string)
	mu                     sync.Mutex
	lastPing               map[string]time.Time
	deadPeers              map[string]bool
	adaptive               *AdaptiveParams
	channels               *ChannelStore
}

// NewNode — создаёт новый узел
func NewNode(cfg Config) *Node {
	port := cfg.Port
	if port == 0 {
		port = 9000
	}
	listenIP := cfg.ListenIP
	if listenIP == "" {
		listenIP = "0.0.0.0"
	}
	return &Node{
		ethHash:                cfg.EthHash,
		configTransports:       cfg.Transports,
		configPort:             port,
		configBootstrap:        cfg.Bootstrap,
		configEnableMDNS:       cfg.EnableMDNS,
		configListenIP:         listenIP,
		configEnableRelayServer: cfg.EnableRelayServer,
		memory: Memory{seen: make(map[string]bool)},
	}
}

// SetMessageHook — устанавливает колбэк для новых сообщений
func (n *Node) SetMessageHook(hook func(string)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.messageHook = hook
}

func (n *Node) getObfuscationKey() []byte {
	hash := sha256.Sum256([]byte(n.ethHash))
	return hash[:]
}

func (n *Node) obfuscate(msg string) string {
	key := n.getObfuscationKey()
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(cryptorand.Reader, nonce)
	ciphertext := gcm.Seal(nonce, nonce, []byte(msg), nil)
	return OBFUSCATION_PREFIX + base64.StdEncoding.EncodeToString(ciphertext)
}

func (n *Node) deobfuscate(data string) (string, bool) {
	if !strings.HasPrefix(data, OBFUSCATION_PREFIX) {
		return data, false
	}
	encoded := strings.TrimPrefix(data, OBFUSCATION_PREFIX)
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return data, false
	}
	key := n.getObfuscationKey()
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return data, false
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return data, false
	}
	return string(plaintext), true
}

func randomDelay(minMs, maxMs int) {
	n, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(maxMs-minMs)))
	delay := time.Duration(minMs+int(n.Int64())) * time.Millisecond
	time.Sleep(delay)
}

func (n *Node) HandlePeerFound(peerInfo peer.AddrInfo) {
	ctx := context.Background()
	if err := n.host.Connect(ctx, peerInfo); err != nil {
		return
	}
	go func() {
		time.Sleep(3 * time.Second)
		n.broadcastLayers()
	}()
}

func (n *Node) handleStream(stream network.Stream) {
	defer stream.Close()
	buf := make([]byte, 2*1024*1024)
	nr, err := stream.Read(buf)
	if err != nil {
		return
	}
	msg := strings.TrimSpace(string(buf[:nr]))

	if strings.HasPrefix(msg, REPLICA_PREFIX) {
		payload := strings.TrimPrefix(msg, REPLICA_PREFIX)
		var replicaMsg Message
		if err := json.Unmarshal([]byte(payload), &replicaMsg); err == nil {
			if plaintext, ok := n.deobfuscate(replicaMsg.Text); ok {
				replicaMsg.Text = plaintext
			}
			replicaMsg.ExpiresAt = time.Time{}
			replicaMsg.ReplicatedAt = time.Now()
			if n.memory.Add(replicaMsg) {
				if n.messageHook != nil {
					data, _ := json.Marshal(replicaMsg)
					n.messageHook(string(data))
				}
			}
		}
		return
	}

	if strings.HasPrefix(msg, RESTORE_PREFIX) {
		nodeID := strings.TrimPrefix(msg, RESTORE_PREFIX)
		replicas := n.memory.GetReplicasFor(nodeID)
		if len(replicas) > 0 {
			for _, r := range replicas {
				data, _ := json.Marshal(r)
				stream.Write([]byte(REPLICA_PREFIX + string(data) + "\n"))
			}
		}
		return
	}

	if strings.HasPrefix(msg, "PEERS:") {
		payload := strings.TrimPrefix(msg, "PEERS:")
		var peerAddrs []string
		if err := json.Unmarshal([]byte(payload), &peerAddrs); err == nil {
			for _, addr := range peerAddrs {
				peerInfo, err := peer.AddrInfoFromString(addr)
				if err == nil {
					n.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, time.Hour*24)
				}
			}
			allAddrs := n.GetKnownPeers()
			myAddrs := n.GetMultiaddrs()
			for _, addr := range myAddrs {
				if !strings.Contains(addr, "127.0.0.1") {
					allAddrs = append(allAddrs, addr)
				}
			}
			data, _ := json.Marshal(allAddrs)
			stream.Write([]byte("PEERS:" + string(data) + "\n"))
		}
		return
	}

	if strings.HasPrefix(msg, "RELAY:") {
		parts := strings.SplitN(strings.TrimPrefix(msg, "RELAY:"), ":", 2)
		if len(parts) == 2 {
			targetPeerID := parts[0]
			actualMsg := parts[1]
			if plaintext, ok := n.deobfuscate(actualMsg); ok {
				actualMsg = plaintext
			}
			targetPID, err := peer.Decode(targetPeerID)
			if err == nil {
				go func() {
					ctx := context.Background()
					s, err := n.host.NewStream(ctx, targetPID, protocolID)
					if err != nil {
						return
					}
					defer s.Close()
					fmt.Fprintf(s, "%s\n", n.obfuscate(actualMsg))
				}()
			}
		}
		return
	}

	if strings.HasPrefix(msg, STEGO_PREFIX) {
		return
	}

	if plaintext, ok := n.deobfuscate(msg); ok {
		msg = plaintext
	}

	if msg == n.ethHash {
		stream.Write([]byte("ACCEPTED\n"))
		return
	}

	remoteID := stream.Conn().RemotePeer().String()

	if strings.HasPrefix(msg, "CHAIN:") {
		parts := strings.SplitN(msg, ":", 3)
		if len(parts) == 3 {
			nextRelays := parts[1]
			actualMsg := parts[2]
			if nextRelays != "" {
				relays := strings.Split(nextRelays, ",")
				n.sendViaRelayChain(relays, actualMsg)
			} else {
				n.processMessageRelayed(actualMsg, remoteID)
			}
			return
		}
	}

	n.processMessage(msg, remoteID, false)
}

func (n *Node) handleReplicaData(data string) {
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(line, REPLICA_PREFIX) {
			payload := strings.TrimPrefix(line, REPLICA_PREFIX)
			var replicaMsg Message
			if err := json.Unmarshal([]byte(payload), &replicaMsg); err == nil {
				if plaintext, ok := n.deobfuscate(replicaMsg.Text); ok {
					replicaMsg.Text = plaintext
				}
				replicaMsg.ExpiresAt = time.Time{}
				replicaMsg.ReplicatedAt = time.Now()
				if n.memory.Add(replicaMsg) {
					if n.messageHook != nil {
						data, _ := json.Marshal(replicaMsg)
						n.messageHook(string(data))
					}
				}
			}
		}
	}
}

func (n *Node) handlePingStream(stream network.Stream) {
	defer stream.Close()
	buf := make([]byte, 64)
	nr, _ := stream.Read(buf)
	msg := strings.TrimSpace(string(buf[:nr]))
	if msg == "PING" {
		stream.Write([]byte("PONG\n"))
	}
}

func (n *Node) pingPeers() {
	go func() {
		for {
			time.Sleep(30 * time.Second)
			if n.host == nil {
				continue
			}
			peers := n.host.Network().Peers()
			for _, p := range peers {
				go func(peerID peer.ID) {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					s, err := n.host.NewStream(ctx, peerID, pingProtocolID)
					if err != nil {
						n.markPeerDead(peerID.String())
						return
					}
					defer s.Close()
					s.Write([]byte("PING\n"))
					buf := make([]byte, 64)
					s.SetReadDeadline(time.Now().Add(5 * time.Second))
					nr, err := s.Read(buf)
					if err != nil || strings.TrimSpace(string(buf[:nr])) != "PONG" {
						n.markPeerDead(peerID.String())
						return
					}
					n.markPeerAlive(peerID.String())
				}(p)
			}
		}
	}()
}

func (n *Node) markPeerAlive(peerID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.lastPing == nil {
		n.lastPing = make(map[string]time.Time)
	}
	if n.deadPeers == nil {
		n.deadPeers = make(map[string]bool)
	}
	n.lastPing[peerID] = time.Now()
	if n.deadPeers[peerID] {
		delete(n.deadPeers, peerID)
	}
}

func (n *Node) markPeerDead(peerID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.deadPeers == nil {
		n.deadPeers = make(map[string]bool)
	}
	if !n.deadPeers[peerID] {
		n.deadPeers[peerID] = true
	}
}

func (n *Node) isPeerDead(peerID string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.deadPeers[peerID]
}

func (n *Node) getPeerWeight(peerID string) float64 {
	msgs := n.memory.GetMessagesFrom(peerID)
	if len(msgs) == 0 {
		return 0.5
	}
	total := 0.0
	for _, m := range msgs {
		total += m.Weight
	}
	return total / float64(len(msgs))
}

func (n *Node) processMessageRelayed(msg string, senderID string) {
	id := generateMsgID(msg)
	newMsg := Message{
		ID:       id,
		Text:     msg,
		Sender:   senderID,
		Time:     time.Now().UTC().Format("2006-01-02T15:04:05"),
		IsOwn:    false,
		Score:    0,
		Weight:   0.5,
		Priority: 0,
		Mode:     1,
		Relayed:  true,
	}
	if n.memory.Add(newMsg) {
		n.replicateMessage(newMsg)
		if n.messageHook != nil {
			data, _ := json.Marshal(newMsg)
			n.messageHook(string(data))
		}
	}
}

func (n *Node) replicateMessage(msg Message) {
	if n.host == nil {
		return
	}
	msg.ReplicatedFrom = msg.Sender
	msg.ExpiresAt = time.Time{}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	peers := n.host.Network().Peers()
	var alive []peer.ID
	for _, p := range peers {
		if p.String() == msg.Sender {
			continue
		}
		if !n.isPeerDead(p.String()) {
			alive = append(alive, p)
		}
	}
	if len(alive) == 0 {
		return
	}
	replicaCount := 2
	if len(alive) < replicaCount {
		replicaCount = len(alive)
	}
	for i := 0; i < replicaCount; i++ {
		go func(peerID peer.ID) {
			randomDelay(10, 30)
			ctx := context.Background()
			s, err := n.host.NewStream(ctx, peerID, protocolID)
			if err != nil {
				return
			}
			defer s.Close()
			fmt.Fprintf(s, "%s%s\n", REPLICA_PREFIX, string(data))
		}(alive[i])
	}
}

func (n *Node) requestRestore() {
	if n.host == nil {
		return
	}
	myID := n.host.ID().String()[:8]
	peers := n.host.Network().Peers()
	for _, p := range peers {
		go func(peerID peer.ID) {
			randomDelay(100, 500)
			ctx := context.Background()
			s, err := n.host.NewStream(ctx, peerID, protocolID)
			if err != nil {
				return
			}
			defer s.Close()
			fmt.Fprintf(s, "%s%s\n", RESTORE_PREFIX, myID)
			buf := make([]byte, 2*1024*1024)
			s.SetReadDeadline(time.Now().Add(5 * time.Second))
			nr, _ := s.Read(buf)
			if nr > 0 {
				n.handleReplicaData(strings.TrimSpace(string(buf[:nr])))
			}
		}(p)
	}
}

func (n *Node) processMessageWithTTL(msg string, senderID string, isOwn bool, expiresAt time.Time) {
	n.processMessageInternal(msg, senderID, isOwn, expiresAt, "")
}

func (n *Node) processMessageWithID(msg string, senderID string, isOwn bool, expiresAt time.Time, id string) {
	n.processMessageInternal(msg, senderID, isOwn, expiresAt, id)
}

func (n *Node) processMessageWithModeAndTTL(msg string, senderID string, isOwn bool, mode int, expiresAt time.Time) {
	if mode == 0 || n.host == nil {
		n.processMessageInternal(msg, senderID, isOwn, expiresAt, "")
		return
	}
	relayCount := 4
	if mode == 2 {
		relayCount = 5
		time.Sleep(10 * time.Second)
	}
	relays := n.selectRelays(relayCount)
	if len(relays) < relayCount {
		n.processMessageInternal(msg, senderID, isOwn, expiresAt, "")
		return
	}
	n.sendViaRelayChain(relays, msg)
}

func (n *Node) processMessageWithMode(msg string, senderID string, isOwn bool, mode int) {
	n.processMessageWithModeAndTTL(msg, senderID, isOwn, mode, time.Time{})
}

func (n *Node) selectRelays(count int) []string {
	peers := n.host.Network().Peers()
	var trusted []peer.ID
	var fallback []peer.ID
	for _, p := range peers {
		if n.isPeerDead(p.String()) {
			continue
		}
		weight := n.getPeerWeight(p.String())
		if weight > 0.7 {
			trusted = append(trusted, p)
		} else {
			fallback = append(fallback, p)
		}
	}
	if len(trusted) >= count {
		perm := mathrand.Perm(len(trusted))
		result := make([]string, count)
		for i := 0; i < count; i++ {
			result[i] = trusted[perm[i]].String()
		}
		return result
	}
	selected := make([]string, 0)
	for _, p := range trusted {
		selected = append(selected, p.String())
	}
	perm := mathrand.Perm(len(fallback))
	for i := 0; len(selected) < count && i < len(fallback); i++ {
		selected = append(selected, fallback[perm[i]].String())
	}
	return selected
}

func (n *Node) sendViaRelayChain(relays []string, msg string) {
	if len(relays) == 0 {
		return
	}
	obfuscated := n.obfuscate(msg)
	chainMsg := fmt.Sprintf("CHAIN:%s:%s", strings.Join(relays[1:], ","), obfuscated)
	relayPeer, err := peer.Decode(relays[0])
	if err != nil {
		return
	}
	randomDelay(10, 50)
	ctx := context.Background()
	s, err := n.host.NewStream(ctx, relayPeer, protocolID)
	if err != nil {
		return
	}
	defer s.Close()
	s.Write([]byte(chainMsg + "\n"))
}

func (n *Node) processMessage(msg string, senderID string, isOwn bool) {
	n.processMessageInternal(msg, senderID, isOwn, time.Time{}, "")
}

func (n *Node) processMessageInternal(msg string, senderID string, isOwn bool, expiresAt time.Time, providedID string) {
	inputVector := textToVector(msg)
	outputVector, _ := forward(inputVector, n.layers)
	answer := vectorToText(outputVector)

	n.mu.Lock()
	similar := n.memory.FindSimilar(msg, 0.7)
	var contextVector []float64
	if len(similar) > 0 {
		contextVector = make([]float64, VectorDim)
		totalWeight := 0.0
		for _, s := range similar {
			sv := textToVector(s.Text)
			w := s.Weight
			for i := range sv {
				contextVector[i] += sv[i] * w
			}
			totalWeight += w
		}
		for i := range contextVector {
			contextVector[i] /= totalWeight
		}
		for i := range inputVector {
			inputVector[i] = inputVector[i]*0.7 + contextVector[i]*0.3
		}
	}
	lr := 0.01
	if n.adaptive != nil {
		lr = n.adaptive.LearningRate
	}
	n.layers = train(n.layers, inputVector, outputVector, inputVector, lr)
	n.layersDirty = true
	n.mu.Unlock()

	go n.broadcastLayers()

	ethicsVec := hashToVector(n.ethHash)
	msgVec := textToVector(msg)
	ethicsScore := cosineSimilarity(ethicsVec, msgVec)
	initialWeight := 0.3 + ethicsScore*0.5

	if n.preHash != "" {
		preVec := hashToVector(n.preHash)
		preScore := cosineSimilarity(preVec, msgVec)
		preWeight := 0.2 + preScore*0.6
		initialWeight = initialWeight*0.4 + preWeight*0.6
	}
	if n.antiHash != "" {
		antiVec := hashToVector(n.antiHash)
		antiScore := cosineSimilarity(antiVec, msgVec)
		antiWeight := 0.3 + antiScore*0.5
		initialWeight = initialWeight * (1.0 - antiWeight*0.8)
		if initialWeight < 0.1 {
			initialWeight = 0.1
		}
	}

	priority := 0
	if isOwn {
		nodeWeight := 0.5
		activeMsgs := n.memory.GetActiveMessages(0.5)
		if len(activeMsgs) > 0 {
			totalWeight := 0.0
			for _, m := range activeMsgs {
				totalWeight += m.Weight
			}
			nodeWeight = totalWeight / float64(len(activeMsgs))
		}
		if nodeWeight > 0.7 {
			priority = 100
		}
	}

	id := providedID
	if id == "" {
		id = generateMsgID(msg)
	}

	newMsg := Message{
		ID:        id,
		Text:      msg,
		Sender:    senderID,
		Time:      time.Now().UTC().Format("2006-01-02T15:04:05"),
		IsOwn:     isOwn,
		Score:     0,
		Weight:    initialWeight,
		Priority:  priority,
		Mode:      0,
		ExpiresAt: expiresAt,
	}
	if n.memory.Add(newMsg) {
		n.replicateMessage(newMsg)
		if n.messageHook != nil {
			data, _ := json.Marshal(newMsg)
			n.messageHook(string(data))
		}
	}

	if answer != "" {
		answerMsg := Message{
			ID:       generateMsgID(answer),
			Text:     answer,
			Sender:   "🌐 Сеть",
			Time:     time.Now().UTC().Format("2006-01-02T15:04:05"),
			IsOwn:    false,
			Score:    0,
			Weight:   0.5,
			Priority: 0,
			Mode:     0,
		}
		if n.memory.Add(answerMsg) {
			n.replicateMessage(answerMsg)
		}
	}

	go n.saveState()
}

func migrateTime(t string) string {
	if strings.Contains(t, "T") {
		return t
	}
	if len(t) == 8 && t[2] == ':' {
		return time.Now().Format("2006-01-02") + "T" + t
	}
	return time.Now().Format("2006-01-02T15:04:05")
}

func (n *Node) loadBootstrapPeers() []string {
	if len(n.configBootstrap) > 0 {
		return n.configBootstrap
	}
	var peers []string
	if envPeers := os.Getenv("ISOTOPE_BOOTSTRAP_PEERS"); envPeers != "" {
		for _, p := range strings.Split(envPeers, ",") {
			if p = strings.TrimSpace(p); p != "" {
				peers = append(peers, p)
			}
		}
	}
	if data, err := os.ReadFile("bootstrap.txt"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				peers = append(peers, line)
			}
		}
	}
	return peers
}

func (n *Node) loadState() error {
	data, err := n.loadStateData()
	if err != nil {
		return err
	}
	var state struct {
		Messages     []Message   `json:"messages"`
		Layers       [][]float64 `json:"layers"`
		MsgCount     int         `json:"msgCount"`
		PreHash      string      `json:"preHash"`
		AntiHash     string      `json:"antiHash"`
		RoutingTable []string    `json:"routingTable"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	for i := range state.Messages {
		state.Messages[i].Time = migrateTime(state.Messages[i].Time)
		state.Messages[i].ExpiresAt = time.Time{}
	}
	for _, msg := range state.Messages {
		n.memory.Add(msg)
	}
	n.layers = state.Layers
	n.msgCount = state.MsgCount
	n.preHash = state.PreHash
	n.antiHash = state.AntiHash
	n.layersDirty = true
	if len(state.RoutingTable) > 0 {
		if n.dhtNode != nil {
			data, _ := json.Marshal(state.RoutingTable)
			n.dhtNode.LoadRoutingTable(data)
		}
	}
	return nil
}

// InitP2P — инициализирует P2P
func (n *Node) InitP2P() error {
	if n.stateFile == "" {
		nodeIDStr := os.Getenv("NODE_ID")
		if nodeIDStr == "" {
			nodeIDStr = "1"
		}
		nodeID, _ := strconv.Atoi(nodeIDStr)
		n.nodeID = nodeID
		n.stateFile = fmt.Sprintf("state/state_node%d.json", nodeID)
	}

	n.adaptive = NewAdaptiveParams()
	n.channels = NewChannelStore()

	if err := n.loadState(); err != nil {
		log.Println("[INIT] Состояние не найдено, начинаем с нуля")
	}

	var priv crypto.PrivKey
	keyBytes, err := n.loadPrivateKey()
	if err != nil {
		priv, _, _ = crypto.GenerateKeyPair(crypto.RSA, 2048)
		keyBytes, _ = crypto.MarshalPrivateKey(priv)
		n.savePrivateKey(keyBytes)
	} else {
		priv, _ = crypto.UnmarshalPrivateKey(keyBytes)
	}

	listenAddr := fmt.Sprintf("/ip4/%s/tcp/%d", n.configListenIP, n.configPort)
	listenWS := fmt.Sprintf("/ip4/%s/tcp/%d/ws", n.configListenIP, n.configPort+1)

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(listenAddr, listenWS),
		libp2p.Identity(priv),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
	}

	if len(n.configTransports) == 0 {
		opts = append(opts, libp2p.Transport(websocket.New))
	} else {
		for _, t := range n.configTransports {
			switch t {
			case "ws":
				opts = append(opts, libp2p.Transport(websocket.New))
			case "tcp":
				opts = append(opts, libp2p.Transport(tcp.NewTCPTransport))
			}
		}
	}

	host, err := libp2p.New(opts...)
	if err != nil {
		return err
	}
	n.host = host
	n.host.SetStreamHandler(protocolID, n.handleStream)
	n.host.SetStreamHandler(syncProtocolID, n.handleSyncStream)
	n.host.SetStreamHandler(pingProtocolID, n.handlePingStream)

	if n.configEnableRelayServer {
		if _, err := relay.New(host); err != nil {
			log.Printf("[RELAY] Failed: %v", err)
		}
	}

	if n.configEnableMDNS {
		mdnsService := mdns.NewMdnsService(n.host, "_isotope._tcp.local", n)
		mdnsService.Start()
	}

	log.Println("[INIT] Node started with ID:", host.ID())

	bootstrapPeers := n.loadBootstrapPeers()
	for _, addr := range bootstrapPeers {
		go func(addr string) {
			peerInfo, err := peer.AddrInfoFromString(addr)
			if err != nil {
				return
			}
			ctx := context.Background()
			n.host.Connect(ctx, *peerInfo)
			go func(peerID peer.ID) {
				time.Sleep(2 * time.Second)
				n.ExchangePeers(peerID.String())
			}(peerInfo.ID)
		}(addr)
	}

	dhtNode, err := NewDHT(host)
	if err == nil {
		n.dhtNode = dhtNode
		dhtNode.JoinDHT(bootstrapPeers)
	}

	if len(n.layers) == 0 {
		n.layers = append(n.layers, make([]float64, VectorDim))
		hashVec := hashToVector(n.ethHash)
		for i := range n.layers[0] {
			if i < len(hashVec) {
				n.layers[0][i] = hashVec[i] * 0.1
			}
		}
	}

	n.pingPeers()
	n.StartAdaptation()

	go func() {
		time.Sleep(5 * time.Second)
		n.requestRestore()
	}()

	return nil
}

// StartHTTP — запускает HTTP-сервер
func (n *Node) StartHTTP(port int) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/send", n.handleSend)
	http.HandleFunc("/messages", n.handleMessages)
	http.HandleFunc("/status", n.handleStatus)
	http.HandleFunc("/dht/find", n.handleDHTFindPeer)
	http.HandleFunc("/dht/provide", n.handleDHTProvide)
	http.HandleFunc("/dht/info", n.handleDHTInfo)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

// StartMobile — запускает узел для мобильного
func (n *Node) StartMobile(stateFile string) error {
	if stateFile == "" {
		return fmt.Errorf("stateFile is required")
	}
	n.stateFile = stateFile
	n.adaptive = NewAdaptiveParams()
	n.channels = NewChannelStore()
	return n.InitP2P()
}

// Stop — корректно завершает узел
func (n *Node) Stop() error {
	if n.dhtNode != nil {
		n.dhtNode.Close()
	}
	n.saveState()
	if n.host != nil {
		return n.host.Close()
	}
	return nil
}

// GetMessages — возвращает все сообщения
func (n *Node) GetMessages() []Message {
	return n.memory.GetAll()
}

// GetPeers — возвращает список активных пиров
func (n *Node) GetPeers() []string {
	if n.host == nil {
		return []string{}
	}
	peers := n.host.Network().Peers()
	result := make([]string, 0, len(peers))
	for _, p := range peers {
		result = append(result, p.String())
	}
	return result
}

// GetWeight — возвращает вес узла
func (n *Node) GetWeight() float64 {
	msgs := n.memory.GetActiveMessages(0.5)
	if len(msgs) == 0 {
		return 0.5
	}
	total := 0.0
	for _, m := range msgs {
		total += m.Weight
	}
	return total / float64(len(msgs))
}

// GetStatus — возвращает JSON-статус
func (n *Node) GetStatus() string {
	if n.host == nil {
		return `{"id":"","peers":0,"memory":0,"layers":0}`
	}
	return fmt.Sprintf(`{"id":"%s","peers":%d,"memory":%d,"layers":%d}`,
		n.host.ID().String(),
		len(n.host.Network().Peers()),
		n.memory.Count(),
		len(n.layers),
	)
}

// GetMultiaddrs — возвращает все адреса узла
func (n *Node) GetMultiaddrs() []string {
	if n.host == nil {
		return []string{}
	}
	var result []string
	for _, addr := range n.host.Addrs() {
		result = append(result, addr.String()+"/p2p/"+n.host.ID().String())
	}
	return result
}

// ConnectToPeer — подключение к пиру
func (n *Node) ConnectToPeer(multiaddr string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	peerInfo, err := peer.AddrInfoFromString(multiaddr)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := n.host.Connect(ctx, *peerInfo); err != nil {
		return err
	}
	go func() {
		time.Sleep(2 * time.Second)
		n.ExchangePeers(peerInfo.ID.String())
	}()
	return nil
}

// SendMessage — отправляет сообщение всем пирам
func (n *Node) SendMessage(text string, ttl int) (string, error) {
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(time.Duration(ttl) * time.Second)
	}
	id := generateMsgID(text)
	n.processMessageWithID(text, n.host.ID().String(), true, expiresAt, id)

	go func() {
		for _, p := range n.host.Network().Peers() {
			randomDelay(5, 25)
			obfuscated := n.obfuscate(text)
			ctx := context.Background()
			s, err := n.host.NewStream(ctx, p, protocolID)
			if err != nil {
				continue
			}
			fmt.Fprintf(s, "%s\n", obfuscated)
			s.Close()
		}
	}()
	return id, nil
}

// SendToPeer — отправляет сообщение конкретному пиру
func (n *Node) SendToPeer(peerID string, text string, ttl int) (string, error) {
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(time.Duration(ttl) * time.Second)
	}
	id := generateMsgID(text)
	n.processMessageWithID(text, n.host.ID().String(), true, expiresAt, id)

	go func() {
		for _, p := range n.host.Network().Peers() {
			randomDelay(5, 25)
			obfuscated := n.obfuscate(text)
			ctx := context.Background()
			s, err := n.host.NewStream(ctx, p, protocolID)
			if err != nil {
				continue
			}
			fmt.Fprintf(s, "RELAY:%s:%s\n", peerID, obfuscated)
			s.Close()
		}
	}()
	return id, nil
}

// GetKnownPeers — возвращает список известных multiaddr
func (n *Node) GetKnownPeers() []string {
	if n.host == nil {
		return []string{}
	}
	var addrs []string
	peers := n.host.Peerstore().Peers()
	for _, p := range peers {
		peerInfo := n.host.Peerstore().PeerInfo(p)
		for _, addr := range peerInfo.Addrs {
			addrStr := addr.String() + "/p2p/" + p.String()
			if strings.Contains(addrStr, "127.0.0.1") {
				continue
			}
			addrs = append(addrs, addrStr)
		}
	}
	return addrs
}

// ExchangePeers — обменивается списками узлов
func (n *Node) ExchangePeers(peerID string) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}
	pid, err := peer.Decode(peerID)
	if err != nil {
		return err
	}
	allKnownAddrs := n.GetKnownPeers()
	myAddrs := n.GetMultiaddrs()
	for _, addr := range myAddrs {
		if !strings.Contains(addr, "127.0.0.1") {
			found := false
			for _, known := range allKnownAddrs {
				if known == addr {
					found = true
					break
				}
			}
			if !found {
				allKnownAddrs = append(allKnownAddrs, addr)
			}
		}
	}
	ctx := context.Background()
	s, err := n.host.NewStream(ctx, pid, protocolID)
	if err != nil {
		return err
	}
	defer s.Close()
	data, _ := json.Marshal(allKnownAddrs)
	fmt.Fprintf(s, "PEERS:%s\n", string(data))
	buf := make([]byte, 256*1024)
	s.SetReadDeadline(time.Now().Add(15 * time.Second))
	nr, _ := s.Read(buf)
	response := strings.TrimSpace(string(buf[:nr]))
	if strings.HasPrefix(response, "PEERS:") {
		payload := strings.TrimPrefix(response, "PEERS:")
		var peerAddrs []string
		if err := json.Unmarshal([]byte(payload), &peerAddrs); err == nil {
			for _, addr := range peerAddrs {
				peerInfo, err := peer.AddrInfoFromString(addr)
				if err == nil {
					n.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, time.Hour*24)
				}
			}
		}
	}
	return nil
}

// GetHost — возвращает libp2p host
func (n *Node) GetHost() host.Host {
	return n.host
}

// SetDHT — устанавливает DHT
func (n *Node) SetDHT(d *DHTNode) {
	n.dhtNode = d
}

// GetDHT — возвращает DHT
func (n *Node) GetDHT() *DHTNode {
	return n.dhtNode
}