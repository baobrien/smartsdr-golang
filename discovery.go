/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady OBrien. All Rights Reserved.
 *
 *
 * Utility to discover FlexRadio devices using the VITA-49 based discovery protocol
 */

package main

import (
	"fmt"
	"os"
)
import (
	"errors"
	"net"
	"time"
)

type Radio struct {
	discoveryProtocolVersion string
	model                    string
	serial                   string
	version                  string
	nickname                 string
	ip                       string
	port                     string
	status                   string
	callsign                 string
}

const DISCOVERY_PORT = 4992

type DiscoveryClient struct {
	errors    chan error
	radios    chan *Radio
	quit      chan int
	udplisten net.PacketConn
}

func (radio *Radio) String() string {
	return fmt.Sprintf("discovery_protocol_version=%s model=%s serial=%s version=%s nickname=%s callsign=%s ip=%s port=%s status=%s\n",
		radio.discoveryProtocolVersion,
		radio.model,
		radio.serial,
		radio.version,
		radio.nickname,
		radio.callsign,
		radio.ip,
		radio.port,
		radio.status)
}

func CreateDiscoveryClient(addr *net.UDPAddr) (*DiscoveryClient, error) {
	discli := &DiscoveryClient{
		errors: make(chan error),
		radios: make(chan *Radio),
		quit:   make(chan int),
	}
	ulistenfile, err := discoveryGetUDPListener(addr)
	if err != nil {
		if ulistenfile != nil {
			ulistenfile.Close()
		}
		return nil, err
	}
	ulisten, err := net.FilePacketConn(ulistenfile)
	if err != nil {
		if ulisten != nil {
			ulisten.Close()
		}
		return nil, err
	}
	discli.udplisten = ulisten
	return discli, nil
}

func parseDiscoveryPacket(buf []byte) (*Radio, error) {
	if len(buf) < 28 {
		return nil, errors.New("parseDiscoveryPacket: packet too short")
	}
	v := &VitaIfDataHeader{}

	ReadVitaHeader(buf, v)

	if v.ClassIDH != 0x00001C2D {
		return nil, errors.New(fmt.Sprintf("parseDiscoveryPacket: Wrong OUI %08x", v.ClassIDH))
	}
	if (v.Header & VITA_HEADER_PACKET_TYPE_MASK) != VITA_PACKET_TYPE_EXT_DATA_WITH_STREAM_ID {
		return nil, errors.New(fmt.Sprintf("parseDiscoveryPacket: Wrong Packet Type %08x", v.Header))
	}
	if (v.ClassIDL & VITA_CLASS_ID_PACKET_CLASS_MASK) != 0xFFFF {
		return nil, errors.New(fmt.Sprintf("parseDiscoveryPacket: Wrong class %08x", v.ClassIDH))
	}

	radio := &Radio{}
	discstr := string(buf[28:])
	for k, v := range detokenize(discstr) {
		switch k {
		case "discovery_protocol_version":
			radio.discoveryProtocolVersion = v
		case "model":
			radio.model = v
		case "serial":
			radio.serial = v
		case "nickname":
			radio.nickname = v
		case "ip":
			radio.ip = v
		case "port":
			radio.port = v
		case "status":
			radio.status = v
		case "version":
			radio.version = v
		case "callsign":
			radio.callsign = v
		}
	}
	return radio, nil
}

func (discli *DiscoveryClient) doDiscoveryListen() {
	buf := make([]byte, 1500)
	for {
		select {
		case <-discli.quit:
			discli.udplisten.Close()
			return
		case <-func() chan int {
			dc := make(chan int)
			go func() {
				n, _, err := discli.udplisten.ReadFrom(buf)
				if err != nil {
					discli.errors <- err
					discli.quit <- 1
				}
				if n == 0 {
					discli.errors <- errors.New("Got EOF in UDP listener")
					discli.quit <- 1
				}
				dc <- n
			}()
			return dc
		}():
			r, err := parseDiscoveryPacket(buf)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			} else {
				discli.radios <- r
			}
		}
	}
}

func (discli *DiscoveryClient) Close() {
	// Send on quit channel if client is listening on the other end
	select {
	case discli.quit <- 1:
	default:
	}
}

/*
 Start a discovery client instance on the default port,
  discover a single radio or time out, and close
*/
func DiscoverRadio(timeout time.Duration) (*Radio, error) {
	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:4992")
	if err != nil {
		return nil, err
	}
	disClient, err := CreateDiscoveryClient(addr)
	if err != nil {
		return nil, err
	}

	go disClient.doDiscoveryListen()

	select {
	case <-time.After(timeout):
		disClient.Close()
		return nil, errors.New("Discovery client timed out")
	case radio := <-disClient.radios:
		disClient.Close()
		return radio, nil
	case err := <-disClient.errors:
		disClient.Close()
		return nil, err
	}

	return nil, nil

}
