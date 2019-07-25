// Copyright 2019 Cloud Privacy Labs, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	mail "gopkg.in/gomail.v2"
)

func TestSend(t *testing.T) {
	var msg *mail.Message
	var cfg FormCfg

	m := Mailer{Send: func(m *mail.Message) error {
		msg = m
		return nil
	}, ConfigGetter: func(name string) *FormCfg {
		return &cfg
	}}
	server := httptest.NewServer(m)
	defer server.Close()

	cfg.From = "from@test"
	cfg.Recipients = []string{"to@test"}
	cfg.Subject = "Subject {{index .form.formId 0}}"

	rsp, err := http.PostForm(server.URL, url.Values{"formId": []string{"test"},
		"field1": []string{"value1"}})
	if err != nil {
		t.Errorf("Cannot post: %v", err)
		return
	}
	rsp.Body.Close()
	if rsp.StatusCode != 200 {
		t.Errorf("Not OK: %s", rsp.Status)
		return
	}

	if msg == nil {
		t.Errorf("No message")
		return
	}

	if len(msg.GetHeader("From")) != 1 || msg.GetHeader("From")[0] != cfg.From {
		t.Errorf("Wrong from: %v", msg.GetHeader("From"))
	}
	if len(msg.GetHeader("To")) != 1 || msg.GetHeader("To")[0] != cfg.Recipients[0] {
		t.Errorf("Wrong to: %v", msg.GetHeader("To"))
	}
	if len(msg.GetHeader("Subject")) != 1 || msg.GetHeader("Subject")[0] != "Subject test" {
		t.Errorf("Wrong subject: %s", msg.GetHeader("Subject"))
	}
	out := bytes.Buffer{}
	msg.WriteTo(&out)
	if strings.Index(out.String(), "field1:") == -1 ||
		strings.Index(out.String(), "value1") == -1 {
		t.Errorf("Incorrect body: %s", out.String())
	}
}
