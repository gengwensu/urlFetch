/***
website searcher: takes a list of url's; fetch home pages of all and find matches of
a regex (case insensitive) con-currently. Limit 20 HTTP Requests at any given time.
***/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type rankURL struct {
	rank   int
	url    string
	result string
}

// file that contains a list of urls.
//var urlFile = "D:/interview/weWork/urls-test.txt"

// MAXCONCURRENCY the # of http requests allowed at a time
var MAXCONCURRENCY = 20

// BLOCKFACTOR multiple of MAXCONCURRENCY to be proccessed each iteration
var BLOCKFACTOR = 5

var ch = make(chan rankURL)
var tokens = make(chan struct{}, MAXCONCURRENCY)

var re *regexp.Regexp
var debug int
var outFile string

func main() {
	inFile := flag.String("infile", "urls.txt", "input file for urls")
	oFile := flag.String("outfile", "out.txt", "output file for urls")
	rePattern := flag.String("regexp", "new.?", "reg exp for matching")
	debugFlag := flag.Int("debugLevel", 0, "0-off,1-info,2-all")
	flag.Parse()

	fmt.Printf("Web search running. input=%q, output=%q, regexp=%q\n", *inFile, *oFile, *rePattern)
	reString := "(?i)" + *rePattern // flag to case insensitive search
	re = regexp.MustCompile(reString)
	debug = *debugFlag
	outFile = *oFile

	urlList := readUrlList(*inFile) // get the urls from file
	start := time.Now()
	tBlock := BLOCKFACTOR * MAXCONCURRENCY
	//processing a block of threads at a time to make sure getting some output
	n := (int)(len(urlList) / tBlock)
	for i := 0; i <= n; i++ {
		startIndex := i * tBlock
		var endIndex int
		if i == n {
			endIndex = startIndex + len(urlList)%tBlock
		} else {
			endIndex = startIndex + tBlock
		}

		fmt.Printf("main: processing %d to %d\n", startIndex, endIndex-1)
		concurrentSearch(urlList[startIndex:endIndex])
	}

	fmt.Printf("main: %.2fs elapsed overall.\n", time.Since(start).Seconds())
}

func concurrentSearch(urlList []rankURL) {
	for _, v := range urlList {
		go fetch(v)
	}

	var outList []rankURL
	for range urlList {
		output := <-ch

		switch debug {
		case 1:
			fmt.Printf("concurrentSearch: got output from %s\n", output.url)
		case 2:
			fmt.Printf("concurrentSearch: got output from %s, output is %v\n", output.url, output)
		}

		outList = append(outList, output)
		if debug > 1 {
			fmt.Println("concurrentSearch: after append.")
		}
	}

	if debug > 1 {
		fmt.Printf("concurrentSearch: before write to file. outList: %v\n", outList)
	}
	writeToFile(outFile, outList)
}

func fetch(ru rankURL) {
	start := time.Now()

	if debug > 0 {
		fmt.Println(ru.url + " is up " + time.Now().String())
	}
	tokens <- struct{}{} // acquire a token
	if debug > 0 {
		fmt.Println(ru.url + " got a token " + time.Now().String())
	}

	c := &http.Client{ // need to prevent no response case
		Timeout: 15 * time.Second,
	}
	resp, err := c.Get("http://" + ru.url)
	if err != nil { // can't just exit, need to take care of unblocking others
		if debug > 0 {
			fmt.Printf("Get %s, fetch error: %v\n", ru.url, err)
		}
		ru.result = fmt.Sprintf("Get %s, fetch error: %v\n", ru.url, err)

		<-tokens
		if debug > 0 {
			fmt.Println(ru.url + " release a token " + time.Now().String())
		}
		ch <- ru
	} else {
		b, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		<-tokens // release the token
		if debug > 0 {
			fmt.Println(ru.url + " release a token " + time.Now().String())
		}

		if err != nil {
			fmt.Printf("%s: fetch: reading error: %v\n", ru.url, err)
			ru.result = fmt.Sprintf("%s: fetch: reading error: %v\n", ru.url, err)
		} else {
			src := string(b)

			match := re.FindAllString(src, -1)
			for _, s := range match {
				ru.result += " " + s
			}
		}

		ch <- ru
	}

	secs := time.Since(start).Seconds()
	if debug > 0 {
		fmt.Printf("%s: done, %.2fs elapsed\n", ru.url, secs)
	}
}

func readUrlList(fName string) []rankURL {
	file, err := os.Open(fName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var result []rankURL

	scanner := bufio.NewScanner(file)
	for scanner.Scan() { // read in the file line by line
		lineOut := scanner.Text()
		if ok, _ := regexp.MatchString("^[0-9]+", lineOut); ok { //only parsing lines starting with a number
			fieldsOut := strings.Split(lineOut, ",")   // split line into fields; delimitted by ","
			s := fieldsOut[1][1 : len(fieldsOut[1])-1] // strip quotes
			var ru rankURL
			ru.rank, _ = strconv.Atoi(fieldsOut[0])
			ru.url = s
			ru.result = ""
			result = append(result, ru)
		}
	}

	return result
}

func writeToFile(fName string, rList []rankURL) {
	sort.Slice(rList, func(i, j int) bool { return rList[i].rank < rList[j].rank })

	file, err := os.OpenFile(fName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	for _, e := range rList {
		fmt.Fprintf(file, "%d, %q, matches found: %s\n", e.rank, e.url, e.result)
	}

}
