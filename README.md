# Leaderboard API service for Fairground incentives (code name TOPGUN)

API service that provides a **sorted leaderboard for incentives/games operating on the Fairground testnet.**

The leaderboard is filtered to include ONLY participants that are found on a verified 'allow-list' provided by an external API service. On which each user will verify their public key using Twitter. This service is known internally as **Social Media Verification** or "Twitter Registration". An optional blacklist can be configured to exclude users from the main public leaderboard, this feature is useful for team members or known bots/scammers.

When running an incentive/game the configuration file for the topgun-service can be configured with the appropriate 'algorithm' to serve up a list of participants on a leaderboard. The choices of algorithm currently includes:

* `ByPartyAccountGeneralBalance` - Sorted by trading account total general balance of given asset
* `ByPartyAccountGeneralBalanceLP` - Sorted by trading account total general balance of given asset and must have submitted LP for configured `MarketID`
* `ByPartyAccountGeneralProfit` - Sorted by profit algorithm ((balanceGeneral - depositTotal)/depositTotal) for given asset
* `ByPartyAccountGeneralProfitLP` - Sorted by profit algorithm (as above) for given asset and must have submitted LP for configured `MarketID`
* `ByPartyGovernanceVotes` - Sorted by trading account governance votes
* `ByLPEquitylikeShare` - Sorted by LP equity like share
* `ByAssetDepositWithdrawal` - Sorted by ERC20 assets deposited and withdrawn (achieved when user deposits and withdraws 2 unique assets) 
* `BySocialRegistration` - Sorted by latest Twitter registrations (used to check that a twitter handle is verified/signed up for incentives)

The service is written in Go and more recent algorithms use MongoDB as a persistence layer.

## How to run the service

**Example:**

`./topgun-service -config custom-config-file.yaml`

The application requires a custom configuration file passed in the argument named `-config`, an example can be found [here](./example-custom-config-file.yaml). Details of the config variables are detailed below:

**Config:**

- listen - the address:port for the service to bind to e.g. 127.0.0.1:8000
- logFormat - format for logging e.g. text
- logLevel - level of logging e.g. Info
- LogMethodName - logging displays method name e.g. False
- socialURL - the http/web URL for the 3rd party social handle to pubkey verifier API service
- vegaGraphQLUrl - endpoint url to send graphql queries to
- gracefulShutdownTimeout - the duration for which the server gracefully waits for existing connections to finish e.g. 15s
- vegapoll - the duration for which the service will poll the Vega API for accounts e.g. 5s
- vegaassets - a collection of one or more Vega asset IDs, e.g. XYZAlpha, etc
- base - Base for price fetching e.g. BTC
- quote - Quote for price fetching e.g. USD
- defaultDisplay - the default display name/data for the leaderboard
- defaultSort - the default sort name/data for the leaderboard
- headers - A collection of custom headers returned with the data in a leaderboard e.g. Asset Total
- startTime - the start time for the incentive period
- endTime - the end time for the incentive period
- twitterBlacklist - a map/list of twitterUserID: twitterHandle that should be excluded from the default leaderboard

**MongoDB:**

- mongoConnectionString - the full connection string for the optional mongodb database
- mongoCollectionName - the collection name for the leaderboard data to be stored
- mongoDatabaseName - the database name for the leaderboard collection

Optionally, algorithms can make use of persisting and sharing data collections stored in MongoDB, useful to preserve 
state of incentives throughout resets and other events like restarts. Currently only the `ByAssetDepositWithdrawal` 
algorithm makes use of mongodb, other algos should set these fields to the value `NA` or similar as shown in the example 
config file.

**Queries:**

- `/status` - useful for health returns 200 if service is up
- `/leaderboard` - returns the leaderboard in json format
   -  `?q={social_handle}` - search query to filter for a specific social handle, case insensitive
   -  `?skip={n}` - skip `n` leaderboard results (pagination)
   -  `?size={n}` - page size `n` leaderboard results (pagination)
   -  `?type={csv|json}` - return type of results, default JSON
   -  `?blacklisted={true|false}` - Return leaderboard of blacklisted users, default: `false`

## Verified socials

A mapping of public key to social handle (Twitter) is provided by an external service, please see the file `verified_example.txt` for an example of the format returned. An attempt to update this list from the 3rd party server happens on each reload of the data from Vega, see `vegapoll` time parameter above. This service is operated by Vega and is known internally as **Social Media Verification** or "Twitter Registration".

## How to file an issue or report a problem

Please use the Issues tab in the topgun-service repository in GitHub.
