package main

import (
	"encoding/json"
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

const (
	apiEndpoint = "https://api.netlify.com/api/v1"
)

var (
	personalAccessToken string
	domain              string
	subdomain           string
	hostname            string
	ttl                 string
	pollInterval        time.Duration
)

func init() {
	var pollIntervalString string

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

	recordIpv4, err := GetCurrentRecordIpv4()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Current IPv4 record: %v", recordIpv4)
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

type DnsZone struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ZoneNotFoundError struct {
	Domain string
}

func (e *ZoneNotFoundError) Error() string {
	return fmt.Sprintf("No zone found for domain %s", e.Domain)
}

func GetZoneId(domain, pat string) (string, error) {
	var (
		zones  []DnsZone
		zoneId string = ""
	)

	client := http.Client{}
	zoneRequest, err := http.NewRequest("GET", fmt.Sprintf("%s/dns_zones?access_token=%s", apiEndpoint, pat), nil)
	if err != nil {
		return zoneId, err
	}
	zoneRequest.Header.Set("Content-Type", "application/json")
	zoneResponse, err := client.Do(zoneRequest)
	if err != nil {
		return zoneId, err
	}
	zoneString, err := BodyToString(zoneResponse)
	if err != nil {
		return zoneId, err
	}

	err = json.Unmarshal([]byte(zoneString), &zones)
	if err != nil {
		return zoneId, err
	}

	for _, zone := range zones {
		if zone.Name == domain {
			zoneId = zone.Id
			break
		}
	}

	if zoneId == "" {
		return zoneId, &ZoneNotFoundError{Domain: domain}
	}

	return zoneId, nil
}

func GetCurrentRecordIpv4() (net.IP, error) {
	zoneId, err := GetZoneId(domain, personalAccessToken)

	if err != nil {
		return nil, err
	}

	log.Printf("Target zone id: %v", zoneId)

	return nil, nil
}
