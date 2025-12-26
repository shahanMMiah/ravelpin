package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func main() {

	url := "https://uk.pinterest.com/pin"
	pinId := 1094937728148153742
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%v/", url, pinId), nil)
	if err != nil {
		log.Print("client: could not create request", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Authorization", "<access_token>")

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		log.Print("client: error making http request", err)
		os.Exit(1)
	}

	data, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Print("client: error making http request", err)
		os.Exit(1)
	}

	test := data.Find("img")

	log.Print(test)

}
