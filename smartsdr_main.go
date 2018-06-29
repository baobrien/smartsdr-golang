/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady O'Brien. All Rights Reserved.
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

func StartVitaEchoer(vif *VitaInterface) {
	ch1 := make(chan []float32, 2)
	ch2 := make(chan []float32, 2)
	vif.Subscribers[0x81000000] = StVitaInputF(ch1)
	go StAccumulatorF(ch1, ch2, 1000)
	templateHeader := &VitaIfDataHeader{
		StreamID:       0x81000000,
		ClassIDH:       0x00001C2D,
		ClassIDL:       SL_VITA_SLICE_AUDIO_CLASS,
		TimestampFracH: 0,
		TimestampFracL: 0,
		TimestampInt:   0,
	}
	go StVitaOutputF(ch1, vif, templateHeader)

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
	go api.PingLoop(time.Second * 10)
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

	/* Set up VITA stream handler */
	connVitaLocal, err := net.ResolveUDPAddr("udp", "0.0.0.0:4999")
	if err != nil {
		topError(err)
	}
	connVitaRadio, err := net.ResolveUDPAddr("udp", radio.ip+":4991")
	if err != nil {
		topError(err)
	}

	vitaListener, err := InitVitaListener(connVitaLocal, connVitaRadio)
	if err != nil {
		topError(err)
	}

	/*vitaListener.Subscribers[0x81000000] = func(pkt *VitaIFData, pool *VitaBufferPool) {
		fmt.Println("Got VITA49 Packet. Samples: ", len(pkt.DataBytes)/8)
		pool.releasePB(pkt.RawPacketBuffer, pkt)
	}*/

	StartVitaEchoer(vitaListener)
	go func() {
		serr := vitaListener.VitaListenLoop()
		if serr != nil {
			fmt.Println("Error:", serr)
		}
	}()

	go vitaListener.VitaSenderLoop()

	time.Sleep(time.Second * 100)
	os.Exit(0)
}
