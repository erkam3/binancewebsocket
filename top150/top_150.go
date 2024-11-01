package top150

import (
	"encoding/json"
	"net/http"
	"sort"
)

type PairWithVolume struct {
	Symbol      string  `json:"symbol"`
	QuoteVolume float64 `json:"quoteVolume,string"`
}

// GetTop150 returns the top 150 symbols by quote volume
// Returns:
//
//	map[string]bool: set the top 150 symbols
//	error: an error if the request fails
func GetTop150() (map[string]bool, error) {
	resp, err := http.Get("https://fapi.binance.com/fapi/v1/ticker/24hr")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pairs []PairWithVolume
	err = json.NewDecoder(resp.Body).Decode(&pairs)
	if err != nil {
		return nil, err
	}

	pairs = filterUSTD(pairs)

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].QuoteVolume > pairs[j].QuoteVolume
	})

	top150 := make(map[string]bool)
	for i := 0; i < 150; i++ {
		top150[pairs[i].Symbol] = true
	}

	return top150, nil
}

func filterUSTD(pairs []PairWithVolume) []PairWithVolume {
	var filteredPairs []PairWithVolume
	for _, pair := range pairs {
		if pair.Symbol[len(pair.Symbol)-4:] == "USDT" {
			filteredPairs = append(filteredPairs, pair)
		}
	}
	return filteredPairs
}
