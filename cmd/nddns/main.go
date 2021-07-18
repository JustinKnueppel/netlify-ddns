package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

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
