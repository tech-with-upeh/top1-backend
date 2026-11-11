package main

import (
	"bytes"
	"fmt"
	"log"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"time"
)

func main() {
	from := "Acme <hello@mail.acme.com>"
	to := "user@example.com"
	subject := "Welcome to Acme"

	// Build a multipart/alternative body: plain text first, then HTML
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	textPart, _ := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=UTF-8"},
	})
	textPart.Write([]byte("Welcome! Thanks for signing up."))

	htmlPart, _ := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/html; charset=UTF-8"},
	})
	htmlPart.Write([]byte("<h1>Welcome!</h1><p>Thanks for signing up.</p>"))
	mw.Close()

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", from)
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&msg, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&msg, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", mw.Boundary())
	msg.Write(body.Bytes())

	//auth := smtp.PlainAuth("", "", "", "localhost:1025")

	var auth smtp.Auth

	err := smtp.SendMail(
		"localhost:1025",
		auth,
		"hello@mail.acme.com", // envelope sender: the bare address
		[]string{to},
		msg.Bytes(),
	)
	if err != nil {
		log.Fatal(err)
	}
}
