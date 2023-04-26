/*
 * Copyright (c) 2020 Joseph Saylor <doug@saylorsolutions.com>
 * Copyright (c) 2023 Lorenzo Delgado <lnsdev@proton.me>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package specfile

import (
	"bytes"
	"fmt"
	"html/template"
)

const specTemplateString = `
{{- if .WithComments}}# Hosts and files to tail{{end}}
hosts:
  host1:
    hostname: remote-host-1
    {{if .WithComments}}# Excluding the username here will default it to the current user name{{end}}
    file: /var/log/syslog
    {{if .WithComments}}# Default SSH port{{end}}
    port: 22
  host2:
    hostname: remote-host-2
    username: me
    file: /var/log/syslog
    port: 22
`

// SpecTemplateConfig config
type SpecTemplateConfig struct {
	WithComments bool
}

// NewSpecTemplate creates a new spec template with the given configuration parameters.
func NewSpecTemplate(withComments bool) (string, error) {

	t, err := template.New("spec-template").Parse(specTemplateString)
	if err != nil {
		return "", fmt.Errorf("Unable to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err = t.Execute(&buf, SpecTemplateConfig{WithComments: withComments}); err != nil {
		return "", fmt.Errorf("Unable to generate template file contents: %v", err)
	}

	return buf.String(), nil
}
