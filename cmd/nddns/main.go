package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	personalAccessToken string
	domain              string
	subdomain           string
	ttl                 string
	pollInterval        time.Duration
)

func init() {
	var pollIntervalString string
	flag.StringVar(&personalAccessToken, "pat", os.Getenv("NDDNS_PAT"), "Netlify personal access token")
	flag.StringVar(&domain, "domain", "", "Netlify controlled domain")
	flag.StringVar(&subdomain, "subdomain", "", "Subdomain to which A record will be added")
	flag.StringVar(&ttl, "ttl", "300", "Time to live for A record")
	flag.StringVar(&pollIntervalString, "poll", "", "How often to poll for changes to IP")

	if personalAccessToken == "" {
		log.Fatalln("Personal access token required. Use -pat flag or set NDDNS_PAT environment variable")
	}

	if domain == "" {
		log.Fatalln("Domain required. Use -domain flag")
	}

	p, err := time.ParseDuration(pollIntervalString)
	if err != nil {
		panic(err)
	}
	pollInterval = p
}

func main() {
	log.Println("Starting ddns service...")
	currentIpv4, err := GetCurrentIpv4()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Current IPv4: %v", currentIpv4)
}

func GetCurrentIpv4() (net.IP, error) {
	res, err := http.Get("https://icanhazip.com")
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	ipString := strings.Trim(string(b), "\t\n ")

	ip := net.ParseIP(ipString)
	if ip == nil {
		return nil, &net.ParseError{
			Type: "IPv4 address",
			Text: ipString,
		}
	}

	return ip, nil
}

func BuildDomain(domain, subdomain string) string {
	if subdomain == "" {
		return domain
	}
	return fmt.Sprintf("%s.%s", subdomain, domain)
}
