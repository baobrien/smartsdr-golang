/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady O'Brien. All Rights Reserved.
 */

package main

// #cgo LDFLAGS: -lcodec2
// #include <codec2/freedv_api.h>
// typedef struct freedv freedv_s;
import "C"

type Freedv struct {
	fdv *C.freedv_s
}
