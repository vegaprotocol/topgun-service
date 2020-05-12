package util

import (
	"fmt"
	"time"
)

func UnixTimestampUtcNowFormatted() string {
	return fmt.Sprintf("%d", time.Now().UTC().Unix())
}
