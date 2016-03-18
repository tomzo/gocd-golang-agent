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
}

func NewUploader(httpClient *http.Client, baseURL string) *Uploader {
	return &Uploader{baseURL: baseURL, httpClient: httpClient}
}

func (u *Uploader) Upload(source, destFile, destURL string) (err error) {
	zipped, err := u.zipSource(source, destFile)
	if err != nil {
		return
	}
	contentType, body, err := u.requestBody(zipped, destFile)
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
		info, _ := os.Stat(zipped)
		err = errors.New(fmt.Sprintf("Artifact upload for file %s (Size: %s) was denied by the server. This usually happens when server runs out of disk space.", source, info.Size()))
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

func (u *Uploader) requestBody(source, destFile string) (contentType string, body bytes.Buffer, err error) {
	md5, err := u.computeMd5(source)

	checksum := fmt.Sprintf("#\n#%v\n%v=%x", time.Now(), destFile, md5)
	writer := multipart.NewWriter(&body)
	err = u.writeFilePart(writer, source, "zipfile")
	if err != nil {
		return
	}
	err = u.writePart(writer, bytes.NewBufferString(checksum), "file_checksum", "checksum_file")
	if err != nil {
		return
	}
	err = writer.Close()
	contentType = writer.FormDataContentType()
	return
}

func (u *Uploader) writeFilePart(writer *multipart.Writer, path, paramName string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return u.writePart(writer, file, paramName, filepath.Base(path))
}

func (u *Uploader) writePart(writer *multipart.Writer, src io.Reader, fieldname, filename string) error {
	part, err := writer.CreateFormFile(fieldname, filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, src)
	return err
}

func (u *Uploader) computeMd5(filePath string) ([]byte, error) {
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

func (u *Uploader) zipSource(path string, dest string) (string, error) {
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
