/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady O'Brien. All Rights Reserved.
 */

package main

// #cgo LDFLAGS: -lcodec2
// #include <codec2/freedv_api.h>
// typedef struct freedv freedv_s;
import "C"
import "errors"

type Freedv struct {
	fdv *C.freedv_s
}

type FreedvMode int

const (
	FREEDV_MODE_1600  FreedvMode = 0
	FREEDV_MODE_700              = 1
	FREEDV_MODE_700B             = 2
	FREEDV_MODE_2400A            = 3
	FREEDV_MODE_2400B            = 4
	FREEDV_MODE_800XA            = 5
	FREEDV_MODE_700C             = 6
	FREEDV_MODE_700D             = 7
)

func FreedvOpen(mode FreedvMode) (*Freedv, error) {
	fdv := C.freedv_open(C.int(mode))
	if fdv == nil {
		return nil, errors.New("Something went wrong opening FreeDV")
	}
	sfdv := &Freedv{fdv: fdv}
	return sfdv, nil
}

func (fdv *Freedv) Close() error {
	if fdv == nil {
		return nil
	}
	if fdv.fdv == nil {
		return nil
	}
	C.freedv_close(fdv.fdv)
	return nil
}

func (fdv *Freedv) Nin() int {
	nin := C.freedv_nin(fdv.fdv)
	return int(nin)
}

func (fdv *Freedv) GetSampleRate() int {
	rs := C.freedv_get_modem_sample_rate(fdv.fdv)
	return int(rs)
}

func (fdv *Freedv) GetMaxModemSamps() int {
	mms := C.freedv_get_n_max_modem_samples(fdv.fdv)
	return int(mms)
}

func (fdv *Freedv) GetNomModemSamps() int {
	nms := C.freedv_get_n_nom_modem_samples(fdv.fdv)
	return int(nms)
}

func (fdv *Freedv) GetNSpeechSamples() int {
	nss := C.freedv_get_n_speech_samples(fdv.fdv)
	return int(nss)
}

func (fdv *Freedv) GetSync() bool {
	sync := C.freedv_get_sync(fdv.fdv)
	return int(sync) > 0
}

func (fdv *Freedv) RxFloat(rxsamp []float32, rxspeech []int16) int {
	nin := fdv.Nin()
	nss := fdv.GetNSpeechSamples()
	if len(rxsamp) < nin {
		return 0
	}
	if len(rxspeech) < nss {
		return 0
	}
	nout := C.freedv_floatrx(fdv.fdv, (*C.short)(&rxspeech[0]), (*C.float)(&rxsamp[0]))
	return int(nout)
}
