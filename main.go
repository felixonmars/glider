package main

import (
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nadoo/glider/common/log"
	"github.com/nadoo/glider/dns"
	"github.com/nadoo/glider/ipset"
	"github.com/nadoo/glider/proxy"
	"github.com/nadoo/glider/rule"
	"github.com/nadoo/glider/strategy"

	_ "github.com/nadoo/glider/proxy/http"
	_ "github.com/nadoo/glider/proxy/kcp"
	_ "github.com/nadoo/glider/proxy/mixed"
	_ "github.com/nadoo/glider/proxy/obfs"
	_ "github.com/nadoo/glider/proxy/reject"
	_ "github.com/nadoo/glider/proxy/socks5"
	_ "github.com/nadoo/glider/proxy/ss"
	_ "github.com/nadoo/glider/proxy/ssr"
	_ "github.com/nadoo/glider/proxy/tcptun"
	_ "github.com/nadoo/glider/proxy/tls"
	_ "github.com/nadoo/glider/proxy/udptun"
	_ "github.com/nadoo/glider/proxy/uottun"
	_ "github.com/nadoo/glider/proxy/vmess"
	_ "github.com/nadoo/glider/proxy/ws"
)

var version = "0.8.2"

func main() {
	// read configs
	confInit()

	// setup a log func
	log.F = func(f string, v ...interface{}) {
		if conf.Verbose {
			stdlog.Printf(f, v...)
		}
	}

	// global rule proxy
	p := rule.NewProxy(conf.rules, strategy.NewProxy(conf.Forward, &conf.StrategyConfig))

	// ipset manager
	ipsetM, _ := ipset.NewManager(conf.rules)

	// check and setup dns server
	if conf.DNS != "" {
		d, err := dns.NewServer(conf.DNS, p, &conf.DNSConfig)
		if err != nil {
			log.Fatal(err)
		}

		// rule
		for _, r := range conf.rules {
			for _, domain := range r.Domain {
				if len(r.DNSServers) > 0 {
					d.SetServers(domain, r.DNSServers...)
				}
			}
		}

		// add a handler to update proxy rules when a domain resolved
		d.AddHandler(p.AddDomainIP)
		if ipsetM != nil {
			d.AddHandler(ipsetM.AddDomainIP)
		}

		d.Start()
	}

	// enable checkers
	p.Check()

	// Proxy Servers
	for _, listen := range conf.Listen {
		local, err := proxy.ServerFromURL(listen, p)
		if err != nil {
			log.Fatal(err)
		}

		go local.ListenAndServe()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
