package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type FlexVersion struct {
	Major int
	Minor int
	DevA  int
	DevB  int
}

type CmdResponse struct {
	RespStr string
	Status  uint32
}

type InflightCmd struct {
	Seq         uint32
	CommandText string
	RespChan    chan *CmdResponse
}

type CommandHandler func([]string) (string, uint32)

type StatusHandler func(uint32, string)

type StatusHandlerLink struct {
	prefix  string
	handler StatusHandler
}

type SmartAPIInterface struct {
	Handle         uint32
	Version        FlexVersion
	CmdSeq         uint32
	InflightCmds   map[uint32]*InflightCmd
	TcpConn        net.Conn
	quit           chan int
	pingQuit       chan int
	errs           chan error
	cmdSend        chan *InflightCmd
	cmdHandlers    map[string]CommandHandler
	statusHandlers []StatusHandlerLink
}

func (vers *FlexVersion) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", vers.Major, vers.Minor, vers.DevA, vers.DevB)
}

func InitAPIInterface(connection net.Conn) (*SmartAPIInterface, error) {
	iface := &SmartAPIInterface{
		InflightCmds:   make(map[uint32]*InflightCmd),
		TcpConn:        connection,
		errs:           make(chan error, 1),
		quit:           make(chan int),
		cmdSend:        make(chan *InflightCmd, 1),
		CmdSeq:         10,
		cmdHandlers:    make(map[string]CommandHandler),
		statusHandlers: make([]StatusHandlerLink, 0),
	}
	return iface, nil
}

func (tcpi *SmartAPIInterface) SendCommand(command string, callback func(string, uint32), timeout time.Duration) {
	cmd := &InflightCmd{
		Seq:         0,
		CommandText: command,
		RespChan:    make(chan *CmdResponse),
	}
	select {
	case tcpi.cmdSend <- cmd:
	default:
		return
	}
	go func() {
		select {
		case <-time.After(timeout):
			return
		case resp := <-cmd.RespChan:
			callback(resp.RespStr, resp.Status)
		}
	}()
}

func (tcpi *SmartAPIInterface) DoCommand(command string, timeout time.Duration) (string, uint32, error) {
	cmd := &InflightCmd{
		Seq:         0,
		CommandText: command,
		RespChan:    make(chan *CmdResponse),
	}
	select {
	case tcpi.cmdSend <- cmd:
	default:
		return "", 0, errors.New("DoCommand: API loop not running")
	}
	select {
	case <-time.After(timeout):
		return "", 0, errors.New("DoCommand: Timeout Reached")
	case resp := <-cmd.RespChan:
		return resp.RespStr, resp.Status, nil
	}
}

func (tcpi *SmartAPIInterface) handleCommand(cmdStr string) {
	cmdSegs := strings.Split(cmdStr, "|")
	if len(cmdSegs) >= 2 {
		cmdSeq, err := strconv.Atoi(cmdSegs[0])
		if err != nil {
			return
		}
		fullCmd := cmdSegs[1]
		var respVal uint32
		var respStr string
		argv := strings.Split(fullCmd, " ")
		if len(argv) >= 1 {
			cmdIdx := argv[0]
			cmdHandler := tcpi.cmdHandlers[cmdIdx]
			if cmdHandler != nil {
				respStr, respVal = cmdHandler(argv)
			} else {
				respStr = ""
				respVal = 0x50000015
			}
			respWire := fmt.Sprintf("R%d|%x|%s\n", cmdSeq, respVal, respStr)
			n, err := io.WriteString(tcpi.TcpConn, respWire)
			if n == 0 {
				tcpi.errs <- errors.New("TCP Socket Closed")
				return
			}
			if err != nil {
				tcpi.errs <- err
				return
			}
		}
	}
}

func (tcpi *SmartAPIInterface) handleStatus(status string) {
	statSeg := strings.Split(status, "|")
	if len(statSeg) >= 2 {
		if len(statSeg[1]) <= 0 {
			return
		}
		statStr := statSeg[1]
		idHandler, err := strconv.ParseUint(statSeg[0], 16, 32)
		if err != nil {
			return
		}
		for _, handlerLink := range tcpi.statusHandlers {
			if strings.HasPrefix(statStr, handlerLink.prefix) {
				handlerLink.handler(uint32(idHandler), statStr)
			}
		}
	}
}

func (tcpi *SmartAPIInterface) RegisterCommandHandler(cmd string, handler CommandHandler) {
	tcpi.cmdHandlers[cmd] = handler
}

func (tcpi *SmartAPIInterface) RegisterStatusHandler(prefix string, handler StatusHandler) {
	handlen := len(tcpi.statusHandlers)
	newHandlers := make([]StatusHandlerLink, handlen+1)
	copy(newHandlers, tcpi.statusHandlers)
	newHandlers[handlen].handler = handler
	newHandlers[handlen].prefix = prefix
	tcpi.statusHandlers = newHandlers
}

func (tcpi *SmartAPIInterface) InterfaceLoop() {
	lineChan := make(chan string)
	tcpErr := make(chan error)
	go func() {
		reader := bufio.NewReader(tcpi.TcpConn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				tcpErr <- err
				return
			}
			lineChan <- line[:len(line)-1]
		}
	}()

	for {
		select {
		case err := <-tcpErr:
			tcpi.errs <- err
			return
		case <-tcpi.quit:
			return
		case cmd := <-tcpi.cmdSend:
			tcpi.CmdSeq++
			seq := tcpi.CmdSeq
			cmdWire := fmt.Sprintf("C%d|%s\n", seq, cmd.CommandText)
			n, err := io.WriteString(tcpi.TcpConn, cmdWire)
			if n == 0 {
				tcpi.errs <- errors.New("TCP Socket Closed")
				return
			}
			if err != nil {
				tcpi.errs <- err
				return
			}
			tcpi.InflightCmds[seq] = cmd
		case line := <-lineChan:
			rdchar := line[0]
			switch rdchar {
			//Parse version string
			case 'V':
				vsegs := strings.Split(line[1:], ".")
				if len(vsegs) >= 4 {
					tcpi.Version.Major, _ = strconv.Atoi(vsegs[0])
					tcpi.Version.Minor, _ = strconv.Atoi(vsegs[1])
					tcpi.Version.DevA, _ = strconv.Atoi(vsegs[2])
					tcpi.Version.DevB, _ = strconv.Atoi(vsegs[3])
				}

			case 'H':
				handle, err := strconv.ParseUint(line[1:], 16, 32)
				if err == nil {
					tcpi.Handle = uint32(handle)
				}
			case 'R':
				respsegs := strings.Split(line[1:], "|")
				if len(respsegs) >= 2 {
					respStr := ""
					respSeq, _ := strconv.Atoi(respsegs[0])
					respVal, _ := strconv.ParseUint(respsegs[1], 16, 32)
					if len(respsegs) >= 3 {
						respStr = respsegs[2]
					}
					cmd := tcpi.InflightCmds[uint32(respSeq)]
					if cmd != nil {
						delete(tcpi.InflightCmds, uint32(respSeq))
						resp := &CmdResponse{respStr, uint32(respVal)}
						select {
						case cmd.RespChan <- resp:
						default:
						}
					}
				}
			case 'C':
				tcpi.handleCommand(line[1:])
			case 'S':
				tcpi.handleStatus(line[1:])
			}
		}
	}

}

func (api *SmartAPIInterface) Close() {
	select {
	case api.quit <- 1:
	default:
	}
	select {
	case api.pingQuit <- 1:
	default:
	}
	return
}

// Loop to send ping commands to radio
func (api *SmartAPIInterface) PingLoop() {
	start := time.Now()
	for {
		elapsed := time.Since(start)
		cmd := fmt.Sprintf("ping ms_timestamp=%f", float32(elapsed/time.Microsecond)/1000)
		api.SendCommand(cmd, func(a string, b uint32) {}, time.Millisecond*100)
		select {
		case <-api.pingQuit:
			break
		default:
		}
		time.Sleep(time.Second * 2)
	}
}
