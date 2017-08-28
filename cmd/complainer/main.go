package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"bytes"
	"github.com/cloudflare/complainer/flags"
	"github.com/cloudflare/complainer/matcher"
	"github.com/cloudflare/complainer/mesos"
	"github.com/cloudflare/complainer/monitor"
	"github.com/cloudflare/complainer/reporter"
	"github.com/cloudflare/complainer/uploader"
	log "github.com/sirupsen/logrus"
)

type regexArrayFlags []*regexp.Regexp

func (a *regexArrayFlags) String() string {
	var l []string
	for _, r := range *a {
		l = append(l, r.String())
	}
	return strings.Join(l, ", ")
}

func (a *regexArrayFlags) Set(value string) error {
	r, err := regexp.Compile(value)
	if r != nil {
		*a = append(*a, r)
	}
	return err
}

func main() {

	// Register & parse all arguments

	name := flags.String("name", "COMPLAINER_NAME", monitor.DefaultName, "complainer name to use (default is implicit)")
	d := flags.Bool("default", "COMPLAINER_DEFAULT", true, "whether to use implicit default reporters")
	u := flags.String("uploader", "COMPLAINER_UPLOADER", "", "uploader to use (example: s3aws,s3goamz,noop)")
	r := flags.String("reporters", "COMPLAINER_REPORTERS", "", "reporters to use (example: sentry,hipchat,slack,file)")
	masters := flags.String("masters", "COMPLAINER_MASTERS", "", "list of master urls: http://host:port,http://host:port")
	listen := flags.String("listen", "COMPLAINER_LISTEN", "", "http listen address")
	help := flags.Bool("help", "", false, "Show usage instruction")
	logFileName := flags.String("logfile", "COMPLAINER_LOGFILE", "/dev/stdout", "name of file to write logs to")
	logLevel := flags.String("loglevel", "COMPLAINER_LOGLEVEL", "Info", "log level; one of Debug, Info, Warn, Error, Fatal, Panic")
	logAllTasks := flags.Bool("log-all-tasks", "COMPLAINER_LOG_ALL_TASKS", false, "log all tasks at Debug level - extremely verbose")
	runOnce := flags.Bool("run-once", "COMPLAINER_RUN_ONCE", false, "Run checks only once and exit.")

	var whitelist regexArrayFlags
	var blacklist regexArrayFlags
	flag.Var(&whitelist, "framework-whitelist", "list of regexes that if a framework name matches, will be reported")
	flag.Var(&blacklist, "framework-blacklist", "list of regexes that if a framework name matches, is ignored")

	uploader.RegisterFlags()
	reporter.RegisterFlags()

	flag.Parse()

	// Help

	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Process logging arguments, initialize logrus

	log.SetFormatter(&simpleLogrusFormatter{})

	switch *logFileName {
	case "/dev/stdout", "-":
		log.SetOutput(os.Stdout)
	case "/dev/stderr":
		log.SetOutput(os.Stderr)
	default:
		logFile, err := os.OpenFile(*logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	logLevelEnum, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(logLevelEnum)

	log.Info("Complainer starting")
	defer log.Info("Complainer stopping") // TODO: catch ^C

	// Process rest of the arguments

	if *u == "" || *r == "" || *masters == "" {
		log.Fatalf("Uploader, reporters and masters are mandatory parameters. Use -help for help.")
		os.Exit(1)
	}

	um, err := uploader.MakerByName(*u)
	if err != nil {
		log.Fatalf("Cannot create uploader by name %q: %s", *u, err)
	}

	up, err := um.Make()
	if err != nil {
		log.Fatalf("Cannot create uploader by name %q: %s", *u, err)
	}

	reporters, err := makeReporters(*r)
	if err != nil {
		log.Fatalf("Cannot create requested reporters: %s", err)
	}

	matcher := matcher.RegexMatcher{Whitelist: whitelist, Blacklist: blacklist}
	cluster := mesos.NewCluster(strings.Split(*masters, ","), *logAllTasks)

	m := monitor.NewMonitor(*name, cluster, up, reporters, *d, &matcher)

	serve(m, *listen)

	log.Info("Startup complete")

	if *runOnce {
		err := m.Run()
		if err != nil {
			log.Fatalf("Error running monitor: %s", err)
		}
		os.Exit(0)
	}

	log.Debug("Entering poll loop")
	for {
		err := m.Run()
		if err != nil {
			log.Errorf("Error running monitor: %s", err)
		}

		time.Sleep(time.Second * 5)
	}
}

func serve(m *monitor.Monitor, listen string) {
	if listen != "" || os.Getenv("PORT") != "" {
		if listen == "" {
			listen = fmt.Sprintf(":%s", os.Getenv("PORT"))
		}

		go func() {
			log.Infof("Serving http on %s", listen)
			if err := m.ListenAndServe(listen); err != nil {
				log.Fatalf("Error serving: %s", err)
			}
		}()
	}
}

func makeReporters(requested string) (map[string]reporter.Reporter, error) {
	reporters := map[string]reporter.Reporter{}

	for _, n := range strings.Split(requested, ",") {
		log.Debugf("Creating reporter: %s", n)

		maker, err := reporter.MakerByName(n)
		if err != nil {
			return nil, fmt.Errorf("cannot create reporter maker by name %q: %s", n, err)
		}

		r, err := maker.Make()
		if err != nil {
			return nil, fmt.Errorf("cannot create reporter by name %q: %s", n, err)
		}

		reporters[n] = r
	}

	return reporters, nil
}

// simpleLogrusFormatter implements a simple plaintext log formatter for logrus
type simpleLogrusFormatter struct {
}

func (f *simpleLogrusFormatter) Format(entry *log.Entry) ([]byte, error) {
	var buffer *bytes.Buffer

	if entry.Buffer != nil {
		buffer = entry.Buffer
	} else {
		buffer = &bytes.Buffer{}
	}

	// Time, level
	buffer.WriteString(fmt.Sprintf("[%s] [%s]: ", entry.Time.Format(time.RFC3339), entry.Level.String()))

	// Simple scoping established with the "module" and "func" fields
	if _, moduleFound := entry.Data["module"]; moduleFound {
		buffer.WriteString(fmt.Sprintf("%s: ", entry.Data["module"]))
	}
	if _, funcFound := entry.Data["func"]; funcFound {
		buffer.WriteString(fmt.Sprintf("%s(): ", entry.Data["func"]))
	}

	// Payload
	buffer.WriteString(entry.Message)

	// Possible extra fields
	for key := range entry.Data {
		if key != "module" && key != "func" {
			buffer.WriteString(fmt.Sprintf(" [%s: %s]", key, entry.Data[key]))
		}
	}

	buffer.WriteByte('\n')
	return buffer.Bytes(), nil
}
