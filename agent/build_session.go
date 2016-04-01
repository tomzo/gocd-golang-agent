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
	"encoding/json"
	"github.com/bmatcuk/doublestar"
	"github.com/gocd-contrib/gocd-golang-agent/protocol"
	"github.com/gocd-contrib/gocd-golang-agent/stream"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	DefaultSecretMask           = "********"
	DefaultCancelCommandTimeout = 25 * time.Second
	CancelCommandTimeout        = DefaultCancelCommandTimeout
	CancelBuildTimeout          = 30 * time.Second
	BuildDebugToConsoleLog      = true
)

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

func (s *BuildSession) doProcess(cmd *protocol.BuildCommand) (err error) {
	s.wd = filepath.Clean(filepath.Join(s.rootDir, cmd.WorkingDirectory))
	s.debugLog("set wd to %v", s.wd)

	if !strings.HasPrefix(s.wd, s.rootDir) {
		return Err("Working directory[%v] is outside the agent sandbox.", s.wd)
	}
	switch cmd.Name {
	case protocol.CommandExport:
		s.processExport(cmd)
	case protocol.CommandEcho:
		s.processEcho(cmd)
	case protocol.CommandSecret:
		s.processSecret(cmd)
	case protocol.CommandReportCurrentStatus, protocol.CommandReportCompleting:
		jobState := cmd.Args["status"]
		s.debugLog("report %v", jobState)
		s.send <- protocol.ReportMessage(cmd.Name, s.Report(jobState))
	case protocol.CommandCompose:
		err = s.processCompose(cmd)
	case protocol.CommandTest:
		err = s.processTest(cmd)
	case protocol.CommandExec:
		err = s.processExec(cmd, s.secrets)
	case protocol.CommandMkdirs:
		err = s.processMkdirs(cmd)
	case protocol.CommandCleandir:
		err = s.processCleandir(cmd)
	case protocol.CommandUploadArtifact:
		err = s.processUploadArtifact(cmd)
	case protocol.CommandDownloadFile:
		err = s.processDownload(cmd)
	case protocol.CommandDownloadDir:
		err = s.processDownload(cmd)
	case protocol.CommandFail:
		err = Err(cmd.Args["0"])
	default:
		s.warn("Golang Agent does not support build comamnd '%v' yet, related GoCD feature will not be supported. More details: https://github.com/gocd-contrib/gocd-golang-agent", cmd.Name)
	}
	return
}

func (s *BuildSession) testFailed(test *protocol.BuildCommandTest) bool {
	if test == nil {
		return false
	}
	s.debugLog("test: %+v", test)
	_, err := s.processTestCommand(test.Command)
	if s.isCanceled() {
		s.debugLog("test is canceled due to build is canceled")
		return true
	}
	success := err == nil
	if success != test.Expectation {
		s.debugLog("test failed with err: %v", err)
		return true
	} else {
		s.debugLog("test matches expectation")
		return false
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

func (s *BuildSession) processSecret(cmd *protocol.BuildCommand) {
	value := cmd.Args["value"]
	substitution := cmd.Args["substitution"]
	if substitution == "" {
		substitution = DefaultSecretMask
	}
	s.debugLog("%v => %v", value, substitution)
	s.secrets.Substitutions[value] = substitution
}

func (s *BuildSession) processCleandir(cmd *protocol.BuildCommand) (err error) {
	path := cmd.Args["path"]
	var allows []string
	err = json.Unmarshal([]byte(cmd.Args["allowed"]), &allows)
	if err != nil {
		return
	}
	fullPath := filepath.Join(s.wd, path)
	s.debugLog("cleandir %v, excludes: %+v", fullPath, allows)
	return Cleandir(fullPath, allows...)
}

func (s *BuildSession) processMkdirs(cmd *protocol.BuildCommand) error {
	path := cmd.Args["path"]
	fullPath := filepath.Join(s.wd, path)
	s.debugLog("mkdirs %v", fullPath)
	return Mkdirs(fullPath)
}

func (s *BuildSession) processUploadArtifact(cmd *protocol.BuildCommand) error {
	src := cmd.Args["src"]
	destDir := cmd.Args["dest"]

	absSrc := filepath.Join(s.wd, src)
	return s.uploadArtifacts(absSrc, strings.Replace(destDir, "\\", "/", -1))
}

func (s *BuildSession) uploadArtifacts(source, destDir string) (err error) {
	if strings.Contains(source, "*") {
		matches, err := doublestar.Glob(source)
		sort.Strings(matches)
		if err != nil {
			return err
		}
		base := BaseDirOfPathWithWildcard(source)
		baseLen := len(base)
		for _, file := range matches {
			fileDir, _ := filepath.Split(file)
			dest := Join("/", destDir, fileDir[baseLen:len(fileDir)-1])
			err = s.uploadArtifacts(file, dest)
			if err != nil {
				return err
			}
		}
		return nil
	}

	srcInfo, err := os.Stat(source)
	if err != nil {
		return
	}
	s.ConsoleLog("Uploading artifacts from %v to %v\n", source, destDescription(destDir))

	var destPath string
	if destDir != "" {
		destPath = Join("/", destDir, srcInfo.Name())
	} else {
		destPath = srcInfo.Name()
	}
	destURL := AppendUrlParam(AppendUrlPath(s.artifactUploadBaseURL, destDir),
		"buildId", s.buildId)
	return s.artifacts.Upload(source, destPath, destURL)
}

func (s *BuildSession) processDownload(cmd *protocol.BuildCommand) error {
	checksumURL, err := config.MakeFullServerURL(cmd.Args["checksumUrl"])
	if err != nil {
		return err
	}
	absChecksumFile := filepath.Join(s.wd, cmd.Args["checksumFile"])
	err = s.artifacts.DownloadFile(checksumURL, absChecksumFile)
	if err != nil {
		return err
	}

	srcURL, err := config.MakeFullServerURL(cmd.Args["url"])
	if err != nil {
		return err
	}
	srcPath := cmd.Args["src"]
	absDestPath := filepath.Join(s.wd, cmd.Args["dest"])
	if cmd.Name == protocol.CommandDownloadDir {
		_, fname := filepath.Split(srcPath)
		absDestPath = filepath.Join(s.wd, cmd.Args["dest"], fname)
	}
	err = s.artifacts.VerifyChecksum(srcPath, absDestPath, absChecksumFile)
	if err == nil {
		s.ConsoleLog("[%v] exists and matches checksum, does not need dowload it from server.\n", srcPath)
		return nil
	}
	s.debugLog("download %v to %v", srcURL, absDestPath)
	if cmd.Name == protocol.CommandDownloadDir {
		err = s.artifacts.DownloadDir(srcURL, absDestPath)
	} else {
		err = s.artifacts.DownloadFile(srcURL, absDestPath)
	}
	if err != nil {
		return err
	}
	return s.artifacts.VerifyChecksum(srcPath, absDestPath, absChecksumFile)
}

func (s *BuildSession) processExec(cmd *protocol.BuildCommand, output io.Writer) error {
	args := cmd.ExtractArgList(len(cmd.Args))
	execCmd := exec.Command(args[0], args[1:]...)
	execCmd.Stdout = output
	execCmd.Stderr = output
	execCmd.Dir = s.wd
	done := make(chan error)
	go func() {
		done <- execCmd.Run()
	}()

	select {
	case <-s.cancel:
		s.debugLog("received cancel signal")
		LogInfo("kill process(%v) %v", execCmd.Process, cmd.Args)
		if err := execCmd.Process.Kill(); err != nil {
			s.ConsoleLog("Kill command %v failed, error: %v\n", cmd.Args, err)
		} else {
			LogInfo("process %v is killed", execCmd.Process)
		}
		return Err("%v is canceled", cmd.Args)
	case err := <-done:
		return err
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
		console:     stream.NopCloser(&output),
		command:     cmd,
		buildStatus: protocol.BuildPassed,
		cancel:      s.cancel,
		done:        make(chan bool),
	}

	err := session.ProcessCommand()
	return output, err
}

func (s *BuildSession) processTest(cmd *protocol.BuildCommand) error {
	flag := cmd.Args["flag"]

	switch flag {
	case "-d":
		targetPath := filepath.Join(s.wd, cmd.Args["left"])
		info, err := os.Stat(targetPath)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		} else {
			return Err("%v is not a directory", targetPath)
		}
	case "-f":
		targetPath := filepath.Join(s.wd, cmd.Args["left"])
		info, err := os.Stat(targetPath)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return Err("%v is not a file", targetPath)
		} else {
			return nil
		}
	case "-eq":
		output, err := s.processTestCommand(cmd.SubCommands[0])
		if err != nil {
			s.debugLog("test -eq exec command error: %v", err)
		}
		expected := strings.TrimSpace(cmd.Args["left"])
		actual := strings.TrimSpace(output.String())
		if expected != actual {
			return Err("expected '%v', but was '%v'", expected, actual)
		}
		return nil
	}

	return Err("unknown test flag")
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

func (s *BuildSession) warn(format string, a ...interface{}) {
	s.ConsoleLog(Sprintf("WARN: %v\n", format), a...)
}

func (s *BuildSession) ReplaceEcho(name string, value interface{}) {
	s.echo.Substitutions[name] = value
}

func (s *BuildSession) processEcho(cmd *protocol.BuildCommand) {
	for _, line := range cmd.ExtractArgList(len(cmd.Args)) {
		s.echo.Write([]byte(line))
		s.echo.Write([]byte{'\n'})
	}
}

func (s *BuildSession) processExport(cmd *protocol.BuildCommand) {
	msg := "setting environment variable '%v' to value '%v'\n"
	name := cmd.Args["name"]
	value, ok := cmd.Args["value"]
	if !ok {
		s.ConsoleLog(msg, name, os.Getenv(name))
		return
	}
	secure := cmd.Args["secure"]
	displayValue := value
	if secure == "true" {
		displayValue = DefaultSecretMask
	}
	_, override := s.envs[name]
	if override || os.Getenv(name) != "" {
		msg = "overriding environment variable '%v' with value '%v'\n"
	}
	s.envs[name] = value
	s.ConsoleLog(msg, name, displayValue)
}

func (s *BuildSession) processCompose(cmd *protocol.BuildCommand) error {
	var err error
	for _, sub := range cmd.SubCommands {
		if err != nil {
			s.process(sub)
		} else {
			err = s.process(sub)
		}
	}
	return err
}

func (s *BuildSession) debugLog(format string, a ...interface{}) {
	LogDebug(Sprintf("%v\n", format), a...)
}

func destDescription(path string) string {
	if path == "" {
		return "[defaultRoot]"
	} else {
		return path
	}
}
