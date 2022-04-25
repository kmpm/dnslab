package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

var dnsTimeout = 10 * time.Second
var useEdns0 bool

func dnsQuery(fqdn string, rtype uint16, nameservers []string, recursive bool) (*dns.Msg, error) {
	m := createDNSMsg(fqdn, rtype, recursive)
	var in *dns.Msg
	var err error
	for _, ns := range nameservers {
		in, err = sendDNSQuery(m, ns)
		if err == nil && len(in.Answer) > 0 {
			break
		}
	}
	return in, err
}

func createDNSMsg(fqdn string, rtype uint16, recursive bool) *dns.Msg {
	m := new(dns.Msg)
	m.SetQuestion(fqdn, rtype)
	if useEdns0 {
		m.SetEdns0(4096, false)
	}
	if !recursive {
		m.RecursionDesired = false
	}
	return m
}

func sendDNSQuery(m *dns.Msg, ns string) (*dns.Msg, error) {
	udp := &dns.Client{Net: "udp", Timeout: dnsTimeout}
	in, _, err := udp.Exchange(m, ns)
	// two kinds of errors we can handle by retrying with TCP:
	// truncation and timeout; see https://github.com/caddyserver/caddy/issues/3639
	truncated := in != nil && in.Truncated
	timeoutErr := err != nil && strings.Contains(err.Error(), "timeout")
	if truncated || timeoutErr {
		tcp := &dns.Client{Net: "tcp", Timeout: dnsTimeout}
		in, _, err = tcp.Exchange(m, ns)
	}
	return in, err
}

func usage() {
	flag.Usage()

}

func main() {
	var server string
	var recursive bool
	flag.BoolVar(&useEdns0, "edns0", false, "enable the use of edns0")
	flag.StringVar(&server, "server", "ns1.namesystem.se", "default nameserver to use")
	flag.BoolVar(&recursive, "recursive", true, "use recursive lookup")
	flag.Parse()

	_, _, err := net.SplitHostPort(server)
	if err != nil {
		server += ":53"
	}
	if flag.Arg(0) == "" {
		fmt.Printf("missing fqdn")
		usage()
		os.Exit(1)
	}
	fqdn := flag.Arg(0)
	if !strings.HasSuffix(fqdn, ".") {
		fqdn += "."
	}

	fmt.Printf("Server: %s\n", server)
	fmt.Printf("Recursive: %t\n", recursive)
	fmt.Printf("edns0: %t\n", useEdns0)
	fmt.Println("----- checking -----")

	m, err := dnsQuery(fqdn, dns.TypeTXT, []string{server}, recursive)
	if err != nil {
		fmt.Printf("error quering dns: %v", err)
		os.Exit(1)
	}
	fmt.Println("----- answers -----")
	for i, a := range m.Answer {
		fmt.Printf("%d: %+v\n", i, a)
	}
	fmt.Println("----- full response -----")
	fmt.Printf("%+v\n", m)
}
