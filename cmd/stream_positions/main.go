package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/benburwell/firehose"
)

func main() {
	var app App
	flag.StringVar(&app.Username, "username", "", "Firehose Username")
	flag.StringVar(&app.Password, "password", "", "Password/API key")
	flag.Float64Var(&app.MinLat, "min-lat", -90, "Bounding box minimum latitude")
	flag.Float64Var(&app.MinLon, "min-lon", -180, "Bounding box minimum longitude")
	flag.Float64Var(&app.MaxLat, "max-lat", 90, "Bounding box maximum latitude")
	flag.Float64Var(&app.MaxLon, "max-lon", 180, "Bounding box maximum longitude")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type App struct {
	Username string
	Password string
	MinLat   float64
	MinLon   float64
	MaxLat   float64
	MaxLon   float64
}

func (a *App) Run(ctx context.Context) error {
	stream, err := firehose.Connect()
	if err != nil {
		return fmt.Errorf("could not establish Firehose connection: %w", err)
	}
	defer stream.Close()

	cmd := firehose.InitCommand{
		Live:     true,
		Username: a.Username,
		Password: a.Password,
		Events:   []firehose.Event{firehose.PositionEvent},
		LatLong: []firehose.Rectangle{{
			LowLat: a.MinLat,
			LowLon: a.MinLon,
			HiLat:  a.MaxLat,
			HiLon:  a.MaxLon,
		}},
	}

	if err := stream.Init(cmd.String()); err != nil {
		return fmt.Errorf("could not initialize firehose: %w", err)
	}

	for {
		msg, err := stream.NextMessage(ctx)
		if errors.Is(err, context.Canceled) {
			return nil
		} else if err != nil {
			return err
		}
		switch m := msg.Payload.(type) {
		case firehose.PositionMessage:
			fmt.Printf("%#v\n", m)
		}
	}
}
