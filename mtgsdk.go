package mtgsdk

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/GrandOichii/appdata"
)

const (
	imageFileFormat    = "jpg"                       // the format for image files
	allCardsFileName   = "all_cards.json"            // the path to to the all_cards json file
	edhrecDataFile     = "edhrec_data.json"          // the path to the file with all the edhrec data for commanders
	imagesFolder       = "images"                    // the folder for the images
	apiURL             = "https://api.scryfall.com/" // the url of the api for fetching card data
	cardIDSearchURL    = apiURL + "/cards/"          // the url for searching for cards
	cardQuerySearchURL = apiURL + "/cards/search?q=" // the query url

	cardPrintWidth  = 40 // width of the card (for terminal)
	cardPrintHeight = 25 // height of the card (for terminal)
	maxCostNum      = 10 // the maximum cost pip

	CardNameKey = "name" // the map key for card name
	SetNameKey  = "set"  // the map key for set name
)

type ImageQuality int

const (
	ImageQualitySmall  ImageQuality = iota // card quality (used in scryfall api)
	ImageQualityNormal                     // card quality (used in scryfall api)
	ImageQualityLarge                      // card quality (used in scryfall api)
)

var (
	adm          appdata.AppDataManager    // the app data manager
	allCardsDict map[string]Card           // the map of all cards
	edhrecData   map[string]map[string]int // the map of all commanders and their reccomendations

	colorMap = map[string]string{ // the map of the colors used to print cards
		"W":    "hiwhite",
		"U":    "cyan",
		"B":    "hiblack",
		"R":    "red",
		"G":    "green",
		"GRAY": "white",
		"GOLD": "yellow",
	}
)

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
	// create the edhrec data file
	err = createEDHRECDataFile()
	if err != nil {
		panic(err)
	}
	// create imagesFolder folder
	err = createCardImagesFolder()
	if err != nil {
		panic(err)
	}
	// load all cards
	err = loadAllCards()
	if err != nil {
		panic(err)
	}
	log.Print("mtgsdk - all cards loaded!")
	// load edhrec data
	err = loadEDHRECData()
	if err != nil {
		panic(err)
	}
	log.Print("mtgsdk - edhrec data loaded")
}

// Loads all the cards from the all_cards json file
func loadAllCards() error {
	// read the data
	existingData, err := adm.ReadFile(allCardsFileName)
	if err != nil {
		return err
	}
	// parse the data
	err = json.Unmarshal(existingData, &allCardsDict)
	if err != nil {
		return err
	}
	return nil
}

// Loads the edhrec data
func loadEDHRECData() error {
	existsingData, err := adm.ReadFile(edhrecDataFile)
	if err != nil {
		return err
	}
	// parse the data
	err = json.Unmarshal(existsingData, &edhrecData)
	if err != nil {
		return err
	}
	return nil
}

// Saves the local cards into the all_cards json file
func saveLocalCardDict() error {
	data, err := json.MarshalIndent(allCardsDict, "", "\t")
	if err != nil {
		return err
	}
	return adm.WriteToFile(allCardsFileName, data)
}

// Saves the edhrec data locally
func saveEDHRECData() error {
	data, err := json.MarshalIndent(edhrecData, "", "\t")
	if err != nil {
		return err
	}
	return adm.WriteToFile(edhrecDataFile, data)
}

// Creates the all_cards json file
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

// Create the edhrec data json file
func createEDHRECDataFile() error {
	exists, err := adm.FileExists(edhrecDataFile)
	if err != nil {
		return err
	}
	if !exists {
		err = adm.WriteToFile(edhrecDataFile, []byte("{}"))
		if err != nil {
			return err
		}
	}
	return nil
}

// Creates the card images folder
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

// Saves card to allCards
//
// First checks whether card is already in dict. If true, doesn't do anything. If false, adds the card to the dict and saves the dict to allCardsPath
func saveCard(card Card) error {
	// if card has no id, don't do anything
	if card.ID == "" {
		log.Printf("mtgsdk - fetched card with name %v, but it doesn't have an id", card.Name)
		return nil
	}
	// add card to dict
	// if key is already in dictionary
	if _, hasid := allCardsDict[card.ID]; hasid {
		return nil
	}
	allCardsDict[card.ID] = card
	// save dict to file
	err := saveLocalCardDict()
	if err != nil {
		return err
	}
	log.Printf("mtgsdk - added card %v to all cards file", card.ID)
	return nil
}

// Parses params to query escaped string
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

// Fetches the cards from the scryfall api
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

// Returns the cards stored in the all_cards json file
func GetCardsOffline(params map[string]string) ([]Card, error) {
	cards := make([]Card, 0, len(allCardsDict))
	for _, card := range allCardsDict {
		cards = append(cards, card)
	}
	result := applyQ(cards, params)
	return result, nil
}

// Searches the cards online, if fails, searches for them locally
func GetCards(params map[string]string) ([]Card, error) {
	// TODO
	cards, err := FetchCards(params)
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		log.Println("mtgsdk - failed to connect to host, looking up cards in allCardsPath")
		return GetCardsOffline(params)
	}
	// some other kind of error
	if err != nil {
		return nil, err
	}
	return cards, nil
}

// Returns a slice of all cards that specify the params
func applyQ(cards []Card, params map[string]string) []Card {
	result := make([]Card, 0, len(cards))
	for _, card := range cards {
		if card.Matches(params) {
			result = append(result, card)
		}
	}
	return result
}

// Downloads the card images that match the params
//
// If deckPath is not empty, selects the cards from the deckPath
func DownloadCardImages(params map[string]string, deckPath string, outPath string, quality ImageQuality) error {
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

// Fetches for the card with the specified id online
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
	// save card
	saveCard(card)
	return card, nil
}

// Checks if the id is in the allCardsDict, if not, searches for it online
func GetCard(id string) (Card, error) {
	if card, found := allCardsDict[id]; found {
		return card, nil
	}
	// failed to fetch locally, going online
	card, err := fetchCardWithID(id)
	if err != nil {
		return Card{}, err
	}
	return card, nil
}

// Reads the deck from the specified path
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
