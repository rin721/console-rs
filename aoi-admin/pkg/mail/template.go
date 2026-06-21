package mail

import (
	"bytes"
	"text/template"
)

// Template 是不含业务语义的邮件文本模板。
type Template struct {
	Subject  string
	TextBody string
	HTMLBody string
}

// RenderedTemplate 是模板渲染后的邮件文本。
type RenderedTemplate struct {
	Subject  string
	TextBody string
	HTMLBody string
}

func (t Template) Render(data any) (RenderedTemplate, error) {
	subject, err := RenderText(t.Subject, data)
	if err != nil {
		return RenderedTemplate{}, err
	}
	textBody, err := RenderText(t.TextBody, data)
	if err != nil {
		return RenderedTemplate{}, err
	}
	htmlBody, err := RenderText(t.HTMLBody, data)
	if err != nil {
		return RenderedTemplate{}, err
	}
	return RenderedTemplate{Subject: subject, TextBody: textBody, HTMLBody: htmlBody}, nil
}

func RenderText(pattern string, data any) (string, error) {
	if pattern == "" {
		return "", nil
	}
	tpl, err := template.New("mail").Option("missingkey=error").Parse(pattern)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
