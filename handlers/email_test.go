package handlers

import (
	"testing"

	"github.com/joho/godotenv"
)

func TestSendEmail(t *testing.T) {
	godotenv.Load("../.env")
	err := sendVerifyEmail("", "TESTCODE")

	if err != nil {
		t.Errorf("error sending email %s", err.Error())
	}
}
