package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/jamesbcook/peepingJim"
)

//flagOpts hold all the possible options a user could pass at the cli
type flagOpts struct {
	url     string
	dir     string
	xml     string
	list    string
	output  string
	threads int
	timeout int
	verbose bool
}

//flags is a function that builds the flagOpts struct
func flags() *flagOpts {
	xmlOpt := flag.String("xml", "", "xml file to parse")
	listOpt := flag.String("list", "", "file that contains a list of URLs")
	dirOpt := flag.String("dir", "", "dir of xml files")
	urlOpt := flag.String("url", "", "single URL to scan")
	threadOpt := flag.Int("threads", 1, "Number of Threads to use")
	outputOpt := flag.String("output", "", "where to write folder")
	timeoutOpt := flag.Int("timeout", 8, "time out in seconds")
	verboseOpt := flag.Bool("verbose", false, "Verbose")
	versionOpt := flag.Bool("version", false, "Print version")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "peepingJim %v by %s \nUsage:\n", peepingJim.Version, peepingJim.Author)
		flag.PrintDefaults()
	}
	flag.Parse()
	if *versionOpt {
		fmt.Println(peepingJim.Version)
		os.Exit(0)
	}
	return &flagOpts{url: *urlOpt, dir: *dirOpt, xml: *xmlOpt, list: *listOpt,
		output: *outputOpt, threads: *threadOpt, timeout: *timeoutOpt,
		verbose: *verboseOpt}
}

func main() {
	//Gather all the cli arguments
	options := flags()
	var dstPath string
	//Creating Directory to store all output from phantom and curl
	if options.output != "" {
		if _, err := os.Stat(options.output); err == nil {
			log.Fatal(options.output + " already exists")
		} else {
			dstPath = options.output
		}
	} else {
		dstPath = "peepingJim_" + time.Now().Format("2006_01_02_15_04_05")
	}
	var targets []string
	if options.xml != "" {
		targets = peepingJim.GetTargets(peepingJim.InputType(peepingJim.XML), options.xml)
	} else if options.list != "" {
		targets = peepingJim.GetTargets(peepingJim.InputType(peepingJim.List), options.list)
	} else if options.dir != "" {
		targets = peepingJim.GetTargets(peepingJim.InputType(peepingJim.Dir), options.dir)
	} else if options.url != "" {
		targets = peepingJim.GetTargets(peepingJim.InputType(peepingJim.Plane), options.url)
	} else {
		log.Fatal("Need an input source")
	}
	app := peepingJim.App{}
	client := peepingJim.Client{}
	app.Threads = options.threads
	client.Output = dstPath
	client.TimeOut = options.timeout
	client.Verbose = options.verbose
	client.Chrome.Path = peepingJim.LocateChrome()
	os.Mkdir(dstPath, 0755)
	//Making a list of targets to scan
	db := []map[string]string{}
	//Report name
	report := "peepingJim.html"
	outFile := fmt.Sprintf("%s/%s", dstPath, report)
	fmt.Printf("Loading %d targets\n", len(targets))
	// capture ctrl+c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Printf("captured %v, stopping scanner and exiting...", sig)
			peepingJim.BuildReport(db, outFile)
			os.Exit(1)
		}
	}()
	queue := make(chan string)
	//spawn workers
	for i := 0; i <= app.Threads; i++ {
		go client.Worker(queue, &db)
	}
	//make work
	for _, target := range targets {
		queue <- target
	}
	//fill queue with finished work
	for n := 0; n <= app.Threads; n++ {
		queue <- ""
	}
	//Building the final html file
	peepingJim.BuildReport(db, outFile)
}
