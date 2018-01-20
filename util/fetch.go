package util

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/jammystuff/gocineworld"
)

const listingsXMLURL = "https://www.cineworld.co.uk/syndication/listings.xml"

func FetchListingsXML() []byte {
	fmt.Print("Fetching listings... ")

	resp, err := http.Get(listingsXMLURL)
	if err != nil {
		fmt.Println("ERROR")
		fmt.Fprintf(os.Stderr, "Unable to fetch listings: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	listingsXML, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ERROR")
		fmt.Fprintf(os.Stderr, "Unable to fetch listings: %v", err)
		os.Exit(1)
	}

	fmt.Println("OK")
	return listingsXML
}

func ParseListingsXML(listingsXML []byte) gocineworld.Listings {
	fmt.Print("Parsing listings... ")

	listings, err := gocineworld.ParseListings(listingsXML)
	if err != nil {
		fmt.Println("ERROR")
		fmt.Fprintf(os.Stderr, "Unable to parse listings: %v", err)
		os.Exit(1)
	}

	fmt.Println("OK")
	return listings
}
