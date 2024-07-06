# FlightAware Firehose Client for Go

This is a client library for [FlightAware's Firehose Flight Data Feed](https://www.flightaware.com/commercial/firehose/), a real-time data feed of global aircraft ADS-B positions and flight status.

This is an _unofficial_ library and is not endorsed or supported by FlightAware.

[![Go Reference](https://pkg.go.dev/badge/github.com/benburwell/firehose.svg)](https://pkg.go.dev/github.com/benburwell/firehose)

**This library is a work in progress!** Currently only `position` messages are supported.

## Getting Started

To use Firehose, you'll need to set up API credentials. Log in to your FlightAware account and visit your [Firehose Dashboard](https://www.flightaware.com/account/manage/firehosedash) to view get your API key.

Once you have your credentials, you can use them in your project. Here's an example:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/benburwell/firehose"
)

func main() {
	// Open a basic connection to Firehose.
	stream, err := firehose.Connect()
	if err != nil {
		log.Fatal(err)
	}

	// Initiate the stream
	init := firehose.InitCommand{
		// Get events starting from the present
		Live:     true,
		// Provide your credentials
		Username: os.Getenv("FIREHOSE_USERNAME"),
		Password: os.Getenv("FIREHOSE_PASSWORD"),
		// Specify the event types you want to receive
		Events:   []firehose.Event{firehose.PositionEvent},
	}
	if err := stream.Init(init.String()); err != nil {
		log.Fatal(err)
	}
	
	for {
		// Iterate over received messages from the stream
		msg, err := stream.NextMessage(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		switch m := msg.Payload.(type) {
		case firehose.PositionMessage:
			fmt.Printf("%s is at %sºN, %sºE\n", m.Ident, m.Lat, m.Lon)
		}
	}
}
```
