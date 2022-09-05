package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	URL "net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/mmpx12/optionparser"
)

var (
	success   = 0
	mu        = &sync.Mutex{}
	thread    = make(chan struct{}, 50)
	wg        sync.WaitGroup
	output    = "found_git.txt"
	proxy     string
	insecure  bool
	version   = "1.1.0"
	timeout   = 5
	userAgent = "Mozilla/5.0 (X11; Linux x86_64)"
)

func WriteToFile(target string) {
	f, _ := os.OpenFile(output, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	defer f.Close()
	fmt.Fprintln(f, target)
}

func VerifyDirListing(resp *http.Response) (ok bool) {
	scan := bufio.NewScanner(resp.Body)
	toFind := []byte("Index of /.git")
	for scan.Scan() {
		if bytes.Contains(scan.Bytes(), toFind) {
			return true
		}
	}
	return false
}

func verifyNonDirListing(client *http.Client, url string) (ok bool) {
	req, err := http.NewRequest("GET", "https://"+url+"/.git/config", nil)
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)

	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return false
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if string(body)[:6] == "[core]" {
		return true
	} else {
		return false
	}
	return false
}

func CheckURL(client *http.Client, i, total int, url string) {
	defer wg.Done()
	fmt.Printf("\033[1K\r\033[31m[\033[33m%d\033[36m/\033[33m%d \033[36m(\033[32m%d\033[36m)\033[31m] \033[35m%s\033[0m", i, total, success, url)
	req, err := http.NewRequest("GET", "https://"+url+"/.git/", nil)
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)

	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		<-thread
		return
	}

	if resp.StatusCode == 200 {
		isGitDir := VerifyDirListing(resp)
		if isGitDir {
			success++
			mu.Lock()
			WriteToFile(resp.Request.URL.String())
			mu.Unlock()
			fmt.Printf("\033[1K\rGIT FOUND: " + resp.Request.URL.String() + "\n")

		}
	} else if resp.StatusCode == 403 {
		isGit := verifyNonDirListing(client, url)
		if isGit {
			success++
			mu.Lock()
			WriteToFile(resp.Request.URL.String())
			mu.Unlock()
			fmt.Printf("\033[1K\rGIT FOUND (non dir listing): " + resp.Request.URL.String() + "\n")
		}
	}
	<-thread
}

func LineNBR(f string) int {
	r, _ := os.Open(f)
	defer r.Close()
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}
	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)
		switch {
		case err == io.EOF:
			return count
		case err != nil:
			return 0
		}
	}
}

func CheckGit(input string) {
	readFile, err := os.Open(input)
	defer readFile.Close()
	if err != nil {
		fmt.Println(err)
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	i := 0
	total := LineNBR(input)

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Duration(timeout) * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   time.Duration(timeout) * time.Second,
			ResponseHeaderTimeout: 3 * time.Second,
			DisableKeepAlives:     true,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: insecure},
		},
	}
	if proxy != "" {
		proxyURL, _ := URL.Parse(proxy)
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	}

	for fileScanner.Scan() {
		target := fileScanner.Text()
		thread <- struct{}{}
		i++
		wg.Add(1)
		go CheckURL(client, i, total, target)
	}
	wg.Wait()
}

func main() {
	var threads, input, time string
	var printversion bool
	op := optionparser.NewOptionParser()
	op.Banner = "Scan for exposed git repos\n\nUsage:\n"
	op.On("-t", "--thread NBR", "Number of threads (default 50)", &threads)
	op.On("-o", "--output FILE", "Output file (default found_git.txt)", &output)
	op.On("-i", "--input FILE", "Input file", &input)
	op.On("-k", "--insecure", "Ignore certificate errors", &insecure)
	op.On("-T", "--timeout SEC", "Set timeout (default 5s)", &time)
	op.On("-u", "--user-agent USR", "Set user agent", &userAgent)
	op.On("-p", "--proxy PROXY", "Use proxy (proto://ip:port)", &proxy)
	op.On("-V", "--version", "Print version and exit", &printversion)
	op.Exemple("xgit -i top-alexa.txt")
	op.Exemple("xgit -p socks5://127.0.0.1:9050 -K -o good.txt -i top-alexa.txt -t 60")
	op.Parse()
	op.Logo("[X-git]", "doom", false)

	if printversion {
		fmt.Println("version:", version)
		os.Exit(1)
	}

	if threads != "" {
		tr, _ := strconv.Atoi(threads)
		thread = make(chan struct{}, tr)
	}

	if time != "" {
		timeout, _ = strconv.Atoi(time)
	}

	if input == "" {
		fmt.Println("\033[31m[!] You must specify an input file\033[0m\n")
		op.Help()
		os.Exit(1)
	}

	log.SetOutput(io.Discard)
	os.Setenv("GODEBUG", "http2client=0")
	CheckGit(input)
	fmt.Printf("\033[1k\rFound %d git repos.", success)
	fmt.Println()
}
