package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/html"
)

func pintrestTest() {
	url := "https://uk.pinterest.com/pin"
	pinId := 262475484528346906
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

	//fmt.Printf("%v data type: %v - attrs %v\n", data.FirstChild.Type, data.Data, data.Attr)

	traverseHTML(res.Body, data, "link", 0.0)

	/*
	* get the images
	* check the name of post
	* check if any metadata could be found
	*  ml? get what kind of garment from image
	* use info to run a search on ravelry
	* use image comparison to help get a selection of ravelry posts

	 */
}

func RavelryTest() {
	const APIUS = "read-da2035a6977ad37ae3438a84cd9c7a70"
	const APIKEY = "uOkEf1NzqTJ7EsSu4cLQjImGwKXOLNmZL4JvhtR4"

	url := "https://api.ravelry.com/patterns/search.json"

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s", url), nil)
	if err != nil {
		log.Print("client: could not create request", err)
		os.Exit(1)
	}
	params := req.URL.Query()
	params.Add("query", "hat")
	params.Add("weight", "cobweb")

	req.URL.RawQuery = params.Encode()

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(APIUS, APIKEY)
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
	data, err := io.ReadAll(res.Body)

	jsonData := make(map[string]interface{}, 0)

	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		log.Print("client: error unmarshalling json", err)
		os.Exit(1)
	}

	patterMap, _ := jsonData["patterns"].([]any)

	for _, items := range patterMap {

		if sItem, found := items.(map[string]any); found {
			if image, found := sItem["first_photo"].(map[string]any); found {
				fmt.Printf("query is %v, %v\n", req.URL.String(), image["medium2_url"])
			}
		}
	}

	//fmt.Printf("code: %v - data: %v\n", res.StatusCode, jsonData["patterns"])

}

func main() {

	RavelryTest()

}

func traverseHTML(body io.Reader, node *html.Node, datatype string, level float64) error {

	//fmt.Printf("level %f node %v --------- data %v ns %v\n", level, node.Data, node.Attr, node.Namespace)

	/*if node.Data == "script" {
		for _, attr := range node.Attr {
			child := node.FirstChild
			if attr.Key == "type" && attr.Val == "text/javascript" {
				fmt.Printf("level %f node %v --------- data %v ns %v\n", level, node.Attr, child.Data, node.Namespace)
				return nil

			}
		}
	}
	*/

	if node.Type == html.ElementNode && node.Data == datatype {

		fmt.Printf("level %f data type: %v - attrss %v\n", level, node.Data, node.Attr)

	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		traverseHTML(body, c, datatype, level+1.0)
	}

	return nil
}
