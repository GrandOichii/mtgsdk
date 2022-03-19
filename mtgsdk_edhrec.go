package mtgsdk

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const (
	edhrecURL = "https://edhrec.com/commanders/%s" // The url for edhrec.com

	cardSelector = "div[class^=\"Card_container__\"]" // The selector for card elements
)

var (
	browser *rod.Browser = nil // The browser that accesses the edhrec website

	dchars = []string{ // Disallowed characters in the card name URL
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
	return fmt.Sprintf(edhrecURL, cname)
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
	html := page.MustHTML()
	err = os.WriteFile("page.html", []byte(html), 0755)
	if err != nil {
		return nil, err
	}
	return page, nil
}

// Extracts the name and the synergy of the card from the text
func extractNameAndSynergy(text string) (string, int, error) {
	lines := strings.Split(text, "\n")
	s, err := strconv.Atoi(strings.Split(lines[5], "%")[0])
	return lines[3], s, err
}

// Returns the map of cards id to synergy
func reccomendCards(name string, synergyThresh int) (map[string]int, error) {
	log.Printf("Searching the best cards for %s", name)
	var err error
	// check if the data exists locally
	data, has := edhrecData[name]
	if has {
		return data, nil
	}
	// data doesn't exist locally, fetching for it online
	// init the browser
	if browser == nil {
		err = initBrowser()
		if err != nil {
			return nil, err
		}
	}
	url := commanderURL(name)
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
		if synergy > synergyThresh {
			result[name] = synergy
		}
	}
	log.Printf("Card stats for %s loaded!", name)
	// save locally
	edhrecData[name] = result
	return result, saveEDHRECData()
}
