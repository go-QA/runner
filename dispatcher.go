package runner

import (
	"fmt"
	//"error"
	//"log"
	//"os"
	//"io"
	"github.com/go-QA/logger"
	//"../logger"
	//"runtime"
	"time"
)

const (
	MAX_WAIT_MATCH_RETURN = 3000 // Maximum time in milliseconds to wait for build return message
)

type RunId int

func (ri RunId) String() string {
	return fmt.Sprintf("%d", ri)
}

type RunInfo struct {
	Id         RunId
	Name       string
	LaunchType string
}

type BuildInfo struct {
	Version       string
	Project       string
	Path          string
	BuildComplete time.Time
}

type Matcher interface {
	FindMatches(buildInfo BuildInfo) []RunInfo
}

type BuildMockMatch struct {
	runNum RunId
}

func (mock *BuildMockMatch) FindMatches(buildInfo BuildInfo) []RunInfo {
	return []RunInfo{
		{Id: mock.runNum,
			Name:       fmt.Sprintf("Runplan_%d_%s", mock.runNum, buildInfo.Version),
			LaunchType: "auto"},
		{Id: mock.runNum + 1,
			Name:       fmt.Sprintf("Runplan_%d_%s", mock.runNum+1, buildInfo.Version),
			LaunchType: "manual"},
		{Id: mock.runNum + 2,
			Name:       fmt.Sprintf("Runplan_%d_%s", mock.runNum+2, buildInfo.Version),
			LaunchType: "auto"},
	}
}

type InternalBuildMatcher struct {
	m_log           *logger.GoQALog
	matcher         Matcher
	chnBuilds       chan InternalCommandInfo
	chnRunplans     *CommandQueue
	chnExit         chan int
	isStopRequested bool
}

func (ibm *InternalBuildMatcher) GetBuildInfo(info InternalCommandInfo) (BuildInfo, error) {
	var buildInfo BuildInfo
	var err error
	if info.Command == CMD_NEW_BUILD {
		buildInfo = BuildInfo{Version: info.Data[0].(string),
			Project: info.Data[1].(string),
			Path:    info.Data[2].(string)}
	} else {
		buildInfo = BuildInfo{}
		err = &myError{mes: "No build info"}
	}
	return buildInfo, err
}

func (ibm *InternalBuildMatcher) CreatRunInfoMes(cmd *InternalCommandInfo, run RunInfo) {
	//cmd := new(InternalCommandInfo)
	cmd.Command = CMD_LAUNCH_RUN
	cmd.ChnReturn = make(chan CommandInfo)
	cmd.Data = []interface{}{run.Id, run.Name, run.LaunchType}
	return
}

func (ibm *InternalBuildMatcher) Init(iMatch Matcher, inChn chan InternalCommandInfo, outChn *CommandQueue, chnExit chan int, log *logger.GoQALog) {
	ibm.matcher = iMatch
	ibm.chnBuilds = inChn
	ibm.chnRunplans = outChn
	ibm.chnExit = chnExit
	ibm.m_log = log
	ibm.isStopRequested = false
}

func (ibm *InternalBuildMatcher) Stop(mes int) bool {
	return true
}

func (ibm *InternalBuildMatcher) OnMessageRecieved(nextMessage InternalCommandInfo) {
	var outMes *InternalCommandInfo
	nextBuild, err := ibm.GetBuildInfo(nextMessage)
	if err == nil {
		newRunplans := ibm.matcher.FindMatches(nextBuild)
		if newRunplans != nil {
			for _, run := range newRunplans {
				outMes = new(InternalCommandInfo)
				//ibm.m_log.LogMessage("BuildMatcher mesOut = %p", outMes)
				ibm.CreatRunInfoMes(outMes, run)
				*ibm.chnRunplans <- *outMes
				go func(mes *InternalCommandInfo) {
					select {
					case resv := <-(*mes).ChnReturn:
						ibm.m_log.LogMessage("BuildMatcher resv = %s %s %p", CmdName(resv.Command), resv.Data[0].(string), mes)
					case <-time.After(time.Millisecond * MAX_WAIT_MATCH_RETURN):
						ibm.m_log.LogMessage("BuildMatcher Timed out  %v %p", mes, mes)
					}
				}(outMes)
			}
			nextMessage.ChnReturn <- GetMessageInfo(CMD_OK, fmt.Sprintf("launched %d runs", len(newRunplans)))
		} else {
			nextMessage.ChnReturn <- GetMessageInfo(CMD_OK, "no runs matched")
		}
	} else {
		ibm.m_log.LogError("GetBuildInfo::%s", err)
		nextMessage.ChnReturn <- GetMessageInfo(CMD_OK, "Build match err", err.Error())
	}
}

func (ibm *InternalBuildMatcher) Run() {

	ibm.isStopRequested = false

	for ibm.isStopRequested == false {
		select {
		case nextMessage := <-ibm.chnBuilds:
			go ibm.OnMessageRecieved(nextMessage)
		case exitMessage := <-ibm.chnExit:
			ibm.isStopRequested = ibm.Stop(exitMessage)
		case <-time.After(time.Millisecond * LOOP_WAIT_TIMER):
		}
		ibm.onProcessEvents()
	}
	ibm.m_log.LogDebug("Out of Main loop")
}

func (ibm *InternalBuildMatcher) onProcessEvents() {
	ibm.m_log.LogDebug("Matcher Process Events")
}
