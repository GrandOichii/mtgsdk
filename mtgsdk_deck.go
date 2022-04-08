package mtgsdk

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/GrandOichii/colorwrapper"
)

// A struct of deck statistics
type DeckStat struct {
	CMCBars        map[float64]int // mana values
	RampCount      int             // The amount of ramp cards
	CardDrawCount  int             // The amount of card draw
	BoardWipeCount int             // The amount of board wipes
	CardCount      int             // The amount of cards
	RemovalCount   int             // The amount of temoval
	LandCount      int             // The amount of lands
}

func (d DeckStat) Print() error {
	fmt.Printf("Card count: %d\n", d.CardCount)
	fmt.Printf("\tRamp: %d\n", d.RampCount)
	fmt.Printf("\tCard draw: %d\n", d.CardDrawCount)
	fmt.Printf("\tBoard wipes: %d\n", d.BoardWipeCount)
	fmt.Printf("\tRemoval: %d\n", d.RemovalCount)
	fmt.Printf("\tLands: %d\n", d.LandCount)
	keys := []float64{0., 1., 2., 3., 4., 5., 6., 7., 8., 9., 10.}
	for _, key := range keys {
		value, has := d.CMCBars[key]
		if !has {
			value = 0
		}
		s := strings.TrimRight(strings.Repeat("# ", value), " ")
		colored, err := colorwrapper.GetColored("normal-cyan", s)
		if err != nil {
			return err
		}
		fmt.Printf("%v %s\n", key, colored)
	}
	return nil
}

// A deck struct
type Deck struct {
	Name    string         // The name of the deck
	cards   []Card         // The cards
	amounts map[string]int // The map of the amounts (card.id -- amount)
}

// Creates a new deck
func CreateDeck(name string) *Deck {
	result := Deck{}
	result.Name = name
	return &result
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
		card, err := GetCardWithExactName(cardName)
		if err != nil {
			return Deck{}, err
		}
		a, err := strconv.Atoi(amount)
		if err != nil {
			return Deck{}, err
		}
		result.AddCard(&card, a)
	}
	return result, nil
}

// Adds a card to the deck
func (d *Deck) AddCard(card *Card, amount int) {
	if d.amounts == nil {
		d.amounts = map[string]int{}
		d.cards = make([]Card, 0)
	}
	d.cards = append(d.cards, *card)
	if _, has := d.amounts[card.ID]; !has {
		d.amounts[card.ID] = 0
	}
	d.amounts[card.ID] += amount
}

// Adds a card if it's not already in the deck
//
// Returns true if added the card
func (d *Deck) AddSingletonCard(card *Card) bool {
	if d.Count(card.ID) == 0 {
		d.AddCard(card, 1)
		return true

	}
	return false
}

// Saves the deck the specified path
func (d Deck) Save(path string) error {
	resultText := ""
	for i, card := range d.cards {
		resultText += fmt.Sprint((d.amounts[card.ID])) + " " + card.Name
		if i != len(d.cards)-1 {
			resultText += "\n"
		}
	}
	return os.WriteFile(path, []byte(resultText), 0755)
}

// Returns a slice of unique cards of the deck
func (d Deck) GetUniqueCards() []Card {
	return d.cards
}

// Returns the number of card instances in deck
func (d Deck) Count(cardID string) int {
	result, has := d.amounts[cardID]
	if !has {
		return 0
	}
	return result
}

// Prints the deck out to the console
func (d Deck) Print() error {
	fmt.Printf("Deck %s\n", d.Name)
	s, err := d.ToString()
	if err != nil {
		return err
	}
	fmt.Println(s)
	return nil
}

// Returns a slice of mana values (index - mana value, value - count)
func (d Deck) getCMCBars() (map[float64]int, error) {
	result := map[float64]int{}
	for _, card := range d.cards {
		amount, has := d.amounts[card.ID]
		if !has {
			return nil, fmt.Errorf("mtgsdk - deck.amounts doesn't contain card %s", card.Name)
		}
		if _, has := result[card.Cmc]; !has {
			result[card.Cmc] = 0
		}
		result[card.Cmc] += amount
	}
	return result, nil
}

// Returns the statistics of the deck
func (d Deck) GetStats() (*DeckStat, error) {
	result := DeckStat{}
	var err error
	result.CMCBars, err = d.getCMCBars()
	if err != nil {
		return nil, err
	}
	for _, card := range d.cards {
		amount, has := d.amounts[card.ID]
		if !has {
			return nil, fmt.Errorf("mtgsdk - deck.amounts doesn't contain card %s", card.Name)
		}
		result.CardCount += amount
		if card.IsRamp() {
			result.RampCount += amount
		}
		if card.IsBoardWipe() {
			result.BoardWipeCount += amount
		}
		if card.IsCardDraw() {
			result.CardDrawCount += amount
		}
		if card.IsRemoval() {
			result.RemovalCount += amount
		}
		if card.IsLand() {
			result.LandCount += amount
		}
	}
	return &result, nil
}

// Recommends basic lands for the deck
//
// Returns a map of basic land name: amount
func (d Deck) RecommendBasicLands(count int) (map[string]int, error) {
	landMap := map[string]string{
		"W": "Plains",
		"U": "Island",
		"B": "Swamp",
		"R": "Mountain",
		"G": "Forest",
	}
	result := map[string]int{
		"Plains":   0,
		"Island":   0,
		"Swamp":    0,
		"Mountain": 0,
		"Forest":   0,
	}
	if count <= 0 {
		return result, nil
	}
	// colorPipMap := map[string]string{
	// 	"W": "Plains",
	// 	"U": "Island",
	// 	"B": "Swamp",
	// 	"R": "Mountain",
	// 	"G": "Forest",
	// }
	totalPipCount := map[string]int{
		"W": 0,
		"U": 0,
		"B": 0,
		"R": 0,
		"G": 0,
	}
	for _, card := range d.cards {
		amount, has := d.amounts[card.ID]
		if !has {
			return nil, fmt.Errorf("mtgsdk - deck.amounts doesn't contain card %s", card.Name)
		}
		pipCount := card.CountColorPips()
		for pip, pipc := range pipCount {
			totalPipCount[pip] += pipc * amount
		}
	}
	allPips := 0
	for _, c := range totalPipCount {
		allPips += c
	}
	if allPips == 0 {
		return result, nil
	}
	for pip, c := range totalPipCount {
		// count - allPips
		// ? - c
		land := landMap[pip]
		result[land] = c * count / allPips
	}
	rcount := 0
	for _, c := range result {
		rcount += c
	}
	diff := count - rcount
	if diff > 0 {
		key := ""
		for k, v := range result {
			if v != 0 {
				key = k
				break
			}
		}
		i := 0
		for diff > i {
			i++
		}
		result[key] += i
	}
	return result, nil
}

// Converts the deck to the standard representation of an mtg deck
func (d Deck) ToString() (string, error) {
	result := ""
	for _, card := range d.cards {
		amount, has := d.amounts[card.ID]
		if !has {
			return "", fmt.Errorf("mtgsdk - deck.amounts doesn't contain card %s", card.Name)
		}
		result += fmt.Sprintf("%d %s\n", amount, card.Name)
	}
	return result, nil
}

// Converts the deck to a json format
func (d Deck) ToJSON() ([]byte, error) {
	return json.MarshalIndent(d.amounts, "", "\t")
}
