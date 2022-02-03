package mtgsdk

/*
methods:
GetCardWithName(string cardName)
*/

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/GrandOichii/appdata"
	"github.com/GrandOichii/box"
	"github.com/GrandOichii/colorwrapper"
)

const (
	imageFileFormat    = "jpg"
	allCardsFileName   = "all_cards.json"
	imagesFolder       = "images"
	apiURL             = "https://api.scryfall.com/"
	cardIDSearchURL    = apiURL + "/cards/"
	cardQuerySearchURL = apiURL + "/cards/search?q="

	cardPrintWidth  = 40
	cardPrintHeight = 25
	maxCostNum      = 10

	CardNameKey = "name"
	SetNameKey  = "set"

	ImageQualitySmall = iota
	ImageQualityNormal
	ImageQualityLarge
)

var (
	adm appdata.AppDataManager

	colorMap = map[string]string{
		"W":    "hiwhite",
		"U":    "cyan",
		"B":    "hiblack",
		"R":    "red",
		"G":    "green",
		"GRAY": "white",
		"GOLD": "yellow",
	}
)

type Card struct {
	ID        string `json:"id"`
	OracleID  string `json:"oracle_id"`
	Name      string `json:"name"`
	ImageUris struct {
		Small  string `json:"small"`
		Normal string `json:"normal"`
		Large  string `json:"large"`
	} `json:"image_uris"`
	ManaCost        string   `json:"mana_cost"`
	Cmc             float64  `json:"cmc"`
	TypeLine        string   `json:"type_line"`
	OracleText      string   `json:"oracle_text"`
	Colors          []string `json:"colors"`
	ColorIdentity   []string `json:"color_identity"`
	Keywords        []string `json:"keywords"`
	SetID           string   `json:"set_id"`
	Set             string   `json:"set"`
	SetName         string   `json:"set_name"`
	SetURI          string   `json:"set_uri"`
	SetSearchURI    string   `json:"set_search_uri"`
	ScryfallSetURI  string   `json:"scryfall_set_uri"`
	RulingsURI      string   `json:"rulings_uri"`
	PrintsSearchURI string   `json:"prints_search_uri"`
	Rarity          string   `json:"rarity"`
	CardBackID      string   `json:"card_back_id"`
	ArtistIds       []string `json:"artist_ids"`
	IllustrationID  string   `json:"illustration_id"`
	BorderColor     string   `json:"border_color"`
	Power           string   `json:"power"`
	Toughness       string   `json:"toughness"`
}

type Deck struct {
	Name    string
	cards   []Card
	amounts map[string]int
}

func (d *Deck) AddCard(card Card, amount int) {
	if d.amounts == nil {
		d.amounts = map[string]int{}
		d.cards = make([]Card, 0)
	}
	d.cards = append(d.cards, card)
	if _, has := d.amounts[card.ID]; !has {
		d.amounts[card.ID] = 0
	}
	d.amounts[card.ID] += amount
}

func (d Deck) Save(path string) {
	resultText := ""
	for i, card := range d.cards {
		resultText += fmt.Sprint((d.amounts[card.ID])) + " " + card.Name
		if i != len(d.cards)-1 {
			resultText += "\n"
		}
	}
	os.WriteFile(path, []byte(resultText), 0755)
}

func (d Deck) GetUniqueCards() []Card {
	return d.cards
}

func (c Card) BasicPrint() {
	fmt.Printf("Name: %v, ID: %v\n", c.Name, c.ID)
}

func (c Card) prettyManaCost() (string, int, error) {
	if c.ManaCost == "" {
		return "", 0, nil
	}
	mchars := strings.Split(c.ManaCost, "}{")
	mchars[0] = mchars[0][1:]
	last := mchars[len(mchars)-1]
	mchars[len(mchars)-1] = last[:len(last)-1]
	lengthResult := len(strings.Join(mchars, " "))
	result := ""
	for i, mchar := range mchars {
		color, has := colorMap[mchar]
		if !has {
			if strings.Contains(mchar, "/") {
				color = colorMap["GOLD"]
			} else {
				color = colorMap["GRAY"]
			}
		}
		colored, err := colorwrapper.GetColored(color, mchar)
		if err != nil {
			return "", 0, err
		}
		result += colored
		if i != len(mchars)-1 {
			result += " "
		}
	}
	return result, lengthResult, nil
}

func (c Card) prettyTypeLine() (string, error) {
	return colorwrapper.GetColored("white-normal", c.TypeLine)
}

func (c Card) prettyPowerToughness() (string, int, error) {
	coloredP, err := colorwrapper.GetColored("red", c.Power)
	if err != nil {
		return "", 0, err
	}
	coloredT, err := colorwrapper.GetColored("cyan", c.Toughness)
	if err != nil {
		return "", 0, err
	}
	coloredSlash, err := colorwrapper.GetColored("white", "/")
	if err != nil {
		return "", 0, err
	}
	resultLen := len(c.Power + " / " + c.Toughness)
	result := fmt.Sprintf("%v %v %v", coloredP, coloredSlash, coloredT)
	return result, resultLen, nil
}

func (c Card) IsCreature() bool {
	return strings.Contains(c.TypeLine, "Creature")
}

func (c Card) prettySplitText() ([]string, error) {
	result := []string{}
	for _, line := range strings.Split(c.OracleText, "\n") {
		for _, sline := range box.StrWidthSplit(line, cardPrintWidth-2) {
			colored, err := colorwrapper.GetColored("white", sline)
			if err != nil {
				return nil, err
			}
			result = append(result, colored)
		}
	}
	return result, nil
}

func (c Card) CardPrint() error {
	var cardColor string
	if len(c.Colors) == 0 {
		// nameColor = nil
		cardColor = colorMap["GRAY"]
	} else if len(c.Colors) > 1 {
		cardColor = colorMap["GOLD"]
	} else {
		cardColor = colorMap[c.Colors[0]]
	}
	nameString, err := colorwrapper.GetColored(cardColor+"-normal-bold", c.Name)
	if err != nil {
		return err
	}
	coloredManaCost, mclength, err := c.prettyManaCost()
	if err != nil {
		return err
	}
	topMiddleSpace := strings.Repeat(" ", cardPrintWidth-len(c.Name)-mclength-2)
	topString := nameString + topMiddleSpace + coloredManaCost
	prettyType, err := c.prettyTypeLine()
	if err != nil {
		return err
	}
	lines := []string{
		topString,
		box.Separator(cardColor),
		prettyType,
		box.Separator(cardColor),
	}
	// adding text
	textLines, err := c.prettySplitText()
	if err != nil {
		return err
	}
	lines = append(lines, textLines...)
	// add the bottom (for creatures)
	if c.IsCreature() {
		// add blank spaces
		whiteSpaces := cardPrintHeight - len(lines) - 4
		for i := 0; i < whiteSpaces; i++ {
			lines = append(lines, "")
		}
		// add bottom separator
		lines = append(lines, box.Separator(cardColor))
		// add bottom line
		pt, ptlen, err := c.prettyPowerToughness()
		if err != nil {
			return err
		}
		line := strings.Repeat(" ", cardPrintWidth-ptlen-2) + pt
		lines = append(lines, line)
	}
	// fmt.Println(lines)
	return box.Draw(cardPrintHeight, cardPrintWidth, lines, cardColor)
}

func (c Card) DownloadImage(outPath string, quality int) error {
	// get request
	imageURL := ""
	q := ""
	switch quality {
	case ImageQualitySmall:
		imageURL = c.ImageUris.Small
		q = "small"
	case ImageQualityNormal:
		imageURL = c.ImageUris.Normal
		q = "normal"
	case ImageQualityLarge:
		imageURL = c.ImageUris.Large
		q = "large"
	}
	fileName := fmt.Sprintf("%v_%v.%v", c.ID, q, imageFileFormat)
	appdataPath := path.Join(imagesFolder, fileName)
	resultPath := path.Join(outPath, fileName)
	// check whether image already exists
	exists, err := adm.FileExists(appdataPath)
	if err != nil {
		return err
	}
	if !exists {
		// file doesn't exist locally, downloading it
		log.Printf("mtgsdk - image of quality %v for card %v doesn't exist locally, downloading it", q, c.ID)
		if imageURL == "" {
			log.Printf("mtgsdk - can't download images for card %v", c.ID)
			return nil
		}
		response, err := http.Get(imageURL)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		// check status code
		if response.StatusCode != 200 {
			return fmt.Errorf("received a non 200 response when downloading from %v", imageURL)
		}
		// create output file
		file, err := os.Create(adm.ConcatPath(appdataPath))
		if err != nil {
			return err
		}
		defer file.Close()
		// copy the data to the file
		_, err = io.Copy(file, response.Body)
		if err != nil {
			return err
		}
	}
	contents, err := adm.ReadFile(appdataPath)
	if err != nil {
		return err
	}
	os.WriteFile(resultPath, contents, 0755)
	log.Printf("mtgsdk - saved image for %v", c.ID)
	return nil
}

func init() {
	log.Print("mtgsdk - initializing package")
	var err error
	adm, err = appdata.CreateAppDataManager("mtgsdk-data")
	if err != nil {
		panic(err)
	}
	// create allCards file
	err = createAllCardsFile()
	if err != nil {
		panic(err)
	}
	// create imagesFolder folder
	err = createCardImagesFolder()
	if err != nil {
		panic(err)
	}
}

func (c Card) Matches(params map[string]string) bool {
	for key, value := range params {
		switch key {
		case CardNameKey:
			if !strings.Contains(c.Name, value) {
				return false
			}
		case SetNameKey:
			if !strings.Contains(c.SetName, value) {
				return false
			}
		}
	}
	return true
}

func getLocalCardDict() (map[string]Card, error) {
	// possibly save this to ram?
	existingData, err := adm.ReadFile(allCardsFileName)
	if err != nil {
		return nil, err
	}
	// var cards []Card
	var cards map[string]Card
	err = json.Unmarshal(existingData, &cards)
	if err != nil {
		return nil, err
	}
	return cards, nil
}

func createAllCardsFile() error {
	exists, err := adm.FileExists(allCardsFileName)
	if err != nil {
		return err
	}
	if !exists {
		err = adm.WriteToFile(allCardsFileName, []byte("{}"))
		if err != nil {
			return err
		}
	}
	return nil
}

func createCardImagesFolder() error {
	exists, err := adm.FileExists(imagesFolder)
	if err != nil {
		return err
	}
	if !exists {
		err = adm.CreateFolder(imagesFolder)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveCard(card Card) error {
	// if card has no id, don't do anything
	if card.ID == "" {
		log.Printf("mtgsdk - fetched card with name %v, but it doesn't have an id", card.Name)
		return nil
	}
	// add card to dict
	cards, err := getLocalCardDict()
	if err != nil {
		return err
	}
	// if key is already in dictionary
	if _, hasid := cards[card.ID]; hasid {
		return nil
	}
	cards[card.ID] = card
	// save dict to file
	data, err := json.MarshalIndent(cards, "", "\t")
	if err != nil {
		return err
	}
	log.Printf("mtgsdk - added card %v to all cards file", card.ID)
	return adm.WriteToFile(allCardsFileName, data)
}

func paramsToQ(params map[string]string) string {
	result := ""
	for key, value := range params {
		if value != "" {

			result += url.QueryEscape(key+"="+value) + "+"
		}
	}
	log.Printf("mtgsdk - parsed %v to %v", params, result)
	return result
}

func FetchCards(params map[string]string) ([]Card, error) {
	log.Printf("mtgsdk - fetching cards with params %v", params)
	parsed := paramsToQ(params)
	url := cardQuerySearchURL + parsed
	resp, err := http.Get(url)
	//  can't connect to host
	// var dnsError *net.DNSError
	// if errors.As(err, &dnsError) {
	// 	log.Println("mtgsdk - failed to connect to host, looking up cards in all_cards_path")
	// 	return getCardsOffline(params)
	// }
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// managed to fetch data
	var cardc struct {
		Cards []Card `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&cardc)
	if err != nil {
		return nil, err
	}
	log.Printf("mtgsdk - fetched %v cards", len(cardc.Cards))
	// save all cards to all cards file
	for _, card := range cardc.Cards {
		err = saveCard(card)
		if err != nil {
			return nil, err
		}
	}
	return cardc.Cards, nil
}

func getCardsOffline(params map[string]string) ([]Card, error) {
	cardDict, err := getLocalCardDict()
	if err != nil {
		return nil, err
	}
	cards := make([]Card, 0, len(cardDict))
	for _, card := range cardDict {
		cards = append(cards, card)
	}
	result := applyQ(cards, params)
	return result, nil
}

func GetCards(params map[string]string) ([]Card, error) {
	// TODO
	cards, err := FetchCards(params)
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		log.Println("mtgsdk - failed to connect to host, looking up cards in allCardsPath")
		return getCardsOffline(params)
	}
	// some other kind of error
	if err != nil {
		return nil, err
	}
	return cards, nil
}

func applyQ(cards []Card, params map[string]string) []Card {
	result := make([]Card, 0, len(cards))
	for _, card := range cards {
		if card.Matches(params) {
			result = append(result, card)
		}
	}
	return result
}

func DownloadCardImages(params map[string]string, deckPath string, outPath string, quality int) error {
	var cards []Card
	var err error
	if deckPath == "" {
		cards, err = GetCards(params)
	} else {
		deck, err := ReadDeck(deckPath)
		deckCards := deck.GetUniqueCards()
		if err != nil {
			return err
		}
		cards = applyQ(deckCards, params)
	}
	if err != nil {
		return err
	}
	// create a wait group
	wg := sync.WaitGroup{}
	wg.Add(len(cards))
	for _, card := range cards {
		c := card
		go func() {
			err = c.DownloadImage(outPath, quality)
			wg.Done()
		}()
	}
	wg.Wait()
	if err != nil {
		return err
	}
	return nil
}

func fetchCardWithID(id string) (Card, error) {
	url := cardIDSearchURL + id
	resp, err := http.Get(url)
	// 	// don't know whether to check for a connection error
	if err != nil {
		return Card{}, err
	}
	defer resp.Body.Close()
	var card Card
	err = json.NewDecoder(resp.Body).Decode(&card)
	if err != nil {
		return Card{}, err
	}
	return card, nil
}

func GetCard(id string) (Card, error) {
	cards, err := getLocalCardDict()
	if err != nil {
		return Card{}, nil
	}
	if card, found := cards[id]; found {
		return card, nil
	}
	// failed to fetch locally, going online
	card, err := fetchCardWithID(id)
	if err != nil {
		return Card{}, err
	}
	return card, nil
}

func ReadDeck(path string) (Deck, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Deck{}, err
	}
	text := string(contents)
	lines := strings.Split(text, "\n")
	result := Deck{}
	for _, line := range lines {
		words := strings.Split(line, " ")
		amount := words[0]
		cardName := strings.Join(words[1:], " ")
		fcards, err := GetCards(map[string]string{CardNameKey: cardName})
		if err != nil {
			return Deck{}, err
		}
		a, err := strconv.Atoi(amount)
		if err != nil {
			return Deck{}, err
		}
		result.AddCard(fcards[0], a)
	}
	return result, nil
}
