package mtgsdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-rod/rod"
)

const (
	commanderSearchURL = "https://edhrec.com/commanders/%s" // The url for edhrec.com
	staplesURL         = "https://edhrec.com/top"           // The url for searching for staples

	cardSelector = "div[class^=\"Card_container__\"]" // The selector for card elements
	// cardSelector = ".Card_name__1MYwa"

	renderLimit = 6
)

var (
	// Disallowed characters in the card name URL
	dchars = []string{
		",",
		"'",
	}
)

// Turns the card name to a valid udhrec url
func commanderURL(cardName string) string {
	names := strings.Split(cardName, " // ")
	cname := strings.ToLower(names[0])
	for _, dchar := range dchars {
		cname = strings.ReplaceAll(cname, dchar, "")
	}
	cname = strings.ReplaceAll(cname, " ", "-")
	return fmt.Sprintf(commanderSearchURL, cname)
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
func recommendCards(cardID string, offline bool) (map[*Card]int, error) {
	logger.Printf("Searching the best cards for %s", cardID)
	var err error
	// check if the data exists locally
	data, has := edhrecData[cardID]
	if has {
		return toCardMap(data)
	}
	if !has && offline {
		return nil, fmt.Errorf("mtgsdk - can't recommend cards for %s: no local data", cardID)
	}
	// data doesn't exist locally, fetching for it online
	// init the browser
	err = initBrowser()
	if err != nil {
		return nil, err
	}
	card, err := GetCard(cardID)
	if err != nil {
		return nil, err
	}
	url := commanderURL(card.Name)
	logger.Printf("Accessing %s...", url)

	var cardElems rod.Elements
	var page *rod.Page
	// access the page
	rc := 0
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
		if len(cardElems) == 0 {
			return nil, fmt.Errorf("mtgsdk - bad url")
		}
		rc++
		if rc == renderLimit {
			return nil, fmt.Errorf("mtgsdk - can't find cards for %s (page not rendered %d times)", card.Name, rc)
		}
	}
	cardElems = cardElems[1:] // Skip the first element - it's the commander itself
	defer page.MustClose()
	amount := len(cardElems)
	logger.Printf("Found %d cards", amount)
	result := make(map[string]int, amount)
	for _, card := range cardElems {
		text, err := card.Text()
		if err != nil {
			return nil, err
		}
		name, synergy, err := extractNameAndSynergy(text)
		// logger.Printf("Extracted card: %s (%d)", name, synergy)
		if err != nil {
			return nil, err
		}
		card, err := GetCardWithExactName(name)
		if err != nil {
			return nil, err
		}
		result[card.ID] = synergy
	}
	logger.Printf("Card stats for %s loaded!", card.Name)
	mutex.Lock()
	// save locally
	edhrecData[cardID] = result
	err = saveEDHRECData()
	if err != nil {
		mutex.Unlock()
		return nil, err
	}
	mutex.Unlock()
	return toCardMap(result)
}

// Returns a slice of all commander staple cards (according to edhrec.com)
func GetEDHRECStaples() ([]Card, error) {
	exists, err := adm.FileExists(edhrecStaplesFile)
	if err != nil {
		return nil, err
	}
	if exists {
		return getLocalEDHRECStaples()
	}
	if browser == nil {
		err = initBrowser()
		if err != nil {
			return nil, err
		}
	}
	logger.Printf("Accessing %s", staplesURL)
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
	logger.Printf("Found %d cards", amount)
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
		card, err := GetCardWithExactName(cardName)
		if err != nil {
			return nil, err
		}
		result = append(result, card)
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
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	err = adm.WriteToFile(edhrecStaplesFile, data)
	return err
}
