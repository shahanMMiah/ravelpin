package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/shahanmmiah/ravelpin/components"
	"github.com/shahanmmiah/ravelpin/internal/recoginition"
	"github.com/shahanmmiah/ravelpin/services"
)

const POSTAMOUNT = 5

func HandleHandler(mux *http.ServeMux, handle http.Handler, endpoint, mthd string) error {

	if endpoint == "" {
		return fmt.Errorf("%v Ns cannot be empty", endpoint)
	}
	if mthd == "" {
		return fmt.Errorf("%v Method cannot be empty", mthd)
	}

	mux.Handle(fmt.Sprintf("%s %s", mthd, endpoint), handle)

	return nil
}

func MiddleWareServeFile(file string) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		http.ServeFile(resp, req, file)
	})
}

func MiddleWareGetRavelLink(clasifyModel *recoginition.ClassifyModel) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		err := req.ParseForm()
		if err != nil {
			log.Printf("could not parse form, %v", err.Error())
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(err.Error()))
			return

		}

		//log.Printf("post body: %v ", ))
		pinLink := string(req.PostForm.Get("pinlink"))

		log.Printf("looking at pinlink %v", pinLink)
		ravelPatterns, err := HandlerFindRavelFromPins(pinLink, MiddleWareModelPreload(clasifyModel, services.ImageClasify))

		if err != nil {
			log.Printf("could not find ravelry links, %v", err.Error())

			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(err.Error()))
			return

		}

		htmlComponents := components.RavelPosts(ravelPatterns)
		w := new(bytes.Buffer)

		err = htmlComponents.Render(context.Background(), w)
		if err != nil {
			log.Printf("could not render html components, %v", err.Error())

			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(err.Error()))
			return

		}
		log.Printf("%v", w.String())

		resp.Header().Set("Content-Type", "text/html")
		resp.Write([]byte(w.String()))
		//templ.Handler(htmlComponents).ServeHTTP(resp, req)
		//http.Redirect(resp, req, ravelUrl, http.StatusSeeOther)
		//resp.WriteHeader(400)

	})
}

func MiddleWareModelPreload(model *recoginition.ClassifyModel, mdl func(string, *recoginition.ClassifyModel) ([]string, error)) func(string) ([]string, error) {
	return func(s string) ([]string, error) {
		return mdl(s, model)
	}
}

func HandlerFindRavelFromPins(link string, clasifyCommand func(string) ([]string, error)) ([]services.RavelryPattern, error) {

	pinLink, err := services.GetPinImageLink(link)

	if err != nil {
		return []services.RavelryPattern{}, err
	}

	searchQueries, err := clasifyCommand(link)

	ravPatterns, err := GetRavelPatterns(searchQueries)
	fmt.Printf("found %v ravelry posts\n", len(ravPatterns))

	if err != nil {
		return []services.RavelryPattern{}, err
	}

	if len(ravPatterns) > 0 {
		fmt.Println("comparing found ravelry posts to pintrest image...")
		bestMatchPatterns, err := services.GetBestsImages(ravPatterns, pinLink, POSTAMOUNT)

		if err != nil {
			return []services.RavelryPattern{}, err
		}

		//ravelURL := fmt.Sprintf("%v%s", os.Getenv("RAVELURL"), bestMatchPattern.Permalink)

		//fmt.Printf("closest match found: %s", ravelURL)
		//replCommand.OpenURL(ravelURL)
		return bestMatchPatterns, nil
	}

	return []services.RavelryPattern{}, fmt.Errorf("no ravelry post found")
}

func GetRavelPatterns(queries []string) ([]services.RavelryPattern, error) {
	godotenv.Load()
	APIUS := os.Getenv("RAVELRYAPIUS")
	APIKEY := os.Getenv("RAVELRYAPIKEY")

	url := os.Getenv("RAVELPATTERNSEARCH")

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s", url), nil)
	if err != nil {
		log.Print("client: could not create request", err)
		os.Exit(1)
	}
	params := req.URL.Query()

	queryString := ""
	for num, query := range queries {
		if num == 0 {
			queryString += fmt.Sprintf("%s", query)
		} else {
			queryString += fmt.Sprintf("&%s", query)
		}

		if num == 0 {
		}

	}
	params.Add("query", queryString)
	params.Add("page_size", "10")
	//params.Add("weight", "cobweb")

	req.URL.RawQuery = params.Encode()

	log.Printf("params to search : %v", params.Encode())

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(APIUS, APIKEY)
	//req.Header.Set("Authorization", "<access_token>")

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return []services.RavelryPattern{}, err

	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)

	jsonData := make(map[string]interface{}, 0)

	//fmt.Println(string(data))

	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return []services.RavelryPattern{}, err

	}

	patterMap, _ := jsonData["patterns"].([]any)
	patterns := make([]services.RavelryPattern, 0)
	for _, items := range patterMap {

		pattern := services.RavelryPattern{}

		patternData, err := json.Marshal(items)
		if err != nil {
			return []services.RavelryPattern{}, err

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

	return patterns, nil
	//fmt.Printf("code: %v - data: %v\n", res.StatusCode, jsonData["patterns"])

}
