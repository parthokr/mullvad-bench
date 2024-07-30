package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/prometheus-community/pro-bing"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type WireguardRelayResponse struct {
	HostName    string `json:"hostname"`
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	CityCode    string `json:"city_code"`
	CityName    string `json:"city_name"`
	Ipv4AddrIn  string `json:"ipv4_addr_in"`
}

type ProcessedRelay struct {
	HostName     string
	CountryCode  string
	CountryName  string
	CityCode     string
	CityName     string
	Ipv4AddrIn   string
	PingDuration time.Duration
}

type Flags struct {
	listCountries       bool
	searchWithinCountry string
	timeout             time.Duration
	outputFileName      string
}

func parseFlags() Flags {
	var flags Flags
	flag.BoolVar(&flags.listCountries, "lc", false, "List all available countries")
	flag.StringVar(&flags.searchWithinCountry, "c", "", "Search within a country")
	flag.DurationVar(&flags.timeout, "t", 1*time.Second, "Timeout for each ping in seconds")
	flag.StringVar(&flags.outputFileName, "o", "bench_result.csv", "Output file name")
	flag.Parse()
	return flags
}

type country struct {
	CountryName string `json:"country_name"`
	CountryCode string `json:"country_code"`
}

func listCountries() []country {
	url := "https://api.mullvad.net/www/relays/wireguard"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	var countries []country
	err = json.NewDecoder(resp.Body).Decode(&countries)
	if err != nil {
		log.Fatalln(err)
	}
	return countries

}

func contains(arr []string, target string) bool {
	for _, item := range arr {
		if item == target {
			return true
		}
	}
	return false
}

func pluralize(s string, count int) string {
	if count == 1 {
		return s
	}
	return s + "s"

}

func main() {
	flags := parseFlags()
	if flags.listCountries {
		countries := listCountries()
		countryMap := make(map[string]string)
		for _, country := range countries {
			countryMap[country.CountryCode] = country.CountryName
		}
		fmt.Printf("%d available countries\n", len(countryMap))
		fmt.Printf("Code \t|\t Name\n")
		for code, name := range countryMap {
			fmt.Printf("%s \t|\t %s\n", code, name)
		}
		return
	}
	searchWithinCountry := flags.searchWithinCountry
	var scope []string
	if searchWithinCountry == "" {
		fmt.Println("No country specified, searching all servers...")
	} else {
		scope = strings.Split(searchWithinCountry, ",")
		for i, code := range scope {
			scope[i] = strings.TrimSpace(code)
			if len(scope[i]) == 0 {
				fmt.Printf("Invalid country code at position %d\n", i)
			}
		}

	}
	timeout := flags.timeout
	outputFileName := flags.outputFileName
	if _, err := os.Stat(outputFileName); err == nil {
		fmt.Println("Default output file already exists, please specify a different file name")
		fmt.Println("Remove the existing file? (y/N)")
		var ch string
		fmt.Scanln(&ch)
		if ch == "y" {
			err := os.Remove(outputFileName)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Println("Removed the existing file")
		} else {
			fmt.Println("Exiting...")
			return
		}
	}
	url := "https://api.mullvad.net/www/relays/wireguard"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	var relayResponse []WireguardRelayResponse
	err = json.NewDecoder(resp.Body).Decode(&relayResponse)
	if err != nil {
		log.Fatalln(err)
	}

	var totalServers int
	for _, relay := range relayResponse {
		if len(scope) > 0 && !contains(scope, relay.CountryCode) {
			continue
		}
		totalServers++
	}

	fmt.Printf("Found %d servers\n", totalServers)

	var processedRelays []ProcessedRelay

	var count int
	for _, relay := range relayResponse {
		if len(scope) > 0 && !contains(scope, relay.CountryCode) {
			continue
		}
		pinger, err := probing.NewPinger(relay.Ipv4AddrIn)
		pinger.Count = 1
		pinger.Timeout = timeout
		if err != nil {
			log.Fatalln(err)
		}
		pinger.OnRecv = func(pkt *probing.Packet) {
			processedRelays = append(processedRelays, ProcessedRelay{
				HostName:     relay.HostName,
				CountryCode:  relay.CountryCode,
				CountryName:  relay.CountryName,
				CityCode:     relay.CityCode,
				CityName:     relay.CityName,
				Ipv4AddrIn:   relay.Ipv4AddrIn,
				PingDuration: pkt.Rtt,
			})
			count += 1
		}
		err = pinger.Run()
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("\r%d %s processed", count, pluralize("server", count))
	}

	sort.Slice(processedRelays, func(i, j int) bool {
		return processedRelays[i].PingDuration < processedRelays[j].PingDuration
	})

	csvFile, err := os.Create(outputFileName)
	if err != nil {
		log.Fatalln(err)
	}
	defer csvFile.Close()
	csvWriter := csv.NewWriter(csvFile)
	err = csvWriter.Write([]string{"Server", "Country", "City", "Ping"})
	for _, relay := range processedRelays {
		err := csvWriter.Write([]string{relay.HostName, relay.CountryName, relay.CityName, relay.PingDuration.String()})
		if err != nil {
			log.Fatalln(err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		log.Fatalln(err)
	}
}
