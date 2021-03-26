package verifier

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type QueryResult struct {
	Parties []Social `json:"parties"`
}

type Social struct {
	Party  string `json:"party"`
	Handle string `json:"handle"`
}

type Service struct {
	mu        sync.RWMutex
	socials   *[]Social
	verifyUrl string
}

func NewVerifierService(verifyUrl string) *Service {
	socials := make([]Social, 0)
	s := Service{
		verifyUrl: verifyUrl,
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

func (s *Service) List() *[]Social {
	return s.socials
}

func (s *Service) Dictionary() map[string]Social {
	result := map[string]Social{}
	for _, m := range *s.socials {
		result[m.Party] = m
	}
	return result
}

func (s *Service) loadVerifiedParties() (*[]Social, error) {
	if s.verifyUrl == "" {
		return nil, errors.New("social verifier URL not specified or empty")
	}

	resp, err := http.Get(s.verifyUrl)
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
		var res QueryResult
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the mapping returned from verifier service")
		}
		found := res.Parties
		return &found, nil
	} else {
		return nil, errors.New(fmt.Sprintf("wrong status code returned from verifier service: %d", resp.StatusCode))
	}
}
