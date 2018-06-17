/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady OBrien. All Rights Reserved.
 */
package main

import (
	"os"

	"golang.org/x/sys/unix"
)

/* Ugly hack to get a UDP listener socket with SO_REUSEADDR bound */
func discoveryGetUDPListener() (*os.File, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	if err != nil {
		return nil, err
	}

	address := &unix.SockaddrInet4{}
	address.Port = 4992
	address.Addr[0] = 0
	address.Addr[1] = 0
	address.Addr[2] = 0
	address.Addr[3] = 0

	err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		return nil, err
	}

	if err = unix.Bind(fd, address); err != nil {
		return nil, err
	}

	if err = unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), ""), nil
}
