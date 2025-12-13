// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	dhcpPort         = 67
	dhcpClientPort   = 68
	dnsPort          = 53
	leaseTimeSeconds = 3600
)

type lease struct {
	ip      net.IP
	mac     net.HardwareAddr
	expires time.Time
}

var (
	leases     = make(map[string]lease)
	leaseMutex = &sync.Mutex{}
)

type dhcpMessage []byte

func (m dhcpMessage) op() byte        { return m[0] }
func (m dhcpMessage) xid() []byte     { return m[4:8] }
func (m dhcpMessage) chaddr() []byte  { return m[28:44] }
func (m dhcpMessage) magic() []byte   { return m[236:240] }
func (m dhcpMessage) options() []byte { return m[240:] }

func (m dhcpMessage) setOp(op byte)        { m[0] = op }
func (m dhcpMessage) setXid(xid []byte)    { copy(m[4:8], xid) }
func (m dhcpMessage) setYIAddr(ip net.IP)  { copy(m[16:20], ip.To4()) }
func (m dhcpMessage) setSIAddr(ip net.IP)  { copy(m[20:24], ip.To4()) }
func (m dhcpMessage) setCHAddr(mac []byte) { copy(m[28:44], mac) }
func (m dhcpMessage) setMagic()            { copy(m[236:240], []byte{99, 130, 83, 99}) }

func getDHCPOption(msg dhcpMessage, option byte) []byte {
	opts := msg.options()
	for i := 0; i < len(opts)-2; {
		if opts[i] == option {
			length := int(opts[i+1])
			return opts[i+2 : i+2+length]
		}
		if opts[i] == 255 {
			break
		}
		if opts[i] == 0 {
			i++
			continue
		}
		i += int(opts[i+1]) + 2
	}
	return nil
}

func getDHCPMessageType(msg dhcpMessage) byte {
	opt := getDHCPOption(msg, 53)
	if opt != nil {
		return opt[0]
	}
	return 0
}

func configureInterface(ifaceName string, ip net.IP, mask net.IPMask) error {
	appLogger.Printf("Attempting to configure %s with %s", ifaceName, ip.String())
	prefixLen, _ := mask.Size()
	addr := fmt.Sprintf("%s/%d", ip.String(), prefixLen)
	var cmd *exec.Cmd
	if runtime.GOOS == "linux" {
		cmd = exec.Command("ip", "addr", "add", addr, "dev", ifaceName)
	} else {
		return fmt.Errorf("unsupported OS")
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to configure IP: %v, output: %s", err, string(output))
	}
	return nil
}

func ipToUint32(ip net.IP) uint32 {
	return binary.BigEndian.Uint32(ip.To4())
}

func uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}

func runDHCPServer(ifaceName string, subnet *net.IPNet, serverIP net.IP, dnsIP net.IP) {
	if err := configureInterface(ifaceName, serverIP, subnet.Mask); err != nil {
		log.Printf("DHCP: WARN: Could not auto-configure IP for %s: %v", ifaceName, err)
	}

	conn, err := net.ListenPacket("udp4", fmt.Sprintf("%s:%d", serverIP.String(), dhcpPort))
	if err != nil {
		log.Fatalf("DHCP: Failed to listen on %s:%d: %v", serverIP.String(), dhcpPort, err)
		return
	}
	defer conn.Close()
	appLogger.Printf("DHCP server started on %s for subnet %s", serverIP, subnet)

	buf := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}
		handleDHCPRequest(conn, buf[:n], subnet, serverIP, dnsIP)
	}
}

func handleDHCPRequest(conn net.PacketConn, data []byte, subnet *net.IPNet, serverIP, dnsIP net.IP) {
	req := dhcpMessage(data)
	if req.op() != 1 || len(req.magic()) < 4 || !bytes.Equal(req.magic(), []byte{99, 130, 83, 99}) {
		return
	}

	msgType := getDHCPMessageType(req)
	macAddr := net.HardwareAddr(req.chaddr()[0:6])
	macStr := macAddr.String()

	var offeredIP net.IP
	leaseMutex.Lock()
	if existingLease, ok := leases[macStr]; ok && existingLease.expires.After(time.Now()) {
		offeredIP = existingLease.ip
	} else {
		startIP := ipToUint32(subnet.IP) | 100
		endIP := (ipToUint32(subnet.IP) | 0xFF) &^ (ipToUint32(net.IP(subnet.Mask)) & 0xFF)
		usedIPs := make(map[uint32]bool)
		for _, l := range leases {
			usedIPs[ipToUint32(l.ip)] = true
		}

		for i := startIP; i <= endIP; i++ {
			if !usedIPs[i] {
				offeredIP = uint32ToIP(i)
				break
			}
		}
	}
	leaseMutex.Unlock()

	if offeredIP == nil {
		appLogger.Printf("DHCP: No available IPs in subnet %s", subnet.String())
		return
	}

	var resp dhcpMessage
	switch msgType {
	case 1: // DISCOVER
		resp = buildDHCPResponse(req, 2, serverIP, offeredIP, subnet.Mask, dnsIP) // OFFER
	case 3: // REQUEST
		resp = buildDHCPResponse(req, 5, serverIP, offeredIP, subnet.Mask, dnsIP) // ACK
		leaseMutex.Lock()
		leases[macStr] = lease{ip: offeredIP, mac: macAddr, expires: time.Now().Add(leaseTimeSeconds * time.Second)}
		leaseMutex.Unlock()
		appLogger.Printf("DHCP: Assigned IP %s to %s", offeredIP, macStr)
	default:
		return
	}

	destAddr := &net.UDPAddr{IP: net.IPv4bcast, Port: dhcpClientPort}
	conn.WriteTo(resp, destAddr)
}

func buildDHCPResponse(req dhcpMessage, msgType byte, serverIP, offeredIP net.IP, mask net.IPMask, dnsIP net.IP) dhcpMessage {
	resp := make(dhcpMessage, 576)
	resp.setOp(2)
	resp.setXid(req.xid())
	resp.setYIAddr(offeredIP)
	resp.setSIAddr(serverIP)
	resp.setCHAddr(req.chaddr())
	resp.setMagic()

	options := new(bytes.Buffer)
	options.Write([]byte{53, 1, msgType})
	leaseBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(leaseBytes, leaseTimeSeconds)
	options.Write(append([]byte{51, 4}, leaseBytes...))
	options.Write(append([]byte{54, 4}, serverIP.To4()...))
	options.Write(append([]byte{1, 4}, mask...))
	options.Write(append([]byte{3, 4}, serverIP.To4()...))
	options.Write(append([]byte{6, 4}, dnsIP.To4()...))
	options.WriteByte(255)

	copy(resp[240:], options.Bytes())
	return resp
}

const (
	queryTypeA   = 1
	queryClassIN = 1
)

type dnsHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

type dnsQuestion struct {
	Name  string
	Type  uint16
	Class uint16
}

func parseDNSQuestion(data []byte) (*dnsQuestion, int) {
	var name strings.Builder
	offset := 0
	for data[offset] != 0 {
		length := int(data[offset])
		offset++
		if name.Len() > 0 {
			name.WriteByte('.')
		}
		name.Write(data[offset : offset+length])
		offset += length
	}
	offset++

	q := &dnsQuestion{Name: name.String()}
	q.Type = binary.BigEndian.Uint16(data[offset:])
	q.Class = binary.BigEndian.Uint16(data[offset+2:])
	return q, offset + 4
}

func runDNSServer(serverIP net.IP, forwarderIP net.IP) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: serverIP, Port: dnsPort})
	if err != nil {
		log.Fatalf("DNS: Failed to listen on %s:%d: %v", serverIP.String(), dnsPort, err)
	}
	defer conn.Close()
	appLogger.Printf("DNS server started on %s (Forwarder: %s)", serverIP, forwarderIP)

	for {
		buf := make([]byte, 512)
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		go handleDNSRequest(conn, addr, buf[:n], serverIP, forwarderIP)
	}
}

func handleDNSRequest(conn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, serverIP, forwarderIP net.IP) {
	var header dnsHeader
	reader := bytes.NewReader(data)
	binary.Read(reader, binary.BigEndian, &header)

	if header.QDCount == 0 {
		return
	}

	question, _ := parseDNSQuestion(data[12:])

	if forwarderIP != nil {
		forwardedResp, err := forwardDNSQuery(forwarderIP, data)
		if err == nil {
			conn.WriteTo(forwardedResp, clientAddr)
			return
		}
		appLogger.Printf("DNS: Forwarder failed: %v. Falling back to sinkhole.", err)
	}

	var resp []byte
	if question.Type == queryTypeA && question.Class == queryClassIN {
		resp = buildDNSResponse(header, *question, serverIP)
	} else {
		resp = buildDNSResponse(header, *question, nil)
	}

	conn.WriteTo(resp, clientAddr)
}

func forwardDNSQuery(forwarderIP net.IP, query []byte) ([]byte, error) {
	fconn, err := net.Dial("udp", fmt.Sprintf("%s:%d", forwarderIP.String(), dnsPort))
	if err != nil {
		return nil, err
	}
	defer fconn.Close()

	fconn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := fconn.Write(query); err != nil {
		return nil, err
	}

	resp := make([]byte, 512)
	n, err := fconn.Read(resp)
	if err != nil {
		return nil, err
	}
	return resp[:n], nil
}

func buildDNSResponse(header dnsHeader, question dnsQuestion, ip net.IP) []byte {
	buf := new(bytes.Buffer)

	respHeader := header
	respHeader.Flags = 0x8180 // Response, authoritative, no error
	if ip != nil {
		respHeader.ANCount = 1
	} else {
		respHeader.Flags = 0x8183 // NXDOMAIN
	}
	binary.Write(buf, binary.BigEndian, respHeader)

	offset := 0
	for _, part := range strings.Split(question.Name, ".") {
		buf.WriteByte(byte(len(part)))
		buf.WriteString(part)
		offset += len(part) + 1
	}
	buf.WriteByte(0)
	binary.Write(buf, binary.BigEndian, question.Type)
	binary.Write(buf, binary.BigEndian, question.Class)

	if ip != nil {
		binary.Write(buf, binary.BigEndian, uint16(0xc00c)) // Pointer to question name
		binary.Write(buf, binary.BigEndian, uint16(queryTypeA))
		binary.Write(buf, binary.BigEndian, uint16(queryClassIN))
		binary.Write(buf, binary.BigEndian, uint32(300)) // TTL
		binary.Write(buf, binary.BigEndian, uint16(4))   // RDLENGTH
		buf.Write(ip.To4())
	}

	return buf.Bytes()
}

func startNetworkServices() {
	rand.Seed(time.Now().UnixNano())

	if len(options.DHCPInterfaces) == 0 {
		return
	}

	if runtime.GOOS != "linux" {
		log.Fatalf("network servers only available on Linux.")
	}

	var dnsForwarder net.IP
	dnsEnabled := options.DNS != ""
	if dnsEnabled && options.DNS != "true" {
		dnsForwarder = net.ParseIP(options.DNS)
		if dnsForwarder == nil {
			log.Fatalf("Invalid DNS forwarder IP: %s", options.DNS)
		}
	}

	var dnsServerIP net.IP
	firstServerIPSet := false

	for _, config := range options.DHCPInterfaces {
		parts := strings.SplitN(config, ":", 2)
		if len(parts) != 2 {
			log.Printf("Invalid --dhcp format, skipping: %s", config)
			continue
		}
		ifaceName, subnetStr := parts[0], parts[1]

		_, subnet, err := net.ParseCIDR(subnetStr)
		if err != nil {
			log.Printf("Invalid subnet CIDR, skipping: %s", subnetStr)
			continue
		}

		netAddrVal := ipToUint32(subnet.IP)
		serverIPVal := netAddrVal + 1
		serverIP := uint32ToIP(serverIPVal)

		dnsIPForThisSubnet := serverIP
		if dnsForwarder != nil {
			dnsIPForThisSubnet = serverIP
		}

		if !firstServerIPSet {
			dnsServerIP = serverIP
			firstServerIPSet = true
		}

		go runDHCPServer(ifaceName, subnet, serverIP, dnsIPForThisSubnet)
	}

	if dnsEnabled && dnsServerIP != nil {
		go runDNSServer(dnsServerIP, dnsForwarder)
	}
}

func getServingIPs() []string {
	var listeningIPs []string

	host := options.WebHost

	if host == "0.0.0.0" || host == "" {
		interfaces, err := net.Interfaces()
		if err != nil {
			return []string{"Error getting interfaces: " + err.Error()}
		}

		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp == 0 {
				continue
			}

			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok {
					if ipnet.IP.To4() != nil {
						listeningIPs = append(listeningIPs, ipnet.IP.String())
					}
				}
			}
		}
	} else if host == "localhost" || host == "127.0.0.1" {
		listeningIPs = []string{"127.0.0.1"}
	} else {
		listeningIPs = []string{host}
	}

	return listeningIPs
}
