# topgun-service

Lightweight API service that provides leaderboard data for topgun.[testnet/stagnet/devnet].vega.[trading/xyz]

The service is written in Go and is designed to poll a Bitstamp for the latest asset price of BTC (used for converting current value of BTC->USD, and to also poll a Vega API node via GraphQL to retrieve account data for parties. The poll rate for both Bitstamp and Vega API queries are configurable.

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

## Whitelists

Due to Vega using public-key identifiers as parties, we need to specify a 'whitelist' when running the service. This ensures we filter out all the bots that are operating on a network from the leaderboard. Vega whitelists can be found in the `/csv/` directory.

## How to file an issue or report a problem

Please use the Issues tab in the topgun-service repository in GitHub.
