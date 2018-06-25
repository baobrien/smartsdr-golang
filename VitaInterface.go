/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady OBrien. All Rights Reserved.
 */

package main

import (
	"net"
)

const MAX_PACKET_LEN = 1500
const BUF_POOL_SIZE = 100

/* Pool to keep a bunch of VITA packet buffers */
type VitaBufferPool struct {
	DecodeBufs  chan []byte
	VitaPackets chan *VitaIFData
}

type StreamSubscriber func(*VitaIFData, *VitaBufferPool)

type VitaInterface struct {
	Conn        net.PacketConn // UDP Connection
	BufBag      *VitaBufferPool
	Subscribers map[uint32]StreamSubscriber
}

func CreateVitaBufferPool(nbufs uint) *VitaBufferPool {
	nchancap := (nbufs * 3) / 2
	bufchan := make(chan []byte, nchancap)
	packetchan := make(chan *VitaIFData, nchancap)
	for i := uint(0); i < nbufs; i++ {
		bufchan <- make([]byte, MAX_PACKET_LEN)
		packetchan <- &VitaIFData{}
	}
	pool := &VitaBufferPool{
		DecodeBufs:  bufchan,
		VitaPackets: packetchan,
	}
	return pool
}

func (pool *VitaBufferPool) grabPB() ([]byte, *VitaIFData) {
	return <-pool.DecodeBufs, <-pool.VitaPackets
}

func (pool *VitaBufferPool) releasePB(buf []byte, pkt *VitaIFData) {
	pool.DecodeBufs <- buf
	pool.VitaPackets <- pkt
}

/*type StreamSubscriber struct {
	StreamID   uint32
	PacketRxed func(packet *VitaIFData)
}*/

func InitVitaListener(localaddr, remoteaddr *net.UDPAddr) (*VitaInterface, error) {
	conn, err := net.DialUDP("udp", localaddr, remoteaddr)
	if err != nil {
		return nil, err
	}
	vitaIface := &VitaInterface{
		Conn:   conn,
		BufBag: CreateVitaBufferPool(BUF_POOL_SIZE),
	}

	return vitaIface, nil
}

func ParseVitaDataPacket(buf []byte, packet *VitaIFData) bool {
	if !ReadVitaHeader(buf, &packet.Header) {
		return false
	}

	packet.DataBytes = buf[28:]
	return true
}

func (vif *VitaInterface) VitaListenLoop() error {

	// Get a packet buffer from the buffer pool
	buffer, pkt := vif.BufBag.grabPB()
	for {
		n, _, err := vif.Conn.ReadFrom(buffer)
		if err != nil {
			return err
		}
		if ParseVitaDataPacket(buffer[:n], pkt) {
			// Add reference to underlying packet buffer slice so we can correctly free to pool later
			pkt.RawPacketBuffer = buffer
			// Do stuff here
			// Grab a new packet from the buffer pool.
			// If the parse gets us a valid packet, the called thing should handle release
			// Otherwise, we  just re-use the packet
			buffer, pkt = vif.BufBag.grabPB()
		}
	}
	return nil
}
