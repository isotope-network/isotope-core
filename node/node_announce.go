package core

// D:\isotope\node\node_announce.go
// node_announce.go — ANNOUNCE, FIND, relay-reservation, GetMultiaddrs,
// ConnectToPeer, ConnectToPeerWithFallback, ExchangePeers, GetKnownPeers.
// Вынесено из node.go.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	client "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
)

func (n *Node) announcePeer(peerID string, multiaddrs []string) {
	if len(multiaddrs) == 0 {
		return
	}
	n.announcedMu.Lock()
	defer n.announcedMu.Unlock()
	if n.announcedPeers == nil {
		n.announcedPeers = make(map[string]announcedPeer)
	}
	n.announcedPeers[peerID] = announcedPeer{
		Multiaddrs: multiaddrs,
		LastSeen:   time.Now(),
	}
	log.Printf("[ANNOUNCE] %s → %d addrs (%v)", peerID, len(multiaddrs), multiaddrs)
}

func (n *Node) lookupPeer(peerID string) ([]string, bool) {
	n.announcedMu.Lock()
	defer n.announcedMu.Unlock()
	p, ok := n.announcedPeers[peerID]
	if !ok {
		return nil, false
	}
	if time.Since(p.LastSeen) > ANNOUNCE_TTL {
		delete(n.announcedPeers, peerID)
		return nil, false
	}
	return p.Multiaddrs, true
}

func (n *Node) cleanupAnnounced() {
	connected := make(map[string]bool)
	if n.host != nil {
		for _, p := range n.host.Network().Peers() {
			connected[p.String()] = true
		}
	}

	n.announcedMu.Lock()
	defer n.announcedMu.Unlock()
	now := time.Now()
	for id, p := range n.announcedPeers {
		if !connected[id] {
			delete(n.announcedPeers, id)
			log.Printf("[ANNOUNCE] peer %s not connected — removed", id)
			continue
		}
		if now.Sub(p.LastSeen) > ANNOUNCE_TTL {
			delete(n.announcedPeers, id)
			log.Printf("[ANNOUNCE] TTL expired: %s", id)
		}
	}
}
func (n *Node) SendAnnounce(multiaddrs []string) {
	if n.host == nil || len(multiaddrs) == 0 {
		return
	}

	n.announceMultiaddrs = multiaddrs

	relayAddrs := n.GetRelayAddrs()
	allAddrs := make([]string, 0, len(multiaddrs)+len(relayAddrs))
	allAddrs = append(allAddrs, multiaddrs...)
	allAddrs = append(allAddrs, relayAddrs...)

	var sb strings.Builder
	sb.WriteString(ANNOUNCE_PREFIX)
	sb.WriteString("\n")
	for _, a := range allAddrs {
		sb.WriteString(a)
		sb.WriteString("\n")
	}
	sb.WriteString(END_PREFIX)
	sb.WriteString("\n")
	payload := sb.String()

	bootstrapPeers := n.loadBootstrapPeers()
	for _, addr := range bootstrapPeers {
		pi, err := peer.AddrInfoFromString(addr)
		if err != nil {
			continue
		}
		go func(pid peer.ID) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s, err := n.host.NewStream(ctx, pid, protocolID)
			if err != nil {
				log.Printf("[ANNOUNCE] send failed to %s: %v", pid, err)
				return
			}
			defer s.Close()
			fmt.Fprintf(s, "%s", payload)
			log.Printf("[ANNOUNCE] sent to %s: %d addrs", pid, len(allAddrs))
		}(pi.ID)
	}

	for _, p := range n.host.Network().Peers() {
		isBootstrap := false
		for _, addr := range bootstrapPeers {
			if pi, err := peer.AddrInfoFromString(addr); err == nil && pi.ID == p {
				isBootstrap = true
				break
			}
		}
		if isBootstrap {
			continue
		}
		go func(pid peer.ID) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s, err := n.host.NewStream(ctx, pid, protocolID)
			if err != nil {
				return
			}
			defer s.Close()
			fmt.Fprintf(s, "%s", payload)
		}(p)
	}
}

func (n *Node) FindPeerByID(targetID string) ([]string, error) {
	if n.host == nil {
		return nil, fmt.Errorf("node not started")
	}

	if addrs, ok := n.lookupPeer(targetID); ok {
		log.Printf("[FIND] local hit: %s → %d addrs: %v", targetID, len(addrs), addrs)
		return addrs, nil
	}

	bootstrapPeers := n.loadBootstrapPeers()
	for _, addr := range bootstrapPeers {
		pi, err := peer.AddrInfoFromString(addr)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		s, err := n.host.NewStream(ctx, pi.ID, protocolID)
		if err != nil {
			cancel()
			log.Printf("[FIND] cannot open stream to %s: %v", pi.ID, err)
			continue
		}
		fmt.Fprintf(s, "%s%s\n", FIND_PREFIX, targetID)
		buf := make([]byte, 64*1024)
		s.SetReadDeadline(time.Now().Add(5 * time.Second))
		nr, _ := s.Read(buf)
		s.Close()
		cancel()

		response := strings.TrimSpace(string(buf[:nr]))
		addrs := parseMultiaddrsResponse(response, FOUND_PREFIX)
		if len(addrs) > 0 {
			log.Printf("[FIND] %s → %d addrs (via %s): %v", targetID, len(addrs), pi.ID, addrs)
			return addrs, nil
		}
		if strings.HasPrefix(response, NOT_FOUND_PREFIX) {
			log.Printf("[FIND] %s not found on %s", targetID, pi.ID)
		}
	}

	return nil, fmt.Errorf("not found")
}

func parseMultiaddrsResponse(response, prefix string) []string {
	lines := strings.Split(response, "\n")
	if len(lines) == 0 {
		return nil
	}
	if strings.TrimSpace(lines[0]) != prefix {
		return nil
	}
	var addrs []string
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" || line == END_PREFIX {
			continue
		}
		addrs = append(addrs, line)
	}
	return addrs
}
func (n *Node) buildRelayAddrFor(targetID string) string {
	if n.host == nil {
		return ""
	}
	myID := n.host.ID().String()
	for _, a := range n.host.Addrs() {
		s := a.String()
		if strings.Contains(s, "127.0.0.1") {
			continue
		}
		if strings.Contains(s, "/ws") {
			return s + "/p2p/" + myID + "/p2p-circuit/p2p/" + targetID
		}
	}
	for _, a := range n.host.Addrs() {
		s := a.String()
		if strings.Contains(s, "127.0.0.1") {
			continue
		}
		return s + "/p2p/" + myID + "/p2p-circuit/p2p/" + targetID
	}
	return ""
}

// GetRelayAddrs — возвращает список relay-адресов для анонса.
func (n *Node) GetRelayAddrs() []string {
	n.relayMu.Lock()
	defer n.relayMu.Unlock()
	if n.relayReservation == nil {
		return nil
	}
	if n.relayPeerInfo.ID == "" {
		return nil
	}
	myID := n.host.ID().String()

	var result []string
	for _, addr := range n.loadBootstrapPeers() {
		pi, err := peer.AddrInfoFromString(addr)
		if err != nil {
			continue
		}
		if pi.ID != n.relayPeerInfo.ID {
			continue
		}
		full := addr + "/p2p-circuit/p2p/" + myID
		result = append(result, full)
		log.Printf("[RELAY] built relay addr: %s", full)
	}
	return result
}

// ReserveRelaySlot — резервирует слот на relay-сервере.
func (n *Node) ReserveRelaySlot(ctx context.Context, relayAddrInfo peer.AddrInfo) error {
	if n.host == nil {
		return fmt.Errorf("node not started")
	}

	resv, err := client.Reserve(ctx, n.host, relayAddrInfo)
	if err != nil {
		n.relayMu.Lock()
		n.lastReservationErr = err
		n.relayMu.Unlock()
		return fmt.Errorf("relay reserve failed: %w", err)
	}

	n.relayMu.Lock()
	n.relayReservation = resv
	n.relayPeerInfo = relayAddrInfo
	n.lastReservationErr = nil
	n.relayMu.Unlock()

	log.Printf("[RELAY] reserved slot, expires=%s", resv.Expiration)
	return nil
}

// relayLoop — периодически проверяет наличие резервации и обновляет её.
func (n *Node) relayLoop() {
	go func() {
		backoff := 2 * time.Second
		const maxBackoff = 30 * time.Second

		for {
			time.Sleep(backoff)

			n.relayMu.Lock()
			resv := n.relayReservation
			relayInfo := n.relayPeerInfo
			n.relayMu.Unlock()

			if relayInfo.ID == "" {
				continue
			}

			needReserve := false
			if resv == nil {
				needReserve = true
			} else if time.Until(resv.Expiration) < 5*time.Minute {
				needReserve = true
			}

			if !needReserve {
				backoff = 2 * time.Second
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := n.ReserveRelaySlot(ctx, relayInfo)
			cancel()
			if err != nil {
				log.Printf("[RELAY] reserve failed (backoff=%s): %v", backoff, err)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
			backoff = 2 * time.Second
		}
	}()
}

// isRelayAddr — true, если remote — наш relay-сервер (bootstrap).
func (n *Node) isRelayAddr(remotePeerID string) bool {
	for _, addr := range n.loadBootstrapPeers() {
		pi, err := peer.AddrInfoFromString(addr)
		if err != nil {
			continue
		}
		if pi.ID.String() == remotePeerID {
			return true
		}
	}
	return false
}

// refreshRelayAndAnnounce — немедленно пересоздаёт резервацию на VPS
// и анонсирует новый адрес.
func (n *Node) refreshRelayAndAnnounce() {
	n.relayMu.Lock()
	if time.Since(n.lastRelayRefresh) < 10*time.Second && n.lastReservationErr == nil {
		n.relayMu.Unlock()
		return
	}
	n.lastRelayRefresh = time.Now()
	relayInfo := n.relayPeerInfo
	n.relayMu.Unlock()

	if relayInfo.ID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err := n.ReserveRelaySlot(ctx, relayInfo)
	cancel()
	if err != nil {
		log.Printf("[RELAY] refresh on reconnect failed: %v", err)
		return
	}
	log.Printf("[RELAY] refreshed on reconnect")

	if len(n.announceMultiaddrs) > 0 {
		n.SendAnnounce(n.announceMultiaddrs)
	}
}

// GetMultiaddrs — возвращает все адреса узла.
func (n *Node) GetMultiaddrs() []string {
	if n.host == nil {
		return []string{}
	}
	myID := n.host.ID().String()

	ifaceAddrs, err := n.host.Network().InterfaceListenAddresses()
	if err != nil {
		ifaceAddrs = nil
	}

	fallbackAddrs := n.host.Addrs()

	seen := make(map[string]bool)
	var result []string

	addAddr := func(addr string) {
		if seen[addr] {
			return
		}
		seen[addr] = true
		result = append(result, addr+"/p2p/"+myID)
	}

	for _, a := range ifaceAddrs {
		s := a.String()
		if strings.Contains(s, "127.0.0.1") {
			continue
		}
		addAddr(s)
	}

	for _, a := range fallbackAddrs {
		s := a.String()
		if strings.Contains(s, "127.0.0.1") {
			continue
		}
		addAddr(s)
	}

	return result
}

// ConnectToPeer — подключение к пиру.
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

// ConnectToPeerWithFallback — пробует все multiaddr параллельно.
func (n *Node) ConnectToPeerWithFallback(multiaddrs []string) (string, error) {
	if len(multiaddrs) == 0 {
		return "", fmt.Errorf("empty multiaddrs list")
	}
	if n.host == nil {
		return "", fmt.Errorf("node not started")
	}

	type result struct {
		addr string
		err  error
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := make(chan result, len(multiaddrs))

	for _, ma := range multiaddrs {
		go func(addr string) {
			peerInfo, err := peer.AddrInfoFromString(addr)
			if err != nil {
				results <- result{addr: addr, err: err}
				return
			}

			dialCtx, dialCancel := context.WithTimeout(ctx, 10*time.Second)
			defer dialCancel()

			if err := n.host.Connect(dialCtx, *peerInfo); err != nil {
				results <- result{addr: addr, err: err}
				return
			}
			results <- result{addr: addr, err: nil}
		}(ma)
	}

	var lastErr error
	for i := 0; i < len(multiaddrs); i++ {
		r := <-results
		if r.err == nil {
			cancel()
			log.Printf("[CONNECT] success %s", r.addr)
			go func() {
				time.Sleep(2 * time.Second)
				peerInfo, err := peer.AddrInfoFromString(r.addr)
				if err == nil {
					n.ExchangePeers(peerInfo.ID.String())
				}
			}()
			return r.addr, nil
		}
		lastErr = r.err
		log.Printf("[CONNECT] failed %s: %v", r.addr, r.err)
	}

	return "", fmt.Errorf("all dials failed: %v", lastErr)
}

// ExchangePeers — обменивается списками узлов.
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

// GetKnownPeers — возвращает список известных multiaddr.
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
