package handlers

import (
	"fmt"
	"net/smtp"
	"os"
)

func sendVerifyEmail(email, passHash, verfiyCode, url string) error {

	auth := smtp.PlainAuth("", os.Getenv("EMAILUSER"), os.Getenv("EMAILPASS"), "smtp.gmail.com")

	to := []string{email}

	subject := "Subject: RavelPin Verify Code \n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	body := fmt.Sprintf(`<html>
		<body>
			<h1 style="color: blue;">Welcome to Ravelpin!!</h1>
			<p> Use this link to Verify your account:</p>
			<br>
			<p style="font-family: 'Courier New', Courier, monospace; font-size: 18px;">
				<a href=%s/verify?email=%s&passhash=%s&vercode=%s> Verify Link</a>
			</p>
			<br>
			<p> This code expires in 10 mins </p>

		</body>
	</html>`, url, email, passHash, verfiyCode)

	msg := []byte(subject + mime + body)

	err := smtp.SendMail(os.Getenv("SMPTSERVER"), auth, os.Getenv("EMAILUSER"), to, msg)

	if err != nil {
		return err
	}

	return nil
}
