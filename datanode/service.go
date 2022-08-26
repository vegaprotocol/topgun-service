package datanode

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	baseUrlStaging    = "https://n02.stagnet2.vega.xyz"
	baseUrlTestnet    = "https://lb.testnet.vega.xyz"
	baseUrl = baseUrlTestnet
)

type RewardResponse struct {
	Rewards []Reward `json:"rewardDetails"`
}

type Reward struct {
	Asset   string `json:"asset"`
	Details []RewardDetail `json:"details"`
	TotalForAsset string `json:"totalForAsset"`
}

type RewardDetail struct {
	AssetID           string `json:"assetId"`
	PartyID           string `json:"partyId"`
	Epoch             string `json:"epoch"`
	Amount            string `json:"amount"`
	PercentageOfTotal string `json:"percentageOfTotal"`
	ReceivedAt        string `json:"receivedAt"`
}

type WithdrawalResponse struct {
	Withdrawals []Withdrawal `json:"withdrawals"`
}

type Withdrawal struct {
	ID                 string `json:"id"`
	PartyID            string `json:"partyId"`
	Amount             string `json:"amount"`
	Asset              string `json:"asset"`
	Status             string `json:"status"`
	Ref                string `json:"ref"`
	Expiry             string `json:"expiry"`
	TxHash             string `json:"txHash"`
	CreatedTimestamp   string `json:"createdTimestamp"`
	WithdrawnTimestamp string `json:"withdrawnTimestamp"`
	Ext                struct {
		Erc20 struct {
			ReceiverAddress string `json:"receiverAddress"`
		} `json:"erc20"`
	} `json:"ext"`
}

type DelegationResponse struct {
	Delegations []Delegation `json:"delegations"`
}

type Delegation struct {
	Party    string `json:"party"`
	NodeID   string `json:"nodeId"`
	Amount   string `json:"amount"`
	EpochSeq string `json:"epochSeq"`
	EpochID  int64
}

type EpochResponse struct {
	Epoch Epoch `json:"epoch"`
}

type Epoch struct {
	Seq         string       `json:"seq"`
	Timestamps  Timestamps   `json:"timestamps"`
	Validators  []Validator  `json:"validators"`
	Delegations []Delegation `json:"delegations"`
}

type Timestamps struct {
	StartTime  string `json:"startTime"`
	ExpiryTime string `json:"expiryTime"`
	EndTime    string `json:"endTime"`
	FirstBlock string `json:"firstBlock"`
	LastBlock  string `json:"lastBlock"`
}

type Validator struct {
	ID                string        `json:"id"`
	PubKey            string        `json:"pubKey"`
	TmPubKey          string        `json:"tmPubKey"`
	EthereumAddress   string        `json:"ethereumAdddress"`
	InfoURL           string        `json:"infoUrl"`
	Location          string        `json:"location"`
	StakedByOperator  string        `json:"stakedByOperator"`
	StakedByDelegates string        `json:"stakedByDelegates"`
	StakedTotal       string        `json:"stakedTotal"`
	MaxIntendedStake  string        `json:"maxIntendedStake"`
	PendingStake      string        `json:"pendingStake"`
	EpochData         interface{}   `json:"epochData"`
	Status            string        `json:"status"`
	Delegations       []Delegation  `json:"delegations"`
	Score             string        `json:"score"`
	NormalisedScore   string        `json:"normalisedScore"`
	Name              string        `json:"name"`
	AvatarURL         string        `json:"avatarUrl"`
}

type ProposalResponse struct {
	Data []Proposal `json:"data"`
}
type NewFreeform struct {
	URL         string `json:"url"`
	Description string `json:"description"`
	Hash        string `json:"hash"`
}
type Terms struct {
	ClosingTimestamp    string      `json:"closingTimestamp"`
	EnactmentTimestamp  string      `json:"enactmentTimestamp"`
	ValidationTimestamp string      `json:"validationTimestamp"`
	NewFreeform         *NewFreeform `json:"newFreeform"`
}
type ProposalDetails struct {
	ID           string `json:"id"`
	Reference    string `json:"reference"`
	PartyID      string `json:"partyId"`
	State        string `json:"state"`
	Timestamp    string `json:"timestamp"`
	Terms        Terms  `json:"terms"`
	Reason       string `json:"reason"`
	ErrorDetails string `json:"errorDetails"`
}
type Vote struct {
	PartyID                     string `json:"partyId"`
	Value                       string `json:"value"`
	ProposalID                  string `json:"proposalId"`
	Timestamp                   string `json:"timestamp"`
	TotalGovernanceTokenBalance string `json:"totalGovernanceTokenBalance"`
	TotalGovernanceTokenWeight  string `json:"totalGovernanceTokenWeight"`
}
type YesParty struct { }
type NoParty struct { }
type Proposal struct {
	Proposal ProposalDetails `json:"proposal"`
	Yes      []Vote          `json:"yes"`
	No       []Vote          `json:"no"`
	YesParty YesParty        `json:"yesParty"`
	NoParty  NoParty          `json:"noParty"`
}

type LiquidityOrder struct {
	Reference  string `json:"reference"`
	Proportion int    `json:"proportion"`
	Offset     string `json:"offset"`
}

type Buys struct {
	LiquidityOrder LiquidityOrder `json:"liquidityOrder"`
}

type Sells struct {
	LiquidityOrder LiquidityOrder `json:"liquidityOrder"`
}

type LiquidityProvision struct {
	ID               string    `json:"id"`
	PartyID          string    `json:"partyId"`
	CommitmentAmount string    `json:"commitmentAmount"`
	CreatedAt        string    `json:"createdAt"`
	UpdatedAt        string    `json:"updatedAt"`
	Status           string    `json:"status"`
	Fee              string    `json:"fee"`
	Version          string    `json:"version"`
	Reference        string    `json:"reference"`
	Buys             []Buys    `json:"buys"`
	Sells            []Sells   `json:"sells"`
}

type LiquidityProvisionResponse struct {
	LiquidityProvisions []LiquidityProvision `json:"liquidityProvisions"`
}

type AccountsResponse struct {
	Accounts []Account `json:"accounts"`
}
type Account struct {
	ID       string `json:"id"`
	Owner    string `json:"owner"`
	Balance  string `json:"balance"`
	Asset    string `json:"asset"`
	MarketID string `json:"marketId"`
	Type     string `json:"type"`
	BalanceF float64
	DepositF float64
	ProfitF   float64
}

type DepositsResponse struct {
	Deposits []Deposit `json:"deposits"`
}
type Deposit struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	PartyID           string `json:"partyId"`
	Asset             string `json:"asset"`
	AssetF            float64
	Amount            string `json:"amount"`
	TxHash            string `json:"txHash"`
	CreditedTimestamp string `json:"creditedTimestamp"`
	CreatedTimestamp  string `json:"createdTimestamp"`
}

func LoadDepositsForParty(partyId string) ([]Deposit, error) {

	//log.Infof(baseUrl  + "/datanode/rest/withdrawals/party/" + partyId)
	log.Infof(baseUrl + "/datanode/rest/parties/" + partyId + "/deposits")

	resp, err := http.Get(baseUrl + "/datanode/rest/parties/" + partyId + "/deposits")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := make([]Deposit, 0)
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var res DepositsResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the deposit data returned from data node")
		}
		if len(res.Deposits) > 0 {
			result = res.Deposits
		}
		return result, nil
	} else {
		return nil,
			errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	}
}

func LoadAccountsForParty(partyId string) ([]Account, error) {
	resp, err := http.Get(baseUrl + "/datanode/rest/parties/" + partyId + "/accounts")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := make([]Account, 0)
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var res AccountsResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the account data returned from data node")
		}
		if len(res.Accounts) > 0 {
			result = res.Accounts
		}
		return result, nil
	} else if resp.StatusCode == http.StatusInternalServerError {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if len(body) > 0 {
			content := string(body)
			if strings.Contains(content, "Not Found") {
				return result, nil
			}
		}
		return result,
			errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	} else {
		return nil,
			errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	}
}

func LoadRewardsForParty(partyId string) ([]Reward, error) {
	resp, err := http.Get(baseUrl + "/datanode/rest/parties/" + partyId + "/rewards")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := make([]Reward, 0)
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var res RewardResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the reward data returned from data node")
		}
		if len(res.Rewards) > 0 {
			result = res.Rewards
		}
		return result, nil
	} else if resp.StatusCode == http.StatusInternalServerError {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if len(body) > 0 {
			content := string(body)
			if strings.Contains(content, "no rewards found for partyid") {
				return result, nil
			}
		}
		return result,
			errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	} else {
		return nil,
			errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	}
}

func LoadLiquidityProvisionsForParty(partyId string, marketId string) ([]LiquidityProvision, error) {
	log.Info(baseUrl + "/datanode/rest/liquidity-provisions/party/" + partyId + "/market/" + marketId)

	resp, err := http.Get(baseUrl + "/datanode/rest/liquidity-provisions/party/" + partyId + "/market/" + marketId)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := make([]LiquidityProvision, 0)
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var res LiquidityProvisionResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the lp data returned from data node")
		}
		if len(res.LiquidityProvisions) > 0 {
			result = res.LiquidityProvisions
		}
		return result, nil
	} else {
		return nil, errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	}
}

func LoadWithdrawalsForParty(partyId string) ([]Withdrawal, error) {

	log.Infof(baseUrl  + "/datanode/rest/withdrawals/party/" + partyId)

	resp, err := http.Get(baseUrl + "/datanode/rest/withdrawals/party/" + partyId)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := make([]Withdrawal, 0)
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var res WithdrawalResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the withdrawal data returned from data node")
		}
		if len(res.Withdrawals) > 0 {
			result = res.Withdrawals
		}
		return result, nil
	} else {
		return nil, errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	}
}

func LoadGovernanceProposals(partyId string) ([]Proposal, error) {
	log.Debugf(baseUrl + "/datanode/rest/parties/" + partyId + "/proposals")
	resp, err := http.Get(baseUrl + "/datanode/rest/parties/" + partyId + "/proposals")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var res ProposalResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the delegation data returned from data node")
		}
		return res.Data, nil
	} else {
		return nil, errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	}
}

func LoadDelegationsForParty(partyId string) ([]Delegation, error) {
	resp, err := http.Get(baseUrl + "/datanode/rest/delegations?party=" + partyId)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := make([]Delegation, 0)
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var res DelegationResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the delegation data returned from data node")
		}
		if len(res.Delegations) > 0 {
			// Convert epoch ID to int and store in extra ID var
			for _, d := range res.Delegations {
				foundEpochVal, err := strconv.ParseInt(d.EpochSeq, 10, 64)
				if err != nil {
					log.WithError(err).Error("Cannot parse the found epoch in delegation from data-node")
					return result, err
				}
				d.EpochID = foundEpochVal
				result = append(result, d)
			}
			// Sort by epoch ID
			sortFunc := func(i, j int) bool {
				return result[i].EpochID > result[j].EpochID
			}
			sort.Slice(result, sortFunc)
		}
		return result, nil
	} else {
		return nil, errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	}
}

func LoadDelegations() ([]Delegation, error) {
	resp, err := http.Get(baseUrl + "/datanode/rest/delegations")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := make([]Delegation, 0)
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var res DelegationResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the delegation data returned from data node")
		}
		if len(res.Delegations) > 0 {
			result = res.Delegations
		}
		return result, nil
	} else {
		return nil, errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	}
}

func LoadEpochForId(id string) (*Epoch, error) {

	log.Infof(baseUrl + "/datanode/rest/epochs?id=" + id)

	resp, err := http.Get(baseUrl + "/datanode/rest/epochs?id=" + id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	//body, err := ioutil.ReadAll(resp.Body)
	//log.Infof(string(body))

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var res EpochResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the delegation data returned from data node")
		}
		return &res.Epoch, nil
	} else {
		return nil, errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	}
}

func LoadCurrentEpoch() (*Epoch, error) {

	log.Infof(baseUrl + "/datanode/rest/epochs")

	resp, err := http.Get(baseUrl + "/datanode/rest/epochs")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var res EpochResponse
		err = json.Unmarshal(body, &res)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal the delegation data returned from data node")
		}
		return &res.Epoch, nil
	} else {
		return nil, errors.New(fmt.Sprintf("unexpected status code returned from vega data node: %d", resp.StatusCode))
	}
}


