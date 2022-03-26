package mtgsdk

import (
	"fmt"
	"log"
	"sort"
)

const (
	RampCountDefault      = 10
	BoardWipeCountDefault = 5
	CardDrawCountDefault  = 10
	RemovalCountDefault   = 8
	LandCountDefault      = 33

	DeckGenRampCountKey      = "ramp"
	DeckGenBoardWipeCountKey = "boardwipes"
	DeckGenCardDrawCount     = "carddraw"
	DeckGenRemovalCount      = "removal"
	DeckGenLandCount         = "land"
)

// Generates a commander deck for a card (the card has to be a legendary creature)
func (c Card) GenerateCommanderDeck(params map[string]interface{}, offline bool) (*Deck, error) {
	if !(c.IsCreature() && c.IsLegendary()) {
		return nil, fmt.Errorf("mtgsdk - %s is not a legendary creature", c.Name)
	}
	log.Printf("Generating deck for %s", c.Name)
	result := CreateDeck(fmt.Sprintf("Commander deck for %s", c.Name))
	// add the commander itself
	result.AddSingletonCard(&c)
	// add staples
	staples, err := GetEDHRECStaples(offline)
	if err != nil {
		return nil, err
	}
	unsortedRecc, err := reccomendCards(c.ID, offline)
	if err != nil {
		return nil, err
	}
	recc := sortRecc(unsortedRecc)
	// add lands
	lr := LandCountDefault
	if amount, has := params[DeckGenLandCount]; has {
		lr = amount.(int)
	}
	rcards := 99 - lr
	for _, pair := range *recc {
		card := pair.Key
		if lr == 0 {
			break
		}
		if card.IsLand() {
			if result.AddSingletonCard(card) {
				lr--
				log.Printf("mtgsdk - adding %s -- land (%d)", card.Name, lr)
			}
		}
	}
	for _, card := range staples {
		if lr == 0 {
			break
		}
		if c.MatchesColorIdentity(card.ColorIdentity) {
			if card.IsLand() {
				if result.AddSingletonCard(&card) {
					lr--
					log.Printf("mtgsdk - Adding %s -- land", card.Name)
				}
			}
		}
	}
	log.Print("mtgsdk - added lands")
	if rcards <= 0 {
		return result, nil
	}
	// add other staples
	rampr := RampCountDefault
	if amount, has := params[DeckGenRampCountKey]; has {
		rampr = amount.(int)
	}
	bwr := BoardWipeCountDefault
	if amount, has := params[DeckGenBoardWipeCountKey]; has {
		bwr = amount.(int)
	}
	cdr := CardDrawCountDefault
	if amount, has := params[DeckGenCardDrawCount]; has {
		cdr = amount.(int)
	}
	rr := RemovalCountDefault
	if amount, has := params[DeckGenRemovalCount]; has {
		rr = amount.(int)
	}
	for _, card := range staples {
		if rcards != 0 && c.MatchesColorIdentity(card.ColorIdentity) {
			add := false
			if rampr != 0 && card.IsRamp() {
				rampr--
				log.Printf("mtgsdk - adding %s -- ramp", card.Name)
				add = true
			}
			if bwr != 0 && card.IsBoardWipe() {
				bwr--
				log.Printf("mtgsdk - adding %s -- board wipe", card.Name)
				add = true
			}
			if cdr != 0 && card.IsCardDraw() {
				cdr--
				log.Printf("mtgsdk - adding %s -- card draw", card.Name)
				add = true
			}
			if rr != 0 && card.IsRemoval() {
				rr--
				log.Printf("mtgsdk - adding %s -- removal", card.Name)
				add = true
			}
			if add {
				if result.AddSingletonCard(&card) {
					rcards--
				}
			}
		}
	}
	// add reccomendations
	for _, pair := range *recc {
		if rcards == 0 {
			break
		}
		card := pair.Key
		if result.AddSingletonCard(card) {
			rcards--
			log.Printf("mtgsdk - added card %s as reccomendation (synergy: %d)\n", card.Name, pair.Value)
		}
	}
	// add remaining basic lands
	blrecc, err := result.ReccomendBasicLands(lr)
	if err != nil {
		return nil, err
	}
	for lname, amount := range blrecc {
		cards, err := GetCards(map[string]string{CardNameKey: lname}, offline)
		if err != nil {
			return nil, err
		}
		for _, card := range cards {
			if amount != 0 && card.Name == lname {
				result.AddCard(&card, amount)
				break
			}
		}
	}
	return result, nil
}

// A pair struct
type pair struct {
	Key   *Card
	Value int
}

// For recc map sorting
type pairList []pair

func (p pairList) Len() int           { return len(p) }
func (p pairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p pairList) Less(i, j int) bool { return p[i].Value > p[j].Value }

func sortRecc(m map[*Card]int) *pairList {
	p := make(pairList, len(m))
	i := 0
	for k, v := range m {
		p[i] = pair{k, v}
		i++
	}
	sort.Sort(p)
	return &p
}
