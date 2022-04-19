package mtgsdk

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/GrandOichii/box"
	"github.com/GrandOichii/colorwrapper"
)

// A card struct
type Card struct {
	ID        string   `json:"id"`        // The id of the card (used as key in maps)
	OracleID  string   `json:"oracle_id"` // The oracle id for the card
	Name      string   `json:"name"`      // The name of the card
	ImageUris struct { // URLs for the card images
		Small  string `json:"small"`
		Normal string `json:"normal"`
		Large  string `json:"large"`
	} `json:"image_uris"`
	ManaCost        string   `json:"mana_cost"`      // The raw manacost of the card
	Cmc             float64  `json:"cmc"`            // The converted manacost of the card
	TypeLine        string   `json:"type_line"`      // The card type line
	OracleText      string   `json:"oracle_text"`    // The oracled text of the card
	Colors          []string `json:"colors"`         // Card colors
	ColorIdentity   []string `json:"color_identity"` // The color identity of the card
	Keywords        []string `json:"keywords"`       // The keywords of the card
	SetID           string   `json:"set_id"`         // The ID of the set of the card
	Set             string   `json:"set"`            // The set codename of the card
	SetName         string   `json:"set_name"`       // The actual set name
	SetURI          string   `json:"set_uri"`        // The URI to the set
	SetSearchURI    string   `json:"set_search_uri"` // The URI to serach for the set
	ScryfallSetURI  string   `json:"scryfall_set_uri"`
	RulingsURI      string   `json:"rulings_uri"`
	PrintsSearchURI string   `json:"prints_search_uri"`
	Rarity          string   `json:"rarity"`          // The rarity of the card
	CardBackID      string   `json:"card_back_id"`    // The ID of the back of the card (if is double-sided)
	ArtistIds       []string `json:"artist_ids"`      // ID of the artist
	IllustrationID  string   `json:"illustration_id"` // ID of the illustration
	BorderColor     string   `json:"border_color"`    // The color of the border
	Power           string   `json:"power"`           // The power of the card
	Toughness       string   `json:"toughness"`       // The toughness of the card
}

// Prints out the card to the console
func (c Card) BasicPrint() {
	fmt.Printf("Name: %v, ID: %v\n", c.Name, c.ID)
}

// Returns the prettified mana cost of the card
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

// Returns the prettified type line of the card
func (c Card) prettyTypeLine() (string, error) {
	return colorwrapper.GetColored("white-normal", c.TypeLine)
}

// Returns the prettified line of power/toughness of the card
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

// Returns the prettified split text of the card
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

// Prints the card as a card
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
	tmsr := cardPrintWidth - len(c.Name) - mclength - 2
	if tmsr < 0 {
		tmsr = 0
	}
	topMiddleSpace := strings.Repeat(" ", tmsr)
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
	return box.Draw(cardPrintHeight, cardPrintWidth, lines, cardColor)
}

// Downloads the card image to the specified path
func (c Card) DownloadImage(outPath string, quality ImageQuality) error {
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
		logger.Printf("mtgsdk - image of quality %v for card %v doesn't exist locally, downloading it", q, c.ID)
		if imageURL == "" {
			logger.Printf("mtgsdk - can't download images for card %v", c.ID)
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
		file, err := os.Create(adm.PathTo(appdataPath))
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
	logger.Printf("mtgsdk - saved image for %v", c.ID)
	return nil
}

// Returns true if the cards matches all the specified params
func (c Card) Matches(params map[string]string) bool {
	for key, value := range params {
		switch key {
		case CardNameKey:
			if !strings.Contains(strings.ToLower(c.Name), strings.ToLower(value)) {
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

// Returns the map of card ids and their synergies (only applies to legendary creatures)
func (c Card) GetRecommendations(synergy int, offline bool) (map[*Card]int, error) {
	if c.IsLegendary() && c.IsCreature() {
		recc, err := recommendCards(c.ID, offline)
		if err != nil {
			return nil, err
		}
		result := make(map[*Card]int)
		for card, syn := range recc {
			if syn >= synergy {
				result[card] = syn
			}
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("mtgsdk - can't get recommendations for non-legendary creature (%s)", c.Name)
	}
}

// Returns true if the card is a creature
func (c Card) IsCreature() bool {
	return strings.Contains(c.TypeLine, "Creature")
}

// Returns true if the card is legendary
func (c Card) IsLegendary() bool {
	return strings.Contains(c.TypeLine, "Legendary")
}

// Returns true if the card is a land
func (c Card) IsLand() bool {
	return strings.Contains(c.TypeLine, "Land")
}

// Returns true if the card is a basic land
func (c Card) IsBasicLand() bool {
	return strings.Contains(c.TypeLine, "Basic Land")
}

// Returns true if the card can generate mana
func (c Card) IsRamp() bool {
	return !c.IsLand() && strings.Contains(c.OracleText, "Add ") || strings.Contains(c.OracleText, "your library for a basic land card, put that card onto the battlefield tapped")
}

// Returns true if the card is a board wipe
func (c Card) IsBoardWipe() bool {
	return strings.Contains(c.OracleText, "Destroy all ") || strings.Contains(c.OracleText, " damage to each creature") || strings.Contains(c.OracleText, "All creatures get -")
}

// Returns true if the card forces the player to draw cards
func (c Card) IsCardDraw() bool {
	return strings.Contains(strings.ToLower(c.OracleText), "draw ")
}

// Returns true if the card is a removal card
func (c Card) IsRemoval() bool {
	return strings.Contains(strings.ToLower(c.OracleText), "destroy target")
}

// Returns true if the colors match the color identity of the card
func (c Card) MatchesColorIdentity(colors []string) bool {
	for _, ci := range colors {
		contains := false
		for _, cci := range c.ColorIdentity {
			if ci == cci {
				contains = true
				break
			}
		}
		if !contains {
			return false
		}
	}
	return true
}

// Returns the map of counted color pips
func (c Card) CountColorPips() map[string]int {
	cpips := strings.Split("WUBRG", "")
	result := map[string]int{}
	for _, pip := range cpips {
		amount := strings.Count(c.ManaCost, pip)
		_, has := result[pip]
		if !has {
			result[pip] = 0
		}
		result[pip] += amount
	}
	return result
}

// Returns true if the card has the specified name
//
// Works on double-faced cards
func (c Card) HasName(name string) bool {
	if c.Name == name {
		return true
	}
	// weird cards that are the same on two sides
	names := strings.Split(c.Name, " // ")
	if len(names) == 2 && names[0] == names[1] {
		return false
	}
	for _, fname := range names {
		if fname == name {
			return true
		}
	}
	return false
}

// func (c Card) GetRusPrice() (int, error) {
// 	return getRusPriceFor(c.Name)
// }
