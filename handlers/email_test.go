package handlers

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestSendEmail(t *testing.T) {
	godotenv.Load("../.env")
	err := sendVerifyEmail(os.Getenv("EMAILUSER"), "testPass", "TESTCODE", "url")

	if err != nil {
		t.Errorf("error sending email %s", err.Error())
	}
}
