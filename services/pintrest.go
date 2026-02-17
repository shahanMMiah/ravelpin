package services

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/shahanmmiah/ravelpin/internal/recoginition"
	"golang.org/x/net/html"
)

func GetPinImageLink(pintrestURL string) (string, error) {

	data, err := ParsePinPage(pintrestURL)

	if err != nil {
		return "", err
	}

	link, err := traverseHTML(data, "link", 0.0, CheckFoundImgLink)

	if err == nil {
		return link, nil
	}

	return "", fmt.Errorf("no link found")

}

func ParsePinPage(pintrestURL string) (*html.Node, error) {
	req, err := http.NewRequest(http.MethodGet, pintrestURL, nil)

	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	//req.Header.Set("Authorization", "<access_token>")

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := html.Parse(res.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func traverseHTML(node *html.Node, datatype string, level float64, checkFunc func(node *html.Node) (string, bool)) (string, error) {

	if node.Type == html.ElementNode && node.Data == datatype {

		if link, found := checkFunc(node); found {
			return link, nil
		}

	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		child, err := traverseHTML(c, datatype, level+1.0, checkFunc)

		if err == nil {
			return child, nil
		}
	}

	return "", fmt.Errorf("no image link found")
}

func CheckFoundTitle(node *html.Node) (string, bool) {

	if node.FirstChild != nil {
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			fmt.Printf("child node %v", c.Data)
		}

		return string(node.FirstChild.Data), true

	}

	return "", false

}

func CheckFoundImgLink(node *html.Node) (string, bool) {

	imgNode := make(map[string]string, 0)

	for _, attr := range node.Attr {
		imgNode[attr.Key] = attr.Val
	}

	link, linkFound := imgNode["href"]
	_, idFound := imgNode["id"]
	as, asFound := imgNode["as"]

	if linkFound && idFound && asFound && as == "image" {
		return string(link), true
	}

	return "", false

}

func ImageClasify(link string, model *recoginition.ClassifyModel) ([]string, error) {

	pinLink, err := GetPinImageLink(link)

	if err != nil {
		return nil, err
	}

	clsItems := make([]string, 0)
	testRavelSearchTerms := map[string]any{
		"garment": []string{"sweater", "pancho", "cardigan", "trousers", "jean", "sock", "sweatshirt", "mitten"},
	}
	classified := model.ClassifyImage(pinLink)

	if len(classified) < 1 {
		return nil, fmt.Errorf("couldnt classify whats in the image :(")
	}
	fmt.Printf("classified labels found %v\n", classified)
	fmt.Println("best guesses of what is in pintrest image:")

	for _, cls := range classified {

		garmList, _ := testRavelSearchTerms["garment"].([]string)

		if slices.Contains(garmList, strings.ToLower(cls.Label)) {
			fmt.Printf("%v\n", cls.Label)
			clsItems = append(clsItems, cls.Label)

		}
	}

	return clsItems, nil
}
