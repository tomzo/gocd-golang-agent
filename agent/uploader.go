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
	"archive/zip"
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type Uploader struct {
	baseURL    string
	httpClient *http.Client
	console    *BuildConsole
}

func NewUploader(console *BuildConsole, httpClient *http.Client, baseURL string) *Uploader {
	return &Uploader{console: console, baseURL: baseURL, httpClient: httpClient}
}

func (u *Uploader) Upload(workingDir, src, destDir, buildId string) (err error) {
	wd, err := filepath.Abs(workingDir)
	if err != nil {
		return
	}
	source := filepath.Join(wd, src)

	finfo, err := os.Stat(source)
	if err != nil {
		return errors.New("Failed to find " + src)
	}
	if finfo.IsDir() {
		return errors.New("Can't handle directory upload yet")
	}
	u.console.WriteLn("Uploading artifacts from %v to %v",
		source, destDescription(destDir))

	var destURI string
	if destDir != "" {
		destURI = filepath.Join(destDir, finfo.Name())
	} else {
		destURI = finfo.Name()
	}

	zipped, err := zipSource(source, destURI)
	if err != nil {
		return
	}
	contentType, body, err := requestBody(zipped, destURI, destDir)
	if err != nil {
		return
	}
	destURL, err := u.buildDestURL(destDir, buildId)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", destURL, &body)
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", contentType)
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusCreated {
		return
	}
	switch resp.StatusCode {
	case http.StatusRequestEntityTooLarge:
		err = errors.New(fmt.Sprintf("Artifact upload for file %s (Size: %s) was denied by the server. This usually happens when server runs out of disk space.", source, finfo.Size()))
	default:
		err = errors.New(fmt.Sprintf("Failed to upload %v. Server response: %v", source, resp.Status))
	}
	return
}

func (u *Uploader) buildDestURL(destDir, buildId string) (string, error) {
	url, err := url.Parse(u.baseURL)
	if err != nil {
		return "", err
	}
	url.RawPath = url.RawPath + "/" + destDir
	values := url.Query()
	values.Set("buildId", buildId)
	url.RawQuery = values.Encode()

	return url.String(), nil
}

func requestBody(source, destURI, destDir string) (contentType string, body bytes.Buffer, err error) {
	md5, err := computeMd5(source)

	checksum := fmt.Sprintf("#\n#%v\n%v=%x", time.Now(), destURI, md5)
	writer := multipart.NewWriter(&body)
	err = writeFilePart(writer, source, "zipfile")
	if err != nil {
		return
	}
	err = writePart(writer, bytes.NewBufferString(checksum), "file_checksum", "checksum_file")
	if err != nil {
		return
	}
	err = writer.Close()
	contentType = writer.FormDataContentType()
	return
}

func writeFilePart(writer *multipart.Writer, path, paramName string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return writePart(writer, file, paramName, filepath.Base(path))
}

func writePart(writer *multipart.Writer, src io.Reader, fieldname, filename string) error {
	part, err := writer.CreateFormFile(fieldname, filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, src)
	return err
}

func computeMd5(filePath string) ([]byte, error) {
	var result []byte
	file, err := os.Open(filePath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return result, err
	}

	return hash.Sum(result), nil
}

func destDescription(path string) string {
	if path == "" {
		return "[defaultRoot]"
	} else {
		return path
	}
}

func zipSource(path string, dest string) (string, error) {
	zipfile, err := ioutil.TempFile("", "tmp.zip")
	if err != nil {
		return "", err
	}
	defer zipfile.Close()
	w := zip.NewWriter(zipfile)
	defer w.Close()
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	writer, err := w.Create(dest)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(writer, file)
	return zipfile.Name(), err
}
