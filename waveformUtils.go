/* SPDX-License-Identifier: GPL-3.0
 *
 * Copyright (C) 2018 Brady O'Brien. All Rights Reserved.
 */

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	st "strings"
	"time"
)

func RegisterWaveform(api *SmartAPIInterface, cfgFile io.Reader) error {
	fileReader := bufio.NewReader(cfgFile)
	// Find Header section
	for {
		line, err := fileReader.ReadString('\n')
		if err != nil {
			return errors.New("Hit end of file without finding [header]")
		}
		line = st.Trim(line, " \n\r")
		if st.HasPrefix(st.ToLower(line), "[header]") {
			break
		}
	}
	// Find minimum version. Ignore for now, since C implementation ignores
	minVersString := ""
	for {
		line, err := fileReader.ReadString('\n')
		if err != nil {
			return errors.New("Hit end of file without finding minimum-version")
		}
		line = st.Trim(line, " \n\r")
		if st.HasPrefix(st.ToLower(line), "minimum-smartsdr-version:") {
			toks := st.Split(line, " ")
			if len(toks) >= 2 {
				minVersString = toks[1]
				break
			}
		}
	}
	fmt.Printf("Minimum version: %s\n", minVersString)
	// Find setup section
	for {
		line, err := fileReader.ReadString('\n')
		if err != nil {
			return errors.New("Hit end of file without finding [setup]")
		}
		line = st.Trim(line, " \n\r")
		if st.HasPrefix(st.ToLower(line), "[setup]") {
			break
		}
	}

	// Relay setup commands to radio
	for {
		line, err := fileReader.ReadString('\n')
		if err != nil {
			return errors.New("Hit end of file without finding [end]")
		}
		line = st.Trim(line, " \n\r")
		if st.HasPrefix(st.ToLower(line), "[end]") {
			break
		}
		if len(line) > 0 {
			a, b, err := api.DoCommand(line, time.Second*1)
			if err == nil {
				fmt.Printf("%x,%s:%s\n", b, a, line)
			}
		}
	}

	// Subscribe to slice info
	cmd := "sub slice all"
	api.SendCommand(cmd, func(a string, b uint32) {
		fmt.Printf("%x,%s:%s\n", b, a, cmd)
	}, time.Second*2)
	return nil
}
