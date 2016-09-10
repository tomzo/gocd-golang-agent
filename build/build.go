package main
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

import (
	"fmt"
	"os"
	"os/exec"
	"io"
	"bytes"
	"io/ioutil"
	"strings"
	"flag"
)

var goAgentFilename = "gocd-golang-agent"

var targetOS = map[string][]string{
	"darwin" : {"amd64"},
	"linux" : {"386", "amd64"},
//	"windows" : {"386", "amd64"},  // Windows build is broken because of undefined syscall.Statfs_t and syscall.Statfs
}

var goAgent = "github.com/gocd-contrib/gocd-golang-agent"

var targetOSmap = map[string]string{
	"darwin" : "MacOS",
	"linux" : "Linux",
	"windows" : "Windows",
	"solaris" : "Solaris",
	"netbsd" : "NetBSD",
	"freebsd" : "FreeBSD",
}

var ext_libraries []string =  []string{
	"golang.org/x/net/websocket",
	"golang.org/x/text",
	"golang.org/x/crypto/ssh",
	"github.com/satori/go.uuid",
	"github.com/xli/assert",
	"github.com/bmatcuk/doublestar"}

var testReport = "testreport.xml"

func shouldSkipDependent(exlib string) bool {
	exlibGoEnv := "GO_REVISION_SRC_" + strings.ToUpper(strings.Join(strings.Split(strings.Join(strings.Split(exlib,"."),"_"),"/"),"_"))
	if len(os.Getenv(exlibGoEnv)) > 0 {
		return true
	}
	return false
}

func getDependencies(excludeLib string) {
	fmt.Println("==================================")
	fmt.Println("Download Dependencies")
	go_args := []string{}
	args := []string{"get","-u"}
	for _, exlib := range ext_libraries {
		if strings.Contains(excludeLib,exlib) {
			fmt.Printf("Exclude from go get. Please manually run : go get %s\n", exlib)
		}else {
			if !shouldSkipDependent(exlib) {
				go_args = append(args, exlib)
				_, err := exec.Command("go", go_args...).Output()
				if err != nil {
					fmt.Println(err)
				}
				fmt.Printf("Get : %s\n", exlib)
			} else {
				fmt.Printf("Skip %s since it is part of GoCD Material\n", exlib)
			}
		}
	}
}

func buildBinary(){
	fmt.Println("==================================")
	fmt.Println("Building Binary")
	os.RemoveAll("output")
	os.Mkdir("output",0755)
	os.Setenv("CGO_ENABLED","0")
	args := []string{"build", "-a", "-o", }
	newArgs := []string{}
	for buildOS, buildArchs := range targetOS {
		for _, buildArch := range buildArchs {
			fmt.Println("---> " + targetOSmap[buildOS] + " - " + buildArch)
			os.Setenv("GOOS", buildOS)
			os.Setenv("GOARCH", buildArch)
			newArgs = append(args, "output/" + goAgentFilename + "_" + buildOS + "_" + buildArch)
			newArgs = append(newArgs, goAgent)
			if _, err := exec.Command("go", newArgs...).CombinedOutput(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
		}
	}

}

func runTest(pwd string){
	fmt.Println("==================================")
	fmt.Println("Running Test")
	args := []string{"get","-u","github.com/jstemmer/go-junit-report"}
	reportCmd := pwd + "/bin/go-junit-report"
	reportCmd_args := []string{" > ", testReport}

	_, err := exec.Command("go", args...).Output()
	if err != nil{
		fmt.Println(err)
	}
	_, err = os.Stat(testReport)
	if err == nil {
		os.Remove(testReport)
	}
	args = []string{"test", "-test.v", goAgent + "..." }
	gotest := exec.Command("go", args...)
	goreport := exec.Command(reportCmd,reportCmd_args...)
	reader, writer := io.Pipe()
	gotest.Stdout = writer
	goreport.Stdin = reader

	var buf bytes.Buffer
	goreport.Stdout = &buf

	gotest.Start()
	goreport.Start()
	gotest.Wait()
	writer.Close()
	goreport.Wait()
	err = ioutil.WriteFile(testReport,buf.Bytes(),0644)
	if err != nil{
		panic(err)
	}
}

func main() {

	var excludeLib string

	flag.StringVar(&excludeLib, "excludelib", "us-west-1", "AWS Regions in comma separated format, eg us-west-1,us-east-1")
	flag.Parse()

	pwd, err := os.Getwd()
	if err == nil {
		os.Setenv("GOPATH",pwd)
		os.Setenv("GOBIN", pwd + "/bin")
	}

	getDependencies(excludeLib)
	runTest(pwd)
	buildBinary()


}