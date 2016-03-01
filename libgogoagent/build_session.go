package libgogoagent

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
)

type BuildSession struct {
	HttpClient            *http.Client
	BuildStatus           string
	Console               *BuildConsole
	ArtifactUploadBaseUrl string
	PropertyBaseUrl       string
	BuildId               string
	Envs                  map[string]string
}

func (s *BuildSession) Process(cmd *BuildCommand) error {
	log.Printf("procssing build command: %v\n", cmd)
	if s.BuildStatus != "" && cmd.RunIfConfig != "any" && cmd.RunIfConfig != s.BuildStatus {
		//skip, no failure
		return nil
	}
	if cmd.Test != nil {
		success := s.Process(cmd.Test.Command) == nil
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
		return s.processReportCurrentStatus(cmd)
	case "end":
		return s.processEnd(cmd)
	default:
		//return "Unknown command: " + cmd.Name
		return s.processEcho(&BuildCommand{Args: []interface{}{cmd.Name}})
	}
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
	execCmd.Stdout = s.Console
	execCmd.Stderr = s.Console
	execCmd.Dir = cmd.WorkingDirectory
	return execCmd.Run()
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

func (s *BuildSession) processEnd(cmd *BuildCommand) error {
	s.Console.Stop <- 1
	return nil
}

func (s *BuildSession) processReportCurrentStatus(cmd *BuildCommand) error {
	return nil
}

func (s *BuildSession) processEcho(cmd *BuildCommand) error {
	for _, arg := range cmd.Args {
		s.Console.WriteLn(arg.(string))
	}
	return nil
}

func (s *BuildSession) processExport(cmd *BuildCommand) error {
	if len(cmd.Args) > 0 {
		newEnvs := cmd.Args[0].(map[string]interface{})
		for key, value := range newEnvs {
			s.Envs[key] = value.(string)
		}
	} else {
		args := make([]interface{}, 0)
		for key, value := range s.Envs {
			args = append(args, fmt.Sprintf("export %v=%v", key, value))
		}
		s.Process(&BuildCommand{
			Name: "echo",
			Args: args,
		})
	}
	return nil
}

func (s *BuildSession) processCompose(cmd *BuildCommand) error {
	var err error
	for _, sub := range cmd.SubCommands {
		if err = s.Process(sub); err != nil {
			s.BuildStatus = "failed"
		}
	}
	return err
}

func (s *BuildSession) processStart(cmd *BuildCommand) error {
	settings, _ := cmd.Args[0].(map[string]interface{})
	SetState("buildLocator", settings["buildLocator"].(string))
	SetState("buildLocatorForDisplay", settings["buildLocatorForDisplay"].(string))
	s.Console = MakeBuildConsole(s.HttpClient, settings["consoleURI"].(string))
	s.ArtifactUploadBaseUrl = settings["artifactUploadBaseUrl"].(string)
	s.PropertyBaseUrl = settings["propertyBaseUrl"].(string)
	s.BuildId = settings["buildId"].(string)
	s.Envs = make(map[string]string)
	s.BuildStatus = "passed"
	return nil
}
