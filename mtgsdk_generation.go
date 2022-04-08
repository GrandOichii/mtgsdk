package mtgsdk

import (
	"fmt"
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
func (c Card) GenerateCommanderDeck(params map[string]int) (*Deck, error) {
	if !(c.IsCreature() && c.IsLegendary()) {
		return nil, fmt.Errorf("mtgsdk - %s is not a legendary creature", c.Name)
	}
	logger.Printf("generating deck for %s", c.Name)
	result := CreateDeck(fmt.Sprintf("Commander deck for %s", c.Name))
	// add the commander itself
	result.AddSingletonCard(&c)
	// add staples
	staples, err := GetEDHRECStaples()
	if err != nil {
		return nil, err
	}
	unsortedRecc, err := recommendCards(c.ID, false)
	if err != nil {
		logger.Println("failed to fetch recommendations online, trying locally...")
		// unsortedRecc, err := reco
	}
	recc := sortRecc(unsortedRecc)
	// add lands
	lr := LandCountDefault
	if amount, has := params[DeckGenLandCount]; has {
		lr = amount
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
				logger.Printf("mtgsdk - adding %s -- land (%d)", card.Name, lr)
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
					logger.Printf("mtgsdk - Adding %s -- land", card.Name)
				}
			}
		}
	}
	logger.Print("mtgsdk - added lands")
	if rcards <= 0 {
		return result, nil
	}
	// add other staples
	rampr := RampCountDefault
	if amount, has := params[DeckGenRampCountKey]; has {
		rampr = amount
	}
	bwr := BoardWipeCountDefault
	if amount, has := params[DeckGenBoardWipeCountKey]; has {
		bwr = amount
	}
	cdr := CardDrawCountDefault
	if amount, has := params[DeckGenCardDrawCount]; has {
		cdr = amount
	}
	rr := RemovalCountDefault
	if amount, has := params[DeckGenRemovalCount]; has {
		rr = amount
	}
	for _, card := range staples {
		if rcards != 0 && c.MatchesColorIdentity(card.ColorIdentity) {
			add := false
			if rampr != 0 && card.IsRamp() {
				rampr--
				logger.Printf("mtgsdk - adding %s -- ramp", card.Name)
				add = true
			}
			if bwr != 0 && card.IsBoardWipe() {
				bwr--
				logger.Printf("mtgsdk - adding %s -- board wipe", card.Name)
				add = true
			}
			if cdr != 0 && card.IsCardDraw() {
				cdr--
				logger.Printf("mtgsdk - adding %s -- card draw", card.Name)
				add = true
			}
			if rr != 0 && card.IsRemoval() {
				rr--
				logger.Printf("mtgsdk - adding %s -- removal", card.Name)
				add = true
			}
			if add {
				if result.AddSingletonCard(&card) {
					rcards--
				}
			}
		}
	}
	// add recommendations
	for _, pair := range *recc {
		if rcards == 0 {
			break
		}
		card := pair.Key
		if result.AddSingletonCard(card) {
			rcards--
			logger.Printf("added card %s as recommendation (synergy: %d)\n", card.Name, pair.Value)
		}
	}
	// add remaining basic lands
	blrecc, err := result.RecommendBasicLands(lr)
	if err != nil {
		return nil, err
	}
	basics, err := GetBasicLands()
	if err != nil {
		return nil, err
	}
	for lname, amount := range blrecc {
		card, has := basics[lname]
		if !has {
			return nil, fmt.Errorf("mtgsdk - no basic land with name %s", lname)
		}
		if amount != 0 && card.HasName(lname) {
			result.AddCard(&card, amount)
			break
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
