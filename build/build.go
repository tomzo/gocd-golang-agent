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
	"runtime"
	"crypto/sha256"
	"encoding/hex"
)

var goAgentFilename = "gocd-golang-agent"

var targetOS = map[string][]string{
	"darwin" : {"amd64"},
	"linux" : {"386", "amd64", "arm", "arm64"},
	"windows" : {"386", "amd64"},
//	"solaris" : {"amd64"},  // Solaris does not have syscall.Statfs_t and syscall.Statfs to find disk usage.
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

func getGitHash(pwd string) string {
	runCmd := exec.Command("git", "rev-parse", "HEAD")
	runCmd.Dir = pwd + "/src/" + goAgent
	out, err := runCmd.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	return strings.TrimSpace(string(out))
}

func buildBinary(pwd string, binAllbinary bool){
	fmt.Println("==================================")
	fmt.Println("Building Binary")
	os.RemoveAll("output")
	os.Mkdir("output",0755)
	os.Setenv("CGO_ENABLED","0")
	if binAllbinary {
		for buildOS, buildArchs := range targetOS {
			for _, buildArch := range buildArchs {
				os.Setenv("GOOS", buildOS)
				os.Setenv("GOARCH", buildArch)
				compileApp(pwd, buildOS, buildArch)
			}
		}
	}else{
		compileApp(pwd, runtime.GOOS, runtime.GOARCH)
	}

}

func compileApp(pwd string, targetOS string, targetArch string){
	var outputName string
	fmt.Println("---> " + targetOSmap[targetOS] + " - " + targetArch)
	ldFlags := "-w -X main.Githash=" + getGitHash(pwd)
	buildVersion := os.Getenv("BUILD_VERSION")
	outputName = "output/" + goAgentFilename + "_" + targetOS + "_" + targetArch
	if len(buildVersion) > 0 {
		ldFlags = ldFlags + " -X main.Version=" + buildVersion
		outputName = outputName + "_" + buildVersion
	}
	if targetOS == "windows"{
		outputName = outputName + ".exe"
	}
	out, err := exec.Command("go", "build", "-a", "-tags", "netgo", "-ldflags", ldFlags, "-o", outputName, goAgent).Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(string(out)) > 0 {
		fmt.Println(out)
	}
	hasher := sha256.New()
	outfileHash, outfileHashErr := ioutil.ReadFile(outputName)
	if outfileHashErr != nil {
		fmt.Println("Cannot read file ", outputName)
		os.Exit(2)
	}
	hasher.Write(outfileHash)
	outputNameSHA256, outputNameSHA256Err := os.Create(outputName + ".SHA256")
	if outputNameSHA256Err != nil {
		fmt.Println("Cannot create ", outputName, " SHA256 checksum file")
		os.Exit(2)
	}
	defer outputNameSHA256.Close()
	_, writeSHA256Err := outputNameSHA256.WriteString(hex.EncodeToString(hasher.Sum(nil)))
	if writeSHA256Err != nil {
		fmt.Println("Cannot write ", outputName, " SHA256 checksum to file")
		os.Exit(2)
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

func uploadToBintray(pwd string, binAllbinary bool)  {

	var (
		bintrayURL string =  "https://api.bintray.com/content/gocd-contrib/gocd_golang_goagent"
		bintrayPackage = "goagent"
	)

	buildVersion := os.Getenv("BUILD_VERSION")
	bintrayUser := os.Getenv("BINTRAY_USER")
	bintrayToken := os.Getenv("BINTRAY_TOKEN")
	if len(buildVersion) <= 0 {
		fmt.Println("Unknown BUILD_VERSION")
		os.Exit(2)
	}
	if len(bintrayUser) <= 0 {
		fmt.Println("Unknown Bintray User")
		os.Exit(2)
	}
	if len(bintrayToken) <= 0 {
		fmt.Println("Unknown Bintray Token")
		os.Exit(2)
	}


	// temporary use curl to upload the file , will change it to use native http/net library

	fmt.Println("==================================")
	fmt.Println("Upload Binary to Bintray")
	for buildOS, buildArchs := range targetOS {
		for _, buildArch := range buildArchs {
			fmt.Println("---> " + targetOSmap[buildOS] + " - " + buildArch)
			outFilename := goAgentFilename + "_" + buildOS + "_" + buildArch + "_" + buildVersion
			if buildOS == "windows"{
				outFilename = outFilename + ".exe"
			}
			if _, err := os.Stat("output/" + outFilename); err == nil {
				if _, err := os.Stat("output/" + outFilename + ".SHA256"); err == nil {
					outFilenameURL :=  bintrayURL + "/" + bintrayPackage +
						"/" + buildVersion + "/" + outFilename + ";bt_package=" + bintrayPackage + ";bt_version=" +
						buildVersion
					fmt.Println("\t", outFilename)
					_, uploadErr := exec.Command("curl", "-T", "output/" + outFilename, "-u" + bintrayUser + ":" + bintrayToken , outFilenameURL).Output()
					if uploadErr != nil {
						fmt.Println("Error upload file : ", outFilename, " to bintray.")
						os.Exit(2)
					}
					outFilenameURL =  bintrayURL + "/" + bintrayPackage +
						"/" + buildVersion + "/" + outFilename + ".SHA256" + ";bt_package=" + bintrayPackage + ";bt_version=" +
						buildVersion
					fmt.Println("\t", outFilename + ".SHA256")
					_, uploadErr = exec.Command("curl", "-T", "output/" + outFilename + ".SHA256", "-u" + bintrayUser + ":" + bintrayToken, outFilenameURL).Output()
					if uploadErr != nil {
						fmt.Println("Error upload file : ", outFilename,  " SHA256 checksum to bintray.")
						os.Exit(2)
					}

				}else{
					fmt.Println("File : " , outFilename, " SHA256 checksum does not exist")
					os.Exit(2)
				}
			}else{
				fmt.Println("File : ", outFilename , " does not exist.")
				os.Exit(2)
			}

		}
	}
}
func main() {

	var (
		excludeLib string
		runAllTest bool
		buildLocalBinary bool
		buildAll bool
		uploadBinary bool
	)

	flag.StringVar(&excludeLib, "excludelib", "", "exclude dependencies in comma separated format, eg github.com/gocd-contrib/fake_agent,github.com/gocd-contrib/fake_server")
	flag.BoolVar(&runAllTest,"runtest", true, "Run all Tests")
	flag.BoolVar(&buildLocalBinary,"buildbinary", true, "Build local GoAgent binary" )
	flag.BoolVar(&buildAll,"buildall", false, "Build GoAgent binary for all platforms" )
	flag.BoolVar(&uploadBinary,"upload", false, "Upload GoAgent binary to bintray")
	flag.Parse()

	pwd, err := os.Getwd()
	if err == nil {
		os.Setenv("GOPATH",pwd)
		os.Setenv("GOBIN", pwd + "/bin")
	}


	getDependencies(excludeLib)

	if runAllTest {
		runTest(pwd)
	}

	if buildAll {
		buildBinary(pwd,true)
	}else{
		if buildLocalBinary {
			buildBinary(pwd, false)
		}
	}

	if uploadBinary {
		uploadToBintray(pwd,true)
	}
}