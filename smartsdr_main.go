/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady OBrien. All Rights Reserved.
 */

package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

func topError(err error) {
	fmt.Printf("Error in main: %v\n", err)
	os.Exit(1)
}

func main() {
	/* Create and start the discovery client */

	addr, err := net.ResolveUDPAddr("udp", "[::]:4992")
	disClient, err := CreateDiscoveryClient(addr)
	if err != nil {
		topError(err)
	}

	go disClient.doDiscoveryListen()
	select {
	case radio := <-disClient.radios:
		fmt.Println("Found Radio:", radio)
		disClient.Close()
		conn, err := net.Dial("tcp", radio.ip+":4992")
		if err != nil {
			topError(err)
		}
		apiface, err := InitTcpInterface(conn)
		if err != nil {
			topError(err)
		}
		restr, restat, err := apiface.DoCommand("info", 10*time.Second)
		if err != nil {
			topError(err)
		}
		fmt.Printf("Command returned status %x\n", restat)
		fmt.Printf("%s\n", restr)
	case err = <-disClient.errors:
		topError(err)
	case <-time.After(time.Second * 30):
		disClient.Close()
		topError(errors.New("Failed to find client after 30 seconds"))
	}

	os.Exit(0)
}
