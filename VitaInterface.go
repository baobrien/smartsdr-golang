/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady OBrien. All Rights Reserved.
 *
 * This implements enough of a subset of ANSI/VITA-49 to communicate
 * waveform streams to and from FlexRadio SmartSDR devices
 */

package main

import (
	b "encoding/binary"
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
	Conn         net.PacketConn // UDP Connection
	BufBag       *VitaBufferPool
	Subscribers  map[uint32]StreamSubscriber
	SendChannel  chan *VitaIFData
	SendCounters map[uint32]uint64
	LocalAddr    net.Addr
	RemoteAddr   net.Addr
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
		Conn:         conn,
		LocalAddr:    localaddr,
		RemoteAddr:   remoteaddr,
		BufBag:       CreateVitaBufferPool(BUF_POOL_SIZE),
		SendChannel:  make(chan *VitaIFData, 10),
		SendCounters: make(map[uint32]uint64),
		Subscribers:  make(map[uint32]StreamSubscriber),
	}

	return vitaIface, nil
}

func ParseVitaDataPacket(buf []byte, packet *VitaIFData) bool {
	correct, payloadWords, headerWords := ReadVitaHeaderStream(buf, &(packet.Header))
	if !correct {
		return false
	}
	if len(buf) < (headerWords+payloadWords)*4 {
		return false
	}
	packet.DataBytes = buf[headerWords*4:]
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
			if sub, ok := vif.Subscribers[pkt.Header.StreamID]; ok {
				// Add reference to underlying packet buffer slice so we can correctly free to pool later
				pkt.RawPacketBuffer = buffer

				sub(pkt, vif.BufBag)
				// Grab a new packet from the buffer pool.
				// If the parse gets us a valid packet, the called thing should handle release
				// Otherwise, we  just re-use the packet
				buffer, pkt = vif.BufBag.grabPB()
			}
		}
	}
	return nil
}

/*
func ReadVitaHeader(rawPkt []byte, header *VitaIfDataHeader) bool {
	if len(rawPkt) < 28 {
		return false
	}
	header.Header = b.BigEndian.Uint32(rawPkt[:])
	header.StreamID = b.BigEndian.Uint32(rawPkt[4:])
	header.ClassIDH = b.BigEndian.Uint32(rawPkt[8:])
	header.ClassIDL = b.BigEndian.Uint32(rawPkt[12:])
	header.TimestampInt = b.BigEndian.Uint32(rawPkt[16:])
	header.TimestampFracH = b.BigEndian.Uint32(rawPkt[20:])
	header.TimestampFracL = b.BigEndian.Uint32(rawPkt[24:])
	return true
}*/

func PackVifSendPacket(packet *VitaIFData, buffer []byte, seq uint32) int {

	var payload_word_count = len(packet.DataBytes) / 4
	var hdrWord uint32 = VITA_PACKET_TYPE_IF_DATA_WITH_STREAM_ID
	hdrWord |= VITA_HEADER_CLASS_ID_PRESENT
	hdrWord |= VITA_TSI_OTHER
	hdrWord |= VITA_TSF_SAMPLE_COUNT
	hdrWord |= (seq & 0xF) << 16
	hdrWord |= (7 + uint32(payload_word_count))
	packetBytes := (7 + payload_word_count) * 4
	if len(buffer) < packetBytes {
		return 0 //Return error?
	}
	/* Pack the output buffer */

	b.BigEndian.PutUint32(buffer[:], hdrWord)
	b.BigEndian.PutUint32(buffer[4:], packet.Header.StreamID)
	b.BigEndian.PutUint32(buffer[8:], packet.Header.ClassIDH)
	b.BigEndian.PutUint32(buffer[12:], packet.Header.ClassIDL)
	b.BigEndian.PutUint32(buffer[16:], packet.Header.TimestampInt)
	b.BigEndian.PutUint32(buffer[20:], packet.Header.TimestampFracH)
	b.BigEndian.PutUint32(buffer[24:], packet.Header.TimestampFracL)

	return packetBytes
}

func (vif *VitaInterface) VitaSenderLoop() error {
	sendBuf := make([]byte, MAX_PACKET_LEN)
	for {
		pkt := <-vif.SendChannel

		/* Increment stream counter and sequence number */
		vif.SendCounters[pkt.Header.StreamID]++
		count := vif.SendCounters[pkt.Header.StreamID]

		n := PackVifSendPacket(pkt, sendBuf, uint32(count))
		if n > 0 {
			m, err := vif.Conn.WriteTo(sendBuf[:n], vif.RemoteAddr)
			if err != nil {
				//TODO: error handling here
				break
			}
			if m != n {
				break
			}
		}
		vif.BufBag.releasePB(pkt.RawPacketBuffer, pkt)
	}
	return nil
}
