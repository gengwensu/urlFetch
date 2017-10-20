/***
website searcher: takes a list of url's; fetch home pages of all and find matches of
a regex (case insensitive) con-currently. Limit 20 HTTP Requests at any given time.
***/

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

// list of urls.
var urlList = [...]string{
	"facebook.com/",
	"twitter.com/",
	"google.com/",
	"youtube.com/",
	"wordpress.org/",
	"adobe.com/",
	"blogspot.com/",
	"wikipedia.org/",
	"linkedin.com/",
	"wordpress.com/",
}

// MAXCONCURRENCY the # of http requests allowed at a time
var MAXCONCURRENCY = 3

var tokens = make(chan struct{}, MAXCONCURRENCY)

func main() {
	start := time.Now()
	ch := make(chan string)

	for i, u := range urlList {
		url := "http://" + u
		go fetch(i, url, ch)
	}
	var result [10]string
	for range urlList {
		output := <-ch
		//fmt.Println(output)
		//fmt.Println(output[:2])
		idx, _ := strconv.Atoi(output[:2])
		//fmt.Printf("%d\n", idx)
		result[idx] = output[3:]
		//fmt.Println(result[idx])
	}

	for i, r := range result {
		fmt.Printf("%d: %s\n", i+1, r)
	}
	fmt.Printf("%.2fs elapsed overall.\n", time.Since(start).Seconds())
}

func fetch(idx int, url string, ch chan<- string) {
	start := time.Now()
	re := regexp.MustCompile("new.?")
	fmt.Println(url + " is up " + time.Now().String())
	tokens <- struct{}{} // acquire a token
	fmt.Println(url + " got a token " + time.Now().String())

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		os.Exit(1)
	}
	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
		os.Exit(1)
	}
	<-tokens // release the token
	fmt.Println(url + " release a token " + time.Now().String())
	src := string(b)
	// fmt.Println(src)
	//fmt.Printf("%s\n", re.FindAllString(src, -1))
	secs := time.Since(start).Seconds()
	ch <- fmt.Sprintf("%02d\t%.2fs elapsed, %s: %v found", idx, secs, url, re.FindAllString(src, -1))
}
