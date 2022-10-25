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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mmpx12/optionparser"
)

var (
	success       = 0
	outdated_repo = 0
	mu            = &sync.Mutex{}
	thread        = make(chan struct{}, 50)
	wg            sync.WaitGroup
	output        = "found_git.txt"
	proxy         string
	insecure      bool
	version       = "1.1.5"
	timeout       = 5
	date          string
	dateFormat    string
	userAgent     = "Mozilla/5.0 (X11; Linux x86_64)"
)

func WriteToFile(target string) {
	f, _ := os.OpenFile(output, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	defer f.Close()
	fmt.Fprintln(f, target)
}

func CheckDate(client *http.Client, url string) (outdated bool) {
	req, err := http.NewRequest("GET", "https://"+url+"/.git/logs/HEAD", nil)
	if err != nil {
		return false
	}
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)

	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return false
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	headDate := strings.Split(string(body), "\n")
	headDate = strings.Split(headDate[len(headDate)-2], "@")
	headDate = strings.Split(headDate[1], " ")
	repodate, _ := strconv.Atoi(headDate[1])
	t, err := time.Parse(dateFormat, date)
	if repodate-int(t.Unix()) >= 0 {
		return true
	} else {
		i, err := strconv.ParseInt(headDate[1], 10, 64)
		if err != nil {
			panic(err)
		}
		t := time.Unix(i, 0)
		dom := string(resp.Request.URL.String()[:len(resp.Request.URL.String())-9])
		fmt.Printf("\033[1K\rOUTDATED: \033[37m " + dom + " \033[31mlast update: " + t.Format("02-01-2006") + "\033[0m\n")
		return false
	}
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
	req, err := http.NewRequest("GET", "https://"+url+"/.git/HEAD", nil)
	if err != nil {
		return false
	}
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)

	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return false
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	r := regexp.MustCompile("(^(([a-fA-F]|[0-9]){40})|^ref: )")
	if r.MatchString(string(body)) {
		return true
	}
	return false
}

func CheckURL(client *http.Client, i, total int, url string) {
	defer wg.Done()
	fmt.Printf("\033[1K\r\033[31m[\033[33m%d\033[36m/\033[33m%d \033[36m(\033[32m%d\033[36m)\033[31m] \033[35m%s\033[0m", i, total, success, url)
	req, err := http.NewRequest("GET", "https://"+url+"/.git/", nil)
	if err != nil {
		<-thread
		return
	}
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
			if date == "" || CheckDate(client, url) {
				success++
				mu.Lock()
				WriteToFile(resp.Request.URL.String())
				mu.Unlock()
				fmt.Printf("\033[1K\rGIT FOUND:\033[36m " + resp.Request.URL.String() + "\033[0m\n")
			}
		}
	} else if resp.StatusCode == 403 {
		isGit := verifyNonDirListing(client, url)
		if isGit {
			if date == "" || CheckDate(client, url) {
				success++
				mu.Lock()
				WriteToFile("[nd] " + resp.Request.URL.String())
				mu.Unlock()
				fmt.Printf("\033[1K\rGIT FOUND:\033[33m " + resp.Request.URL.String() + "\033[0m\n")
			}
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
		os.Exit(1)
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
	var threads, input, timeOut string
	var printversion bool
	op := optionparser.NewOptionParser()
	op.Banner = "Scan for exposed git repos\n\nUsage:\n"
	op.On("-t", "--thread NBR", "Number of threads (default 50)", &threads)
	op.On("-o", "--output FILE", "Output file (default found_git.txt)", &output)
	op.On("-i", "--input FILE", "Input file", &input)
	op.On("-d", "--check-date DATE", "repo only after date (mm-dd-yyyy, yyyy, mm-yyyy)", &date)
	op.On("-k", "--insecure", "Ignore certificate errors", &insecure)
	op.On("-T", "--timeout SEC", "Set timeout (default 5s)", &timeOut)
	op.On("-u", "--user-agent USR", "Set user agent", &userAgent)
	op.On("-p", "--proxy PROXY", "Use proxy (proto://ip:port)", &proxy)
	op.On("-V", "--version", "Print version and exit", &printversion)
	op.Exemple("xgit -i top-alexa.txt")
	op.Exemple("xgit -p socks5://127.0.0.1:9050 -k -o good.txt -i top-alexa.txt -t 60")
	op.Output("GIT FOUND:\033[36m https://localhost/.git/\033[0m (directory listing enable)")
	op.Output("GIT FOUND:\033[33m https://localhost/.git/\033[0m (directory listing disable)")
	op.Output("-OUTDATED:\033[37m https://localhost/.git/\033[31m last update: 02-01-2006\033[0m\n")
	op.Parse()
	fmt.Printf("\033[31m")
	op.Logo("[X-git]", "doom", false)
	fmt.Printf("\033[0m")

	if printversion {
		fmt.Println("version:", version)
		os.Exit(1)
	}

	if threads != "" {
		tr, _ := strconv.Atoi(threads)
		thread = make(chan struct{}, tr)
	}

	if timeOut != "" {
		timeout, _ = strconv.Atoi(timeOut)
	}

	if input == "" {
		fmt.Println("\033[31m[!] You must specify an input file\033[0m\n")
		op.Help()
		os.Exit(1)
	}

	if date != "" {
		switch len(date) {
		case 7:
			dateFormat = "01-2006"
		case 4:
			dateFormat = "2006"
		case 10:
			dateFormat = "01-02-2006"
		default:
			fmt.Println("[!] Date format error\n")
			op.Help()
			os.Exit(1)
		}
	}

	log.SetOutput(io.Discard)
	os.Setenv("GODEBUG", "http2client=0")
	CheckGit(input)
	fmt.Printf("\033[1K\rFound %d git repos.\n", success)
}
