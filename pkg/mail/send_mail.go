// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public Licensee as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public Licensee for more details.
//
// You should have received a copy of the GNU Affero General Public Licensee
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package mail

import (
	"embed"
	"io"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/version"

	"github.com/wneessen/go-mail"
)

// Opts holds infos for a mail
type Opts struct {
	From        string
	To          string
	Subject     string
	Message     string
	HTMLMessage string
	ContentType ContentType
	Boundary    string
	Headers     []*header
	Embeds      map[string]io.Reader
	EmbedFS     map[string]*embed.FS
}

// ContentType represents mail content types
type ContentType int

// Enumerate all the team rights
const (
	ContentTypePlain ContentType = iota
	ContentTypeHTML
	ContentTypeMultipart
)

type header struct {
	Field   mail.Header
	Content string
}

// SendTestMail sends a test mail to a recipient.
// It works without a queue.
func SendTestMail(opts *Opts) error {
	if config.MailerHost.GetString() == "" {
		log.Warning("Mailer seems to be not configured! Please see the config docs for more details.")
		return nil
	}

	c, err := getClient()
	if err != nil {
		return err
	}

	m := getMessage(opts)

	return c.DialAndSend(m)
}

func getMessage(opts *Opts) *mail.Msg {
	m := mail.NewMsg()
	m.SetUserAgent("Vikunja " + version.Version)
	if opts.From == "" {
		opts.From = "Vikunja <" + config.MailerFromEmail.GetString() + ">"
	}
	_ = m.From(opts.From)
	_ = m.To(opts.To)
	m.Subject(opts.Subject)

	for _, h := range opts.Headers {
		m.SetGenHeader(h.Field, h.Content)
	}

	for name, content := range opts.Embeds {
		m.EmbedReader(name, content)
	}

	for name, fs := range opts.EmbedFS {
		err := m.EmbedFromEmbedFS(name, fs)
		if err != nil {
			log.Errorf("Error embedding %s via embed.FS into mail: %v", err)
		}
	}

	switch opts.ContentType {
	case ContentTypePlain:
		m.SetBodyString("text/plain", opts.Message)
	case ContentTypeHTML:
		m.SetBodyString("text/html", opts.Message)
	case ContentTypeMultipart:
		m.SetBodyString("text/plain", opts.Message)
		m.AddAlternativeString("text/html", opts.HTMLMessage)
	}

	return m
}

// SendMail puts a mail in the queue
func SendMail(opts *Opts) {
	if isUnderTest {
		sentMails = append(sentMails, opts)
		return
	}

	m := getMessage(opts)
	Queue <- m
}
