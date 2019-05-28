package utils

import (
	"time"

	"github.com/beevik/ntp"
	"github.com/sirupsen/logrus"
)

var log = logrus.New().WithField("module", "ntp")

// TimeOffset is the offset from the current time of the computer.
var TimeOffset time.Duration

// CheckNTP queries an NTP server and asks for the current time
func CheckNTP() {
	res, err := ntp.Query("pool.ntp.org")
	if err != nil {
		log.Warn("could not connect to NTP server to check time offset")
		return
	}

	log.WithField("offset", res.ClockOffset).Info("got clock offset from NTP server")
	TimeOffset = res.ClockOffset
}

// Now gets the true time (not relying on computer time)
func Now() time.Time {
	return time.Now().Add(TimeOffset)
}
