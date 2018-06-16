/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady OBrien. All Rights Reserved.
 */

package main

import "strings"

func detokenize(tokenString string) map[string]string {
	tokenMap := make(map[string]string)
	for _, seg := range strings.Split(tokenString, " ") {
		parts := strings.Split(seg, "=")
		if len(parts) == 2 {
			tokenMap[parts[0]] = parts[1]
		}
	}
	return tokenMap
}
