package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
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
	ttl                 int64
	pollInterval        time.Duration
)

func init() {
	var (
		pollIntervalString string
		ttlString          string
	)

	flag.StringVar(&personalAccessToken, "pat", os.Getenv("NDDNS_PAT"), "Netlify personal access token")
	flag.StringVar(&domain, "domain", "", "Netlify controlled domain")
	flag.StringVar(&subdomain, "subdomain", "", "Subdomain to which A record will be added")
	flag.StringVar(&ttlString, "ttl", "300", "Time to live for A record")
	flag.StringVar(&pollIntervalString, "poll", "30m", "How often to poll for changes to IP")

	flag.Parse()

	if personalAccessToken == "" {
		log.Fatalln("Personal access token required. Use -pat flag or set NDDNS_PAT environment variable")
	}

	if domain == "" {
		log.Fatalln("Domain required. Use -domain flag")
	}
	hostname = BuildHostname(domain, subdomain)

	t, err := strconv.ParseInt(ttlString, 10, 32)
	if err != nil {
		log.Fatalln("Failed to parse ttl as number")
	}

	if t <= 0 {
		log.Fatalf("TTL must be positive. Found: %v", t)
	}

	ttl = t

	p, err := time.ParseDuration(pollIntervalString)
	if err != nil {
		panic(err)
	}
	pollInterval = p
}

func main() {
	log.Println("Starting ddns service...")

	for {
		err := PollForChanges()
		if err != nil {
			log.Printf("Error occurred: %v", err)
		}

		time.Sleep(pollInterval)
	}
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

type DnsRecord struct {
	Id       string `json:"id"`
	Hostname string `json:"hostname"`
	Value    string `json:"value"`
}

func GetCurrentRecord(zoneId string) (*DnsRecord, error) {
	client := http.Client{}
	recordsRequest, err := http.NewRequest("GET", fmt.Sprintf("%s/dns_zones/%s/dns_records?access_token=%s", apiEndpoint, zoneId, personalAccessToken), nil)
	if err != nil {
		return nil, err
	}
	recordsRequest.Header.Set("Content-Type", "application/json")

	res, err := client.Do(recordsRequest)
	if err != nil {
		return nil, err
	}

	recordsString, err := BodyToString(res)
	if err != nil {
		return nil, err
	}

	var (
		dnsRecords []DnsRecord
	)
	err = json.Unmarshal([]byte(recordsString), &dnsRecords)

	if err != nil {
		return nil, err
	}

	for _, record := range dnsRecords {
		if record.Hostname == hostname {
			return &record, nil
		}
	}

	return nil, nil
}

type CreateRecordBody struct {
	Type     string `json:"type"`
	Hostname string `json:"hostname"`
	Value    string `json:"value"`
	Ttl      int64  `json:"ttl"`
}

type CreateRecordResponse struct {
	Type     string `json:"type"`
	Hostname string `json:"hostname"`
	Value    string `json:"value"`
	Ttl      int64  `json:"ttl"`
}

func CreateIPv4Record(zoneId string, target net.IP) error {
	client := http.Client{}

	body := CreateRecordBody{
		Type:     "A",
		Hostname: hostname,
		Value:    target.String(),
		Ttl:      ttl,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/dns_zones/%s/dns_records?access_token=%s", apiEndpoint, zoneId, personalAccessToken), bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	resString, err := BodyToString(res)
	if err != nil {
		return err
	}

	var resBody CreateRecordResponse
	err = json.Unmarshal([]byte(resString), &resBody)
	if err != nil {
		return err
	}

	if resBody.Hostname != hostname ||
		resBody.Type != "A" ||
		resBody.Value != target.String() ||
		resBody.Ttl != ttl {
		return errors.New("failed to create record")
	}

	return nil
}

func DeleteIpv4Record(zoneId, recordId string) error {
	client := http.Client{}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/dns_zones/%s/dns_records/%s?access_token=%s", apiEndpoint, zoneId, recordId, personalAccessToken), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.Status != "204" {
		return fmt.Errorf("failed to delete DNS record %s, got status code %s", recordId, res.Status)
	}
	return nil
}

func PollForChanges() error {
	currentIpv4, err := GetCurrentIpv4()
	if err != nil {
		return err
	}
	log.Printf("Current IPv4: %v", currentIpv4)

	zoneId, err := GetZoneId(domain, personalAccessToken)

	if err != nil {
		return err
	}

	log.Printf("Target zone id: %v", zoneId)

	dnsRecord, err := GetCurrentRecord(zoneId)
	if err != nil {
		return err
	}

	if dnsRecord == nil {
		log.Printf("No DNS record found for %s. Updating to current IPv4 of: %v", hostname, currentIpv4)
		CreateIPv4Record(zoneId, currentIpv4)
		return nil
	}

	if dnsRecord.Value == currentIpv4.String() {
		log.Println("Value of record matches current IP address")
		return nil
	}

	log.Printf("Record value of %s differs from current IP of %v", dnsRecord.Value, currentIpv4)
	log.Printf("Deleting DNS record: %s", dnsRecord.Id)
	DeleteIpv4Record(zoneId, dnsRecord.Id)

	log.Printf("Creating DNS A record for %s with value %v", hostname, currentIpv4)
	CreateIPv4Record(zoneId, currentIpv4)

	return nil
}
