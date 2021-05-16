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

func main() {
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "Usage of %s: [flag] config\n", os.Args[0])
		fmt.Fprintln(out, "  -h: show this help")
		flag.PrintDefaults()
	}
	flag.Parse()

	logger := log.New(os.Stdout, progName+": ", log.LstdFlags)
	if err := run(context.Background(), logger); err != nil {
		logger.Fatalln(err)
		os.Exit(2)
	}
}

func run(ctx context.Context, logger *log.Logger) error {
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

	for {
		updaters.Update(ctx, logger)
		time.Sleep(sleepTime)
	}
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
