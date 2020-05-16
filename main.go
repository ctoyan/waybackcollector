package main

import (
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// Wayback CDX Server rate limits requests
const MAX_REQUESTS_COUNT = 28

// Pause the requests every 4 seconds
const PAUSE_INTERVAL = 4

type HistoryItem struct {
	Timestamp string
	Digest    string
	Length    string
}

func main() {
	url := flag.String("url", "", "URL pattern to collect responses for")
	dateFrom := flag.String("from", "", "Date on which to start collecing responses. Inclusive. Format: yyyyMMddhhmmss. Defaults to first ever record.")
	dateTo := flag.String("to", "", "Date on which to end collecing responses. Inclusive. Format: yyyyMMddhhmmss. Defaults to last ever record.")
	limit := flag.Int("limit", 0, "Limit the results")
	filter := flag.String("filter", "", "Filter your search, using the wayback cdx filters (find more here: https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server#filtering)")
	collapse := flag.String("collapse", "", "A form of filtering, with which you can collaps adjasent fields(find more here: https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server#collapsing)")

	estimateTime := flag.Bool("time", false, "Show how much time it would take to make all requests for the current query")
	printUrls := flag.Bool("print-urls", false, "Print to stdout only a list of historic URLs, which you can request later")
	unique := flag.Bool("unique", false, "Print to stdout only unique reponses")
	output := flag.String("output", "", "Path to a folder where the tool will safe all unique responses in uniquely named files per response (meg style output)")

	flag.Parse()

	if *url == "" {
		log.Fatal("url argument is required")
	}

	if (*printUrls && *unique) ||
		(*printUrls && *output != "") ||
		(*unique && *output != "") {
		log.Fatal("you can only set one of the following arguments: print-urls, unique, output")
	}

	requestUrl := fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=%v&output=json&fl=timestamp,digest,length", *url)

	if *dateFrom != "" {
		requestUrl += fmt.Sprintf("&from=%v", *dateFrom)
	}
	if *dateTo != "" {
		requestUrl += fmt.Sprintf("&to=%v", *dateTo)
	}
	if *limit != 0 {
		requestUrl += fmt.Sprintf("&limit=%v", *limit)
	}
	if *collapse != "" {
		requestUrl += fmt.Sprintf("&collapse=%v", *collapse)
	}
	if *filter != "" {
		requestUrl += fmt.Sprintf("&filter=%v", *filter)
	}

	historyItems := getHistoryItems(requestUrl)
	if *estimateTime {
		requestsCount := len(historyItems)
		duration, err := time.ParseDuration(fmt.Sprintf("%vs", requestsCount/(MAX_REQUESTS_COUNT/PAUSE_INTERVAL)))
		if err != nil {
			log.Fatalf("error parsing duration: %v", err)
		}
		fmt.Printf("All %v requests will take %v", requestsCount, duration)
	}

	var allHistoryUrls []string
	var wg sync.WaitGroup
	historicResponses := make([][]byte, len(historyItems))
	for i, hi := range historyItems {
		historyUrl := fmt.Sprintf("https://web.archive.org/web/%vif_/%v", hi.Timestamp, *url)

		if *printUrls {
			allHistoryUrls = append(allHistoryUrls, historyUrl)
			continue
		}

		wg.Add(1)
		if i%MAX_REQUESTS_COUNT == 0 {
			time.Sleep(PAUSE_INTERVAL * time.Second)
		}

		go func() {
			historicResponses = append(historicResponses, get(historyUrl))
			wg.Done()
		}()
	}
	wg.Wait()

	uniqueResponses := make(map[[20]byte][]byte)
	for _, res := range historicResponses {
		if *output != "" || *unique {
			uniqueResponses[sha1.Sum(res)] = res
		}

		if !*unique && !*printUrls && *output == "" {
			fmt.Println(string(res))
		}
	}

	if *unique {
		for k, _ := range uniqueResponses {
			fmt.Println(string(uniqueResponses[k]))
		}
		return
	}

	if *printUrls {
		for _, au := range allHistoryUrls {
			fmt.Println(au)
		}
		return
	}

	if *output != "" {
		os.MkdirAll(*output, 0700)
		for k, _ := range uniqueResponses {
			err := ioutil.WriteFile(fmt.Sprintf("%v/%x", *output, k), uniqueResponses[k], 0644)
			if err != nil {
				log.Fatalf("error writing to file: %v", err)
			}
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

func getHistoryItems(requestUrl string) []HistoryItem {
	body := get(requestUrl)

	var timestamps2d [][]string
	err := json.Unmarshal(body, &timestamps2d)
	if err != nil {
		log.Fatalf("error parsing timestamps: %v", err)
	}

	var timestamps []HistoryItem
	for i, val := range timestamps2d {
		if i == 0 {
			continue
		}
		timestamps = append(timestamps, HistoryItem{
			Timestamp: val[0],
			Digest:    val[1],
			Length:    val[2],
		})
	}
	return timestamps
}
