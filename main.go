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
	dateFrom := flag.String("from", "", "Date on which to start collecing responses. Inclusive. Format: yyyyMMddhhmmss. Defaults to first ever record")
	dateTo := flag.String("to", "", "Date on which to end collecing responses. Inclusive. Format: yyyyMMddhhmmss. Defaults to last ever record")
	limit := flag.Int("limit", 0, "Limit the results")
	filter := flag.String("filter", "", "Filter your search, using the wayback cdx filters (find more here: https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server#filtering)")
	collapse := flag.String("collapse", "", "A form of filtering, with which you can collaps adjasent fields(find more here: https://github.com/internetarchive/wayback/tree/master/wayback-cdx-server#collapsing)")

	requestsPerSecond := flag.Int("req-per-sec", 1, "Requests per second. Defaults to 1")
	estimateTime := flag.Bool("time", false, "Show how much time it would take to make all requests for the current query and exit (without response time take into account)")
	printUrls := flag.Bool("print-urls", false, "Print to stdout only a list of historic URLs, which you can request later")
	unique := flag.Bool("unique", false, "Print to stdout only unique reponses")
	outputFolder := flag.String("output-folder", "", "Path to a folder where the tool will safe all unique responses, in uniquely named files")
	logSuccessFile := flag.String("log-success-file", "", "Path to log file. Log successful request urls only")
	logFailFile := flag.String("log-fail-file", "", "Path to log file. Log failed requests only")
	verbose := flag.Bool("verbose", false, "Show more detailed output")

	flag.Parse()

	if *url == "" {
		log.Fatal("url argument is required")
	}

	if (*printUrls && *unique) ||
		(*printUrls && *outputFolder != "") ||
		(*unique && *outputFolder != "") {
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

	printVerbose(fmt.Sprintf("[*] Constructed main request url: %v", requestUrl), *verbose)

	historyItems := getHistoryItems(requestUrl)
	requestsCount := len(historyItems)
	duration, err := time.ParseDuration(fmt.Sprintf("%vs", requestsCount/(*requestsPerSecond)))
	if err != nil {
		log.Fatalf("error parsing duration: %v", err)
	}

	printVerbose(fmt.Sprintf("[*] Making %v requests for approx. %v", requestsCount, duration), *verbose)

	if *estimateTime {
		if *unique || *outputFolder != "" {
			fmt.Println("[!] With the unique or output options enabled, the request count(an therefore the time) will be much lower than what's show below")
		}
		fmt.Printf("All %v requests will be made in %v", requestsCount, duration)
		return
	}

	var wg sync.WaitGroup
	uniqueResponses := make(map[[20]byte][]byte)
	for i, hi := range historyItems {
		historyUrl := fmt.Sprintf("https://web.archive.org/web/%vif_/%v", hi.Timestamp, *url)

		if *printUrls {
			fmt.Println(historyUrl)
			continue
		}

		if i%*requestsPerSecond == 0 {
			printVerbose(fmt.Sprintf("[*] Throttoling for a second at request number %v", i), *verbose)
			time.Sleep(1 * time.Second)
		}

		wg.Add(1)
		go func() {
			response, err := get(historyUrl)
			if err != nil {
				fmt.Printf("error making history item request: %v", err)
				appendToFile(historyUrl, *logFailFile)
				return
			} else {
				appendToFile(historyUrl, *logSuccessFile)
			}

			hashedResponse := sha1.Sum(response)

			if !*printUrls && !*unique && *outputFolder == "" {
				fmt.Println(string(response))
			}

			if *unique {
				uniqueResponse := string(uniqueResponses[hashedResponse])
				if uniqueResponse == "" {
					printVerbose("[*] Found new unique response", *verbose)
					uniqueResponses[hashedResponse] = response
					fmt.Println(string(response))
				}
			}

			if *outputFolder != "" && !fileExists(fmt.Sprintf("%v/%x", *outputFolder, hashedResponse)) {
				printVerbose(fmt.Sprintf("[*] Writing new unique response to file with name %x", hashedResponse), *verbose)
				os.MkdirAll(*outputFolder, os.ModePerm)
				appendToFile(string(response), fmt.Sprintf("%v/%x", *outputFolder, hashedResponse))
			}

			wg.Done()
		}()
	}
	wg.Wait()
}

func appendToFile(data string, filePath string) {
	if filePath != "" {
		f, err := os.OpenFile(filePath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("error writing to file: %v", err)
		}
		defer f.Close()
		if _, err := f.WriteString(data + "\n"); err != nil {
			log.Fatalf("error writing to file: %v", err)
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

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func printVerbose(msg string, verbose bool) {
	if verbose {
		fmt.Println(msg)
	}
}
