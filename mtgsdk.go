package mtgsdk

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/GrandOichii/appdata"
)

const (
	imageFileFormat     = "jpg"                          // the format for image files
	allCardsFileName    = "all_cards.json"               // the path to to the all_cards json file
	edhrecDataFile      = "edhrec_data.json"             // the path to the file with all the edhrec data for commanders
	edhrecStaplesFile   = "edhrec_staples.json"          // The path to the file with all the ids of staple cards for commander
	cardplacePricesFile = "cardplace_prices.json"        // The path to the file with all the scraped prices for cards (in rub)
	imagesFolder        = "images"                       // the folder for the images
	apiURL              = "https://api.scryfall.com/"    // the url of the api for fetching card data
	cardIDSearchURL     = apiURL + "/cards/"             // the url for searching for cards
	cardQuerySearchURL  = apiURL + "/cards/named?fuzzy=" // the query url

	cardPrintWidth  = 40 // width of the card (for terminal)
	cardPrintHeight = 25 // height of the card (for terminal)
	maxCostNum      = 10 // the maximum cost pip

	CardNameKey = "name" // the map key for card name
	SetNameKey  = "set"  // the map key for set name

)

// The quality of the image
type ImageQuality int

const (
	ImageQualitySmall  ImageQuality = iota // card image quality (used in scryfall api)
	ImageQualityNormal                     // card image quality (used in scryfall api)
	ImageQualityLarge                      // card image quality (used in scryfall api)
)

var (
	adm           *appdata.AppDataManager // the app data manager
	logger        *log.Logger             // the logger
	mutex         sync.Mutex
	allCardsMap   map[string]Card           // the map of all cards
	edhrecData    map[string]map[string]int // the map of all commanders and their recommendations (card.id -- synergy)
	cardplaceData map[string]int            // the map of all prices for all cards (in rub)

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

// Sets the logger of the package
func SetLogger(l *log.Logger) {
	logger = l
}

func init() {
	logger = log.New(os.Stdout, "mtgsdk", log.Ltime|log.Lshortfile)
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
	// create the edhrec files
	err = createEDHRECFiles()
	if err != nil {
		panic(err)
	}
	// create the cardplace files
	err = createCardplaceFile()
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
	// load edhrec data
	err = loadEDHRECData()
	if err != nil {
		panic(err)
	}
}

func CardCount() int {
	return len(allCardsMap)
}

// Loads all the cards from the all_cards json file
func loadAllCards() error {
	// read the data
	existingData, err := adm.ReadFile(allCardsFileName)
	if err != nil {
		return err
	}
	// parse the data
	err = json.Unmarshal(existingData, &allCardsMap)
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

// Saves the edhrec data locally
func saveEDHRECData() error {
	data, err := json.Marshal(edhrecData)
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
		return UpdateBulkData()
	}
	return nil
}

// Create the edhrec data json file
func createEDHRECFiles() error {
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
	exists, err = adm.FileExists(edhrecStaplesFile)
	if err != nil {
		return err
	}
	if !exists {
		err = adm.WriteToFile(edhrecStaplesFile, []byte("[]"))
		if err != nil {
			return err
		}
	}
	return nil
}

// Creates the cardplace files
func createCardplaceFile() error {
	exists, err := adm.FileExists(cardplacePricesFile)
	if err != nil {
		return err
	}
	if !exists {
		err = adm.WriteToFile(cardplacePricesFile, []byte("{}"))
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

// Returns the cards stored in the all_cards json file
func GetCards(params map[string]string) ([]Card, error) {
	cards := make([]Card, 0, len(allCardsMap))
	for _, card := range allCardsMap {
		cards = append(cards, card)
	}
	result := applyQ(cards, params)
	return result, nil
}

// Returns the card with the exact name of the card
func GetCardWithExactName(cardName string) (Card, error) {
	cards, err := GetCards(map[string]string{CardNameKey: cardName})
	if err != nil {
		return Card{}, err
	}
	for _, card := range cards {
		if card.HasName(cardName) {
			return card, nil
		}
	}
	type cardc struct {
		Cards []Card `json:"data"`
	}
	resp, err := http.Get(cardQuerySearchURL + cardName)
	if err != nil {
		return Card{}, err
	}
	var cc cardc
	err = json.NewDecoder(resp.Body).Decode(&cc)
	if err != nil {
		return Card{}, err
	}
	for _, card := range cc.Cards {
		if card.HasName(cardName) {
			return card, nil
		}
	}
	return Card{}, fmt.Errorf("mtgsdk - no card with exact name %s", cardName)
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
func DownloadCardImages(params map[string]string, deckPath string, outPath string, quality ImageQuality, offline bool) error {
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

// Checks if the id is in the allCardsDict, if not, searches for it online
func GetCard(id string) (Card, error) {
	if card, found := allCardsMap[id]; found {
		return card, nil
	}
	logger.Printf("couldn't find card %s locally, fetching for it online", id)
	// didn't find locally, fetching for it online
	res, err := http.Get(cardIDSearchURL + id)
	if err != nil {
		return Card{}, err
	}
	var result Card
	err = json.NewDecoder(res.Body).Decode(&result)
	return result, err
}

// Returns the map of basic lands
func GetBasicLands() (map[string]Card, error) {
	blnames := []string{"Plains", "Island", "Swamp", "Mountain", "Forest"}
	result := map[string]Card{}
	for _, blname := range blnames {
		card, err := GetCardWithExactName(blname)
		if err != nil {
			return nil, err
		}
		result[blname] = card
	}
	return result, nil
}
