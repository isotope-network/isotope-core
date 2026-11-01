package core

// D:\isotope\node\node_p2p.go
// node_p2p.go — сеть: handleStream, ping, reconnect, announce, cleanup,
// sendToRecipient, fallback, replicateMessage, processMessageInternal,
// Sphinx-получатель (handleSphinxReplicaMsg).
// Вынесено из node.go.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	mathrand "math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

// nodeNotifiee — Notifiee libp2p: реакция на connect/disconnect.
type nodeNotifiee struct {
	node *Node
}

func (nn *nodeNotifiee) Connected(net network.Network, conn network.Conn) {
	remote := conn.RemotePeer().String()
	log.Printf("[NOTIFY] connected to %s", remote)
	go func() {
		time.Sleep(1 * time.Second)

		if nn.node.hasPending() {
			log.Printf("[NOTIFY] flushing pending after connect to %s", remote)
			nn.node.flushPending()
		}

		if nn.node.isRelayAddr(remote) {
			nn.node.refreshRelayAndAnnounce()
		}
	}()
}

func (nn *nodeNotifiee) Disconnected(net network.Network, conn network.Conn) {
	remote := conn.RemotePeer().String()
	log.Printf("[NOTIFY] disconnected from %s", remote)
	nn.node.mu.Lock()
	if nn.node.deadPeers != nil {
		delete(nn.node.deadPeers, remote)
	}
	nn.node.mu.Unlock()
}

func (nn *nodeNotifiee) Listen(net network.Network, addr ma.Multiaddr)      {}
func (nn *nodeNotifiee) ListenClose(net network.Network, addr ma.Multiaddr) {}

// handleStream — обработчик входящего stream.
func (n *Node) handleStream(stream network.Stream) {
	defer stream.Close()

	stream.SetReadDeadline(time.Now().Add(15 * time.Second))

	var full []byte
	buf := make([]byte, 64*1024)
	for {
		nr, err := stream.Read(buf)
		if nr > 0 {
			full = append(full, buf[:nr]...)
			if bytes.Contains(buf[:nr], []byte("\n")) {
				break
			}
		}
		if err != nil {
			break
		}
		if len(full) > 4*1024*1024 {
			break
		}
	}
	msg := strings.TrimSpace(string(full))

	if strings.HasPrefix(msg, ANNOUNCE_PREFIX) {
		remoteID := stream.Conn().RemotePeer().String()
		addrs := parseMultiaddrsResponse(msg, ANNOUNCE_PREFIX)
		if len(addrs) > 0 {
			n.announcePeer(remoteID, addrs)
		}
		return
	}

	if strings.HasPrefix(msg, FIND_PREFIX) {
		targetID := strings.TrimPrefix(msg, FIND_PREFIX)
		targetID = strings.TrimSpace(targetID)

		addrs, ok := n.lookupPeer(targetID)

		if !ok && n.host != nil {
			targetPID, err := peer.Decode(targetID)
			if err == nil {
				connected := false
				for _, p := range n.host.Network().Peers() {
					if p == targetPID {
						connected = true
						break
					}
				}
				if connected {
					relayAddr := n.buildRelayAddrFor(targetID)
					if relayAddr != "" {
						addrs = []string{relayAddr}
						ok = true
						log.Printf("[FIND] fallback to relay for connected peer %s", targetID)
					}
				}
			}
		}

		if ok && len(addrs) > 0 {
			var sb strings.Builder
			sb.WriteString(FOUND_PREFIX)
			sb.WriteString("\n")
			for _, a := range addrs {
				sb.WriteString(a)
				sb.WriteString("\n")
			}
			sb.WriteString(END_PREFIX)
			sb.WriteString("\n")
			stream.Write([]byte(sb.String()))
		} else {
			stream.Write([]byte(NOT_FOUND_PREFIX + "\n"))
		}
		return
	}
	log.Printf("[SPHINX] incoming packet: len=%d prefix=%q", len(msg), truncate(msg, 20))
	if strings.HasPrefix(msg, SPHINX_PREFIX) {
		payload := strings.TrimPrefix(msg, SPHINX_PREFIX)
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload))
		if err != nil {
			log.Printf("[SPHINX] base64 decode failed: %v", err)
			return
		}
		myID := n.host.ID().String()
		nextPeer, mixing, dummy, remainder, err := n.processSphinxPacket(raw)
		if err == nil {
			log.Printf("[SPHINX] relay: next=%s size=%d mixing=%v dummy=%v", nextPeer, len(remainder), mixing, dummy)
			if mixing {
				sleepExponential(5*time.Second, 30*time.Second)
			}
			peerAddrs := n.host.Peerstore().Addrs(nextPeer)
			if len(peerAddrs) == 0 {
				if addrs, ferr := n.FindPeerByID(nextPeer.String()); ferr == nil {
					var direct []ma.Multiaddr
					for _, a := range addrs {
						maAddr, perr := ma.NewMultiaddr(a)
						if perr != nil {
							continue
						}
						if strings.Contains(maAddr.String(), "/p2p-circuit") {
							continue
						}
						direct = append(direct, maAddr)
					}
					if len(direct) == 0 {
						log.Printf("[SPHINX] relay: no direct addrs for next hop %s, drop", nextPeer)
						return
					}
					n.host.Peerstore().AddAddrs(nextPeer, direct, peerstore.TempAddrTTL)
					log.Printf("[SPHINX] relay: next=%s direct_addrs=%d", nextPeer, len(direct))
				} else {
					log.Printf("[SPHINX] relay: FindPeerByID(%s) failed: %v, drop", nextPeer, ferr)
					return
				}
			} else {
				var direct []ma.Multiaddr
				for _, a := range peerAddrs {
					if strings.Contains(a.String(), "/p2p-circuit") {
						continue
					}
					direct = append(direct, a)
				}
				if len(direct) == 0 {
					log.Printf("[SPHINX] relay: no direct addrs in peerstore for next hop %s, drop", nextPeer)
					return
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), SphinxTimeout)
			s, serr := n.host.NewStream(ctx, nextPeer, protocolID)
			cancel()
			if serr != nil {
				log.Printf("[SPHINX] relay NewStream to %s failed: %v", nextPeer, serr)
				return
			}
			defer s.Close()
			fmt.Fprintf(s, "%s%s\n", SPHINX_PREFIX, base64.StdEncoding.EncodeToString(remainder))
			return
		}
		log.Printf("[SPHINX] processSphinxPacket failed: %v", err)
		plain, err := n.extractSphinxPayload(raw)
		if err != nil {
			log.Printf("[SPHINX] neither relay nor recipient: %v", err)
			return
		}
		var replicaMsg Message
		if err := json.Unmarshal(plain, &replicaMsg); err != nil {
			log.Printf("[SPHINX] recipient unmarshal failed: %v", err)
			return
		}
		if replicaMsg.Recipient != myID {
			log.Printf("[SPHINX] recipient mismatch: %s != %s", replicaMsg.Recipient, myID)
			return
		}
		n.handleSphinxReplicaMsg(replicaMsg)
		return
	}

	if strings.HasPrefix(msg, REPLICA_PREFIX) {
		payload := strings.TrimPrefix(msg, REPLICA_PREFIX)

		var replicaMsg Message
		if err := json.Unmarshal([]byte(payload), &replicaMsg); err == nil {
			if n.isServiceType(replicaMsg.Type) {
				myID := n.host.ID().String()
				if replicaMsg.Recipient == myID {
					n.handleServiceMessage(replicaMsg)
				} else if n.isRelay {
					go n.replicateMessage(replicaMsg)
				}
				return
			}
			if replicaMsg.Version < 2 {
				if plaintext, ok := n.deobfuscate(replicaMsg.Text); ok {
					replicaMsg.Text = plaintext
				}
			}
			replicaMsg.ReplicatedAt = time.Now()
			replicaMsg.IsOwn = false

			if replicaMsg.Version == MESSAGE_VERSION_E2E {
				myID := n.host.ID().String()
				if replicaMsg.Recipient == myID {
					if replicaMsg.MediaType == "batch" {
						plaintext, err := n.decryptFromSender(replicaMsg.Sender, replicaMsg.Text)
						if err != nil {
							log.Printf("[BATCH] decrypt failed from %s: %v", replicaMsg.Sender, err)
							return
						}
						var batch struct {
							Texts []struct {
								ID   string `json:"id"`
								Text string `json:"text"`
							} `json:"texts"`
						}
						if err := json.Unmarshal([]byte(plaintext), &batch); err != nil {
							log.Printf("[BATCH] parse failed from %s: %v", replicaMsg.Sender, err)
							return
						}
						senderName := n.nameForPeer(replicaMsg.Sender)
						for _, t := range batch.Texts {
							msg := Message{
								ID:           t.ID,
								Text:         t.Text,
								Sender:       replicaMsg.Sender,
								SenderName:   senderName,
								Recipient:    replicaMsg.Recipient,
								Type:         TypeMessage,
								MediaType:    "text",
								Version:      MESSAGE_VERSION_E2E,
								Time:         time.Now().UTC().Format("2006-01-02T15:04:05"),
								IsOwn:        false,
								Weight:       0.5,
								ReplicatedAt: time.Now(),
							}
							if n.memory.Add(msg) {
								if n.messageHook != nil {
									data, _ := json.Marshal(msg)
									n.messageHook(string(data))
								}
							}
							n.queueDelivered(replicaMsg.Sender, t.ID)
						}
						return
					}
					if isChunkedMediaType(replicaMsg.MediaType) && replicaMsg.ChunkTotal > 0 {
						raw, err := n.decryptBytesFromSender(replicaMsg.Sender, replicaMsg.Text)
						if err != nil {
							log.Printf("[REPLICA] decrypt bytes failed from %s: %v", replicaMsg.Sender, err)
							return
						}
						replicaMsg.Text = base64.StdEncoding.EncodeToString(raw)
					} else {
						plaintext, err := n.decryptFromSender(replicaMsg.Sender, replicaMsg.Text)
						if err != nil {
							log.Printf("[REPLICA] decrypt failed from %s: %v", replicaMsg.Sender, err)
							return
						}
						replicaMsg.Text = plaintext
					}
					replicaMsg.SenderName = n.nameForPeer(replicaMsg.Sender)
					added := n.memory.Add(replicaMsg)
					if added {
						if isChunkedMediaType(replicaMsg.MediaType) && replicaMsg.ChunkTotal > 0 && replicaMsg.MediaID != "" {
							n.acceptedCountMu.Lock()
							n.acceptedCount[replicaMsg.MediaID]++
							accepted := n.acceptedCount[replicaMsg.MediaID]
							n.acceptedCountMu.Unlock()
							n.pushFileStatus(replicaMsg.MediaID, accepted, replicaMsg.ChunkTotal)
						}
						if n.messageHook != nil {
							hookMsg := replicaMsg
							if isChunkedMediaType(hookMsg.MediaType) && hookMsg.ChunkTotal > 0 {
								hookMsg.Text = ""
								hookMsg.PlainText = ""
								n.pendingHookAcksMu.Lock()
								w := n.pendingHookAcks[hookMsg.MediaID]
								if w == nil {
									w = &hookAckWindow{Indices: make(map[int]time.Time)}
									n.pendingHookAcks[hookMsg.MediaID] = w
								}
								w.Indices[hookMsg.ChunkIndex] = time.Now()
								n.pendingHookAcksMu.Unlock()
							}
							data, _ := json.Marshal(hookMsg)
							n.messageHook(string(data))
						}
					}
					n.queueDelivered(replicaMsg.Sender, replicaMsg.ID)
					return
				}
				if n.isRelay {
					go n.replicateMessage(replicaMsg)
				}
				return
			}

			if replicaMsg.Recipient != "" && n.host != nil {
				myID := n.host.ID().String()
				if replicaMsg.Recipient != myID {
					if n.isRelay {
						log.Printf("[REPLICA] relay forward to %s", replicaMsg.Recipient)
						go n.replicateMessage(replicaMsg)
					} else {
						log.Printf("[REPLICA] not for us (recipient=%s), dropping", replicaMsg.Recipient)
					}
					return
				}
				replicaMsg.SenderName = n.nameForPeer(replicaMsg.Sender)
				if n.memory.Add(replicaMsg) {
					log.Printf("[REPLICA] received addressed message for us: %s", replicaMsg.ID)
					if n.messageHook != nil {
						data, _ := json.Marshal(replicaMsg)
						n.messageHook(string(data))
					}
				}
				return
			}

			if n.memory.Add(replicaMsg) {
				go n.replicateMessage(replicaMsg)
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

// handleSphinxReplicaMsg — обработка расшифрованного Sphinx-пакета получателем.
func (n *Node) handleSphinxReplicaMsg(replicaMsg Message) {
	if replicaMsg.MediaType == "dummy" {
		log.Printf("[DUMMY] dropped from %s", replicaMsg.Sender)
		return
	}
	if replicaMsg.Version == MESSAGE_VERSION_E2E {
		if replicaMsg.MediaType == "batch" {
			plaintext, err := n.decryptFromSender(replicaMsg.Sender, replicaMsg.Text)
			if err != nil {
				log.Printf("[SPHINX] batch decrypt failed from %s: %v", replicaMsg.Sender, err)
				return
			}
			var batch struct {
				Texts []struct {
					ID   string `json:"id"`
					Text string `json:"text"`
				} `json:"texts"`
			}
			if err := json.Unmarshal([]byte(plaintext), &batch); err != nil {
				log.Printf("[SPHINX] batch parse failed from %s: %v", replicaMsg.Sender, err)
				return
			}
			senderName := n.nameForPeer(replicaMsg.Sender)
			for _, t := range batch.Texts {
				msg := Message{
					ID:           t.ID,
					Text:         t.Text,
					Sender:       replicaMsg.Sender,
					SenderName:   senderName,
					Recipient:    replicaMsg.Recipient,
					Type:         TypeMessage,
					MediaType:    "text",
					Version:      MESSAGE_VERSION_E2E,
					Time:         time.Now().UTC().Format("2006-01-02T15:04:05"),
					IsOwn:        false,
					Weight:       0.5,
					ReplicatedAt: time.Now(),
				}
				if n.memory.Add(msg) {
					if n.messageHook != nil {
						data, _ := json.Marshal(msg)
						n.messageHook(string(data))
					}
				}
				n.queueDelivered(replicaMsg.Sender, t.ID)
			}
			return
		}
		if isChunkedMediaType(replicaMsg.MediaType) && replicaMsg.ChunkTotal > 0 {
			raw, err := n.decryptBytesFromSender(replicaMsg.Sender, replicaMsg.Text)
			if err != nil {
				log.Printf("[SPHINX] decrypt bytes failed from %s: %v", replicaMsg.Sender, err)
				return
			}
			replicaMsg.Text = base64.StdEncoding.EncodeToString(raw)
		} else {
			plaintext, err := n.decryptFromSender(replicaMsg.Sender, replicaMsg.Text)
			if err != nil {
				log.Printf("[SPHINX] decrypt failed from %s: %v", replicaMsg.Sender, err)
				return
			}
			replicaMsg.Text = plaintext
		}
	}
	replicaMsg.SenderName = n.nameForPeer(replicaMsg.Sender)
	replicaMsg.ReplicatedAt = time.Now()
	replicaMsg.IsOwn = false
	added := n.memory.Add(replicaMsg)
	if added {
		if isChunkedMediaType(replicaMsg.MediaType) && replicaMsg.ChunkTotal > 0 && replicaMsg.MediaID != "" {
			n.acceptedCountMu.Lock()
			n.acceptedCount[replicaMsg.MediaID]++
			accepted := n.acceptedCount[replicaMsg.MediaID]
			n.acceptedCountMu.Unlock()
			n.pushFileStatus(replicaMsg.MediaID, accepted, replicaMsg.ChunkTotal)
		}
		if n.messageHook != nil {
			hookMsg := replicaMsg
			if isChunkedMediaType(hookMsg.MediaType) && hookMsg.ChunkTotal > 0 {
				hookMsg.Text = ""
				hookMsg.PlainText = ""
			}
			data, _ := json.Marshal(hookMsg)
			n.messageHook(string(data))
		}
	}
	n.queueDelivered(replicaMsg.Sender, replicaMsg.ID)
}

// handlePingStream — обработчик PING.
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
					if n.markPeerAlive(peerID.String()) {
						go func() {
							if n.hasPending() {
								log.Printf("[PING] peer %s alive again, flushing pending", peerID)
								n.flushPending()
							}
						}()
					}
				}(p)
			}
		}
	}()
}

func (n *Node) reconnectLoop() {
	go func() {
		backoff := time.Second
		const maxBackoff = 30 * time.Second

		for {
			time.Sleep(backoff)
			if n.host == nil {
				continue
			}
			peers := n.host.Network().Peers()
			if len(peers) > 0 {
				if backoff != time.Second {
					backoff = time.Second
				}
				continue
			}
			log.Printf("[RECONNECT] Нет пиров, переподключаюсь к bootstrap (backoff=%s)", backoff)

			bootstrapPeers := n.loadBootstrapPeers()
			connected := false
			for _, addr := range bootstrapPeers {
				peerInfo, err := peer.AddrInfoFromString(addr)
				if err != nil {
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				err = n.host.Connect(ctx, *peerInfo)
				cancel()
				if err != nil {
					log.Printf("[RECONNECT] Ошибка подключения к %s: %v", addr, err)
					continue
				}
				log.Printf("[RECONNECT] Подключён к bootstrap: %s", addr)
				go func(peerID peer.ID) {
					time.Sleep(2 * time.Second)
					n.ExchangePeers(peerID.String())
					n.flushPending()
				}(peerInfo.ID)
				connected = true
				break
			}

			if connected {
				backoff = time.Second
			} else {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
		}
	}()
}
func (n *Node) announceLoop() {
	go func() {
		for {
			time.Sleep(4 * time.Minute)
			if len(n.announceMultiaddrs) > 0 {
				n.SendAnnounce(n.announceMultiaddrs)
			}
			n.flushPending()
		}
	}()
}

func (n *Node) cleanupLoop() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		diagTicker := time.NewTicker(10 * time.Second)
		defer diagTicker.Stop()
		for {
			select {
			case <-ticker.C:
				n.cleanupAnnounced()
				if removed := n.memory.DeleteExpired(); removed > 0 {
					log.Printf("[TTL] purged %d expired messages", removed)
					n.scheduleSaveState()
				}
			case <-diagTicker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				log.Printf("[DIAG] goroutines=%d, heap=%d KB, sys=%d KB",
					runtime.NumGoroutine(), m.HeapAlloc/1024, m.Sys/1024)
			}
		}
	}()
}

func (n *Node) cleanupPendingAcksLoop() {
	go func() {
		const hookAckTimeout = 120 * time.Second
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			n.pendingHookAcksMu.Lock()
			for mediaID, w := range n.pendingHookAcks {
				for idx, addedAt := range w.Indices {
					if now.Sub(addedAt) > hookAckTimeout {
						log.Printf("[ACK_TIMEOUT] mediaID=%s idx=%d", mediaID, idx)
						delete(w.Indices, idx)
					}
				}
				if len(w.Indices) == 0 {
					delete(n.pendingHookAcks, mediaID)
				}
			}
			n.pendingHookAcksMu.Unlock()
		}
	}()
}

func (n *Node) cleanupSentLoop() {
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			n.cleanupSentOnce()
		}
	}()
}

func (n *Node) cleanupSentOnce() {
	dir := n.sentDirPath()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	type fileStatus struct {
		total     int
		delivered int
		createdAt time.Time
	}
	statuses := make(map[string]*fileStatus)
	all := n.memory.GetAll()
	for _, m := range all {
		if !isChunkedMediaType(m.MediaType) || m.ChunkTotal <= 0 || m.MediaID == "" {
			continue
		}
		st, ok := statuses[m.MediaID]
		if !ok {
			st = &fileStatus{total: m.ChunkTotal}
			statuses[m.MediaID] = st
		}
		if m.Status >= StatusDelivered {
			st.delivered++
		}
	}

	cutoff := time.Now().Add(-24 * time.Hour)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".bin") {
			continue
		}
		mediaID := strings.TrimSuffix(e.Name(), ".bin")
		fullPath := n.sentPath(mediaID)

		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(fullPath); err == nil {
				log.Printf("[SENT] removed %s (older than 24h)", mediaID)
			}
			continue
		}

		if st, ok := statuses[mediaID]; ok && st.total > 0 && st.delivered >= st.total {
			if err := os.Remove(fullPath); err == nil {
				log.Printf("[SENT] removed %s (all %d chunks delivered)", mediaID, st.total)
			}
		}
	}
}

func (n *Node) markPeerAlive(peerID string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.lastPing == nil {
		n.lastPing = make(map[string]time.Time)
	}
	if n.deadPeers == nil {
		n.deadPeers = make(map[string]bool)
	}
	wasDead := n.deadPeers[peerID]
	n.lastPing[peerID] = time.Now()
	if wasDead {
		delete(n.deadPeers, peerID)
	}
	return wasDead
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

// isPeerDead — true, только если peer сейчас в Network().Peers()
// и помечен мёртвым. Если peer вне Peers() — не «мёртв», а «нет
// прямого пути». Для таких — сразу fallback (в sendToRecipient).
func (n *Node) isPeerDead(peerID string) bool {
	if n.host == nil {
		return false
	}
	inPeers := false
	for _, p := range n.host.Network().Peers() {
		if p.String() == peerID {
			inPeers = true
			break
		}
	}
	if !inPeers {
		return false
	}
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
	newMsg.SenderName = n.nameForPeer(newMsg.Sender)
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
	isFileChunk := isChunkedMediaType(msg.MediaType) && msg.ChunkTotal > 0
	isTextForSphinxBatch := (msg.MediaType == "" || msg.MediaType == "text") && msg.Recipient != ""
	if isTextForSphinxBatch && n.useOnion && !n.isRelay {
		n.addToBatch(msg.Recipient, msg.ID, msg.PlainText)
		return
	}
	msg.ReplicatedFrom = msg.Sender
	msg.IsOwn = false
	msg.PlainText = ""
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	if msg.Recipient != "" {
		if n.useOnion && !n.isRelay && !isFileChunk {
			if n.sendViaSphinx(msg) {
				return
			}
			log.Printf("[SPHINX] failed, fallback to direct (recipient=%s)", msg.Recipient)
		}
		if isFileChunk && !n.isRelay {
			n.chunkQueue <- msg
		} else {
			if isFileChunk && n.isRelay {
				n.sendSem <- struct{}{}
				defer func() { <-n.sendSem }()
			}
			n.sendToRecipient(msg, data, isFileChunk)
		}
		if !n.isRelay && !n.isServiceType(msg.Type) && !isFileChunk {
			n.enqueuePending(msg)
		}
		return
	}

	peers := n.host.Network().Peers()
	log.Printf("[REPLICA] broadcast msg id=%s sender=%s: %d peers", msg.ID, msg.Sender, len(peers))

	var alive []peer.ID
	for _, p := range peers {
		if p.String() == msg.Sender {
			continue
		}
		alive = append(alive, p)
	}
	log.Printf("[REPLICA] alive=%d", len(alive))

	if len(alive) == 0 {
		if n.isRelay {
			return
		}
		log.Printf("[REPLICA] SKIP: no alive peers — enqueue")
		if !isFileChunk {
			n.enqueuePending(msg)
		}
		return
	}
	replicaCount := 2
	if len(alive) < replicaCount {
		replicaCount = len(alive)
	}
	for i := 0; i < replicaCount; i++ {
		go n.sendReplicaToPeer(alive[i], data, isFileChunk)
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
	n.processMessageInternal(msg, senderID, isOwn, expiresAt, "", "", 0, "", TypeMessage, false, 0, "")
}

func (n *Node) processMessageWithID(msg string, senderID string, isOwn bool, expiresAt time.Time, id string) {
	n.processMessageInternal(msg, senderID, isOwn, expiresAt, id, "", 0, "", TypeMessage, false, 0, "")
}

func (n *Node) processMessageWithRecipient(msg string, senderID string, isOwn bool, expiresAt time.Time, id string, recipient string, version int) {
	n.processMessageInternal(msg, senderID, isOwn, expiresAt, id, recipient, version, "", TypeMessage, false, 0, "")
}

func (n *Node) processMessageWithModeAndTTL(msg string, senderID string, isOwn bool, mode int, expiresAt time.Time) {
	if mode == 0 || n.host == nil {
		n.processMessageInternal(msg, senderID, isOwn, expiresAt, "", "", 0, "", TypeMessage, false, 0, "")
		return
	}
	relayCount := 4
	if mode == 2 {
		relayCount = 5
		time.Sleep(10 * time.Second)
	}
	relays := n.selectRelays(relayCount)
	if len(relays) < relayCount {
		n.processMessageInternal(msg, senderID, isOwn, expiresAt, "", "", 0, "", TypeMessage, false, 0, "")
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
	n.processMessageInternal(msg, senderID, isOwn, time.Time{}, "", "", 0, "", TypeMessage, false, 0, "")
}

func (n *Node) processMessageInternal(msg string, senderID string, isOwn bool, expiresAt time.Time, providedID string, recipient string, version int, plainText string, msgType MessageType, viaBootstrap bool, ttlPeriodSeconds int, ttlMode string) {
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

	readEnabled := n.myReadEnabled
	mixingEnabled := false
	if n.settingsStore != nil {
		mixingEnabled = n.settingsStore.GetMixingEnabled() && n.useOnion
	}
	newMsg := Message{
		ID:               id,
		Text:             msg,
		PlainText:        plainText,
		Sender:           senderID,
		Recipient:        recipient,
		Version:          version,
		Type:             msgType,
		Time:             time.Now().UTC().Format("2006-01-02T15:04:05"),
		IsOwn:            isOwn,
		Score:            0,
		Weight:           initialWeight,
		Priority:         priority,
		Mode:             0,
		ExpiresAt:        expiresAt,
		ReadEnabled:      &readEnabled,
		TtlPeriodSeconds: ttlPeriodSeconds,
		TtlMode:          ttlMode,
		MixingEnabled:    mixingEnabled,
	}
	if isOwn && isChunkedMediaType(newMsg.MediaType) && newMsg.ChunkTotal > 0 {
		newMsg.Text = ""
		newMsg.PlainText = ""
	}
	if n.memory.Add(newMsg) {
		if isOwn && version == MESSAGE_VERSION_E2E && newMsg.Type == TypeMessage {
			n.setMessageStatus(id, StatusSent)
		}
		if viaBootstrap {
			if err := n.sendServiceViaBootstrap(newMsg); err != nil {
				log.Printf("[SERVICE] viaBootstrap failed for %s: %v", newMsg.ID, err)
			}
		} else {
			n.replicateMessage(newMsg)
		}
		if n.messageHook != nil {
			data, _ := json.Marshal(newMsg)
			n.messageHook(string(data))
		}
	}

	if answer != "" && recipient == "" {
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
	var raw []string
	if len(n.configBootstrap) > 0 {
		raw = n.configBootstrap
	} else {
		if envPeers := os.Getenv("ISOTOPE_BOOTSTRAP_PEERS"); envPeers != "" {
			for _, p := range strings.Split(envPeers, ",") {
				if p = strings.TrimSpace(p); p != "" {
					raw = append(raw, p)
				}
			}
		}
		if data, err := os.ReadFile("bootstrap.txt"); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					raw = append(raw, line)
				}
			}
		}
	}

	expanded := make([]string, 0, len(raw)*2)
	for _, addr := range raw {
		expanded = append(expanded, expandTransportAlternatives(addr)...)
	}
	return expanded
}

func expandTransportAlternatives(addr string) []string {
	result := []string{addr}

	const tcpMark = "/tcp/"
	i := strings.Index(addr, tcpMark)
	if i < 0 {
		return result
	}
	portStart := i + len(tcpMark)
	portEnd := portStart
	for portEnd < len(addr) && addr[portEnd] >= '0' && addr[portEnd] <= '9' {
		portEnd++
	}
	port, err := strconv.Atoi(addr[portStart:portEnd])
	if err != nil {
		return result
	}

	if strings.Contains(addr, "/ws") {
		altPort := strconv.Itoa(port - 1)
		prefix := addr[:portStart]
		suffix := addr[portEnd:]
		suffix = strings.Replace(suffix, "/ws", "", 1)
		result = append(result, prefix+altPort+suffix)
	} else {
		altPort := strconv.Itoa(port + 1)
		prefix := addr[:portStart]
		suffix := addr[portEnd:]
		result = append(result, prefix+altPort+"/ws"+suffix)
	}

	return result
}
