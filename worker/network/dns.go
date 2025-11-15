package network

import (
	"fmt"

	"github.com/miekg/dns"
)

const resolvConfPath = "/etc/resolv.conf"

func DNSServer() (*dns.Server, error) {
	resolvconf
	resolveConf, err := dns.ClientConfigFromFile(resolvConfPath)
	if err != nil {
		return nil, fmt.Errorf("parsing resolv config: %w", err)
	}

	client := &dns.Client{}

	mux := dns.NewServeMux()
	mux.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		var (
			err error
			msg *dns.Msg
		)

		for _, server := range resolveConf.Servers {
			address := fmt.Sprintf("%s:%s", server, resolveConf.Port)

			msg, _, err = client.Exchange(r, address)
			if err == nil {
				msg.Compress = true
				_ = w.WriteMsg(msg)
				break
			}
		}

		if err != nil {
			msg = new(dns.Msg)
			msg.SetRcode(r, dns.RcodeRefused)
			_ = w.WriteMsg(msg)
		}
	})

	server := &dns.Server{
		Addr:    ":53",
		Net:     "udp",
		Handler: mux,
	}

	return server, nil
}
