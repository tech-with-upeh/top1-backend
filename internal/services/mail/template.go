package mail

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed templates/verification.html
var emailTemplates embed.FS

type VerificationEmailData struct {
	FirstName       string
	OTP             string
	VerificationURL string
}

func RenderVerificationEmail(data VerificationEmailData) (string, error) {
	tmpl, err := template.ParseFS(
		emailTemplates,
		"templates/verification.html",
	)
	if err != nil {
		return "", err
	}

	var body bytes.Buffer

	if err := tmpl.Execute(&body, data); err != nil {
		return "", err
	}

	return body.String(), nil
}
