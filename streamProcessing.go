/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady O'Brien. All Rights Reserved.
 */

package main

/*
 * StVitaInputF creates a stream subscriber function. The returned function
 * takes a VITA packet, extracts float samples, and shoves it down the
 * channel
 */
func StVitaInputF(outputChan chan []float32) StreamSubscriber {
	return func(pkt *VitaIFData, pool *VitaBufferPool) {
		samps := VitaToFloat(pkt)
		pool.releasePB(pkt.RawPacketBuffer, pkt)
		outputChan <- samps
	}
}

/*
 * Creates a function meant to be run in a goroutine which waits for input
 * buffers on InputChan, packs a frame with them, and sends it on it's way
 * into the VitaInterface
 */
func StVitaOutputF(inputChan chan []float32, vif *VitaInterface, headerPrototype *VitaIfDataHeader) {
	for {
		/* Nil buffer signals quit */
		bufIn := <-inputChan
		if bufIn == nil {
			break
		}
		n := 0
		for n < len(bufIn) {
			/* Grab a packet and buffer */
			buf, pkt := vif.BufBag.grabPB()
			pkt.RawPacketBuffer = buf
			pkt.DataBytes = buf
			/* Copy prototype header data in */
			pkt.Header = *headerPrototype

			n += FloatToVitaFrame(pkt, bufIn)

			vif.SendChannel <- pkt
		}
	}

}

/*
 * Create a stream processor function which accumulates some number of samples before sending a buffer off
 */
func StAccumulatorF(inputChan chan []float32, outputChan chan []float32, naccum int) {
	accumulator := make([]float32, naccum)
	nInBuf := 0
	for {
		bufIn := <-inputChan
		if bufIn == nil {
			outputChan <- nil
			break
		}
		for len(bufIn) > 0 {
			n := copy(accumulator[nInBuf:], bufIn[:])
			nInBuf += n
			if nInBuf == naccum {
				outputChan <- accumulator
				accumulator = make([]float32, naccum)
				nInBuf = 0
			}
			bufIn = bufIn[n:]
		}
	}
}
