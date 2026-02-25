package test

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

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
