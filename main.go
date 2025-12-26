package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/html"
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

	defer res.Body.Close()

	data, err := html.Parse(res.Body)
	if err != nil {
		log.Print("client: error making http request", err)
		os.Exit(1)
	}

	traverseHTML(data)

}

func traverseHTML(node *html.Node) error {

	if node.Type == html.ElementNode && node.Data == "img" {
		fmt.Printf("data type: %v - attrs %v\n", node.Data, node.Attr)

	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		traverseHTML(c)
	}

	return nil
}
