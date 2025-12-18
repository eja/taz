// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type Peer struct {
	IP       string `json:"ip"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	LastSeen int64  `json:"last_seen"`
}

var (
	peers      = make(map[string]Peer)
	peersMutex = sync.RWMutex{}
)

func startDiscovery() {
	host := options.WebHost
	if host != "0.0.0.0" && !isPrivateIP(host) {
		return
	}

	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", options.WebPort))
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		appLogger.Printf("Discovery: Error binding UDP: %v", err)
		return
	}

	appLogger.Printf("Discovery started on port %d\n", options.WebPort)

	go func() {
		defer conn.Close()

		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				broadcastShout(conn)
				cleanupOldPeers()
			}
		}()

		buf := make([]byte, 1024)
		for {
			n, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				appLogger.Printf("Discovery socket error: %v", err)
				return
			}

			msg := string(buf[:n])
			if msg == "TAZ_DISCOVER" {
				resp := fmt.Sprintf("TAZ_IDENT|%s|%s", appLabel, appVersion)
				conn.WriteToUDP([]byte(resp), remoteAddr)
			} else if strings.HasPrefix(msg, "TAZ_IDENT|") {
				parts := strings.Split(msg, "|")
				if len(parts) >= 3 {
					updatePeer(remoteAddr.IP.String(), parts[1], parts[2])
				}
			}
		}
	}()

}

func broadcastShout(conn *net.UDPConn) {
	dest := &net.UDPAddr{IP: net.IPv4bcast, Port: 35248}
	conn.WriteToUDP([]byte("TAZ_DISCOVER"), dest)
}

func updatePeer(ip, name, version string) {
	peersMutex.Lock()
	defer peersMutex.Unlock()
	peers[ip] = Peer{
		IP:       ip,
		Name:     name,
		Version:  version,
		LastSeen: time.Now().Unix(),
	}
}

func cleanupOldPeers() {
	peersMutex.Lock()
	defer peersMutex.Unlock()
	now := time.Now().Unix()
	for ip, peer := range peers {
		if now-peer.LastSeen > 45 {
			delete(peers, ip)
		}
	}
}

func getDiscoveredPeers() []Peer {
	peersMutex.RLock()
	defer peersMutex.RUnlock()
	list := []Peer{}
	for _, p := range peers {
		list = append(list, p)
	}
	return list
}
