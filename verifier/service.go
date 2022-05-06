package verifier

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Socials struct {
	Socials []Social
}

type Social struct {
	PartyID       string `json:"party_id"`
	TwitterHandle string `json:"twitter_handle"`
	TwitterUserID int64  `json:"twitter_user_id"`
	CreatedAt     int64  `json:"created"`
	UpdatedAt     int64  `json:"last_modified"`
	IsBlacklisted bool   `json:"is_blacklisted"`
}

type Service struct {
	mu         sync.RWMutex
	blacklist  map[string]string
	socialList *Socials
	verifyURL  url.URL
}

func NewVerifierService(verifyURL url.URL, blacklist map[string]string) *Service {
	socialList := make([]Social, 0)
	socialHolder := Socials{Socials: socialList}
	s := Service{
		verifyURL:  verifyURL,
		socialList: &socialHolder,
		blacklist: blacklist,
	}
	return &s
}

func (s *Service) UpdateVerifiedParties() {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Info("Syncing verified parties from external social verifier API service")
	socials, err := s.loadVerifiedParties()
	previousSocialList := s.getSocialList()

	foundTotal := 0
	if err != nil {
		log.Error(errors.Wrap(err, "failed to update/sync verified parties"))
	} else {
		log.Info("Verified parties loaded from external API service")
		foundTotal = len(socials.Socials)
		s.socialList = socials
	}

	log.Infof("Parties found: %d, last total: %d", foundTotal, len(previousSocialList))
}

func (s *Service) List() []Social {
	soc := *s.socialList
	return soc.Socials
}

func (s *Service) processBlacklisted(socials []Social) []Social {
	if socials == nil {
		return nil
	}
	socialList := make([]Social, 0)
	for _, soc := range socials {
		sUID:= strconv.FormatInt(soc.TwitterUserID, 10)
		if _, found := s.blacklist[sUID]; found {
			log.Infof("Found blacklisted user: %s - %d", soc.TwitterHandle, soc.TwitterUserID)
			soc.IsBlacklisted = true
		}
		socialList = append(socialList, soc)
	}
	return socialList
}

func (s *Service) getSocialList() []Social {
	socialList := Socials{}
	if s.socialList != nil {
		socialList = *s.socialList
	}
	return socialList.Socials
}

func (s *Service) PubKeysToSocials() map[string]Social {
	result := map[string]Social{}
	socialList := s.getSocialList()
	for _, m := range socialList {
		result[m.PartyID] = m
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
		var res []Social
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the mapping returned from verifier service")
		}
		found := res
		return &Socials{Socials: s.processBlacklisted(found)}, nil
	} else {
		return nil, errors.New(fmt.Sprintf("wrong status code returned from verifier service: %d", resp.StatusCode))
	}
}
