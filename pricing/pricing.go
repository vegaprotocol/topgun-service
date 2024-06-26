package pricing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	ppconfig "code.vegaprotocol.io/priceproxy/config"
	ppservice "code.vegaprotocol.io/priceproxy/service"
	"github.com/pkg/errors"
)

// Engine represents a pricing engine. Do not use this directly. Use New() and an interface.
type Engine struct {
	address url.URL
	client  http.Client
}

// NewEngine creates a new pricing engine
func NewEngine(address url.URL) *Engine {
	e := Engine{
		address: address,
		client:  http.Client{},
	}

	return &e
}

// GetPrice fetches a live/recent price from the price proxy.
func (e *Engine) GetPrice(pricecfg ppconfig.PriceConfig) (pi ppservice.PriceResponse, err error) {
	v := url.Values{}
	if pricecfg.Source != "" {
		v.Set("source", pricecfg.Source)
	}
	if pricecfg.Base != "" {
		v.Set("base", pricecfg.Base)
	}
	if pricecfg.Quote != "" {
		v.Set("quote", pricecfg.Quote)
	}
	v.Set("wander", fmt.Sprintf("%v", pricecfg.Wander))
	relativeURL := &url.URL{RawQuery: v.Encode()}
	fullURL := e.address.ResolveReference(relativeURL).String()
	req, _ := http.NewRequest(http.MethodGet, fullURL, nil)

	var resp *http.Response
	resp, err = e.client.Do(req)
	if err != nil {
		err = errors.Wrap(err, "failed to perform HTTP request")
		return
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = errors.Wrap(err, "failed to read HTTP response body")
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad response: HTTP %d %s", resp.StatusCode, string(content))
		return
	}

	var response ppservice.PricesResponse
	if err = json.Unmarshal(content, &response); err != nil {
		err = errors.Wrap(err, "failed to parse HTTP response as JSON")
		return
	}

	if len(response.Prices) == 0 {
		err = errors.New("zero-length price list from Price Proxy")
		return
	}

	return *response.Prices[0], nil
}
