/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady O'Brien. All Rights Reserved.
 */

package main

const scaleShort = float32(8000)

func StFreedvRxF(inputChan, outputChan chan []float32, fdv *Freedv) {
	nMax := fdv.GetMaxModemSamps()
	nin := fdv.Nin()
	accumulator := make([]float32, nMax)
	speechS := make([]int16, fdv.GetNSpeechSamples())
	nInBuf := 0
	for {
		bufIn := <-inputChan
		if bufIn == nil {
			outputChan <- nil
			break
		}
		for len(bufIn) > 0 {
			n := copy(accumulator[nInBuf:nin], bufIn[:])
			nInBuf += n
			if nInBuf == nin {
				nout := fdv.RxFloat(accumulator, speechS)
				speech := make([]float32, nout)
				for i := 0; i < nout; i++ {
					speech[i] = float32(speechS[i]) / scaleShort
				}
				nin = fdv.Nin()
				outputChan <- speech
				nInBuf = 0
			}
			bufIn = bufIn[n:]
		}
	}
}
