/*
 * Copyright 2016 ThoughtWorks, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agent

import (
	"bytes"
	"github.com/gocd-contrib/gocd-golang-agent/protocol"
	"github.com/gocd-contrib/gocd-golang-agent/stream"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultSecretMask           = "********"
	DefaultCancelCommandTimeout = 25 * time.Second
)

var (
	CancelCommandTimeout   = DefaultCancelCommandTimeout
	CancelBuildTimeout     = 30 * time.Second
	BuildDebugToConsoleLog = true
)

type Executor func(session *BuildSession, cmd *protocol.BuildCommand) error

func Executors() map[string]Executor {
	return map[string]Executor{
		protocol.CommandExport:              CommandExport,
		protocol.CommandEcho:                CommandEcho,
		protocol.CommandSecret:              CommandSecret,
		protocol.CommandReportCurrentStatus: CommandReport,
		protocol.CommandReportCompleting:    CommandReport,
		protocol.CommandCompose:             CommandCompose,
		protocol.CommandTest:                CommandTest,
		protocol.CommandExec:                CommandExec,
		protocol.CommandMkdirs:              CommandMkdirs,
		protocol.CommandCleandir:            CommandCleandir,
		protocol.CommandUploadArtifact:      CommandUploadArtifact,
		protocol.CommandDownloadFile:        CommandDownloadArtifact,
		protocol.CommandDownloadDir:         CommandDownloadArtifact,
		protocol.CommandFail:                CommandFail,
		protocol.CommandGenerateTestReport:  CommandGenerateTestReport,
		protocol.CommandGenerateProperty:    NotImplemented,
	}
}

type BuildSession struct {
	send                  chan *protocol.Message
	console               io.WriteCloser
	artifacts             *Artifacts
	command               *protocol.BuildCommand
	artifactUploadBaseURL *url.URL

	envs    map[string]string
	cancel  chan bool
	done    chan bool
	echo    *stream.SubstituteWriter
	secrets *stream.SubstituteWriter

	buildId     string
	buildStatus string

	rootDir string
	wd      string

	executors map[string]Executor
}

func MakeBuildSession(buildId string,
	command *protocol.BuildCommand,
	console io.WriteCloser,
	artifacts *Artifacts,
	artifactUploadBaseURL *url.URL,
	send chan *protocol.Message,
	rootDir string) *BuildSession {

	secrets := stream.NewSubstituteWriter(console)
	return &BuildSession{
		buildId:               buildId,
		buildStatus:           protocol.BuildPassed,
		console:               console,
		artifacts:             artifacts,
		artifactUploadBaseURL: artifactUploadBaseURL,
		command:               command,
		send:                  send,
		envs:                  make(map[string]string),
		cancel:                make(chan bool),
		done:                  make(chan bool),
		secrets:               secrets,
		echo:                  stream.NewSubstituteWriter(secrets),
		rootDir:               rootDir,
		executors:             Executors(),
	}
}

func (s *BuildSession) Close() error {
	return closeAndWait(s.cancel, s.done, CancelBuildTimeout)
}

func (s *BuildSession) isCanceled() bool {
	if s.buildStatus == protocol.BuildCanceled {
		return true
	}
	if isClosedChan(s.cancel) {
		s.buildStatus = protocol.BuildCanceled
		return true
	} else {
		return false
	}
}

func (s *BuildSession) Run() error {
	defer func() {
		s.console.Close()
		s.send <- protocol.CompletedMessage(s.Report(""))
		LogInfo("Build completed")
	}()
	LogInfo("Build started, root directory: %v", s.rootDir)
	return s.ProcessCommand()
}

func (s *BuildSession) ProcessCommand() error {
	defer func() {
		close(s.done)
	}()

	return s.process(s.command)
}

func (s *BuildSession) process(cmd *protocol.BuildCommand) (err error) {
	defer s.onCancel(cmd)

	if s.isCanceled() {
		s.debugLog("build canceled, ignore %v", cmd.Name)
		return nil
	}

	if !cmd.RunIfAny() && !cmd.RunIfMatch(s.buildStatus) {
		s.debugLog("ignore %v: build[%v] != runIf[%v]", cmd.Name, s.buildStatus, cmd.RunIfConfig)
		//skip, no failure
		return nil
	}
	s.debugLog("process: %v", cmd.Name)
	if s.testFailed(cmd.Test) {
		return nil
	}

	err = s.doProcess(cmd)
	if s.isCanceled() {
		LogInfo("build canceled")
		s.buildStatus = protocol.BuildCanceled
	} else if err != nil && s.buildStatus != protocol.BuildFailed {
		s.buildStatus = protocol.BuildFailed
		errMsg := Sprintf("ERROR: %v\n", err)
		LogInfo(errMsg)
		s.ConsoleLog(errMsg)
	}

	return
}

func (s *BuildSession) doProcess(cmd *protocol.BuildCommand) error {
	s.wd = filepath.Clean(filepath.Join(s.rootDir, cmd.WorkingDirectory))
	s.debugLog("set wd to %v", s.wd)

	if !strings.HasPrefix(s.wd, s.rootDir) {
		return Err("Working directory[%v] is outside the agent sandbox.", s.wd)
	}
	_, err := os.Stat(s.wd)
	if err != nil {
		if os.IsNotExist(err) {
			return Err("Working directory \"%v\" is not a directory", s.wd)
		} else {
			return err
		}
	}

	exec := s.executors[cmd.Name]
	if exec == nil {
		return Err("Unknown build command: %v", cmd.Name)
	} else {
		return exec(s, cmd)
	}
}

func (s *BuildSession) testFailed(test *protocol.BuildCommand) bool {
	if test == nil {
		return false
	}
	s.debugLog("test: %+v", test)
	_, err := s.processTestCommand(test)
	if s.isCanceled() {
		s.debugLog("test is canceled due to build is canceled")
		return true
	}
	if err == nil {
		s.debugLog("test matches expectation")
		return false
	} else {
		s.debugLog("test failed with err: %v", err)
		return true
	}
}

func (s *BuildSession) onCancel(cmd *protocol.BuildCommand) {
	if cmd.OnCancel == nil || !s.isCanceled() {
		return
	}
	cancel := &BuildSession{
		buildId:               s.buildId,
		console:               s.console,
		artifacts:             s.artifacts,
		artifactUploadBaseURL: s.artifactUploadBaseURL,
		send:        s.send,
		envs:        s.envs,
		secrets:     s.secrets,
		echo:        s.echo,
		rootDir:     s.rootDir,
		executors:   s.executors,
		command:     cmd.OnCancel,
		buildStatus: protocol.BuildPassed,
		cancel:      make(chan bool),
		done:        make(chan bool),
	}
	go func() {
		cancel.ProcessCommand()
	}()
	select {
	case <-cancel.done:
	case <-time.After(CancelCommandTimeout):
		s.warn("Kill cancel task because it did not finish in %v.", CancelCommandTimeout)
		cancel.Close()
	}
}

func (s *BuildSession) processTestCommand(cmd *protocol.BuildCommand) (bytes.Buffer, error) {
	var output bytes.Buffer
	session := &BuildSession{
		buildId:               s.buildId,
		artifacts:             s.artifacts,
		artifactUploadBaseURL: s.artifactUploadBaseURL,
		send:        s.send,
		envs:        s.envs,
		secrets:     s.secrets.Filter(&output),
		echo:        s.echo.Filter(&output),
		rootDir:     s.rootDir,
		executors:   s.executors,
		console:     stream.NopCloser(&output),
		command:     cmd,
		buildStatus: protocol.BuildPassed,
		cancel:      s.cancel,
		done:        make(chan bool),
	}

	err := session.ProcessCommand()
	return output, err
}

func (s *BuildSession) Report(jobState string) *protocol.Report {
	return &protocol.Report{
		AgentRuntimeInfo: GetAgentRuntimeInfo(),
		BuildId:          s.buildId,
		JobState:         jobState,
		Result:           s.buildStatus,
	}
}

func (s *BuildSession) ConsoleLog(format string, a ...interface{}) {
	s.console.Write([]byte(Sprintf(format, a...)))
}

func (s *BuildSession) ReplaceEcho(name string, value interface{}) {
	s.echo.Substitutions[name] = value
}

func (s *BuildSession) warn(format string, a ...interface{}) {
	s.ConsoleLog(Sprintf("WARN: %v\n", format), a...)
}

func (s *BuildSession) debugLog(format string, a ...interface{}) {
	LogDebug(Sprintf("%v\n", format), a...)
}
