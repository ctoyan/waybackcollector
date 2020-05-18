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

	requestsPerSecond := flag.Int("req-per-sec", 0, "Requests per second. 0 means no one request at a time")
	estimateTime := flag.Bool("time", false, "Show how much time it would take to make all requests for the current query")
	printUrls := flag.Bool("print-urls", false, "Print to stdout only a list of historic URLs, which you can request later")
	unique := flag.Bool("unique", false, "Print to stdout only unique reponses")
	output := flag.String("output", "", "Path to a folder where the tool will safe all unique responses in uniquely named files per response")
	logFile := flag.String("log-file", "", "Log every wayback history request url, but not the response")

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

	addToLog("Main request url: "+requestUrl, *logFile)

	historyItems := getHistoryItems(requestUrl)

	if *estimateTime {
		requestsCount := len(historyItems)
		duration, err := time.ParseDuration(fmt.Sprintf("%vs", requestsCount/(*requestsPerSecond)))
		if err != nil {
			log.Fatalf("error parsing duration: %v", err)
		}
		fmt.Printf("All %v requests will be made in %v", requestsCount, duration)
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

		if i%*requestsPerSecond == 0 {
			time.Sleep(1 * time.Second)
		}

		wg.Add(1)
		go func() {
			addToLog("Hisotry item request url: "+historyUrl, *logFile)

			response, err := get(historyUrl)
			if err != nil {
				fmt.Printf("error making history item request: %v", err)
				addToLog(fmt.Sprintf("ERROR fetching %v: %v", historyUrl, err), *logFile)
			}

			if !*printUrls && !*unique && *output == "" {
				fmt.Println(string(response))
			}

			historicResponses = append(historicResponses, response)

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

func addToLog(logRow string, logFile string) {
	if logFile != "" {
		f, err := os.OpenFile(logFile,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("error writing to log file: %v", err)
		}
		defer f.Close()
		if _, err := f.WriteString(logRow + "\n"); err != nil {
			log.Fatalf("error writing to log file: %v", err)
		}
	}
}

func get(url string) (body []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func getHistoryItems(requestUrl string) []HistoryItem {
	body, err := get(requestUrl)
	if err != nil {
		log.Fatalf("error getting history items: %v", err)
	}

	var timestamps2d [][]string
	err = json.Unmarshal(body, &timestamps2d)
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
