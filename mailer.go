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
	"fmt"
	"net/http"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"

	mail "gopkg.in/gomail.v2"
)

// Mailer handles form posts and sends mails
type Mailer struct {
	Send         func(*mail.Message) error
	ConfigGetter func(string) *FormCfg
}

type FormCfg struct {
	From                 string
	Domain               string
	Subject              string
	Recipients           []string
	AllowCustomRecipient bool
	Honeypot             string
	Body                 string
}

const DefaultBody = `
{{range $key, $value :=  .form -}}
{{if eq $key "ok" "err" "formId" "recipient" -}}
{{else -}}
{{$key}}: {{range $value -}}
{{.}}

{{end -}}
{{end -}}
{{end -}}
`

func eval(tmpl string, data interface{}) string {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		log.Errorf("Cannot parse template: %v", err)
	}
	out := bytes.Buffer{}
	err = t.Execute(&out, data)
	if err != nil {
		log.Errorf("Cannot execute template: %v", err)
	}
	return out.String()
}

// ServeHTTP processes POST requests
func (m Mailer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	fail := func(statusCode int, msg string, args ...interface{}) {
		log.Errorf(msg, args...)
		writer.WriteHeader(statusCode)
	}
	log.Debugf("Handling a request")
	if request.Method != http.MethodPost {
		fail(http.StatusMethodNotAllowed, "Rejecting %s", request.Method)
		return
	}
	err := request.ParseForm()
	if err != nil {
		fail(http.StatusBadRequest, "Cannot parse form: %v", err)
		return
	}

	form := request.Form
	log.Debugf("Html form: %v", form)

	succ := func() {}
	succUrl := form.Get("ok")
	if len(succUrl) != 0 {
		succ = func() { http.Redirect(writer, request, succUrl, http.StatusFound) }
		form.Del("ok")
	}

	errUrl := form.Get("err")
	if len(errUrl) != 0 {
		fail = func(statusCode int, msg string, args ...interface{}) {
			log.Warnf(msg, args...)
			http.Redirect(writer, request, errUrl, statusCode)
		}
		form.Del("err")
	}

	// Expect to see these variables in the submitted form
	formId := form.Get("formId")
	log.Debugf("Form: %s", formId)

	config := m.ConfigGetter(formId)
	if config == nil {
		fail(http.StatusNotFound, "No form %s", formId)
		return
	}

	recipient := form.Get("recipient")
	log.Debugf("Recipient: %s", recipient)
	if strings.Index(recipient, "@") != -1 {
		fail(http.StatusForbidden, "Invalid recipient: %s", recipient)
		return
	}
	if !config.AllowCustomRecipient && len(recipient) > 0 {
		fail(http.StatusForbidden, "Custom recipient not allowed")
		return
	}

	// Recipient cannot have @ in it. The domain for the recipient will
	// be read from formId, otherwise this becomes a spam engine
	if len(recipient) != 0 {
		recipient = fmt.Sprintf("%s@%s", recipient, config.Domain)
	}
	if len(recipient) == 0 && len(config.Recipients) == 0 {
		fail(http.StatusBadRequest, "No recipients")
		return
	}
	log.Debugf("Recipient: %s", recipient)

	if len(config.Honeypot) != 0 {
		if len(form.Get(config.Honeypot)) > 0 {
			fail(http.StatusNotAcceptable, "Non-empty honeypot")
			return
		}
	}

	data := map[string]interface{}{"config": config, "form": form}

	msg := mail.NewMessage()
	msg.SetHeader("From", config.From)
	to := append([]string{}, config.Recipients...)
	if len(recipient) > 0 {
		to = append(to, recipient)
	}
	msg.SetHeader("To", to...)
	if len(config.Subject) > 0 {
		msg.SetHeader("Subject", eval(config.Subject, data))
	} else {
		msg.SetHeader("Subject", fmt.Sprintf("From %s", formId))
	}

	bodyTemplate := DefaultBody
	if len(config.Body) > 0 {
		bodyTemplate = config.Body
	}
	body := eval(bodyTemplate, data)

	log.Debugf("Sending: %v", msg)
	log.Debugf("Body: %s", body)
	msg.SetBody("text/plain", body)
	err = m.Send(msg)
	if err != nil {
		fail(http.StatusInternalServerError, "Send error: %v", err)
	} else {
		log.Debugf("Sent")
		succ()
	}
}
