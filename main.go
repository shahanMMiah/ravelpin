package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/shahanmmiah/ravelpin/internal/recoginition"
	"golang.org/x/net/html"
)

type RavelPhoto struct {
	MediumURL string `json:"medium_url"`
}

type RavelryPattern struct {
	Id         int        `json:"id"`
	Name       string     `json:"name"`
	Permalink  string     `json:"permalink"`
	FirstPhoto RavelPhoto `json:"first_photo"`
}

func pintrestTest() string {
	url := "https://uk.pinterest.com/pin"
	pinId := 815573814800236964
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

	link, err := traverseHTML(res.Body, data, "link", 0.0)

	if err == nil {
		return link
	}

	return ""
	/*
	* get the images
	* check the name of post
	* check if any metadata could be found
	*  ml? get what kind of garment from image
	* use info to run a search on ravelry
	* use image comparison to help get a selection of ravelry posts

	 */

}

func RavelryTest(query string) []RavelryPattern {
	godotenv.Load()
	APIUS := os.Getenv("RAVELRYAPIUS")
	APIKEY := os.Getenv("RAVELRYAPIKEY")

	url := "https://api.ravelry.com/patterns/search.json"

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s", url), nil)
	if err != nil {
		log.Print("client: could not create request", err)
		os.Exit(1)
	}
	params := req.URL.Query()
	params.Add("query", query)
	params.Add("page_size", "10")
	//params.Add("weight", "cobweb")

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

	//fmt.Println(string(data))

	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		log.Print("client: error unmarshalling json", err)
		os.Exit(1)
	}

	patterMap, _ := jsonData["patterns"].([]any)
	patterns := make([]RavelryPattern, 0)
	for _, items := range patterMap {

		pattern := RavelryPattern{}

		patternData, err := json.Marshal(items)
		if err != nil {
			log.Print("client: error re marshalling json", err)
			os.Exit(1)
		}

		json.Unmarshal(patternData, &pattern)

		patterns = append(patterns, pattern)

		/*if sItem, found := items.(map[string]any); found {
			fmt.Println(sItem)

			if image, found := sItem["first_photo"].(map[string]any); found {
				fmt.Printf("item is  is %v, %v\n", image,image["medium2_url"])
			}

		}
		*/
	}

	return patterns
	//fmt.Printf("code: %v - data: %v\n", res.StatusCode, jsonData["patterns"])

}

func CompareRavelImages(patterns []RavelryPattern, trgpath string) (RavelryPattern, error) {

	store := recoginition.CreateStore()

	trgHash, err := recoginition.CreateHash(trgpath)
	if err != nil {
		return RavelryPattern{}, err

	}
	for _, pattern := range patterns {
		recoginition.AddToStore(store, pattern, pattern.FirstPhoto.MediumURL)

	}

	testPhoto := RavelPhoto{MediumURL: "https://images4-f-cdn.ravelrycache.com/uploads/RenardeEndormie/928625511/IMG-1677_medium.jpg"}
	test := RavelryPattern{Id: 1234, Name: "sunflowerSock", Permalink: "sunflower-fields-socks", FirstPhoto: testPhoto}
	recoginition.AddToStore(store, test, test.FirstPhoto.MediumURL)

	matches := store.Query(trgHash)
	sort.Sort(matches)
	pattern, _ := matches[0].ID.(RavelryPattern)
	return pattern, nil

}

func traverseHTML(body io.Reader, node *html.Node, datatype string, level float64) (string, error) {

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

		//fmt.Printf("level %f data type: %v - attrss %v\n", level, node.Data, node.Attr)

		imgNode := make(map[string]string, 0)

		for _, attr := range node.Attr {
			imgNode[attr.Key] = attr.Val
		}

		link, linkFound := imgNode["href"]
		_, idFound := imgNode["id"]
		as, asFound := imgNode["as"]

		if linkFound && idFound && asFound && as == "image" {
			return string(link), nil
		}

	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		child, err := traverseHTML(body, c, datatype, level+1.0)

		if err == nil {
			return child, nil
		}
	}

	return "", fmt.Errorf("no image link found")
}

func main() {

	link := pintrestTest()
	//testlink := "https://i.pinimg.com/736x/06/78/c1/0678c12bb5acb9d93854013af00613a0.jpg"
	if link != "" {
		fmt.Println(link)

		testRavelSearchTerms := map[string]any{
			"garment": []string{"sweater", "pancho", "cardigan", "trousers", "jean", "sock"},
		}
		classified := recoginition.ClasifyImageTest(link)

		fmt.Printf("found %v\n", classified)
		fmt.Println("best guesses are:")

		for _, cls := range classified {

			garmList, _ := testRavelSearchTerms["garment"].([]string)

			if slices.Contains(garmList, strings.ToLower(cls.Label)) {
				fmt.Printf("%v\n", cls.Label)
				ravPatterns := RavelryTest(cls.Label)

				bestMatchPattern, _ := CompareRavelImages(ravPatterns, link)
				fmt.Println(bestMatchPattern)
			}
		}

	}

	//recoginition.TestPrint()

}
