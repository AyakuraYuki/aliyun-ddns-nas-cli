package main

import (
	"github.com/miekg/dns"
	"github.com/mr-karan/doggo/pkg/resolvers"
	"github.com/mr-karan/doggo/pkg/utils"
	"strings"
	"time"
)

var mResolvers = make([]resolvers.Resolver, 0)

func init() {
	var upstreams = []string{
		"tls://223.5.5.5:853",
		"tls://223.6.6.6:853",
		"https://223.5.5.5/dns-query",
		"https://223.6.6.6/dns-query",
		"tls://1.12.12.12:853",
		"https://120.53.53.53/dns-query",
		// "tls://1.12.12.12:853",
		// "tls://120.53.53.53:853",
		// "https://1.12.12.12/dns-query",
		// "https://120.53.53.53/dns-query",
	}
	var opts = resolvers.Options{
		Timeout: 2000 * time.Millisecond,
		Logger:  utils.InitLogger(),
	}
	var dotOpts = resolvers.ClassicResolverOpts{
		UseTLS: true,
		UseTCP: true,
	}

	for _, upstream := range upstreams {
		var ns resolvers.Resolver
		switch {
		case strings.HasPrefix(upstream, "https://"):
			ns, _ = resolvers.NewDOHResolver(upstream, opts)
		case strings.HasPrefix(upstream, "tls://"):
			ns, _ = resolvers.NewClassicResolver(upstream[6:], dotOpts, opts)
		default:
			continue
		}
		mResolvers = append(mResolvers, ns)
	}
}

func ResolveIPv4(domain string) (ip string) {
	return resolve(domain, dns.TypeA)
}

func ResolveIPv6(domain string) (ip string) {
	return resolve(domain, dns.TypeAAAA)
}

func resolve(domain string, qtype uint16) (ip string) {
	var (
		dnsMap   = make(map[string]int, len(mResolvers))
		channel  = make(chan string, len(mResolvers))
		maxCount = -1
	)

	for i := range mResolvers {
		go func(resolver *resolvers.Resolver) {
			addr := ""
			resp, err := (*resolver).Lookup(dns.Question{Name: domain, Qtype: qtype, Qclass: dns.ClassINET})
			if err == nil && len(resp.Answers) > 0 {
				for _, it := range resp.Answers {
					if it.Type == dns.TypeToString[qtype] {
						addr = it.Address
						break
					}
				}
			}
			channel <- addr
		}(&mResolvers[i])
	}

	for i := 0; i < len(mResolvers); i++ {
		v := <-channel
		if len(v) == 0 {
			continue
		}
		if dnsMap[v] >= len(mResolvers)/2 {
			return v
		}
		dnsMap[v]++
	}

	for k, v := range dnsMap {
		if v > maxCount {
			maxCount = v
			ip = k
		}
	}
	return
}
