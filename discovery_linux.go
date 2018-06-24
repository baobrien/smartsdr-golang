/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady OBrien. All Rights Reserved.
 */
package main

import (
	"net"
	"os"

	"golang.org/x/sys/unix"
)

/* Ugly hack to get a UDP listener socket with SO_REUSEADDR bound */
func discoveryGetUDPListener(addr *net.UDPAddr) (*os.File, error) {

	var fd int
	var err error
	var sockaddr unix.Sockaddr
	ip := addr.IP
	if v4 := ip.To4(); v4 != nil {
		sockaddr4 := &unix.SockaddrInet4{}
		sockaddr4.Port = addr.Port
		copy(sockaddr4.Addr[:], v4[:])
		fd, err = unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
		if err != nil {
			return nil, err
		}
		sockaddr = sockaddr4
	} else {
		sockaddr6 := &unix.SockaddrInet6{}
		sockaddr6.Port = addr.Port
		copy(sockaddr6.Addr[:], ip[:])
		fd, err = unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
		if err != nil {
			return nil, err
		}
		sockaddr = sockaddr6
	}
	err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		return nil, err
	}

	if err = unix.Bind(fd, sockaddr); err != nil {
		return nil, err
	}

	if err = unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), ""), nil
}
