package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/codegangsta/cli"
	"github.com/coreos/go-semver/semver"
	"github.com/fatih/color"
	"github.com/octoblu/register-vulcan/healthchecker"
	"github.com/octoblu/register-vulcan/vctl"
	De "github.com/tj/go-debug"
)

var debug = De.Debug("register-vulcan:main")

func main() {
	app := cli.NewApp()
	app.Name = "register-vulcan"
	app.Version = version()
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "backend-id, b",
			EnvVar: "REGISTER_VULCAN_BACKEND_ID",
			Usage:  "Backend id to register server to",
		},
		cli.StringFlag{
			Name:   "server-id, s",
			EnvVar: "REGISTER_VULCAN_SERVER_ID",
			Usage:  "Server ID",
		},
		cli.DurationFlag{
			Name:   "ttl, t",
			EnvVar: "REGISTER_VULCAN_TTL",
			Usage:  "ttl for server keys (in case of unexpected register-vulcan death)",
			Value:  11 * time.Second,
		},
		cli.StringFlag{
			Name:   "uri, u",
			EnvVar: "REGISTER_VULCAN_URI",
			Usage:  "URI to healthcheck, must return status 200",
		},
		cli.StringFlag{
			Name:   "vulcan-uri, v",
			EnvVar: "REGISTER_VULCAN_VULCAN_URI",
			Usage:  "VULCAN URI to register server to",
		},
	}
	app.Run(os.Args)
}

func run(context *cli.Context) {
	vulcanURI, serverID, backendID, uri, ttl := getOpts(context)

	control := make(chan bool)
	go loop(vulcanURI, serverID, backendID, uri, ttl, control)
	go iterate(control)

	sigTerm := make(chan os.Signal)
	signal.Notify(sigTerm, syscall.SIGTERM)

	sigHup := make(chan os.Signal)
	signal.Notify(sigHup, syscall.SIGHUP)

	go func() {
		for {
			<-sigHup
			fmt.Println("SIGHUP received, deregistering")
			control <- false
			onNotHealthy(vulcanURI, serverID, backendID)
			fmt.Println("deregistered, paused for 5 seconds")
			time.Sleep(5 * time.Second)
			go loop(vulcanURI, serverID, backendID, uri, ttl, control)
		}
	}()

	<-sigTerm
	fmt.Println("SIGTERM received, cleaning up")
	control <- false
	onNotHealthy(vulcanURI, serverID, backendID)
	os.Exit(0)
}

func loop(vulcanURI, serverID, backendID, uri string, ttl time.Duration, control <-chan bool) {
	for {
		if !<-control {
			return
		}
		healthcheck(vulcanURI, serverID, backendID, uri, ttl)
	}
}

func iterate(control chan<- bool) {
	for {
		control <- true
		time.Sleep(5 * time.Second)
	}
}

func healthcheck(vulcanURI, serverID, backendID, uri string, ttl time.Duration) {
	if healthchecker.Healthy(fmt.Sprintf("%v/healthcheck", uri)) {
		onHealthy(vulcanURI, serverID, backendID, uri, ttl)
	} else {
		onNotHealthy(vulcanURI, serverID, backendID)
	}
}

func onHealthy(vulcanURI, serverID, backendID, uri string, ttl time.Duration) {
	debug("onHealthy")

	vctlClient, err := vctl.New(vulcanURI)
	FatalIfError("vctl.New vulcanURI", err)

	err = vctlClient.ServerUpsert(serverID, backendID, uri, ttl)
	FatalIfError("vctlClient.ServerUpsert", err)
}

func onNotHealthy(vulcanURI, serverID, backendID string) {
	debug("onNotHealthy")

	vctlClient, err := vctl.New(vulcanURI)
	FatalIfError("vctl.New vulcanURI", err)

	err = vctlClient.ServerRm(serverID, backendID)
	FatalIfError("vctlClient.ServerRm", err)
}

func getOpts(context *cli.Context) (string, string, string, string, time.Duration) {
	backendID := context.String("backend-id")
	serverID := context.String("server-id")
	ttl := context.Duration("ttl")
	vulcanURI := context.String("vulcan-uri")
	uri := context.String("uri")

	if backendID == "" || serverID == "" || vulcanURI == "" || uri == "" {
		cli.ShowAppHelp(context)

		if backendID == "" {
			color.Red("  Missing required flag --backend-id or REGISTER_VULCAN_BACKEND_ID")
		}
		if serverID == "" {
			color.Red("  Missing required flag --server-id or REGISTER_VULCAN_SERVER_ID")
		}
		if uri == "" {
			color.Red("  Missing required flag --uri or REGISTER_VULCAN_URI")
		}
		if vulcanURI == "" {
			color.Red("  Missing required flag --vulcan-uri or REGISTER_VULCAN_VULCAN_URI")
		}
		os.Exit(1)
	}

	validateURI(uri)
	validateURI(vulcanURI)

	return vulcanURI, serverID, backendID, uri, ttl
}

func validateURI(uriStr string) {
	uri, err := url.Parse(uriStr)
	FatalIfError(fmt.Sprintf("Failed to parse uri: %v", uriStr), err)

	if uri.Scheme != "http" && uri.Scheme != "https" {
		log.Fatalf("uri protocol must be one of http/https: %v\n", uriStr)
	}

	parts := strings.Split(uri.Host, ":")

	if len(parts) != 2 || parts[1] == "" {
		log.Fatalf("uri must contain a port: %v\n", uriStr)
	}
}

func version() string {
	version, err := semver.NewVersion(VERSION)
	if err != nil {
		errorMessage := fmt.Sprintf("Error with version number: %v", VERSION)
		log.Panicln(errorMessage, err.Error())
	}
	return version.String()
}

// FatalIfError prints error and dies if error is non nil
func FatalIfError(msg string, err error) {
	if err == nil {
		return
	}

	log.Fatalf("ERROR(%v):\n\n%v", msg, err)
}
