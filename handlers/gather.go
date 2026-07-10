package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/shahanmmiah/ravelpin/internal/services"
)

func (cfg *ApiConfig) SearchYnRavelPosts(
	yarnWeights []services.YarnWeight,
	pageNum,
	pagesize int,
	idsChan chan []int,
	postChan chan []services.RavelryPatternFull,
	wgtMap map[string]string,
	cmbMap map[string][]string) {

	for _, yn := range yarnWeights {

		go GetherSearchYnPosts(strings.ReplaceAll(yn.Name, " ", "-"), pageNum, pagesize, idsChan)

		go GetRavelryPatternFull(idsChan, postChan)

		go DownloadImages(postChan, wgtMap, cmbMap)
	}
}

func GetherSearchYnPosts(yn string, page, pageSize int, ch chan []int) {

	godotenv.Load()
	APIUS := os.Getenv("RAVELRYAPIUS")
	APIKEY := os.Getenv("RAVELRYAPIKEY")
	ids := []int{}
	url := os.Getenv("RAVELPATTERNSEARCH")

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s", url), nil)

	if err != nil {
		slog.Error(err.Error())
		ch <- ids
	}

	params := req.URL.Query()
	params.Add("query", "sweater")
	params.Add("page_size", strconv.Itoa(pageSize))
	params.Add("page", strconv.Itoa(page))
	params.Add("weight", yn)
	params.Add("sort", "popularity")

	req.URL.RawQuery = params.Encode()

	slog.InfoContext(req.Context(), fmt.Sprintf("params to search : %v", req.URL.RawQuery))

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(APIUS, APIKEY)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Info(err.Error())
		ch <- ids

	}
	if res.StatusCode < 400 {

		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)

		jsonData := make(map[string]interface{}, 0)

		//slog.InfoContext(req.Context(), string(data))

		err = json.Unmarshal(data, &jsonData)
		if err != nil {
			slog.Info(err.Error())
			ch <- ids

		}

		patterMap, _ := jsonData["patterns"].([]any)

		for _, items := range patterMap {
			ids = append(ids, int(items.(map[string]interface{})["id"].(float64)))
		}

		ch <- ids
	}
}

func (cfg *ApiConfig) GatherRavelPosts(inc, amount int) {

	for amt := inc; amt <= amount; amt += inc {
		idsChan := make(chan []int, amount-inc)
		postChan := make(chan []services.RavelryPatternFull, amount-inc)

		go GetRavelIds(amt, amt-inc, idsChan)

		go GetRavelryPatternFull(idsChan, postChan)

		go cfg.AddRavelhash(postChan)

	}

}

func GetRavelIds(amount, startNum int, ch chan []int) {
	godotenv.Load()
	APIUS := os.Getenv("RAVELRYAPIUS")
	APIKEY := os.Getenv("RAVELRYAPIKEY")
	ids := make([]int, 0)

	url := os.Getenv("RAVELPATTERNSEARCH")
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s", url), nil)

	if err != nil {
		slog.Error(fmt.Sprintf("client: could not create request %v", err.Error()))
		ch <- ids
		return
	}
	params := req.URL.Query()

	queryString := ""

	for num := startNum; num < amount; num += 1 {
		if num == startNum {
			queryString += fmt.Sprintf("%s", strconv.Itoa(num))
		} else {
			queryString += fmt.Sprintf("|%s", strconv.Itoa(num))
		}

	}

	params.Add("pattern-id", queryString)
	req.URL.RawQuery = params.Encode()

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(APIUS, APIKEY)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error(fmt.Sprintf("error doing request: %s", err.Error()))
		ch <- ids
		return

	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error(fmt.Sprintf("error reading response body: %s", err.Error()))
		ch <- ids
		return
	}

	jsonData := make(map[string]interface{}, 0)

	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		slog.Error(fmt.Sprintf("error unmarshalling data: %s", err.Error()))
		ch <- ids
		return

	}

	patterMap, _ := jsonData["patterns"].([]any)

	for _, items := range patterMap {
		ids = append(ids, int(items.(map[string]interface{})["id"].(float64)))

	}

	ch <- ids
}

func GetRavelryPatternFull(idCh chan []int, pstCh chan []services.RavelryPatternFull) {
	godotenv.Load()
	APIUS := os.Getenv("RAVELRYAPIUS")
	APIKEY := os.Getenv("RAVELRYAPIKEY")

	url := os.Getenv("RAVELIDSEARCH")
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s", url), nil)
	patterns := make([]services.RavelryPatternFull, 0)

	ids := <-idCh

	if err != nil {
		slog.Error(fmt.Sprintf("client: could not create request %v", err.Error()))
		pstCh <- patterns
		return
	}
	params := req.URL.Query()

	queryString := ""

	for _, num := range ids {
		if num == 0 {
			queryString += fmt.Sprintf("%s", strconv.Itoa(num))
		} else {
			queryString += fmt.Sprintf(" %s", strconv.Itoa(num))
		}

	}

	params.Add("ids", queryString)
	req.URL.RawQuery = params.Encode()

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(APIUS, APIKEY)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error(fmt.Sprintf("error doing request: %s", err.Error()))
		pstCh <- patterns
		return
	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error(fmt.Sprintf("error parsing resp data: %s", err.Error()))
		pstCh <- patterns
		return
	}

	jsonData := make(map[string]interface{}, 0)

	//fmt.Println(res.Status)

	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		slog.Error(fmt.Sprintf("error unmarshalling data: %s", err.Error()))
		pstCh <- patterns
		return
	}

	patterMap, _ := jsonData["patterns"].(map[string]any)
	for _, items := range patterMap {

		pattern := services.RavelryPatternFull{}

		patternData, err := json.Marshal(items)
		if err != nil {
			slog.Error(err.Error())
			continue
		}
		json.Unmarshal(patternData, &pattern)
		slog.Info(fmt.Sprintf("ravel pattern found %v of yn %v", pattern.Name, pattern.YarnWeights.Name))

		patterns = append(patterns, pattern)

	}

	pstCh <- patterns
}

func GetRavelYarnWeights() ([]services.YarnWeight, error) {

	APIUS := os.Getenv("RAVELRYAPIUS")
	APIKEY := os.Getenv("RAVELRYAPIKEY")

	url := "https://api.ravelry.com/yarn_weights.json"

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s", url), nil)
	if err != nil {
		slog.Info(fmt.Sprintf("client: could not create request", err))
		return nil, err
	}
	req.SetBasicAuth(APIUS, APIKEY)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Info(fmt.Sprintf("client: could not create request", err))
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)

	if err != nil {
		slog.Info(fmt.Sprintf("client: could not create request", err))
		return nil, err
	}

	//fmt.Println(string(data))

	yarnWeights := make([]services.YarnWeight, 0)
	jsonData := make(map[string][]any, 0)
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		slog.Info(fmt.Sprintf("client: could not parse json data", err))
		return nil, err
	}

	weightData, err := json.Marshal(jsonData["yarn_weights"])
	if err != nil {
		slog.Info(fmt.Sprintf("client: could not parse json data", err))
		return nil, err
	}

	err = json.Unmarshal(weightData, &yarnWeights)

	if err != nil {
		slog.Info(fmt.Sprintf("client: could not parse yarn weight data", err))
		return nil, err
	}
	//fmt.Println(yarnWeights)

	return yarnWeights, nil
}

func DownloadImages(postChan chan []services.RavelryPatternFull, weightMap map[string]string, cmbMap map[string][]string) {

	post := <-postChan
	//<-postChan

	for _, pst := range post {
		for num, phto := range pst.Photos {
			grp := strings.ReplaceAll(pst.YarnWeights.Name, " ", "-")
			for grpNme, mp := range cmbMap {
				for _, nme := range mp {
					if grp == nme {
						grp = grpNme
					}
				}
			}

			if _, exists := weightMap[grp]; exists && phto.MediumURL != "" {

				nme := fmt.Sprintf("%s/%s_%d.jpg",
					grp, SanatizeName(pst.Name), num)
				nme = strings.ReplaceAll(nme, " ", "_")

				fmt.Printf("saving file %s\n", nme)
				imgPath := fmt.Sprintf("./assets/datasets/yarnweights/%s", nme)

				err := downloadImage(phto.MediumURL, imgPath)

				if err != nil {
					fmt.Println(err)
					continue
				}

				fileInfo, err := os.Stat(imgPath)
				if err != nil || fileInfo.Size() <= 0 {
					fmt.Println(fmt.Errorf("removing %s", imgPath).Error())
					if fileExists(imgPath) {
						os.Remove(imgPath)
					}
				} else {
					fmt.Printf("%s size is %v\n", imgPath, fileInfo.Size())
				}
			}
		}
	}

}

func SanatizeName(name string) string {
	sanName := ""
	for _, char := range name {
		if (char >= 48 && char <= 57) ||
			(char >= 65 && char <= 90) ||
			(char >= 97 && char <= 122) {
			sanName += string(char)
		}
	}

	return sanName
}

func downloadImage(url, filepath string) error {

	isImg, err := CheckIfImage(url)

	if !isImg || err != nil {

		return fmt.Errorf("issue with image link %s\n", url)
	}

	resp, err := http.Get(url)

	if err != nil {
		return err
	}

	if exst := fileExists(filepath); !exst {

		out, err := os.Create(filepath)
		if err != nil {
			return err
		}
		defer out.Close()

		img, err := jpeg.Decode(resp.Body)
		if err != nil {
			os.Remove(filepath)
			return err
		}

		err = jpeg.Encode(out, img, &jpeg.Options{Quality: 90})
		if err != nil {
			os.Remove(filepath)
			return err
		}

		return nil
	}

	return nil

}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true // File exists
	}
	if errors.Is(err, os.ErrNotExist) {
		return false // File does not exist
	}

	return false
}
