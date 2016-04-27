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

var UNIT_TEST_REPORT_TEMPLATE = `<div class="tests">
  <p>Tests run:
    <span class="tests_total_count">{{.Tests}}</span>
    , Failures:
    <span class="tests_failed_count">{{.Failures}}</span>
    , Not run:
    <span class="tests_ignored_count">{{.Skipped}}</span>
    , Time:
    <span class="tests_total_duration">{{.Time}}</span>
    seconds.
  </p>
</div>

{{if .Failures }}
<table class="section-table" cellpadding="2" cellspacing="0" border="0" width="98%">
  {{range .TestCases}}
    {{if .Failure}}
    <tr>
      <td class="section-data">Failure</td>
      <td class="section-data">{{ .Name }}</td>
    </tr>
    {{end}}
    {{if .Error}}
    <tr>
      <td class="section-data">Error</td>
      <td class="section-data">{{ .Name }}</td>
    </tr>
    {{end}}
  {{end}}
</table>
{{end}}
{{if .Failures}}
<table class="section-table" cellpadding="2" cellspacing="0" border="0" width="98%">
  <tr>
    <td colspan="2" class="sectionheader">Unit Test Failure and Error Details ({{ .Failures }})</td>
  </tr>
  {{range .TestCases}}
    {{if .Failure}}
    <tr>
      <td class="section-data">Test:</td>
      <td class="section-data">{{ .Name }}</td>
    </tr>
    <tr>
      <td class="section-data">Type:</td>
      <td class="section-data">Failure</td>
    </tr>
    <tr>
      <td class="section-data">Message:</td>
      <td class="section-data">{{ .Failure.StackTrace }}</td>
    </tr>
    {{end}}
    {{if .Error}}
    <tr>
      <td class="section-data">Test:</td>
      <td class="section-data">{{ .Name }}</td>
    </tr>
    <tr>
      <td class="section-data">Type:</td>
      <td class="section-data">Error</td>
    </tr>
    <tr>
      <td class="section-data">Message:</td>
      <td class="section-data">{{ .Error.StackTrace }}</td>
    </tr>
    {{end}}
  {{end}}
</table>
{{end}}
`
