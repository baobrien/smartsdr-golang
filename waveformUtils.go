package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	st "strings"
	"time"
)

func RegisterWaveform(api *SmartAPIInterface, cfgFile *os.File) error {
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
	//minVersString := ""
	for {
		line, err := fileReader.ReadString('\n')
		if err != nil {
			return errors.New("Hit end of file without finding minimum-version")
		}
		line = st.Trim(line, " \n\r")
		if st.HasPrefix(st.ToLower(line), "minimum-smartsdr-version:") {
			toks := st.Split(line, " ")
			if len(toks) >= 2 {
				//minVersString = toks[1]
				break
			}
		}
	}

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
		api.SendCommand(line, func(a string, b uint32) { fmt.Printf("%x/%s:%s\n", b, a, line) }, time.Second*5)
	}

	return nil
}
