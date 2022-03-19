package mtgsdk

import (
	"fmt"
	"os"
)

// A deck struct
type Deck struct {
	Name    string         // The name of the deck
	cards   []Card         // The cards
	amounts map[string]int // The map of the amounts
}

// Creates a new deck
func CreateDeck(name string) *Deck {
	result := Deck{}
	result.Name = name
	return &result
}

// Adds a card to the deck
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
