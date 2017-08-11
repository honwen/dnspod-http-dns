package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chenhw2/dnspod-http-dns/dnspod"
	"github.com/golang/glog"
	"github.com/miekg/dns"
	"github.com/urfave/cli"
)

var (
	version = "MISSING build version [git hash]"

	listenAddress   string
	listenProtocols []string

	dnsProvider *dnspod.DNSPOD
)

func serve(net, addr string) {
	log.Printf("starting %s service on %s", net, addr)

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	server := &dns.Server{Addr: addr, Net: net, TsigSecret: nil}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			glog.Errorf("Failed to setup the %s server: %s\n", net, err.Error())
			sig <- syscall.SIGTERM
		}
	}()

	// serve until exit
	<-sig

	log.Printf("shutting down %s on interrupt\n", net)
	if err := server.Shutdown(); err != nil {
		log.Printf("got unexpected error %s", err.Error())
	}
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	app := cli.NewApp()
	app.Name = "dnspod-http-dns"
	app.Usage = "A DNS-protocol proxy for DNSPOD's DNS-over-HTTP service."
	app.Version = version
	// app.HideVersion = true
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "listen, l",
			Value: ":5300",
			Usage: "Serve address",
		},
		cli.StringFlag{
			Name:  "edns, e",
			Usage: "Extension mechanisms for DNS (EDNS) is parameters of the Domain Name System (DNS) protocol",
		},
		cli.BoolFlag{
			Name:  "udp, U",
			Usage: "Listen on UDP",
		},
		cli.BoolFlag{
			Name:  "tcp, T",
			Usage: "Listen on TCP",
		},
	}
	app.Action = func(c *cli.Context) error {
		listenAddress = c.String("listen")
		if c.Bool("tcp") {
			listenProtocols = append(listenProtocols, "tcp")
		}
		if c.Bool("udp") {
			listenProtocols = append(listenProtocols, "udp")
		}
		if 0 == len(listenProtocols) {
			cli.ShowAppHelp(c)
			os.Exit(0)
		}
		dnsProvider = dnspod.NewDNSPOD(c.String("edns"))
		return nil
	}
	app.Run(os.Args)

	dns.HandleFunc(".", dnsProvider.DNSHandleFunc)
	servers := make(chan bool)
	for _, protocol := range listenProtocols {
		go func(protocol string) {
			serve(protocol, listenAddress)
			servers <- true
		}(protocol)
	}

	// wait for servers to exit
	for range listenProtocols {
		<-servers
	}

}
