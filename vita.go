/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady OBrien. All Rights Reserved.
 *
 * Definitions and decoders required to implement a subset of the VITA 49 protocol
 */

package main

import (
	b "encoding/binary"
	"math"
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

func ReadVitaHeaderStream(rawPkt []byte, header *VitaIfDataHeader) (bool, int, int) {
	if len(rawPkt) < 4 {
		return false, 0, 0
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
		headerWords += 2
		hasCID = true
	}

	if headerWord&VITA_HEADER_T_MASK > 0 {
		trailerWords += 1
	}

	if headerWord&VITA_HEADER_TSF_MASK != VITA_TSF_NONE {
		headerWords += 2
		hasTSF = true
	}

	if headerWord&VITA_HEADER_TSI_MASK != VITA_TSI_NONE {
		headerWords += 1
		hasTSI = true
	}

	if len(rawPkt) < headerWords*4 {
		return false, 0, 0
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

	return true, payloadWords - headerWords - trailerWords, headerWords
}

func FltMean(in []float32) float32 {
	var sum float32 = 0.0
	for _, v := range in {
		sum += (v * v)
	}
	return sum / (float32(len(in)))
}

const MAX_SAMP_PER_FRAME = 128

/* TODO: More efficent complex unpacking */
/* TODO: Buffer pool ? */
/* Extract a buffer of complex numbers from a raw VITA-49 packet */
func VitaToComplex(vpkt *VitaIFData) []complex64 {
	pktSamps := len(vpkt.DataBytes) / 8
	samples := make([]complex64, pktSamps)

	for i := 0; i < pktSamps; i++ {
		realu32 := b.BigEndian.Uint32(vpkt.DataBytes[(i * 8):])
		imagu32 := b.BigEndian.Uint32(vpkt.DataBytes[(i*8)+4:])
		samples[i] = complex(math.Float32frombits(realu32), math.Float32frombits(imagu32))
	}

	return samples
}

/* Extract a buffer of float32 from a raw VITA-49 packet */
func VitaToFloat(vpkt *VitaIFData) []float32 {
	pktSamps := len(vpkt.DataBytes) / 8
	samples := make([]float32, pktSamps)

	for i := 0; i < pktSamps; i++ {
		/* Flex only uses the real part for float-only streams */
		realu32 := b.BigEndian.Uint32(vpkt.DataBytes[(i*8)+4:])
		samples[i] = math.Float32frombits(realu32)
	}
	return samples
}

/* Pack complex samples into a VITA frame and return the number of samples packed */
func ComplexToVitaFrame(vpkt *VitaIFData, buf []complex64) int {
	nSamp := len(buf)
	if nSamp > MAX_SAMP_PER_FRAME {
		nSamp = MAX_SAMP_PER_FRAME
	}
	for i := 0; i < nSamp; i++ {
		realu32 := math.Float32bits(real(buf[i]))
		imagu32 := math.Float32bits(imag(buf[i]))
		b.BigEndian.PutUint32(vpkt.DataBytes[i*8:], realu32)
		b.BigEndian.PutUint32(vpkt.DataBytes[(i*8)+4:], imagu32)

	}
	vpkt.DataBytes = vpkt.DataBytes[:nSamp*8]
	return nSamp
}

/* Pack float samples into a VITA frame and return the number of samples packed */
func FloatToVitaFrame(vpkt *VitaIFData, buf []float32) int {
	nSamp := len(buf)
	if nSamp > MAX_SAMP_PER_FRAME {
		nSamp = MAX_SAMP_PER_FRAME
	}
	for i := 0; i < nSamp; i++ {
		realu32 := math.Float32bits(buf[i])
		b.BigEndian.PutUint32(vpkt.DataBytes[i*8:], realu32)
		b.BigEndian.PutUint32(vpkt.DataBytes[(i*8)+4:], realu32)

	}
	vpkt.DataBytes = vpkt.DataBytes[:nSamp*8]
	return nSamp
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

const SL_VITA_INFO_CLASS uint32 = 0x534C
const SL_CLASS_SAMPLING_24KHZ uint32 = 0x03
const SL_CLASS_32BPS uint32 = (3 << 5)
const SL_CLASS_AUDIO_STEREO uint32 = (0x3 << 7)
const SL_CLASS_IEEE_754 uint32 = (0x1 << 9)
const SL_VITA_SLICE_AUDIO_CLASS uint32 = (SL_VITA_INFO_CLASS << 16) | SL_CLASS_SAMPLING_24KHZ | SL_CLASS_32BPS | SL_CLASS_AUDIO_STEREO | SL_CLASS_IEEE_754
