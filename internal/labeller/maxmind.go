package labeller

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"

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

// Read the content from the secret directory. URL is given as a non empty string. Others can be.
// 1. First try to create the maxmind struct from command line arguments.
// 2. If they are not specified, look for the environment variables.
// 3. If they are also not specified, check the secrets mounted.
func NewMaxMindFromSecret(url, accountId, key string) (MaxMind, error) {
	// Try to build from command line arguments
	if url != "" && accountId != "" && key != "" {
		return maxMind{
			url:       url,
			accountId: accountId,
			key:       key,
		}, nil
	}

	// Get them from environment variables.
	accountId, key = os.Getenv("MAXMIND_ACCOUNTID"), os.Getenv("MAXMIND_KEY")
	if url != "" && accountId != "" && key != "" {
		return maxMind{
			url:       url,
			accountId: accountId,
			key:       key,
		}, nil
	}

	// Get them from secrets mounted by kubernetes.
	accountIdPath := "/var/run/secrets/edge-net.io/maxmind-secret/maxmind_accountid"
	keyPath := "/var/run/secrets/edge-net.io/maxmind-secret/maxmind_token"

	// Last resort, return error if there is one.
	accountIdBytes, err := os.ReadFile(accountId)
	if err != nil {
		return nil, err
	}

	// Last resort, return error if there is one.
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	accountIdBytes, err = base64.StdEncoding.DecodeString(string(accountIdBytes))

	if err != nil {
		return nil, err
	}

	keyBytes, err = base64.StdEncoding.DecodeString(string(keyBytes))

	if err != nil {
		return nil, err
	}

	accountId = string(accountIdBytes)
	key = string(keyBytes)

	// Try to build from command line arguments
	if url != "" && accountId != "" && key != "" {
		return maxMind{
			url:       url,
			accountId: accountId,
			key:       key,
		}, nil
	}

	return nil, errors.New("cannot read the maxmind secrets")
}

// This is for performing a lookup on the IP Address
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
