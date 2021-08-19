# Leaderboard API service for Fairground incentives (code name TOPGUN)

API service that provides a **sorted leaderboard for incentives/games operating on the Fairground testnet.**

The leaderboard is filtered to include ONLY participants that are found on a verified 'allow-list' provided by an external API service. On which each user will verify their public key using Twitter. This service is known internally as **Social Media Verification** or "Twitter Registration".

When running an incentive/game the configuration file for the topgun-service can be configured with the appropriate 'algorithm' to serve up a list of participants on a leaderboard. The choices of algorithm currently includes:

* `ByPartyAccountGeneralBalance` - Sorted by trading account total general balance of given asset
* `ByPartyGovernanceVotes` - Sorted by trading account governance votes
* `ByLPEquitylikeShare` - Sorted by LP equity like share
* `ByAssetDepositWithdrawal` - Sorted by ERC20 assets deposited and withdrawn (achieved when user deposits and withdraws 2 unique assets) 
* `BySocialRegistration` - Sorted by latest Twitter registrations (used to check that a twitter handle is verified/signed up for incentives)

The service is written in Go and more recent algorithms use MongoDB as a perisistence layer.

## How to run the service

**Example:**

`./topgun-service -config custom-config-file.yaml`

The application requires a custom configuration file passed in the argument named `-config`, an example can be found here. Details of the config variables are detailed below:

**Config:**

- listen - the address:port for the service to bind to e.g. 127.0.0.1:8000]
- logFormat - format for logging e.g. text
- logLevel - level of logging e.g. Info
- LogMethodName - logging displays method name e.g. False
- socialURL - the http/web URL for the 3rd party social handle to pubkey verifier API service
- vegaGraphQLUrl - endpoint url to send graphql queries to
- gracefulShutdownTimeout - the duration for which the server gracefully waits for existing connections to finish e.g. 15s
- vegapoll - the duration for which the service will poll the Vega API for accounts e.g. 5s
- vegaasset - Vega asset, e.g. tDAI
- base - Base for price fetching e.g. BTC
- quote - Quote for price fetching e.g. USD
- headers - A collection of custom headers returned with the data in a leaderboard e.g. Asset Total

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

## Verified socials

A mapping of public key to social handle (Twitter) is provided by an external service, please see the file `verified_example.txt` for an example of the format returned. An attempt to update this list from the 3rd party server happens on each reload of the data from Vega, see `vegapoll` time parameter above. This service is operated by Vega and is known internally as **Social Media Verification** or "Twitter Registration".

## How to file an issue or report a problem

Please use the Issues tab in the topgun-service repository in GitHub.
