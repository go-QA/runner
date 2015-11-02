package main

import (
	"fmt"
	//"error"
	//"log"
	"os"
	//"io"
	"../../logger"
	"../../runner"
	//"github.com/go-QA/logger"
	//"github.com/go-QA/runner"
	"time"
	//"net"
	//"encoding/json"
)

const (
	CLENT_DELAY1 = 1000
	CLENT_DELAY2 = 1000	
	BUILDER_COUNT = 3
	MESSAGE_COUNT = 2
)

const (
	MES_ADDRESS = "localhost"
	MES_PORT = 1414
)

func GetStatusRun(log *logger.GoQALog) {
	sender := runner.MessageSender{}
	encoder := runner.JSON_Encode{}
	sender.Init("localhost", 1414, log, &encoder)
	for {
		time.Sleep(time.Millisecond * CLENT_DELAY1)
		rsvMessage, err := sender.Send(runner.CMD_GET_RUN_STATUS, "ALL")
		if err == nil {
			log.LogDebug("CLIENT RCV: %v\n", rsvMessage)
		} else {
			log.LogDebug("CLIENT RCV ERROR::: ", err)
		}
		
	}
}

func BuildRun(chnBuild chan runner.InternalCommandInfo, log *logger.GoQALog) {
	var buildInfo runner.InternalCommandInfo
	verNum := 1
	for {		
		time.Sleep(time.Millisecond * CLENT_DELAY1)
		buildInfo = runner.GetInternalMessageInfo(runner.CMD_NEW_BUILD, make(chan runner.CommandInfo), fmt.Sprintf("T1.0_%d", verNum), "Fun", "~/projects/fun", time.Now().String())
		verNum++
		chnBuild <- buildInfo
		go func(chnRet chan runner.CommandInfo) {
			ret := <- chnRet
			log.LogMessage("ClientRun::%s %s", runner.CmdName(ret.Command), ret.Data[0])
			}(buildInfo.ChnReturn)
	}
}


func main() {

	chnExit := make(chan int)

	commandQueue := make(runner.CommandQueue, 100)
	log := logger.GoQALog{}
	log.Init()
	log.Add("default", logger.LOGLEVEL_ALL, os.Stdout)
	//log.SetDebug(true)

	messageListener := runner.TCPConnector{}
	messageListener.Init(&log, &runner.JSON_Encode{}, MES_ADDRESS, MES_PORT)
	listener := runner.ExternalConnector{}
	listener.Init(&messageListener, &commandQueue, chnExit, &log)

	MockMatch := runner.BuildMockMatch{}
	chnBuildIn := make(chan runner.InternalCommandInfo)
	BuildMatcher := runner.InternalBuildMatcher{}
	BuildMatcher.Init(&MockMatch, chnBuildIn, &commandQueue, chnExit, &log)

	master := runner.Master{}
	master.Init(&listener, &commandQueue, chnExit, &log)
	//time.Sleep(time.Second * 1)

	//go runner.RunListener(&listener, &commandQueue, &chnExit)

	go BuildMatcher.Run()
	//time.Sleep(time.Second * 1)
	for i := 0; i < BUILDER_COUNT; i++ {
		go BuildRun(chnBuildIn, &log)
		time.Sleep(time.Millisecond * 200)
	}
	for i := 0; i < MESSAGE_COUNT; i++ {
		go GetStatusRun(&log)
		time.Sleep(time.Millisecond * 100)
	}

	log.LogMessage("Running master...")
	master.Run()

	log.LogMessage("Leaving Program....")
	time.Sleep(time.Second * 1)
}
