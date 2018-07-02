/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady O'Brien. All Rights Reserved.
 */

package main

import (
	"fmt"
	"time"
)

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
			bufSend := bufIn[n:]
			/* Grab a packet and buffer */
			buf, pkt := vif.BufBag.grabPB()
			pkt.RawPacketBuffer = buf
			pkt.DataBytes = buf
			/* Copy prototype header data in */
			pkt.Header = *headerPrototype

			n += FloatToVitaFrame(pkt, bufSend)
			vif.SendChannel <- pkt
		}
	}

}

/*
 * Create a stream processor function which accumulates some number of samples before sending a buffer off
 */
func StAccumulatorF(inputChan, outputChan chan []float32, naccum int) {
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

/* */
func SampCtrF(inputChan, outputChan chan []float32, name string, interval time.Duration) uint64 {
	sampCount := uint64(0)
	buf := <-inputChan
	start := time.Now()
	lastPrint := time.Now()
	for {
		if buf == nil {
			outputChan <- nil
			break
		}
		sampCount += uint64(len(buf))
		outputChan <- buf

		if time.Since(lastPrint) > interval && interval > 0 {
			sF := float64(sampCount)
			tF := float64(time.Since(start)) / float64(time.Second)
			rate := sF / tF
			fmt.Printf("%s: %v samples, %f samples/s\n", name, sampCount, rate)
			lastPrint = time.Now()
			sampCount = 0
			start = lastPrint
		}
		buf = <-inputChan
	}
	return sampCount
}

func StDelatentizerF(i1, o1, i2, o2 chan []float32, maxdisp int) {
	dispCnt := make(chan int, 10)

	// Input Track Process
	go func() {
		for {
			buf := <-i1
			if buf == nil {
				o1 <- nil
				break
			}
			dispCnt <- len(buf)
			o1 <- buf
		}
	}()

	// Output Track Process
	go func() {
		rundisp := 0
		last := time.Now()
		for {
			select {
			case rsc := <-dispCnt:
				rundisp += rsc
				if rundisp > maxdisp {
					fmt.Printf("Correcting RD by %d\n", rundisp)
					o2 <- make([]float32, rundisp)
					rundisp = 0
				}
			case buf := <-i2:
				if buf == nil {
					o2 <- nil
					break
				}
				rundisp -= len(buf)
				o2 <- buf
			}
			if time.Since(last) > time.Second {
				fmt.Printf("RD is %d\n", rundisp)
				last = time.Now()
			}
		}
	}()
}
