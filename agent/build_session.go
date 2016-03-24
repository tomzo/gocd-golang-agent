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
	"encoding/json"
	"github.com/bmatcuk/doublestar"
	"github.com/gocd-contrib/gocd-golang-agent/protocal"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

var (
	DefaultSecretMask = "********"
)

type BuildSession struct {
	send          chan *protocal.Message
	buildStatus   string
	console       *BuildConsole
	artifacts     *Uploader
	command       *protocal.BuildCommand
	buildId       string
	envs          map[string]string
	cancel        chan bool
	done          chan bool
	echoVariables map[string]string
	secrets       map[string]string
}

func MakeBuildSession(buildId string, command *protocal.BuildCommand,
	console *BuildConsole, artifacts *Uploader,
	send chan *protocal.Message) *BuildSession {
	return &BuildSession{
		buildId:       buildId,
		buildStatus:   "passed",
		console:       console,
		artifacts:     artifacts,
		command:       command,
		send:          send,
		envs:          make(map[string]string),
		cancel:        make(chan bool),
		done:          make(chan bool),
		echoVariables: make(map[string]string),
		secrets:       make(map[string]string),
	}
}

func (s *BuildSession) Close() {
	close(s.cancel)
	<-s.done
}

func (s *BuildSession) isCanceled() bool {
	select {
	case <-s.cancel:
		return true
	default:
		return false
	}
}

func (s *BuildSession) Process() error {
	defer func() {
		s.console.Close()
		close(s.done)
	}()

	LogInfo("start process build command:")
	LogInfo(s.command.String())

	err := s.process(s.command)
	if err != nil {
		s.fail(err)
	}
	return err
}

func (s *BuildSession) process(cmd *protocal.BuildCommand) error {
	if s.isCanceled() {
		LogDebug("Ignored command %v, because build is canceled", cmd.Name)
		return nil
	}

	LogDebug("procssing build command: %v\n", cmd)
	if s.buildStatus != "" && cmd.RunIfConfig != "any" && cmd.RunIfConfig != s.buildStatus {
		//skip, no failure
		return nil
	}
	if cmd.Test != nil {
		success := s.process(cmd.Test.Command) == nil
		if success != cmd.Test.Expectation {
			return nil
		}
	}

	switch cmd.Name {
	case protocal.CommandCompose:
		return s.processCompose(cmd)
	case protocal.CommandExport:
		return s.processExport(cmd)
	case protocal.CommandTest:
		return s.processTest(cmd)
	case protocal.CommandExec:
		return s.processExec(cmd)
	case protocal.CommandEcho:
		return s.processEcho(cmd)
	case protocal.CommandMkdirs:
		return s.processMkdirs(cmd)
	case protocal.CommandCleandir:
		return s.processCleandir(cmd)
	case protocal.CommandUploadArtifact:
		return s.processUploadArtifact(cmd)
	case protocal.CommandSecret:
		return s.processSecret(cmd)
	case protocal.CommandFail:
		return Err(cmd.Args["0"])
	case protocal.CommandReportCurrentStatus, protocal.CommandReportCompleting, protocal.CommandReportCompleted:
		jobState := cmd.Args["jobState"]
		s.send <- protocal.ReportMessage(cmd.Name, s.statusReport(jobState))
	default:
		s.ConsoleLog("TBI command: %v", cmd.Name)
	}
	return nil
}

func (s *BuildSession) processSecret(cmd *protocal.BuildCommand) (err error) {
	value := cmd.Args["value"]
	substitution := cmd.Args["substitution"]
	if substitution == "" {
		substitution = DefaultSecretMask
	}
	s.secrets[value] = substitution
	return nil
}

func (s *BuildSession) processCleandir(cmd *protocal.BuildCommand) (err error) {
	path := cmd.Args["path"]
	wd, err := filepath.Abs(cmd.WorkingDirectory)
	if err != nil {
		return
	}
	var allows []string
	err = json.Unmarshal([]byte(cmd.Args["allowed"]), &allows)
	if err != nil {
		return
	}
	return Cleandir(filepath.Join(wd, path), allows...)
}

func (s *BuildSession) processMkdirs(cmd *protocal.BuildCommand) (err error) {
	path := cmd.Args["path"]
	wd, err := filepath.Abs(cmd.WorkingDirectory)
	if err != nil {
		return err
	}
	return Mkdirs(filepath.Join(wd, path))
}

func (s *BuildSession) processUploadArtifact(cmd *protocal.BuildCommand) (err error) {
	src := cmd.Args["src"]
	destDir := cmd.Args["dest"]

	wd, err := filepath.Abs(cmd.WorkingDirectory)
	if err != nil {
		return
	}
	absSrc := filepath.Join(wd, src)
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
	s.ConsoleLog("Uploading artifacts from %v to %v", source, destDescription(destDir))

	var destPath string
	if destDir != "" {
		destPath = Join("/", destDir, srcInfo.Name())
	} else {
		destPath = srcInfo.Name()
	}
	destURL := AppendUrlParam(AppendUrlPath(s.artifacts.BaseURL, destDir),
		"buildId", s.buildId)
	return s.artifacts.Upload(source, destPath, destURL)
}

func (s *BuildSession) processExec(cmd *protocal.BuildCommand) error {
	outputFilter := &SecretOutput{session: s}
	args := cmd.ExtractArgList(len(cmd.Args))
	execCmd := exec.Command(args[0], args[1:]...)
	execCmd.Stdout = outputFilter
	execCmd.Stderr = outputFilter
	execCmd.Dir = cmd.WorkingDirectory
	done := make(chan error)
	go func() {
		done <- execCmd.Run()
	}()

	select {
	case <-s.cancel:
		LogDebug("received cancel signal")
		LogInfo("killing process(%v) %v", execCmd.Process, cmd.Args)
		if err := execCmd.Process.Kill(); err != nil {
			s.ConsoleLog("kill command %v failed, error: %v", cmd.Args, err)
		} else {
			LogInfo("Process %v is killed", execCmd.Process)
		}
		return Err("%v is canceled", cmd.Args)
	case err := <-done:
		return err
	}
}

func (s *BuildSession) processTest(cmd *protocal.BuildCommand) error {
	flag := cmd.Args["flag"]
	targetPath := cmd.Args["path"]

	if "-d" == flag {
		_, err := os.Stat(targetPath)
		return err
	}
	return Err("unknown test flag")
}

func (s *BuildSession) statusReport(jobState string) *protocal.Report {
	return &protocal.Report{
		AgentRuntimeInfo: GetAgentRuntimeInfo(),
		BuildId:          s.buildId,
		JobState:         jobState,
		Result:           capitalize(s.buildStatus),
	}
}

func capitalize(str string) string {
	a := []rune(str)
	a[0] = unicode.ToUpper(a[0])
	return string(a)
}

func (s *BuildSession) ConsoleLog(format string, a ...interface{}) {
	s.console.Write([]byte(Sprintf(format, a...)))
}

func (s *BuildSession) ReplaceEcho(name, value string) {
	s.echoVariables[name] = value
}

func (s *BuildSession) processEcho(cmd *protocal.BuildCommand) error {
	for _, line := range cmd.ExtractArgList(len(cmd.Args)) {
		s.ConsoleLog(s.maskSecrets(s.substituteVariables(line)))
	}
	return nil
}

func (s *BuildSession) substituteVariables(str string) string {
	for k, v := range s.echoVariables {
		str = strings.Replace(str, k, v, -1)
	}
	dateF := time.Now().Format("2006-01-02 15:04:05 PDT")
	return strings.Replace(str, "${date}", dateF, -1)
}

func (s *BuildSession) maskSecrets(str string) string {
	for k, v := range s.secrets {
		str = strings.Replace(str, k, v, -1)
	}
	return str
}

func (s *BuildSession) processExport(cmd *protocal.BuildCommand) error {
	msg := "setting environment variable '%v' to value '%v'"
	name := cmd.Args["name"]
	value, ok := cmd.Args["value"]
	if !ok {
		s.ConsoleLog(msg, name, os.Getenv(name))
		return nil
	}
	secure := cmd.Args["secure"]
	displayValue := value
	if secure == "true" {
		displayValue = DefaultSecretMask
	}
	_, override := s.envs[name]
	if override || os.Getenv(name) != "" {
		msg = "overriding environment variable '%v' with value '%v'"
	}
	s.envs[name] = value
	s.ConsoleLog(msg, name, displayValue)
	return nil
}

func (s *BuildSession) processCompose(cmd *protocal.BuildCommand) error {
	var err error
	for _, sub := range cmd.SubCommands {
		if err = s.process(sub); err != nil {
			s.fail(err)
		}
	}
	return err
}

func (s *BuildSession) fail(err error) {
	if s.buildStatus != "failed" {
		s.ConsoleLog(err.Error())
		s.buildStatus = "failed"
	}
}

func destDescription(path string) string {
	if path == "" {
		return "[defaultRoot]"
	} else {
		return path
	}
}

type SecretOutput struct {
	session *BuildSession
}

func (o *SecretOutput) Write(out []byte) (int, error) {
	o.session.ConsoleLog(o.session.maskSecrets(string(out)))
	return len(out), nil
}
