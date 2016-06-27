package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/complainer/flags"
	"github.com/cloudflare/complainer/mesos"
	"github.com/cloudflare/complainer/monitor"
	"github.com/cloudflare/complainer/reporter"
	"github.com/cloudflare/complainer/uploader"
)

func main() {
	name := flags.String("name", "COMPLAINER_NAME", monitor.DefaultName, "complainer name to use (default is implicit)")
	u := flags.String("uploader", "COMPLAINER_UPLOADER", "", "uploader to use (example: s3aws,s3goamz,noop)")
	r := flags.String("reporters", "COMPLAINER_REPORTERS", "", "reporters to use (example: sentry,hipchat,slack,file)")
	masters := flags.String("masters", "COMPLAINER_MASTERS", "", "list of master urls: http://host:port,http://host:port")
	listen := flags.String("listen", "COMPLAINER_LISTEN", "", "http listen address")

	uploader.RegisterFlags()
	reporter.RegisterFlags()

	flag.Parse()

	if *u == "" || *r == "" || *masters == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	um, err := uploader.MakerByName(*u)
	if err != nil {
		log.Fatalf("Cannot create uploader by name %q: %s", *u, err)
	}

	up, err := um.Make()
	if err != nil {
		flag.PrintDefaults()
		log.Fatalf("Cannot create uploader by name %q: %s", *u, err)
	}

	reporters, err := makeReporters(*r)
	if err != nil {
		log.Fatalf("Cannot create requested reporters: %s", err)
	}

	cluster := mesos.NewCluster(strings.Split(*masters, ","))

	m := monitor.NewMonitor(*name, cluster, up, reporters)

	if *listen != "" {
		go func() {
			log.Printf("Serving http on %s", *listen)
			if err := m.ListenAndServe(*listen); err != nil {
				log.Fatalf("Error serving: %s", err)
			}
		}()
	}

	for {
		err := m.Run()
		if err != nil {
			log.Printf("Error running monitor: %s", err)
		}

		time.Sleep(time.Second * 5)
	}
}

func makeReporters(requested string) (map[string]reporter.Reporter, error) {
	reporters := map[string]reporter.Reporter{}

	for _, n := range strings.Split(requested, ",") {
		maker, err := reporter.MakerByName(n)
		if err != nil {
			return nil, fmt.Errorf("cannot create reporter by name %q: %s", n, err)
		}

		r, err := maker.Make()
		if err != nil {
			return nil, fmt.Errorf("cannot create reporter by name %q: %s", n, err)
		}

		reporters[n] = r
	}

	return reporters, nil
}
