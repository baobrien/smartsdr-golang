/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady OBrien. All Rights Reserved.
 */

package main

import (
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

	/* Discover a radio */
	radio, err := DiscoverRadio(10 * time.Second)
	if err != nil {
		topError(err)
	}

	fmt.Println("Found radio:", radio)

	/* Connect to radio and start API interface */
	conn, err := net.Dial("tcp", radio.ip+":4992")
	if err != nil {
		topError(err)
	}
	api, err := InitAPIInterface(conn)
	time.Sleep(1 * time.Second)
	go api.InterfaceLoop()
	go api.PingLoop()
	/* Simple loop to print API errors */
	go func() {
		for {
			err := <-api.errs
			fmt.Println(err)
		}
	}()
	/* Register status handler to print all status messages */
	api.RegisterStatusHandler("", func(handle uint32, status string) {
		fmt.Println(status)
	})

	/* Open waveform configuration file and configure waveform */
	fmt.Println("Setting up Waveform:")
	configFile, err := os.Open("FreeDV.cfg")
	if err != nil {
		topError(err)
	}
	defer configFile.Close()
	err = RegisterWaveform(api, configFile)
	if err != nil {
		topError(err)
	}
	time.Sleep(time.Second * 100)
	os.Exit(0)
}
