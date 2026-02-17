package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/a-h/templ"
	"github.com/joho/godotenv"
	"github.com/shahanmmiah/ravelpin/components"
	"github.com/shahanmmiah/ravelpin/handlers"
	"github.com/shahanmmiah/ravelpin/internal/ratelimit"
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

	link, err := traverseHTML(res.Body, data, "title", 0.0)

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
	//godotenv.Load()
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

func ravelParamTest() {

	APIUS := os.Getenv("RAVELRYAPIUS")
	APIKEY := os.Getenv("RAVELRYAPIKEY")

	url := "https://api.ravelry.com/pattern_attributes/groups.json"

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s", url), nil)
	if err != nil {
		log.Print("client: could not create request", err)
		os.Exit(1)
	}
	req.SetBasicAuth(APIUS, APIKEY)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Print("client: could not create request", err)
		os.Exit(1)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)

	if err != nil {
		log.Print("client: could not create request", err)
		os.Exit(1)
	}

	log.Println(string(data))

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

	matches := store.Query(trgHash)
	sort.Sort(matches)
	pattern, _ := matches[0].ID.(RavelryPattern)
	return pattern, nil

}

func traverseHTML(body io.Reader, node *html.Node, datatype string, level float64) (string, error) {

	//fmt.Printf("level %f node %v\n", level, node.Data)

	/*
		if node.Data == "script" {
			for _, attr := range node.Attr {
				child := node.FirstChild
				if attr.Key == "type" && attr.Val == "text/javascript" {
					fmt.Printf("level %f node %v --------- data %v ns %v\n", level, node.Attr, child.Data, node.Namespace)

				}
			}
		}
	*/

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		traverseHTML(body, c, datatype, level+1.0)

	}

	if node.Type == html.ElementNode && node.Data == datatype {

		if node.FirstChild != nil {

			fmt.Printf("level %f node %v\n", level, node.FirstChild.Data)
		}
	}
	/*


			fmt.Printf("level %f name %v data type: %v - attrss %v\n", level, *node, node.Data, node.Attr)


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



				return "", fmt.Errorf("no image link found")

		}
	*/

	return "", nil

}

type Server struct {
	Mux       *http.ServeMux
	RateLimit *ratelimit.RateLimiter
}

func (serv *Server) limitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		if !serv.RateLimit.GetClientRateLimit(ip) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		log.Printf("token left for you %v", serv.RateLimit.GetTokenAmount(ip))

		next.ServeHTTP(w, r)
	})
}

func (serv *Server) templTest(clasifyModel *recoginition.ClassifyModel) {

	fileServerHandle := handlers.MiddleWareServeFile("./assets/js/htmx.min.js")
	handlers.HandleHandler(serv.Mux, fileServerHandle, "/assets/js/htmx.min.js", "GET")

	component := components.HomePage()

	serv.Mux.Handle("/{$}", templ.Handler(component))

	fmt.Println("Listening on :3000")

	err := handlers.HandleHandler(serv.Mux, serv.limitMiddleware(handlers.MiddleWareGetRavelLink(clasifyModel)), "/link", "POST")
	if err != nil {
		log.Panicf("error handler %v", err.Error())
	}

	server := http.Server{
		Handler: serv.Mux,
		Addr:    ":3000",
	}

	server.ListenAndServe()

}

func main() {

	//ravelParamTest()
	//pintrestTest()
	godotenv.Load()
	imageClassifyModel := recoginition.NewModel()

	imageClassifyModel.LoadModel(fmt.Sprintf("./model/%s", os.Getenv("CLASSIFYMODELNAME")))

	server := Server{
		Mux:       http.NewServeMux(),
		RateLimit: ratelimit.NewRateLimiter(),
	}

	server.templTest(imageClassifyModel)

	/*
		testStr := "Seasalt Socks pattern by The Wool Barn Knits."

		err := recoginition.Ner(testStr)
		if err != nil {
			fmt.Println(err)
		}




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
	*/

}
