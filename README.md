# topgun-service

API service that provides a **sorted leaderboard for a single Vega Asset** e.g. `tDAI` or `xyzAlpha`, it also provides a quote value that is retrieved via the Vega price proxy if a base/quote pair (e.g. `BTC/USD`) are provided. The price is then used to generate a total quote value for the underlying Vega Asset (e.g. `USD`).

The leaderboard is filtered to include ONLY participants that are found on a verified 'allow-list' provided by an external API service. On which each user will verify their public key using Twitter.

The service is written in Go.

**Important: The design of this service was updated in March 2021 to match requirements for Fairground internal games. Service should be run with different configuration settings, in multiple instances, behind a caddy proxy to provide multiple leaderboards for a single Vega Asset e.g. xyzAlpha, xyzBeta, etc.**

## How to run the service

**Example:**

`./topgun-service -verifyurl=https://twitter-verifier.vega.trading/pubkeys.json -endpoint=https://lb.testnet.vega.xyz/query -base=BTC -quote=USD -vegaasset=tDAI `

**Arguments:**

- verifyurl - the http/web URL for the 3rd party social handle to pubkey verifier API service [required]
- addr - address:port to bind the service to [optional, default: localhost:8000]
- endpoint - endpoint url to send graphql queries to [required]
- timeout - the duration for which the server gracefully waits for existing connections to finish [optional, default: 15s]
- vegapoll - the duration for which the service will poll the Vega API for accounts [optional, default: 5s]
- vegaasset - Vega asset, e.g. tDAI [required]
- base - Base for price fetching e.g. BTC [optional, recommended]
- quote - Quote for price fetching e.g. USD [optional, recommended]

**Queries:**

- `/status` - useful for health returns 200 if service is up.
- `/leaderboard` - returns the leaderboard in json format, example below:

```
{
  "lastUpdate": "1616774855",
  "base": "BTC",
  "quote": "USD",
  "asset": "tDAI",
  "traders": [
    {
      "order": 1,
      "publicKey": "ac9d9fe2e5904308d9c0f6fe758f8a4f4dd9636ab35584f95909010b7ec7edc9",
      "twitterHandle": "fuzzydunlop99",
      "balanceGeneral": 96665.03674,
      "balanceMargin": 1183.04528,
      "balanceTotal": 97848.08202,
      "quoteGeneral": 5133820937.179082,
      "quoteMargin": 62830810.73491857,
      "quoteTotal": 5196651747.9140005
    },
    {
      "order": 2,
      "publicKey": "6e0c7741220ba99187b59a0b52271b16e02dabd4e38c75e7cfa128f0f784e8a7",
      "twitterHandle": "crypt0wenm00n",
      "balanceGeneral": 56665.03674,
      "balanceMargin": 2183.04528,
      "balanceTotal": 58848.08202,
      "quoteGeneral": 4333820937.179082,
      "quoteMargin": 23830810.73491857,
      "quoteTotal": 4496651747.9140005
    },
    ...
  ]
}

```

## How to file an issue or report a problem

Please use the Issues tab in the topgun-service repository in GitHub.
