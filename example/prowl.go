package main

import (
	"flag"
	"fmt"
	prowl "github.com/cj123/goprowl"
	"os"
	"strings"
)

var (
	apikey, application, event, url, providerKey string
	priority                                     int
)

func init() {
	flag.StringVar(&providerKey, "providerkey", "", "Your provider key")
	flag.StringVar(&apikey, "apikey", "", "Your API key")
	flag.StringVar(&application, "app", "goprowl", "Your application name")
	flag.StringVar(&event, "event", "", "Prowl event")
	flag.IntVar(&priority, "pri", 0, "Prowl priority (-2 to 2)")
	flag.StringVar(&apikey, "url", "", "URL to send")
}

func main() {
	flag.Parse()

	client := prowl.NewProwlClient(providerKey)

	n := prowl.Notification{
		Application: application,
		Description: strings.Join(flag.Args(), " "),
		Event:       event,
		Priority:    priority,
		URL:         url,
	}

	if err := n.AddKey(apikey); err != nil {
		fmt.Fprintf(os.Stderr, "Error registering key:  %v\n", err)
		os.Exit(1)
	}

	if err := client.Push(n); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending message:  %v\n", err)
		os.Exit(1)
	}
}
