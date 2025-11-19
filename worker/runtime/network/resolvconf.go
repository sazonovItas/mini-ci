package network

import (
	"bytes"
	"os"
	"regexp"
	"sync"
)

const (
	defaultResolvConfPath     = "/etc/resolv.conf"
	alternativeResolvConfPath = "/run/systemd/resolve/resolv.conf"
)

const (
	IP = iota
	IPv4
	IPv6
)

var (
	detectSystemdResolvConfOnce         sync.Once
	resolvConfPathAfterSysmtedDetection = defaultResolvConfPath
)

func resolvConfPath() string {
	detectSystemdResolvConfOnce.Do(func() {
		candidateResolvConf, err := os.ReadFile(defaultResolvConfPath)
		if err != nil {
			return
		}

		ns := getNameservers(candidateResolvConf, IP)
		if len(ns) == 1 && ns[0] == "127.0.0.53" {
			resolvConfPathAfterSysmtedDetection = alternativeResolvConfPath
		}
	})

	return resolvConfPathAfterSysmtedDetection
}

const (
	ipv4NumBlock = `(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`
	ipv4Address  = `(` + ipv4NumBlock + `\.){3}` + ipv4NumBlock
	ipv6Address  = `([0-9A-Fa-f]{0,4}:){2,7}([0-9A-Fa-f]{0,4})(%\w+)?`
)

var (
	nsRegexp          = regexp.MustCompile(`^\s*nameserver\s*((` + ipv4Address + `)|(` + ipv6Address + `))\s*$`)
	nsIPv6Regexpmatch = regexp.MustCompile(`^\s*nameserver\s*((` + ipv6Address + `))\s*$`)
	nsIPv4Regexpmatch = regexp.MustCompile(`^\s*nameserver\s*((` + ipv4Address + `))\s*$`)
)

// getLines parses input into lines and strips away comments.
func getLines(input []byte, commentMarker []byte) [][]byte {
	lines := bytes.Split(input, []byte("\n"))
	var output [][]byte
	for _, currentLine := range lines {
		commentIndex := bytes.Index(currentLine, commentMarker)
		if commentIndex == -1 {
			output = append(output, currentLine)
		} else {
			output = append(output, currentLine[:commentIndex])
		}
	}
	return output
}

func getNameservers(resolvConf []byte, kind int) []string {
	nameservers := []string{}
	for _, line := range getLines(resolvConf, []byte("#")) {
		var ns [][]byte
		switch kind {
		case IP:
			ns = nsRegexp.FindSubmatch(line)
		case IPv4:
			ns = nsIPv4Regexpmatch.FindSubmatch(line)
		case IPv6:
			ns = nsIPv6Regexpmatch.FindSubmatch(line)
		}
		if len(ns) > 0 {
			nameservers = append(nameservers, string(ns[1]))
		}
	}
	return nameservers
}

func buildResolvConf(dns []string) ([]byte, error) {
	content := bytes.NewBuffer(nil)
	for _, dns := range dns {
		if _, err := content.WriteString("nameserver " + dns + "\n"); err != nil {
			return nil, err
		}
	}

	return content.Bytes(), nil
}
