package labeller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/savaki/geoip2"
)

type MaxMind interface {
	MaxMindLookup(address string) (*geoip2.Response, error)
}

type maxMind struct {
	MaxMind
	url       string
	accountId string
	key       string
}

// Read the content from the secret directory
func NewMaxMindFromSecret() (MaxMind, error) {
	return nil, errors.New("not yet implemented")
}

// This is for
func (mm maxMind) MaxMindLookup(address string) (*geoip2.Response, error) {
	req, err := http.NewRequest("GET", mm.url+address, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(mm.accountId, mm.key)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		v := geoip2.Error{}
		err := json.NewDecoder(res.Body).Decode(&v)
		if err != nil {
			return nil, err
		}
		return nil, v
	}
	response := &geoip2.Response{}
	err = json.NewDecoder(res.Body).Decode(response)
	return response, err
}
