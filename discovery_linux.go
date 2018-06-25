/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady O'Brien. All Rights Reserved.
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
	/* Set up an IPv4 or IPv6 sockaddr and socket for UDP */
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
	/* Set reuseaddr socket option so we can bind multiple times to the same port */
	err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	if err != nil {
		return nil, err
	}

	/* Bind the socket to the port*/
	if err = unix.Bind(fd, sockaddr); err != nil {
		return nil, err
	}

	/* Set socket nonblocking so go scheduler can work with it */
	if err = unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}

	/* Return go file around file descriptor */
	return os.NewFile(uintptr(fd), ""), nil
}
