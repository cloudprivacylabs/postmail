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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	mail "gopkg.in/gomail.v2"
)

func main() {
	flag.String("cfg", "", "Configuration file")

	flag.Bool("debug", false, "Print debug level logs")

	flag.String("smtp-host", "", "SMTP server host")
	flag.Int("smtp-port", 587, "SMTP server port")
	flag.String("smtp-user", "", "User name")
	flag.String("smtp-pwd", "", "Password")
	flag.String("smtp-cert", "", "Certificate for SMTP TLS configuration")
	flag.String("smtp-key", "", "Certificate key for SMTP TLS configuration")
	flag.String("smtp-ca", "", "CA cert for SMTP TLS configuration")

	flag.Int("http-port", 80, "HTTP port")
	flag.String("http-cert", "", "Certificate for HTTP TLS configuration")
	flag.String("http-key", "", "Certificate key for HTTP TLS configuration")
	flag.String("http-ca", "", "CA cert for HTTP TLS configuration")

	flag.Parse()
	viper.BindPFlags(flag.CommandLine)

	viper.SetEnvPrefix("POSTMAIL")
	viper.AutomaticEnv()

	if viper.GetBool("debug") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if len(viper.GetString("cfg")) > 0 {
		viper.SetConfigFile(viper.GetString("cfg"))
		if err := viper.ReadInConfig(); err != nil {
			panic(err)
		}
		log.Debugf("Read config file")
	}

	if len(viper.GetString("smtp-host")) == 0 ||
		viper.GetInt("smtp-port") == 0 {
		flag.Usage()
		return
	}

	smtpTLS, err := configureTLS(viper.GetString("smtp-cert"), viper.GetString("smtp-key"), viper.GetString("smtp-ca"))
	if err != nil {
		panic(err)
	}

	smtpDialer := mail.NewDialer(viper.GetString("smtp-host"), viper.GetInt("smtp-port"), viper.GetString("smtp-user"), viper.GetString("smtp-pwd"))

	log.Debugf("Dialer: %+v", smtpDialer)

	if smtpTLS != nil {
		smtpDialer.TLSConfig = smtpTLS
	}

	httpTLS, err := configureTLS(viper.GetString("http-cert"), viper.GetString("http-key"), viper.GetString("http-ca"))
	if err != nil {
		panic(err)
	}

	if httpTLS != nil {
		httpTLS.MinVersion = tls.VersionTLS12
		httpTLS.CurvePreferences = []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256}
		httpTLS.PreferServerCipherSuites = true
		httpTLS.CipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		}
	}

	configGetter := func(ID string) *FormCfg {
		err := viper.ReadInConfig()
		if err != nil {
			log.Errorf("Cannot re-read config: %v", err)
		}
		prefix := fmt.Sprintf("forms.%s", ID)
		if len(viper.GetString(prefix+".domain")) == 0 {
			log.Warnf("Not in config: %s", prefix)
			return nil
		}
		return &FormCfg{Domain: viper.GetString(prefix + ".domain"),
			From:                 viper.GetString(prefix + ".from"),
			Subject:              viper.GetString(prefix + ".subject"),
			Recipients:           viper.GetStringSlice(prefix + ".recipients"),
			AllowCustomRecipient: viper.GetBool(prefix + ".allowCustomRecipient"),
			Body:                 viper.GetString(prefix + ".body"),
			Honeypot:             viper.GetString(prefix + ".honeypot")}
	}

	server := http.Server{Addr: fmt.Sprintf(":%d", viper.GetInt("http-port")),
		Handler: Mailer{Send: func(m *mail.Message) error {
			return smtpDialer.DialAndSend(m)
		}, ConfigGetter: configGetter},
		TLSConfig:    httpTLS,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0)}

	log.Debugf("Starting listener")
	if httpTLS != nil {
		panic(server.ListenAndServeTLS("", ""))
	} else {
		panic(server.ListenAndServe())
	}
}

// Loads certs and builds tls config. May return nil if there is no
// certs
func configureTLS(cert, key, ca string) (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	hasCfg := false
	if len(cert) > 0 {
		cert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		hasCfg = true
	}

	if len(ca) > 0 {
		caCert, err := ioutil.ReadFile(ca)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = pool
		hasCfg = true
	}
	if hasCfg {
		return tlsConfig, nil
	}
	return nil, nil
}
