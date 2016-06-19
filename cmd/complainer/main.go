package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/complainer/mesos"
	"github.com/cloudflare/complainer/monitor"
	"github.com/cloudflare/complainer/reporter"
	"github.com/cloudflare/complainer/uploader"
)

func main() {
	name := flag.String("name", monitor.DefaultName, "complainer name to use (default is implicit)")
	u := flag.String("uploader", "", "uploader to use (example: s3aws,s3goamz,noop)")
	r := flag.String("reporters", "", "reporters to use (example: sentry,hipchat,slack,file)")
	masters := flag.String("masters", "", "list of master urls: http://host:port,http://host:port")

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

	masterList := cleanupURLList(strings.Split(*masters, ","))
	if len(masterList) == 0 {
		log.Fatal("After URL cleanup, there is no Mesos master left over. Please check -masters argument")
	}
	cluster := mesos.NewCluster(masterList)

	m := monitor.NewMonitor(*name, cluster, up, reporters)

	for {
		err := m.Run()
		if err != nil {
			log.Fatal(err)
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

// cleanupURLList will clean up a list of URLs.
// It does two things
// 	1. Check if the url is parsable
//	2. Ensures that the url has no / at the end
// A clean list of urls and the last error (if there is one)
// of the url.Parse action will be returned .
func cleanupURLList(urls []string) ([]string) {
	var clean []string
	var err error
	var u *url.URL

	for _, singleURL := range urls {
		trimmedURL := strings.TrimSpace(singleURL)
		// Skip empty URLs
		if len(trimmedURL) == 0 {
			continue
		}

		u, err = url.Parse(trimmedURL)
		// If we have an error during url parsing
		// we just skip this url, because this
		// url won`t be callable anyway
		if err != nil {
			continue
		}

		s := strings.TrimSuffix(u.String(), "/")
		clean = append(clean, s)
	}

	return clean
}
