/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady OBrien. All Rights Reserved.
 *
 * 24Khz to 8Khz filters
 */

package main

const rs_ratio = 3

/* *****************************************************************************
 *	Coefficients for 4 kHz low pass filter at 24 ksps for 48 tap FIR filter
 *
 * 	Generated using fir1(47, 1/3) in MatLab (Octave)
 */

var fdmdv_os24_filter = [...]float32{
	-0.000565842330864509,
	-0.00119184233667459,
	-0.000686550128357081,
	0.000939738560355487,
	0.00235824811185176,
	0.00149083509882116,
	-0.00207002114214581,
	-0.00516284617910486,
	-0.00318858060009128,
	0.00422846062091092,
	0.0102371199934064,
	0.00615820273645780,
	-0.00786127965697296,
	-0.0187253107816201,
	-0.0111560475540299,
	0.0139752338625282,
	0.0334879967920482,
	0.0202917237268834,
	-0.0258029481868858,
	-0.0651503052036609,
	-0.0430343789277145,
	0.0624453219256916,
	0.210663786004670,
	0.318319285594497,
	0.318319285594497,
	0.210663786004670,
	0.0624453219256916,
	-0.0430343789277145,
	-0.0651503052036609,
	-0.0258029481868858,
	0.0202917237268834,
	0.0334879967920482,
	0.0139752338625282,
	-0.0111560475540299,
	-0.0187253107816201,
	-0.00786127965697296,
	0.00615820273645780,
	0.0102371199934064,
	0.00422846062091092,
	-0.00318858060009128,
	-0.00516284617910486,
	-0.00207002114214581,
	0.00149083509882116,
	0.00235824811185176,
	0.000939738560355487,
	-0.000686550128357081,
	-0.00119184233667459,
	-0.000565842330864509,
}

//func Resample8to24F(out24k, in8k []float32)

/*void fdmdv_24_to_8(float out8k[], float in24k[], int n)
{
    int i,j;

    for(i=0; i<n; i++)
    {
		out8k[i] = 0.0;
		for(j=0; j<FDMDV_OS_TAPS; j++)
	    	out8k[i] += fdmdv_os24_filter[j]*in24k[i*FDMDV_OS-j];
    }

    // update filter memory
    for(i=-FDMDV_OS_TAPS; i<0; i++)
	in24k[i] = in24k[i + n*FDMDV_OS];
}*/

func Resample24to8F(out8k, in24k []float32, n int) {
	for i := 0; i < n; i++ {
		out8k[i] = 0
		for j, tap := range fdmdv_os24_filter {
			out8k[i] += tap * in24k[i*rs_ratio-j]
		}
	}

}

/* TODO: Make pluggable with different filter taps and resampling rates */
func StResamp24to8F(inputChan chan []float32, outputChan chan []float32, naccum int) {
	// Round naccum to next smallest 3rd, since this is 3x downsampling filter
	taps := fdmdv_os24_filter
	Rs := rs_ratio

	naccum -= naccum % Rs
	nTaps := len(taps)
	filtMem := make([]float32, naccum+nTaps)
	nInBuf := 0
	for {
		// get an input buffer
		bufIn := <-inputChan
		if bufIn == nil {
			outputChan <- nil
			break
		}
		for len(bufIn) > 0 {
			n := copy(filtMem[nTaps+nInBuf:], bufIn[:])
			nInBuf += n
			/* Filter and send out when we get enough data */
			if nInBuf == naccum {
				outBuf := make([]float32, naccum/3)

				/* Actually run the filter */
				for i := range outBuf {
					v := float32(0)
					for j, tap := range taps {
						v += tap * filtMem[(i*rs_ratio)-j+nTaps]
					}
					outBuf[i] = v
				}
				outputChan <- outBuf
				/* Update filter memory */
				copy(filtMem[:], filtMem[len(filtMem)-nTaps:])
				nInBuf = 0
			}
			bufIn = bufIn[n:]
		}
	}
}

/*
void fdmdv_8_to_24(float out24k[], float in8k[], int n)
{
    int i,j,k,l;

    // make sure n is an integer multiple of the oversampling rate, otherwise
	// this function breaks

    // assert((n % FDMDV_OS) == 0);

    for(i=0; i<n; i++)
    {
		for(j=0; j<FDMDV_OS; j++)
		{
	    	out24k[i*FDMDV_OS+j] = 0.0;
	    	for(k=0,l=0; k<FDMDV_OS_TAPS; k+=FDMDV_OS,l++)
				out24k[i*FDMDV_OS+j] += fdmdv_os24_filter[k+j]*in8k[i-l];
	    	out24k[i*FDMDV_OS+j] *= FDMDV_OS;
		}
    }

    // update filter memory
    for(i=-(FDMDV_OS_TAPS/FDMDV_OS); i<0; i++)
		in8k[i] = in8k[i + n];
}*/

/* TODO: Make pluggable with different filter taps and resampling rates */
func StResamp8to24F(inputChan chan []float32, outputChan chan []float32, naccum int) {
	// Round naccum to next smallest 3rd, since this is 3x downsampling filter
	taps := fdmdv_os24_filter
	Rs := rs_ratio

	naccum -= naccum % Rs
	nTaps := len(taps)
	nMem := nTaps / Rs
	filtMem := make([]float32, naccum+nMem)
	nInBuf := 0
	for {
		// get an input buffer
		bufIn := <-inputChan
		if bufIn == nil {
			outputChan <- nil
			break
		}
		for len(bufIn) > 0 {
			n := copy(filtMem[nMem+nInBuf:], bufIn[:])
			nInBuf += n
			/* Filter and send out when we get enough data */
			if nInBuf == naccum {
				outBuf := make([]float32, naccum*3)

				/* Actually run the filter */
				for i := 0; i < naccum; i++ {
					for j := 0; j < Rs; j++ {
						k := 0
						l := 0
						v := float32(0)
						for ; k < nTaps; k += Rs {
							v += taps[k+j] * filtMem[i-l+nMem]
							l++
						}
						outBuf[i*Rs+j] = v * float32(Rs)
					}
				}

				outputChan <- outBuf
				/* Update filter memory */
				copy(filtMem[:], filtMem[len(filtMem)-nMem:])
				nInBuf = 0
			}
			bufIn = bufIn[n:]
		}
	}
}
