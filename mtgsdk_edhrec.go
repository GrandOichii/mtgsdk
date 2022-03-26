package mtgsdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const (
	commanderSearchURL = "https://edhrec.com/commanders/%s" // The url for edhrec.com
	staplesURL         = "https://edhrec.com/top"           // The url for searching for staples

	cardSelector = "div[class^=\"Card_container__\"]" // The selector for card elements
)

var (
	browser *rod.Browser = nil // The browser that accesses the edhrec website

	// Disallowed characters in the card name URL
	dchars = []string{
		",",
	}
)

// Turns the card name to a valid udhrec url
func commanderURL(cardName string) string {
	cname := strings.ToLower(cardName)
	for _, dchar := range dchars {
		cname = strings.ReplaceAll(cname, dchar, "")
	}
	cname = strings.ReplaceAll(cname, " ", "-")
	return fmt.Sprintf(commanderSearchURL, cname)
}

// Initializes the browser
func initBrowser() error {
	u := launcher.New().
		// Headless(false).
		Set("--blink-settings=imagesEnabled=false").
		MustLaunch()

	browser = rod.New().ControlURL(u)
	err := browser.Connect()
	if err != nil {
		return err
	}
	log.Println("Connected to browser")
	return nil
}

// Navigates the browser to the specified url
func nav(url string) (*rod.Page, error) {
	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, err
	}
	waitFunc := page.MustWaitNavigation()
	err = page.Navigate(url)
	if err != nil {
		return nil, err
	}
	log.Println("Connected to the page, rendering...")
	waitFunc()
	log.Println("Page rendered!")
	return page, nil
}

// Extracts the name and the synergy of the card from the text
func extractNameAndSynergy(text string) (string, int, error) {
	lines := strings.Split(text, "\n")
	s, err := strconv.Atoi(strings.Split(lines[5], "%")[0])
	return lines[3], s, err
}

func toCardMap(data map[string]int) (map[*Card]int, error) {
	result := make(map[*Card]int, len(data))
	for id, syn := range data {
		card, err := GetCard(id)
		if err != nil {
			return nil, err
		}
		result[&card] = syn
	}
	return result, nil
}

// Returns the map of cards id to synergy
func reccomendCards(cardID string, offline bool) (map[*Card]int, error) {
	log.Printf("Searching the best cards for %s", cardID)
	var err error
	// check if the data exists locally
	data, has := edhrecData[cardID]
	if has {
		return toCardMap(data)
	}
	if !has && offline {
		return nil, fmt.Errorf("mtgsdk - can't reccomend cards for %s: no local data", cardID)
	}
	// data doesn't exist locally, fetching for it online
	// init the browser
	if browser == nil {
		err = initBrowser()
		if err != nil {
			return nil, err
		}
	}
	card, err := GetCard(cardID)
	if err != nil {
		return nil, err
	}
	url := commanderURL(card.Name)
	log.Printf("Accessing %s...", url)

	var cardElems rod.Elements
	var page *rod.Page
	// access the page
	for {
		page, err = nav(url)
		if err != nil {
			return nil, err
		}
		// scrape the elements
		cardElems = page.MustElements(cardSelector)
		if len(cardElems) > 1 {
			break
		}
	}
	cardElems = cardElems[1:] // Skip the first element - it's the commander itself
	defer page.MustClose()
	amount := len(cardElems)
	log.Printf("Found %d cards", amount)
	result := make(map[string]int, amount)
	for _, card := range cardElems {
		text, err := card.Text()
		if err != nil {
			return nil, err
		}
		name, synergy, err := extractNameAndSynergy(text)
		// log.Printf("Extracted card: %s (%d)", name, synergy)
		if err != nil {
			return nil, err
		}
		cards, err := GetCards(map[string]string{CardNameKey: name}, offline)
		if err != nil {
			return nil, err
		}
		card := cards[0]
		result[card.ID] = synergy
	}
	log.Printf("Card stats for %s loaded!", card.Name)
	// save locally
	edhrecData[cardID] = result
	err = saveEDHRECData()
	if err != nil {
		return nil, err
	}
	return toCardMap(result)
}

// Returns a slice of all commander staple cards (according to edhrec.com)
func GetEDHRECStaples(offline bool) ([]Card, error) {
	if offline {
		return getLocalEDHRECStaples()
	}
	var err error
	if browser == nil {
		err = initBrowser()
		if err != nil {
			return nil, err
		}
	}
	log.Printf("Accessing %s", staplesURL)
	var cardElems rod.Elements
	var page *rod.Page
	// access the page
	for {
		page, err = nav(staplesURL)
		if err != nil {
			navErr := error(&rod.ErrNavigation{})
			if errors.As(err, &navErr) {
				return getLocalEDHRECStaples()
			}
			return nil, err
		}
		// scrape the elements
		cardElems = page.MustElements(cardSelector)
		if len(cardElems) > 1 {
			break
		}
	}
	defer page.MustClose()
	amount := len(cardElems)
	log.Printf("Found %d cards", amount)
	result := []Card{}
	for _, card := range cardElems {
		text, err := card.Text()
		if err != nil {
			return nil, err
		}
		lines := strings.Split(text, "\n")
		if len(lines) < 4 {
			continue
		}
		cardName := lines[3]
		cards, err := GetCards(map[string]string{CardNameKey: cardName}, offline)
		if err != nil {
			return nil, err
		}
		if len(cards) == 0 {
			continue
		}
		result = append(result, cards[0])
	}
	err = saveEDHRECStaples(result)
	return result, err
}

// Reads the local edhrec staple cards
func getLocalEDHRECStaples() ([]Card, error) {
	data, err := adm.ReadFile(edhrecStaplesFile)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	err = json.Unmarshal(data, &ids)
	result := make([]Card, len(ids))
	for i, id := range ids {
		result[i], err = GetCard(id)
		if err != nil {
			return nil, err
		}
	}
	return result, err
}

// Saves edhrec staples to the edhrec staples file
func saveEDHRECStaples(cards []Card) error {
	ids := make([]string, len(cards))
	for i, card := range cards {
		ids[i] = card.ID
	}
	data, err := json.MarshalIndent(ids, "", "\t")
	if err != nil {
		return err
	}
	err = adm.WriteToFile(edhrecStaplesFile, data)
	return err
}
