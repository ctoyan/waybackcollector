package main

import (
	"flag"
	"log"
	"fmt"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"crypto/sha1"
)

type HistoryItem struct {
	Timestamp string
	Digest 		string
	Length 		string
}

func main() {
	url 		   := flag.String("url", "", "URL pattern to collect responses for")
	dateFrom   := flag.String("from", "", "Date on which to start collecing responses. Inclusive. Format: yyyyMMddhhmmss. Defaults to first ever record.")
	dateTo 	   := flag.String("to", "", "Date on which to end collecing responses. Inclusive. Format: yyyyMMddhhmmss. Defaults to last ever record.")
	limit 	   := flag.Int("limit", 0, "Limit the results")
	filter 	 	 := flag.String("filter", "", "Filter your search, using the wayback cdx filters (find more here: https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server#filtering)")
	urlsOnly 	 := flag.Bool("urls-only", false, "Get only a list of historic URLs")
	uniqueOnly := flag.Bool("unique-only", false, "Show only unique reponses")
	output 		 := flag.String("output", "", "Path to a folder where the tool will safe all unique responses in uniquely named files per response (meg style output)")

	flag.Parse()

	if *url == "" {
		log.Fatal("url argument is required")
	}

	requestUrl := fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=%v&output=json&fl=timestamp,digest,length", *url)

	if *dateFrom != ""  { requestUrl += fmt.Sprintf("&from=%v", *dateFrom) }
	if *dateTo 	 != ""  { requestUrl += fmt.Sprintf("&to=%v", *dateTo) }
	if *limit 	 != 0 	{ requestUrl += fmt.Sprintf("&limit=%v", *limit) }
	if *filter 	 != ""  { requestUrl += fmt.Sprintf("&filter=%v", *filter) }
	
	uniqueResponses := make(map[[20]byte][]byte)
	var allHistoryUrls []string

	historyItems := getHistoryItems(requestUrl)
	for _, hi := range historyItems {
		historyUrl := fmt.Sprintf("https://web.archive.org/web/%vif_/%v", hi.Timestamp, *url)

		if *urlsOnly {
			allHistoryUrls = append(allHistoryUrls, historyUrl)
			continue
			return
		}

		body := get(historyUrl)

		if *uniqueOnly || *output != "" {
			uniqueResponses[sha1.Sum(body)] = body
		}

		if !*uniqueOnly && !*urlsOnly {
			fmt.Println(string(body))
		}
	}

	if *output != "" {
		for k, _ := range uniqueResponses {
			err := ioutil.WriteFile(fmt.Sprintf("%v/%x", *output, k), uniqueResponses[k], 0644)
			if err != nil {
				log.Fatalf("error writing to file: %v", err)
			}
		}
	}

	if *urlsOnly {
		for _, au := range allHistoryUrls {
			fmt.Println(au)
		}
		return
	}

	if *uniqueOnly {
		for k, _ := range uniqueResponses {
			fmt.Println(string(uniqueResponses[k]))
		}
	}
}

func get(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("error making get request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
		log.Fatalf("error reading response body: %v", err)
  }

	return body
}

func getHistoryItems (requestUrl string) []HistoryItem {
	body := get(requestUrl)

	var timestamps2d [][]string
	err := json.Unmarshal(body, &timestamps2d)
	if err != nil {
		log.Fatalf("error parsing timestamps: %v", err)
	}

	var timestamps []HistoryItem
	for i, val := range timestamps2d {
		if i == 0 { continue }
		timestamps = append(timestamps, HistoryItem {
			Timestamp: val[0],
			Digest: 	 val[1],
			Length:		 val[2],
		})
	}
	return timestamps
}
