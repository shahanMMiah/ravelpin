package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/shahanmmiah/ravelpin/repl/replCommand"
)

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

func MiddleWareGetRavelLink() http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		err := req.ParseForm()
		if err != nil {
			log.Println("could not parse form, %v", err.Error())

		}

		if err != nil {
			log.Println("could not parse form, %v", err.Error())

		}

		//log.Printf("post body: %v ", ))
		pinLink := string(req.PostForm.Get("pinlink"))
		ravelUrl, err := HandlerFindRavelFromPin(pinLink, replCommand.ImageSearch)

		if err != nil {
			log.Println("could not parse form, %v", err.Error())

		}

		log.Printf("\nRedirecting to %v", ravelUrl)

		http.Redirect(resp, req, ravelUrl, http.StatusSeeOther)
		//resp.WriteHeader(400)

	})
}

func HandlerFindRavelFromPin(link string, clasifyCommand func(string) ([]string, error)) (string, error) {

	pinLink, err := replCommand.GetPinImageLink(link)

	if err != nil {
		return "", err
	}

	searchQueries, err := clasifyCommand(link)

	ravPatterns, err := replCommand.GetRavelPatterns(searchQueries)
	fmt.Printf("found %v ravelry posts\n", len(ravPatterns))

	if err != nil {
		return "", err
	}

	if len(ravPatterns) > 0 {
		fmt.Println("comparing found ravelry posts to pintrest image...")
		bestMatchPattern, _ := replCommand.CompareRavelImages(ravPatterns, pinLink)
		ravelURL := fmt.Sprintf("https://www.ravelry.com/patterns/library/%s", bestMatchPattern.Permalink)

		fmt.Printf("closest match found: %s", ravelURL)
		//replCommand.OpenURL(ravelURL)
		return ravelURL, nil
	}

	return "", fmt.Errorf("no ravelry post found")
}
