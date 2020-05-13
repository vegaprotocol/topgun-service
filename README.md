# topgun-service

Lightweight API service that provides sorted leaderboard data for topgun.[testnet/stagnet/devnet].vega.xyz

The service is written in Go and is designed to poll a Bitstamp for the latest asset price of BTC (used for converting current value of BTC->USD, and to also poll a Vega API node via GraphQL to retrieve account data for parties. The poll rate for both Bitstamp and Vega API queries are configurable. If Bitstamp is not available a fallback asset price is used, we refer to asset price as the last BTC->USD price from the exchange. The service caches the list of accounts for the `vegapoll` time, it will retry on failure.

Note: Only parties on Vega that are on a whitelist, and have made a trade (deployed either VBTC or USD from their initial allowance) will appear on the response.

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
{
  "lastUpdate": "1589287357",
  "traders": [
    {
      "order": 1,
      "publicKey": "41fe7f57d6d8a05756f1109caaffbeb0fa0623f7c91ec830d9d823ac1031c3cb",
      "usdVal": 5000,
      "usd": "5000.00000",
      "btcVal": 10.001,
      "btc": "10.00100",
      "usdDeployedVal": 0,
      "usdDeployed": "0.00000",
      "btcDeployedVal": 0,
      "btcDeployed": "0.00000",
      "totalUsdVal": 92654.56457999999,
      "totalUsd": "92654.56458",
      "totalUsdDeployedVal": 92654.56457999999,
      "totalUsdDeployed": "92654.56458"
    },
    {
      "order": 2,
      "publicKey": "dc06bf6329ec5779f5e254468895958f3b7ac1d65e8aa82bac6fc0ed3d068a95",
      "usdVal": 4999.61701,
      "usd": "4999.61701",
      "btcVal": 9.77604,
      "btc": "9.77604",
      "usdDeployedVal": 0.37899,
      "usdDeployed": "0.37899",
      "btcDeployedVal": 0.19496,
      "btcDeployed": "0.19496",
      "totalUsdVal": 90682.50167320001,
      "totalUsd": "90682.50167",
      "totalUsdDeployedVal": 92391.62318000001,
      "totalUsdDeployed": "92391.62318"
    },
    {
      "order": 3,
      "publicKey": "41a4a9ed049863a05969847abc1abf36f5e731f912ddffddd7cb66723dae79bb",
      "usdVal": 344.46977,
      "usd": "344.46977",
      "btcVal": 10,
      "btc": "10.00000",
      "usdDeployedVal": 3776.46823,
      "usdDeployed": "3776.46823",
      "btcDeployedVal": 0,
      "btcDeployed": "0.00000",
      "totalUsdVal": 87990.26977,
      "totalUsd": "87990.26977",
      "totalUsdDeployedVal": 91766.738,
      "totalUsdDeployed": "91766.73800"
    }
  ]
}
```


## Whitelists

Due to Vega using public-key identifiers as parties, we need to specify a 'whitelist' when running the service. This ensures we filter out all the bots that are operating on a network from the leaderboard. Whitelists are a simple list with one pubkey per newline that should be included in the leaderboard.

## How to file an issue or report a problem

Please use the Issues tab in the topgun-service repository in GitHub.
