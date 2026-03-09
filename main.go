package main

import (
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pltanton/lingti-bot/cmd"
)

// Build is set via ldflags at build time
var Build = "unknown"

func main() {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: "https://80fde52f4b2be8cd4158bd0599a738d3@o336326.ingest.us.sentry.io/4511012077699072",
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	defer sentry.Flush(2 * time.Second)

	cmd.SetBuild(Build)
	cmd.Execute()
}
