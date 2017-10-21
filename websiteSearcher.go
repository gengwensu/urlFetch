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

var ch = make(chan rankURL)
var tokens = make(chan struct{}, MAXCONCURRENCY)

var re *regexp.Regexp

func main() {
	inFile := flag.String("infile", "urls.txt", "input file for urls")
	outFile := flag.String("outfile", "out.txt", "output file for urls")
	rePattern := flag.String("regexp", "new.?", "reg exp for matching")
	flag.Parse()
	reString := "(?i)" + *rePattern // flag to case insensitive search
	re = regexp.MustCompile(reString)

	urlList := readUrlList(*inFile) // get the urls from file
	start := time.Now()
	for _, v := range urlList {
		go fetch(v.url)
	}

	for range urlList {
		output := <-ch
		key := output.url[7:]
		//fmt.Printf("got output from %s, match %s\n", key, output.result)
		output.rank = urlList[key].rank
		urlList[key] = output
		//fmt.Printf("assign to urlList[%s], match %s\n", key, urlList[key].result)
	}

	writeToFile(*outFile, urlList)
	fmt.Printf("%.2fs elapsed overall.\n", time.Since(start).Seconds())
}

func fetch(url string) {
	start := time.Now()
	var ru rankURL
	ru.url = url
	//fmt.Println(url + " is up " + time.Now().String())
	tokens <- struct{}{} // acquire a token
	//fmt.Println(url + " got a token " + time.Now().String())

	resp, err := http.Get(url)
	if err != nil {
		ru.result = fmt.Sprintf("Get %s, fetch error: %v\n", url, err)
		<-tokens
		ch <- ru
	} else {
		b, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
			os.Exit(1)
		}
		<-tokens // release the token
		//fmt.Println(url + " release a token " + time.Now().String())
		src := string(b)
		// fmt.Println(src)
		//fmt.Printf("%s\n", re.FindAllString(src, -1))

		match := re.FindAllString(src, -1)
		for _, s := range match {
			ru.result += " " + s
		}
		ch <- ru
	}

	secs := time.Since(start).Seconds()
	fmt.Printf("%s done, %.2fs elapsed\n", url, secs)
}

func readUrlList(fName string) map[string]rankURL {
	file, err := os.Open(fName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var result = make(map[string]rankURL)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() { // read in the file line by line
		lineOut := scanner.Text()
		if ok, _ := regexp.MatchString("^[0-9]+", lineOut); ok { //only parsing lines starting with a number
			fieldsOut := strings.Split(lineOut, ",")   // split line into fields; delimitted by ","
			s := fieldsOut[1][1 : len(fieldsOut[1])-1] // strip quotes
			var ru rankURL
			ru.rank, _ = strconv.Atoi(fieldsOut[0])
			ru.url = "http://" + s
			ru.result = ""
			result[s] = ru
		}
	}

	return result
}

type kv struct { // data structure for sorting the map result
	key   string
	value int
}

func writeToFile(fName string, m map[string]rankURL) {
	var ss []kv
	for k, v := range m { // need to sort the output by rank
		ss = append(ss, kv{k, v.rank})
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].value < ss[j].value })

	file, err := os.OpenFile(fName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	for _, e := range ss {
		ue := m[e.key]
		fmt.Fprintf(file, "%d, %q, matches found: %s\n", ue.rank, e.key, ue.result)
	}

}
