/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady OBrien. All Rights Reserved.
 *
 * Definitions and decoders required to implement a subset of the VITA 49 protocol
 */

package main

import (
	b "encoding/binary"
)

/* Header of VITA-49 packet without payload */
type VitaIfDataHeader struct {
	Header         uint32
	StreamID       uint32
	ClassIDH       uint32
	ClassIDL       uint32
	TimestampInt   uint32
	TimestampFracH uint32
	TimestampFracL uint32
}

/* Vita packet with data */
type VitaIFData struct {
	Header          VitaIfDataHeader
	BytesValid      int
	DataBytes       []byte
	RawPacketBuffer []byte
}

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
}

func ReadVitaHeaderStream(rawPkt []byte, header *VitaIfDataHeader, payloadSize *int) bool {
	if len(rawPkt) < 4 {
		return false
	}
	headerWord := b.BigEndian.Uint32(rawPkt[:])
	payloadWords := int(headerWord & VITA_HEADER_PACKET_SIZE_MASK)
	wordPtr := 4
	headerWords := 1
	trailerWords := 0
	hasSID, hasCID, hasTSI, hasTSF := false, false, false, false

	switch headerWord & VITA_HEADER_PACKET_TYPE_MASK {
	case VITA_PACKET_TYPE_EXT_DATA_WITH_STREAM_ID:
	case VITA_PACKET_TYPE_IF_DATA_WITH_STREAM_ID:
		headerWords += 1
		hasSID = true
	case VITA_PACKET_TYPE_EXT_DATA:
	case VITA_PACKET_TYPE_IF_DATA:
		break
	}

	if headerWord&VITA_HEADER_C_MASK > 0 {
		headerWords += 1
		hasCID = true
	}

	if headerWord&VITA_HEADER_T_MASK > 0 {
		trailerWords += 1
	}

	if headerWord&VITA_HEADER_TSF_MASK != VITA_TSF_NONE {
		headerWords += 1
		hasTSF = true
	}

	if headerWord&VITA_HEADER_TSI_MASK != VITA_TSI_NONE {
		headerWords += 1
		hasTSI = true
	}

	if len(rawPkt) < headerWords*4 {
		return false
	}

	*header = VitaIfDataHeader{0, 0, 0, 0, 0, 0, 0}
	header.Header = headerWord

	if hasSID {
		header.StreamID = b.BigEndian.Uint32(rawPkt[wordPtr:])
		wordPtr += 4
	}

	if hasCID {
		header.ClassIDH = b.BigEndian.Uint32(rawPkt[wordPtr:])
		header.ClassIDL = b.BigEndian.Uint32(rawPkt[wordPtr+4:])
		wordPtr += 8
	}

	if hasTSI {
		header.TimestampInt = b.BigEndian.Uint32(rawPkt[wordPtr:])
		wordPtr += 4
	}

	if hasTSF {
		header.TimestampFracH = b.BigEndian.Uint32(rawPkt[wordPtr:])
		header.TimestampFracL = b.BigEndian.Uint32(rawPkt[wordPtr+4:])
		wordPtr += 8
	}

	if payloadSize != nil {
		*payloadSize = payloadWords - headerWords - trailerWords
	}

	return true
}

/* Constants taken from vita.h */
const VITA_HEADER_PACKET_TYPE_MASK uint32 = 0xF0000000
const VITA_PACKET_TYPE_IF_DATA uint32 = 0x00000000
const VITA_PACKET_TYPE_IF_DATA_WITH_STREAM_ID uint32 = 0x10000000
const VITA_PACKET_TYPE_EXT_DATA uint32 = 0x20000000
const VITA_PACKET_TYPE_EXT_DATA_WITH_STREAM_ID uint32 = 0x30000000
const VITA_PACKET_TYPE_CONTEXT uint32 = 0x40000000
const VITA_PACKET_TYPE_EXT_CONTEXT uint32 = 0x50000000
const VITA_HEADER_C_MASK uint32 = 0x08000000
const VITA_HEADER_CLASS_ID_PRESENT uint32 = 0x08000000
const VITA_HEADER_T_MASK uint32 = 0x04000000
const VITA_HEADER_TRAILER_PRESENT uint32 = 0x04000000
const VITA_HEADER_TSI_MASK uint32 = 0x00C00000
const VITA_TSI_NONE uint32 = 0x00000000
const VITA_TSI_UTC uint32 = 0x00400000
const VITA_TSI_GPS uint32 = 0x00800000
const VITA_TSI_OTHER uint32 = 0x00C00000
const VITA_HEADER_TSF_MASK uint32 = 0x00300000
const VITA_TSF_NONE uint32 = 0x00000000
const VITA_TSF_SAMPLE_COUNT uint32 = 0x00100000
const VITA_TSF_REAL_TIME uint32 = 0x00200000
const VITA_TSF_FREE_RUNNING uint32 = 0x00300000
const VITA_HEADER_PACKET_COUNT_MASK uint32 = 0x000F0000
const VITA_HEADER_PACKET_SIZE_MASK uint32 = 0x0000FFFF
const VITA_CLASS_ID_OUI_MASK uint32 = 0x00FFFFFF
const VITA_CLASS_ID_INFORMATION_CLASS_MASK uint32 = 0xFFFF0000
const VITA_CLASS_ID_PACKET_CLASS_MASK uint32 = 0x0000FFFF
