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

package notifications

import (
	"bytes"
	"embed"
	_ "embed"
	templatehtml "html/template"
	templatetext "text/template"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/mail"
	"code.vikunja.io/api/pkg/utils"

	"github.com/yuin/goldmark"
)

const mailTemplatePlain = `
{{ .Greeting }}
{{ range $line := .IntroLines}}
{{ $line }}
{{ end }}
{{ if .ActionURL }}{{ .ActionText }}:
{{ .ActionURL }}{{end}}
{{ range $line := .OutroLines}}
{{ $line }}
{{ end }}`

const mailTemplateHTML = `
<!doctype html>
<html style="width: 100%; height: 100%; padding: 0; margin: 0;">
<head>
    <meta name="viewport" content="width: display-width;">
</head>
<body style="width: 100%; padding: 0; margin: 0; background: #f3f4f6">
<div style="width: 100%; font-family: 'Open Sans', sans-serif; text-rendering: optimizeLegibility">
    <div style="width: 600px; margin: 0 auto; text-align: justify;">
        <h1 style="font-size: 30px; text-align: center;">
            <img src="cid:logo.png" style="height: 75px;" alt="Vikunja"/>
        </h1>
        <div style="border: 1px solid #dbdbdb; -webkit-box-shadow: 0.3em 0.3em 0.8em #e6e6e6; box-shadow: 0.3em 0.3em 0.8em #e6e6e6; color: #4a4a4a; padding: 5px 25px; border-radius: 3px; background: #fff;">
<p>
	{{ .Greeting }}
</p>

{{ range $line := .IntroLinesHTML}}
	{{ $line }}
{{ end }}

{{ if .ActionURL }}
	<a href="{{ .ActionURL }}" title="{{ .ActionText }}"
		style="position: relative;text-decoration:none;display: block;border-radius: 4px;cursor: pointer;padding-bottom: 8px;padding-left: 14px;padding-right: 14px;padding-top: 8px;width:280px;margin:10px auto;text-align: center;white-space: nowrap;border: 0;text-transform: uppercase;font-size: 14px;font-weight: 700;-webkit-box-shadow: 0 3px 6px rgba(107,114,128,.12),0 2px 4px rgba(107,114,128,.1);box-shadow: 0 3px 6px rgba(107,114,128,.12),0 2px 4px rgba(107,114,128,.1);background-color: #1973ff;border-color: transparent;color: #fff;">
		{{ .ActionText }}
	</a>
{{end}}

{{ range $line := .OutroLinesHTML}}
	{{ $line }}
{{ end }}

{{ if .ActionURL }}
	<p style="color: #9CA3AF;font-size:12px;border-top: 1px solid #dbdbdb;margin-top:20px;padding-top:20px;">
		If the button above doesn't work, copy the url below and paste it in your browser's address bar:<br/>
		{{ .ActionURL }}
	</p>
{{ end }}
</div>
</div>
</div>
</body>
</html>
`

//go:embed logo.png
var logo embed.FS

// RenderMail takes a precomposed mail message and renders it into a ready to send mail.Opts object
func RenderMail(m *Mail) (mailOpts *mail.Opts, err error) {

	var htmlContent bytes.Buffer
	var plainContent bytes.Buffer

	plain, err := templatetext.New("mail-plain").Parse(mailTemplatePlain)
	if err != nil {
		return nil, err
	}

	html, err := templatehtml.New("mail-plain").Parse(mailTemplateHTML)
	if err != nil {
		return nil, err
	}

	boundary := "np" + utils.MakeRandomString(13)

	data := make(map[string]interface{})

	data["Greeting"] = m.greeting
	data["IntroLines"] = m.introLines
	data["OutroLines"] = m.outroLines
	data["ActionText"] = m.actionText
	data["ActionURL"] = m.actionURL
	data["Boundary"] = boundary
	data["FrontendURL"] = config.ServiceFrontendurl.GetString()

	var introLinesHTML []templatehtml.HTML
	for _, line := range m.introLines {
		md := []byte(templatehtml.HTMLEscapeString(line))
		var buf bytes.Buffer
		err = goldmark.Convert(md, &buf)
		if err != nil {
			return nil, err
		}
		//#nosec - the html is escaped few lines before
		introLinesHTML = append(introLinesHTML, templatehtml.HTML(buf.String()))
	}
	data["IntroLinesHTML"] = introLinesHTML

	var outroLinesHTML []templatehtml.HTML
	for _, line := range m.outroLines {
		md := []byte(templatehtml.HTMLEscapeString(line))
		var buf bytes.Buffer
		err = goldmark.Convert(md, &buf)
		if err != nil {
			return nil, err
		}
		//#nosec - the html is escaped few lines before
		outroLinesHTML = append(outroLinesHTML, templatehtml.HTML(buf.String()))
	}
	data["OutroLinesHTML"] = outroLinesHTML

	err = plain.Execute(&plainContent, data)
	if err != nil {
		return nil, err
	}
	err = html.Execute(&htmlContent, data)
	if err != nil {
		return nil, err
	}

	mailOpts = &mail.Opts{
		From:        m.from,
		To:          m.to,
		Subject:     m.subject,
		ContentType: mail.ContentTypeMultipart,
		Message:     plainContent.String(),
		HTMLMessage: htmlContent.String(),
		Boundary:    boundary,
		EmbedFS: map[string]*embed.FS{
			"logo.png": &logo,
		},
	}

	return mailOpts, nil
}
