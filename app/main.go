package main

import (
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	flr "github.com/LukeEuler/funnel-log-reporter"
	"github.com/LukeEuler/funnel-log-reporter/config"
	"github.com/LukeEuler/funnel-log-reporter/log"
)

func main() {
	log.AddConsoleOut(5)

	configFile := flag.String("c", "config.toml", "set the config file path")
	flag.Parse()

	config.New(*configFile)

	p, err := flr.NewProcessor()
	if err != nil {
		log.Entry.WithError(err).Fatal(err)
	}

	doLoopJobs(p.Loop)
}

func doLoopJobs(jobs ...func(chan struct{})) {
	shutdown := make(chan struct{})
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range signals {
			if sig == os.Interrupt || sig == syscall.SIGTERM {
				log.Entry.Infof("received signal [%v], preparing to quit", sig)
				close(shutdown)
			}
		}
	}()

	var wg sync.WaitGroup

	for index := range jobs {
		wg.Add(1)
		job := jobs[index]
		go func() {
			defer wg.Done()
			job(shutdown)
		}()
	}

	wg.Wait()
}
