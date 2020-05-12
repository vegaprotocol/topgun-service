# topgun-service

Lightweight API service that provides sorted leaderboard data for topgun.[testnet/stagnet/devnet].vega.[trading/xyz]

The service is written in Go and is designed to poll a Bitstamp for the latest asset price of BTC (used for converting current value of BTC->USD, and to also poll a Vega API node via GraphQL to retrieve account data for parties. The poll rate for both Bitstamp and Vega API queries are configurable. If Bitstamp is not available a fallback asset price is used, we refer to asset price as the last BTC->USD price from the exchange. The service caches the list of accounts for the `vegapoll` time, it will retry on failure.

## How to run the service

**Example:**

`./topgun-service -whitelist=./csv/whitelist-nicenet.csv -endpoint=https://lb.n.vega.xyz/query`

**Arguments:**

- whitelist - the path to the csv file containing partyIDs to whitelist for the leaderboard [required]
- addr - address:port to bind the service to [optional, default: localhost:8000]
- endpoint - endpoint url to send graphql queries to [required]
- timeout - the duration for which the server gracefully waits for existing connections to finish [optional, default: 15s]
- assetpoll - the duration for which the service will poll the exchange for asset price [optional, default: 30s]
- vegapoll - the duration for which the service will poll the Vega API for accounts [optional, default: 5s]

**Queries:**

- `/status` - useful for health returns 200 if service is up.
- `/leaderboard` - returns the leaderboard in json format, example below:

```
[...
{"PartyID":"41fe7f57d6d8a05756f1109caaffbeb0fa0623f7c91ec830d9d823ac1031c3cb","BalanceUSD":5000,"BalanceBTC":10.001,"DeployedUSD":0,"DeployedBTC":0,"TotalUSD":92466.74579999999,"TotalUSDWithDeployed":92466.74579999999},
{"PartyID":"8014938747bdbeb70b28a9c77f16bbd86a1068bc98db40dc935aa0416c94b6ea","BalanceUSD":5000,"BalanceBTC":10,"DeployedUSD":0,"DeployedBTC":0,"TotalUSD":92458,"TotalUSDWithDeployed":92458},
...]
```


## Whitelists

Due to Vega using public-key identifiers as parties, we need to specify a 'whitelist' when running the service. This ensures we filter out all the bots that are operating on a network from the leaderboard. Vega whitelists can be found in the `/csv/` directory.

## How to file an issue or report a problem

Please use the Issues tab in the topgun-service repository in GitHub.
