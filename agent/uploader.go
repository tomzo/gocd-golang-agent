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
	"strconv"
	"strings"
	"time"
)

type Uploader struct {
	BaseURL    *url.URL
	httpClient *http.Client
}

func NewUploader(httpClient *http.Client, baseURL *url.URL) *Uploader {
	return &Uploader{BaseURL: baseURL, httpClient: httpClient}
}

func (u *Uploader) Upload(source, destPath string, destURL *url.URL) (err error) {
	zipped, checksum, err := u.zipSource(source, destPath)
	defer os.Remove(zipped)
	if err != nil {
		return
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	err = u.writeFilePart(writer, zipped, "zipfile")
	if err != nil {
		return
	}
	err = u.writePart(writer, bytes.NewBufferString(checksum), "file_checksum", "checksum_file")
	if err != nil {
		return
	}
	err = writer.Close()
	if err != nil {
		return
	}

	attempt := 1
tryPost:
	attemptUrl := AppendUrlParam(destURL, "attempt", strconv.Itoa(attempt))
	statusCode, err := u.post(source, writer.FormDataContentType(), attemptUrl, &body)
	// client side errors, no retry
	if err != nil {
		return
	}
	// success
	if statusCode == http.StatusCreated {
		return
	}
	// handle errors
	if statusCode == http.StatusRequestEntityTooLarge {
		info, _ := os.Stat(zipped)
		return errors.New(fmt.Sprintf("Artifact upload for file %s (Size: %d) was denied by the server. This usually happens when server runs out of disk space.", source, info.Size()))
	}
	// retry for other errors
	if attempt < 3 {
		attempt++
		goto tryPost
	}
	return errors.New(fmt.Sprintf("Failed to upload %v. Server response: %v", source, statusCode))
}

func (u *Uploader) post(source, contentType string, destURL *url.URL, body *bytes.Buffer) (statusCode int, err error) {
	req, err := http.NewRequest("POST", destURL.String(), body)
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", contentType)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return
	}
	return resp.StatusCode, nil
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

func (u *Uploader) zipSource(source string, dest string) (string, string, error) {
	zipfile, err := ioutil.TempFile("", "tmp.zip")
	if err != nil {
		return "", "", err
	}
	defer zipfile.Close()
	w := zip.NewWriter(zipfile)
	defer w.Close()

	var checksum bytes.Buffer
	checksum.WriteString(fmt.Sprintf("#\n#%v\n", time.Now()))
	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		destFile := dest
		if path != source {
			// source is a directory, find relative path
			// from source and attach to dest path
			rel := path[len(source):]
			if strings.HasPrefix(rel, "/") {
				rel = rel[1:]
			}
			if dest == "" {
				destFile = rel
			} else {
				destFile = dest + "/" + rel
			}
		}

		md5, err := u.computeMd5(path)
		if err != nil {
			return err
		}
		checksum.WriteString(fmt.Sprintf("%v=%x\n", destFile, md5))

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		writer, err := w.Create(destFile)
		if err != nil {
			return err
		}

		_, err = io.Copy(writer, file)
		return err
	})
	return zipfile.Name(), checksum.String(), err
}
