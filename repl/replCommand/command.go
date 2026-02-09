package replCommand

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/shahanmmiah/ravelpin/internal/recoginition"
	"golang.org/x/net/html"
)

type Command struct {
	Name string
	Args []string
}

type CommandMap map[string]func(Command) error

type Commands struct {
	Cmds  CommandMap
	Helps map[string]string
}

func (cmds *Commands) Register(name, help string, f func(Command) error) error {

	_, exists := cmds.Cmds[name]
	if exists {
		return fmt.Errorf("cannot Register %s, already exists", name)
	}

	cmds.Cmds[name] = f
	cmds.Helps[name] = help

	return nil
}

func CreateCommand(scanner *bufio.Scanner) (Command, error) {

	scanner.Scan()
	words := cleanInput(scanner.Text())

	if len(words) < 1 {
		err := fmt.Errorf("Did not write a valid command..")
		return Command{}, err
	}

	cmd := Command{Name: words[0]}

	if len(words) > 1 {
		cmd.Args = words[1:]
	}

	return cmd, nil
}

func (cmds *Commands) Run(cmd Command) error {

	funcName, exists := cmds.Cmds[cmd.Name]
	if !exists {
		return fmt.Errorf("command does not exists")
	}
	err := funcName(cmd)
	if err != nil {
		return err
	}

	return nil
}

func cleanInput(text string) []string {
	sliced := strings.Fields(text)

	for num := range sliced {
		sliced[num] = strings.ToLower(sliced[num])
	}

	return sliced
}

func HandlerQuit(cmd Command) error {
	fmt.Println("exiting repl, bye for now :)")
	os.Exit(0)
	return nil
}

func HandlerHelp(cmd Command, cmds Commands) error {
	fmt.Println("commands for tool:")
	for coms, help := range cmds.Helps {
		fmt.Printf("\t %v: %v\n", coms, help)
	}
	return nil

}

func MiddleWareHelp(handler func(c Command, cmd Commands) error, cmds Commands) func(Command) error {

	return func(cmd Command) error {
		return HandlerHelp(cmd, cmds)
	}
}

func HandlerFindRavelFromPin(cmd Command, clasifyCommand func(string) ([]string, error)) error {

	if len(cmd.Args) < 1 {
		return fmt.Errorf("Need to specify a link")
	}

	link := cmd.Args[0]
	pinLink, err := GetPinImageLink(link)

	if err != nil {
		return err
	}

	searchQueries, err := clasifyCommand(link)

	ravPatterns, err := GetRavelPatterns(searchQueries)
	fmt.Printf("found %v ravelry posts\n", len(ravPatterns))

	if err != nil {
		return err
	}

	if len(ravPatterns) > 0 {
		fmt.Println("comparing found ravelry posts to pintrest image...")
		bestMatchPattern, _ := CompareRavelImages(ravPatterns, pinLink)
		ravelURL := fmt.Sprintf("https://www.ravelry.com/patterns/library/%s", bestMatchPattern.Permalink)

		fmt.Printf("closest match found: %s", ravelURL)
		OpenURL(ravelURL)
	}

	return nil
}

func MiddlewareClassifyImage() func(Command) error {

	return func(c Command) error {
		return HandlerFindRavelFromPin(c, ImageSearch)
	}

}

func ImageSearch(link string) ([]string, error) {

	pinLink, err := GetPinImageLink(link)

	if err != nil {
		return nil, err
	}

	clsItems := make([]string, 0)
	testRavelSearchTerms := map[string]any{
		"garment": []string{"sweater", "pancho", "cardigan", "trousers", "jean", "sock", "sweatshirt", "mitten"},
	}
	classified := recoginition.ClasifyImageTest(pinLink)

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

func NerSearch(link string) ([]string, error) {

	pinTitle, err := GetPinTitle(link)

	if err != nil {
		return nil, err
	}

	nerMap, err := recoginition.Ner(pinTitle)

	log.Printf("named objects found -- %v \n", nerMap)

	if err != nil {
		return nil, err
	}

	searchQuiries := make([]string, 0)

	for _, wrd := range strings.Split(pinTitle, " ") {

		if _, exist := nerMap[wrd]; exist {

			searchQuiries = append(searchQuiries, wrd)
		}
	}
	// get title and photo from pin page
	// run ner model
	// search using found entities

	return searchQuiries, nil
}

func MiddlewareTitleNer() func(Command) error {

	return func(c Command) error {
		return HandlerFindRavelFromPin(c, NerSearch)
	}
}

func GetPinTitle(pintrestURL string) (string, error) {

	data, err := ParsePinPage(pintrestURL)
	if err != nil {
		return "", err
	}

	title, err := traverseHTML(data, "title", 0.0, CheckFoundTitle)

	if err != nil || title == "" {
		return "", err

	}

	return title, nil

}

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

type RavelPhoto struct {
	MediumURL string `json:"medium_url"`
}

type RavelryPattern struct {
	Id         int        `json:"id"`
	Name       string     `json:"name"`
	Permalink  string     `json:"permalink"`
	FirstPhoto RavelPhoto `json:"first_photo"`
}

func GetRavelPatterns(queries []string) ([]RavelryPattern, error) {
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
		return []RavelryPattern{}, err

	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)

	jsonData := make(map[string]interface{}, 0)

	//fmt.Println(string(data))

	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return []RavelryPattern{}, err

	}

	patterMap, _ := jsonData["patterns"].([]any)
	patterns := make([]RavelryPattern, 0)
	for _, items := range patterMap {

		pattern := RavelryPattern{}

		patternData, err := json.Marshal(items)
		if err != nil {
			return []RavelryPattern{}, err

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

func OpenURL(url string) error {

	cmd := "cmd.exe"
	args := []string{"/c", "start", url}
	if len(args) > 1 {
		args = append(args[:1], append([]string{""}, args[1:]...)...)
	}
	return exec.Command(cmd, args...).Start()
}
