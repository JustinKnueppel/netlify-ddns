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
	hostname            string
	ttl                 string
	pollInterval        time.Duration
)

func init() {
	var (
		domain             string
		subdomain          string
		pollIntervalString string
	)
	flag.StringVar(&personalAccessToken, "pat", os.Getenv("NDDNS_PAT"), "Netlify personal access token")
	flag.StringVar(&domain, "domain", "", "Netlify controlled domain")
	flag.StringVar(&subdomain, "subdomain", "", "Subdomain to which A record will be added")
	flag.StringVar(&ttl, "ttl", "300", "Time to live for A record")
	flag.StringVar(&pollIntervalString, "poll", "30m", "How often to poll for changes to IP")

	flag.Parse()

	if personalAccessToken == "" {
		log.Fatalln("Personal access token required. Use -pat flag or set NDDNS_PAT environment variable")
	}

	if domain == "" {
		log.Fatalln("Domain required. Use -domain flag")
	}
	hostname = BuildHostname(domain, subdomain)

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

func BodyToString(res *http.Response) (string, error) {
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func GetCurrentIpv4() (net.IP, error) {
	res, err := http.Get("https://icanhazip.com")
	if err != nil {
		return nil, err
	}

	resString, err := BodyToString(res)
	if err != nil {
		return nil, err
	}

	ipString := strings.Trim(resString, "\t\n ")

	ip := net.ParseIP(ipString)
	if ip == nil {
		return nil, &net.ParseError{
			Type: "IPv4 address",
			Text: ipString,
		}
	}

	return ip, nil
}

func BuildHostname(domain, subdomain string) string {
	if subdomain == "" {
		return domain
	}
	return fmt.Sprintf("%s.%s", subdomain, domain)
}
