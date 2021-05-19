package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/YoRyan/dsddns/updater"

	"gopkg.in/yaml.v3"
)

const (
	progName  = "dsddns"
	sleepTime = 5 * time.Minute
)

type mode int

const (
	runRepeating mode = iota
	dryRun
)

func main() {
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "Usage of %s: [flag] config\n", os.Args[0])
		flag.PrintDefaults()
	}
	var opDryRun bool
	flag.BoolVar(&opDryRun, "dryrun", false, "read the configuration file, but do not push any updates")
	flag.Parse()

	var op mode
	if opDryRun {
		op = dryRun
	} else {
		op = runRepeating
	}
	logger := log.New(os.Stdout, progName+": ", log.LstdFlags)
	if err := run(context.Background(), logger, op); err != nil {
		logger.Fatalln(err)
		os.Exit(2)
	}
}

func run(ctx context.Context, logger *log.Logger, op mode) error {
	path := flag.Arg(0)
	if path == "" {
		return errors.New("missing path to a configuration file")
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	updaters, err := loadConfig(file)
	if err != nil {
		return err
	}

	if op == dryRun {
		updaters.DryRun(ctx, logger)
	} else if op == runRepeating {
		for {
			updaters.Update(ctx, logger)
			time.Sleep(sleepTime)
		}
	}
	return nil
}

func loadConfig(r io.Reader) (updater.Updaters, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var config struct {
		Records updater.Updaters
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return config.Records, nil
}
