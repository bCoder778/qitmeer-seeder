package main

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/Qitmeer/qitmeer/core/blockdag"
	"github.com/Qitmeer/qitmeer/core/message"
	"github.com/Qitmeer/qitmeer/core/protocol"
	"github.com/Qitmeer/qitmeer/p2p/connmgr"
	"github.com/Qitmeer/qitmeer/p2p/peer"
)

var (
	ipMap  map[string]bool
	mutex  sync.Mutex
	custom = []string{
		"121.196.55.29",
		"121.196.28.213",
		"121.196.54.163",
		"47.114.183.16",
		"47.114.184.240",
	}
)

func creep() {
	defer globalWg.Done()

	onaddr := make(chan struct{})
	verack := make(chan struct{})

	newestGSFunc := func() (gs *blockdag.GraphState, err error) {
		gs = blockdag.NewGraphState()
		gs.GetTips().Add(activeNetParams.GenesisHash)
		gs.SetTotal(1)
		return gs, err
	}

	onAddrFunc := func(p *peer.Peer, msg *message.MsgAddr) {
		n := make([]net.IP, 0, len(msg.AddrList))
		for _, addr := range msg.AddrList {
			n = append(n, addr.IP)
		}
		added := manager.AddAddresses(n)
		log.Printf("Peer %v sent %v addresses, %d new",
			p.Addr(), len(msg.AddrList), added)
		onaddr <- struct{}{}
	}

	onVerAckFunc := func(p *peer.Peer, msg *message.MsgVerAck) {
		log.Printf("Adding peer %v with services %v",
			p.NA().IP.String(), p.Services())

		verack <- struct{}{}
	}

	messageListener := peer.MessageListeners{
		OnAddr:   onAddrFunc,
		OnVerAck: onVerAckFunc,
	}

	peerConfig := peer.Config{
		NewestGS:          newestGSFunc,
		UserAgentName:     "qitmeer-seeder",
		UserAgentVersion:  "0.3.1",
		UserAgentComments: []string{"qitmeer", "seeder"},
		ChainParams:       activeNetParams,
		DisableRelayTx:    true,
		Services:          protocol.Full,
		ProtocolVersion:   protocol.ProtocolVersion,
		Listeners:         messageListener,
	}

	var wg sync.WaitGroup
	for {
		ips := manager.Addresses()
		if len(ips) == 0 {
			log.Printf("No stale addresses -- sleeping for %v",
				defaultAddressTimeout)
			time.Sleep(defaultAddressTimeout)
			continue
		}

		wg.Add(len(ips))

		for _, ip := range ips {
			go func(ip net.IP) {
				defer wg.Done()

				host := net.JoinHostPort(ip.String(),
					activeNetParams.DefaultPort)
				p, err := peer.NewOutboundPeer(&peerConfig, host)
				if err != nil {
					log.Printf("NewOutboundPeer on %v: %v",
						host, err)
					return
				}
				manager.Attempt(ip)
				conn, err := net.DialTimeout("tcp", p.Addr(),
					defaultNodeTimeout)
				if err != nil {
					return
				}

				c := connmgr.NewConnReq()
				c.SetConn(conn)

				p.AssociateConnection(c)

				// Wait for the verack message or timeout in case of
				// failure.
				select {
				case <-verack:
					// Mark this peer as a good node.
					manager.Good(p.NA().IP, p.Services())

					// Ask peer for some addresses.
					p.QueueMessage(message.NewMsgGetAddr(), nil)

				case <-time.After(defaultNodeTimeout):
					log.Printf("verack timeout on peer %v",
						p.Addr())
					p.Disconnect()
					return
				}

				select {
				case <-onaddr:

				case <-time.After(defaultNodeTimeout):
					log.Printf("getaddr timeout on peer %v",
						p.Addr())
					p.Disconnect()
					return
				}
				p.Disconnect()
			}(ip)
		}
		wg.Wait()
	}
}

func creepOne(ip net.IP) []net.IP {
	retureIps := make([]net.IP, 0)
	onaddr := make(chan struct{})
	verack := make(chan struct{})

	newestGSFunc := func() (gs *blockdag.GraphState, err error) {
		gs = blockdag.NewGraphState()
		gs.GetTips().Add(activeNetParams.GenesisHash)
		gs.SetTotal(1)
		return gs, err
	}

	onAddrFunc := func(p *peer.Peer, msg *message.MsgAddr) {
		n := make([]net.IP, 0, len(msg.AddrList))
		for _, addr := range msg.AddrList {
			n = append(n, addr.IP)
			retureIps = append(n, addr.IP)
		}
		added := manager.AddAddresses(n)
		log.Printf("Peer %v sent %v addresses, %d new",
			p.Addr(), len(msg.AddrList), added)
		onaddr <- struct{}{}
	}

	onVerAckFunc := func(p *peer.Peer, msg *message.MsgVerAck) {
		log.Printf("Adding peer %v with services %v",
			p.NA().IP.String(), p.Services())

		verack <- struct{}{}
	}

	messageListener := peer.MessageListeners{
		OnAddr:   onAddrFunc,
		OnVerAck: onVerAckFunc,
	}

	peerConfig := peer.Config{
		NewestGS:          newestGSFunc,
		UserAgentName:     "qitmeer-seeder",
		UserAgentVersion:  "0.3.1",
		UserAgentComments: []string{"qitmeer", "seeder"},
		ChainParams:       activeNetParams,
		DisableRelayTx:    true,
		Services:          protocol.Full,
		ProtocolVersion:   protocol.ProtocolVersion,
		Listeners:         messageListener,
	}

	host := net.JoinHostPort(ip.String(),
		activeNetParams.DefaultPort)
	p, err := peer.NewOutboundPeer(&peerConfig, host)
	if err != nil {
		log.Printf("NewOutboundPeer on %v: %v",
			host, err)
		return retureIps
	}
	conn, err := net.DialTimeout("tcp", p.Addr(),
		defaultNodeTimeout)
	if err != nil {
		conn, err = net.DialTimeout("udp", p.Addr(),
			defaultNodeTimeout)
		if err != nil {
			return retureIps
		}
	}

	c := connmgr.NewConnReq()
	c.SetConn(conn)

	p.AssociateConnection(c)

	// Wait for the verack message or timeout in case of
	// failure.
	select {
	case <-verack:

		// Ask peer for some addresses.
		p.QueueMessage(message.NewMsgGetAddr(), nil)

	case <-time.After(defaultNodeTimeout):
		log.Printf("verack timeout on peer %v",
			p.Addr())
		p.Disconnect()
		return retureIps
	}

	select {
	case <-onaddr:

	case <-time.After(defaultNodeTimeout):
		log.Printf("getaddr timeout on peer %v",
			p.Addr())
		p.Disconnect()
		return retureIps
	}
	p.Disconnect()
	return retureIps
}
