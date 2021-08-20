package util

import (
	"fmt"
	"time"
)

func UnixTimestampUtcNowFormatted() string {
	return fmt.Sprintf("%d", time.Now().UTC().Unix())
}

func TimeFromUnixTimeStamp(unixTimestamp int64) time.Time {
	return time.Unix(unixTimestamp, 0)
}
