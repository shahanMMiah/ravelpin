package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/shahanmmiah/ravelpin/components"
	"github.com/shahanmmiah/ravelpin/internal/logging"
	"github.com/shahanmmiah/ravelpin/internal/recoginition"
	"github.com/shahanmmiah/ravelpin/internal/services"
)

// RAVELRY HANDLERS

func (cfg *ApiConfig) MiddleWareGetRavelLink(clasifyModel *recoginition.ImageClassifier) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		_, err := cfg.CheckJwtToken(req)

		if err != nil {
			slog.ErrorContext(req.Context(), fmt.Sprintf("jwt error %v", err.Error()))
			cfg.HandlerRefresh().ServeHTTP(resp, req)
			return
		}

		requestId := uuid.New()

		logObj, err := logging.MakeLoggerObject(requestId)
		if err != nil {
			slog.ErrorContext(req.Context(), fmt.Sprintf("could not log request, %v", err.Error()))

			resp.WriteHeader(http.StatusInternalServerError)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return

		}

		logger := slog.New(logObj.LogHandler)
		slog.SetDefault(logger)
		err = req.ParseForm()
		if err != nil {
			slog.InfoContext(req.Context(), fmt.Sprintf("could not parse form, %v", err.Error()))
			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return

		}

		pinLink := string(req.PostForm.Get("pinlink"))

		slog.InfoContext(req.Context(), fmt.Sprintf("looking at pinlink at %v", pinLink))

		imageLink, err := services.GetPinImageLink(pinLink)

		if err != nil {
			slog.InfoContext(req.Context(), fmt.Sprintf("could not find image links, %v", err.Error()))

			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return

		}

		// get links from database or classify seach
		slog.InfoContext(req.Context(), "tring to get hash")
		ravelPatterns, err := cfg.GetRavelHashes(imageLink, POSTAMOUNT)
		slog.InfoContext(req.Context(), fmt.Sprintf("hash - %v", ravelPatterns))

		if err != nil {
			slog.InfoContext(req.Context(), fmt.Sprintf("could not find ravelry hash in database, %v", err.Error()))

			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return

		}

		if len(ravelPatterns) == 0 {

			slog.InfoContext(req.Context(), "could not find image in database, trying clasify search")
			ravelPatterns, err = ImageClasifySearch(imageLink, clasifyModel)

			if err != nil {
				slog.InfoContext(req.Context(), fmt.Sprintf("could not find ravelry links, %v", err.Error()))

				resp.WriteHeader(http.StatusBadRequest)
				resp.Header().Set("Content-Type", "text/plain")

				resp.Write([]byte(err.Error()))
				return

			}

		}

		// render found posts

		htmlComponents := components.RavelPosts(ravelPatterns)
		w, err := MarhalComponent(htmlComponents)

		if err != nil {
			slog.ErrorContext(req.Context(), fmt.Sprintf("could not render html components, %v", err.Error()))

			resp.WriteHeader(http.StatusBadRequest)
			resp.Header().Set("Content-Type", "text/plain")

			resp.Write([]byte(err.Error()))
			return

		}

		resp.WriteHeader(http.StatusOK)
		resp.Header().Set("Content-Type", "text/html")
		resp.Write(w.Bytes())

		slog.InfoContext(req.Context(), "Request Complete", slog.String("URI", req.RequestURI), slog.String("input", req.FormValue("pinlink")))
		logObj.CloseLogFile()

	})
}

func ImageClasifySearch(link string, imageClasifier *recoginition.ImageClassifier) ([]services.RavelryPattern, error) {

	searchQueries, err := imageClasifier.GetClasifyLabels(link)

	ravPatterns, err := GetRavelPatterns(searchQueries, 10, 1)

	if err != nil {
		return []services.RavelryPattern{}, err
	}

	if len(ravPatterns) > 0 {
		slog.Info("comparing found ravelry posts to pintrest image...", slog.Any("postFound", ravPatterns))
		bestMatchPatterns, err := services.GetBestsImages(ravPatterns, link, POSTAMOUNT)

		if err != nil {
			return []services.RavelryPattern{}, err
		}

		return bestMatchPatterns, nil
	}

	return []services.RavelryPattern{}, fmt.Errorf("no ravelry post found")
}

func GetRavelPatterns(queries []string, pageSize, page int) ([]services.RavelryPattern, error) {
	godotenv.Load()
	APIUS := os.Getenv("RAVELRYAPIUS")
	APIKEY := os.Getenv("RAVELRYAPIKEY")

	url := os.Getenv("RAVELPATTERNSEARCH")

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s", url), nil)
	if err != nil {
		slog.Error(fmt.Sprintf("client: could not create request %v", err.Error()))
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
	params.Add("page_size", strconv.Itoa(pageSize))
	params.Add("page", strconv.Itoa(page))
	//params.Add("weight", "cobweb")

	req.URL.RawQuery = params.Encode()

	slog.InfoContext(req.Context(), fmt.Sprintf("params to search : %v", params.Encode()))

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(APIUS, APIKEY)
	//req.Header.Set("Authorization", "<access_token>")

	res, err := http.DefaultClient.Do(req)
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

	}

	return patterns, nil

}
