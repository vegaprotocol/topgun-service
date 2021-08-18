package verifier

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Socials []struct {
	PartyID       string `json:"party_id"`
	TwitterHandle string `json:"twitter_handle"`
	CreatedAt     int64 `json:"created"`
	UpdatedAt     int64 `json:"last_modified"`
}

type Service struct {
	mu        sync.RWMutex
	socials   *Socials
	verifyURL url.URL
}

func NewVerifierService(verifyURL url.URL) *Service {
	socials := make(Socials, 0)
	s := Service{
		verifyURL: verifyURL,
		socials:   &socials,
	}
	return &s
}

func (s *Service) UpdateVerifiedParties() {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Info("Syncing verified parties from external social verifier API service")
	socials, err := s.loadVerifiedParties()
	foundTotal := 0
	if err != nil {
		log.Error(errors.Wrap(err, "failed to update/sync verified parties"))
	} else {
		log.Info("Verified parties loaded from external API service")
		foundTotal = len(*s.socials)
		s.socials = socials
	}

	log.Infof("Parties found: %d, last total: %d", len(*s.socials), foundTotal)
}

func (s *Service) List() *Socials {
	return s.socials
}

func (s *Service) PubKeysToTwitterHandles() map[string]string {
	result := map[string]string{}
	for _, m := range *s.socials {
		result[m.PartyID] = m.TwitterHandle
	}
	return result
}

func (s *Service) loadVerifiedParties() (*Socials, error) {
	resp, err := http.Get(s.verifyURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		// Decode the result
		var res Socials
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the mapping returned from verifier service")
		}
		found := res
		return &found, nil
	} else {
		return nil, errors.New(fmt.Sprintf("wrong status code returned from verifier service: %d", resp.StatusCode))
	}
}
