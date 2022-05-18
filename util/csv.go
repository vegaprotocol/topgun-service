package util

import (
	"time"
)

type ParticipantCsvEntry struct { // Note: use "-" to ignore a field
	Position      int       `csv:"position"`
	TwitterHandle string    `csv:"twitter_handle"`
	TwitterID     int64     `csv:"twitter_user_id"`
	CreatedAt     time.Time `csv:"created_at"`
	UpdatedAt     time.Time `csv:"updated_at"`
	VegaPubKey    string    `csv:"vega_pubkey"`
	VegaData      string    `csv:"vega_data"`
}
