package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

var dnsTimeout = 10 * time.Second
var useEdns0 uint

func dnsQuery(fqdn string, rtype uint16, nameservers []string, recursive bool) (*dns.Msg, error) {
	m := createDNSMsg(fqdn, rtype, recursive)
	var in *dns.Msg
	var err error
	for _, ns := range nameservers {
		in, err = sendDNSQuery(m, ns)
		if err == nil && len(in.Answer) > 0 {
			break
		}
		if recursive && !in.RecursionAvailable {
			log.Println("recursion unavailable")
		}
		if err == nil && recursive && !in.RecursionAvailable {
			fmt.Printf("retry server %s without recursion: %s\n", ns, fqdn)
			in, err = dnsQuery(fqdn, rtype, []string{ns}, false)
			if err == nil && len(in.Answer) > 0 {
				break
			}
		}
	}
	return in, err
}

func createDNSMsg(fqdn string, rtype uint16, recursive bool) *dns.Msg {
	m := new(dns.Msg)
	m.SetQuestion(fqdn, rtype)
	if useEdns0 > 0 {
		m.SetEdns0(uint16(useEdns0), false)
	}
	fmt.Printf("msg edns: %d, recursive: %t, rtype: %d\n", useEdns0, recursive, rtype)
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
	var recursive bool = true
	var noRecurse bool
	flag.UintVar(&useEdns0, "edns0", 1232, "enable the use of edns0")
	flag.StringVar(&server, "server", "ns1.namesystem.se", "default nameserver to use")
	flag.BoolVar(&noRecurse, "no-recurse", false, "use recursive lookup")
	flag.Parse()
	recursive = !noRecurse
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
	fmt.Printf("edns0: %d\n", useEdns0)
	log.Println("----- quering -----")

	m, err := dnsQuery(fqdn, dns.TypeTXT, []string{server}, recursive)
	if err != nil {
		log.Printf("error quering dns: %v", err)
		os.Exit(1)
	}
	// fmt.Println("----- answers -----")
	// for i, a := range m.Answer {
	// 	fmt.Printf("%d: %+v\n", i, a)
	// }
	log.Println("----- response -----")
	fmt.Printf("%+v\n", m)
}
