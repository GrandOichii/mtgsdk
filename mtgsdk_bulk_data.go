package mtgsdk

import (
	"encoding/json"
	"net/http"
)

const (
	bulkDataURI = "https://api.scryfall.com/bulk-data"

	oracleCardsType = "oracle_cards"
)

type bulkObject struct {
	Type        string `json:"type"`
	UpdatedAt   string `json:"updated_at"`
	DownloadURI string `json:"download_uri"`
}

type bulkData struct {
	Data []*bulkObject `json:"data"`
}

// Fetches fot the bulk data from the scryfall api
func fetchBulkData() (*bulkData, error) {
	logger.Printf("fetching for bulk data from %s", bulkDataURI)
	response, err := http.Get(bulkDataURI)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var result bulkData
	err = json.NewDecoder(response.Body).Decode(&result)
	return &result, err
}

// Updates the bulk data
func UpdateBulkData() error {
	logger.Printf("updating bulk data")
	bd, err := fetchBulkData()
	if err != nil {
		return err
	}
	for _, bo := range bd.Data {
		// update oracle cards
		if bo.Type == oracleCardsType {
			err = downloadCards(bo.DownloadURI)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Downloads the bulk oracle cards
func downloadCards(uri string) error {
	logger.Print("downloading all_cards file...")
	resp, err := http.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var cards []Card
	err = json.NewDecoder(resp.Body).Decode(&cards)
	if err != nil {
		return err
	}
	// transform the data to a map
	allCardsMap = map[string]Card{}
	for _, card := range cards {
		allCardsMap[card.ID] = card
	}
	// save to file
	data, err := json.Marshal(allCardsMap)
	if err != nil {
		return err
	}
	err = adm.WriteToFile(allCardsFileName, data)
	if err != nil {
		return err
	}
	logger.Print("done!")
	return nil
}
