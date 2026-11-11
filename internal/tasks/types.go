package tasks

const (
	QueueEmails = "emails"

	TypeSendEmail = "email:send"
)

type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	IsHTML  bool   `json:"is_html"`
}
