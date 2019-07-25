# Simple Form Mailer

Sends HTTP form submissions as emails. Useful for contact forms in
static sites.

## Building

Just use:

```
go build
```

## How it works

Postmail works with form configurations. A form configuration has the
recipients for the email, its subject, and an optional body
template. For instance, we have separate forms for "Contact us" in the
web page, "Request beta" for our beta, and a "Submit feedback" form
for logged in users.  For form configurations are stored in a
configuration file. Each form has a unique ID. In your static website,
you can create an HTML form containing any number of fields, with form
method=POST and action that is of the form:

```
 https://<postmail url>?formId=<id>&ok=<redirect url>&err=<error redirect url>
```

Postmail loads the form template using formId, builds an email based
on the configuration and sends it. If everything goes fine, it sends a
redirect to the "ok" URL. Otherwise, it sends a redirect to the "err"
URL with a query parameteter "msg=error message".

## Form Configuration

All forms are defined under "/forms". The configuration structure is:

```
forms:
  id1:
    <config>
  id2:
    <config>
```

Here, formId=id1 will select the first form, formId=id2 will select
the second, etc.

Each form has these elements:

```
forms:
  id1:
    from: string
    subject: template
    recipients:
    - recipient1 string
    - recipient 2 string
    honeypot: string
    body: template
    allowCustomRecipient: boolean
    domain: string
```

 * from: This is going to appear in the "from" field of the mail
   sent. This field is a string (not a template)
 * subject: This is the subject of the email. It is a go template.
 * recipients: This is the fixed set of recipient emails, in the form
   recipient@domain.com. If the form allows custom recipients, the
   email is sent to these recipients and the custom ones.
 * honeypot: Name of the honeypot field. If this field is nonempty,
   mail won't be sent.
 * body: The body template. If empty, the default body template is
   used that lists all fields with their values.
 * allowCustomRecipients: If this is true, then the form can specify
   recipient names, and the email will be sent to those people. The
   form **cannot** specify email domains, only names. The recipients
   are read from the "recipient" form field
 * domain: If custom recipients are allowed, this is their email
   domain.


## Templates

Mail body and subject fields are go templates that are evaluated using
the following data structure:

```
form: <HTML form>
config: <form config>
```

Note that all fields in the HTML form are string arrays. So, if you
want to print the subject, you have to do something like:

```
{{index .form.subject 0}}
```

which will print the first subject submitted.


The default body template is:

```
{{range $key, $value :=  .form -}}
{{if eq $key "ok" "err" "formId" "recipient" -}}
{{else -}}
{{$key}}: {{range $value -}}
{{.}}

{{end -}}
{{end -}}
{{end -}}
```

## Configuration

You can specify STMP server and HTTP properties in the command line,
or in the configuration file:

```
smtp-host: localhost
smtp-port: 25

http-port: 2048

debug: true
```

These are the configuration options:

  * cfg: The configuration file (required) (only command-line)
  * debug: true or false
  * smtp-host
  * smtp-port: default 587
  * smtp-user
  * smtp-pwd
  * smtp-cert: Certificate for SMTP TLS configuration
  * smtp-key: Certificate key for SMTP TLS configuration
  * smtp-ca: CA cert for SMTP TLS configuration
  * http-port: HTTP port, default 80
  * http-cert: Certificate for TLS configuration
  * http-key: Certificate key for HTTP TLS configuration
  * http-ca: CA cert for HTTP TLS configuration
  
