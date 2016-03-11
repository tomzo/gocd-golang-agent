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

package libgocdgolangagent

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"unicode"
)

type BuildSession struct {
	HttpClient *http.Client
	Send       chan *Message

	buildStatus           string
	console               *BuildConsole
	artifactUploadBaseUrl string
	propertyBaseUrl       string
	buildId               string
	envs                  map[string]string
	cancel                chan int
	done                  chan int
}

func MakeBuildSession(httpClient *http.Client, send chan *Message) *BuildSession {
	return &BuildSession{
		HttpClient: httpClient,
		Send:       send,
		cancel:     make(chan int),
		done:       make(chan int),
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

func (s *BuildSession) Process(cmd *BuildCommand) error {
	defer func() {
		if s.console != nil {
			s.console.Close()
		}
		close(s.done)
	}()
	return s.process(cmd)
}

func (s *BuildSession) process(cmd *BuildCommand) error {
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
	case "start":
		return s.processStart(cmd)
	case "compose":
		return s.processCompose(cmd)
	case "export":
		return s.processExport(cmd)
	case "test":
		return s.processTest(cmd)
	case "exec":
		return s.processExec(cmd)
	case "echo":
		return s.processEcho(cmd)
	case "reportCurrentStatus":
		s.Send <- MakeMessage(cmd.Name,
			"com.thoughtworks.go.websocket.Report",
			s.statusReport(cmd.Args[0].(string)))
	case "reportCompleting", "reportCompleted":
		s.Send <- MakeMessage(cmd.Name,
			"com.thoughtworks.go.websocket.Report",
			s.statusReport(""))
	case "end":
		// nothing to do
	default:
		s.console.WriteLn("TBI command: %v", cmd.Name)
	}
	return nil
}

func convertToStringSlice(slice []interface{}) []string {
	ret := make([]string, len(slice))
	for i, element := range slice {
		ret[i] = element.(string)
	}
	return ret
}

func (s *BuildSession) processExec(cmd *BuildCommand) error {
	arg0 := cmd.Args[0].(string)
	args := convertToStringSlice(cmd.Args[1:])
	execCmd := exec.Command(arg0, args...)
	execCmd.Stdout = s.console
	execCmd.Stderr = s.console
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
			s.console.WriteLn("kill command %v failed, error: %v", cmd.Args, err)
		} else {
			LogInfo("Process %v is killed", execCmd.Process)
		}
		return errors.New(fmt.Sprintf("%v is canceled", cmd.Args))
	case err := <-done:
		if err != nil {
			s.console.WriteLn(err.Error())
		}
		return err
	}
}

func (s *BuildSession) processTest(cmd *BuildCommand) error {
	flag := cmd.Args[0].(string)
	targetPath := cmd.Args[1].(string)

	if "-d" == flag {
		_, err := os.Stat(targetPath)
		return err
	}
	return errors.New("unknown test flag")
}

func (s *BuildSession) statusReport(jobState string) map[string]interface{} {
	ret := map[string]interface{}{
		"agentRuntimeInfo": AgentRuntimeInfo(),
		"buildId":          s.buildId,
		"jobState":         jobState,
		"result":           capitalize(s.buildStatus)}
	return ret
}

func capitalize(str string) string {
	a := []rune(str)
	a[0] = unicode.ToUpper(a[0])
	return string(a)
}

func (s *BuildSession) processEcho(cmd *BuildCommand) error {
	for _, arg := range cmd.Args {
		s.console.WriteLn(arg.(string))
	}
	return nil
}

func (s *BuildSession) processExport(cmd *BuildCommand) error {
	if len(cmd.Args) > 0 {
		newEnvs := cmd.Args[0].(map[string]interface{})
		for key, value := range newEnvs {
			s.envs[key] = value.(string)
		}
	} else {
		args := make([]interface{}, 0)
		for key, value := range s.envs {
			args = append(args, fmt.Sprintf("export %v=%v", key, value))
		}
		s.process(&BuildCommand{
			Name: "echo",
			Args: args,
		})
	}
	return nil
}

func (s *BuildSession) processCompose(cmd *BuildCommand) error {
	var err error
	for _, sub := range cmd.SubCommands {
		if err = s.process(sub); err != nil {
			s.buildStatus = "failed"
		}
	}
	return err
}

func (s *BuildSession) processStart(cmd *BuildCommand) error {
	settings, _ := cmd.Args[0].(map[string]interface{})
	SetState("buildLocator", settings["buildLocator"].(string))
	SetState("buildLocatorForDisplay", settings["buildLocatorForDisplay"].(string))
	s.console = MakeBuildConsole(s.HttpClient, config.MakeFullServerURL(settings["consoleURI"].(string)))
	s.artifactUploadBaseUrl = config.MakeFullServerURL(settings["artifactUploadBaseUrl"].(string))
	s.propertyBaseUrl = config.MakeFullServerURL(settings["propertyBaseUrl"].(string))
	s.buildId = settings["buildId"].(string)
	s.envs = make(map[string]string)
	s.buildStatus = "passed"
	return nil
}
