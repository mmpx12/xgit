package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	URL "net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/mmpx12/optionparser"
)

var (
	success  = 0
	mu       = &sync.Mutex{}
	wg       = sync.WaitGroup{}
	thread   = make(chan struct{}, 50)
	output   = "found_git.txt"
	proxy    string
	insecure bool
	version  = "1.0.2"
	timeout  = 5
)

func WriteToFile(target string) {
	f, _ := os.OpenFile(output, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	defer f.Close()
	fmt.Fprintln(f, target)
}

func Verify(resp *http.Response) (ok bool) {
	scan := bufio.NewScanner(resp.Body)
	toFind := []byte("Index of /.git")
	defer resp.Body.Close()
	for scan.Scan() {
		if bytes.Contains(scan.Bytes(), toFind) {
			return true
		}
	}
	return false
}

func CheckURL(i, total int, url string) {
	defer wg.Done()
	fmt.Printf("\033[1K\r\033[31m[\033[33m%d\033[36m/\033[33m%d \033[36m(\033[32m%d\033[36m)\033[31m] \033[35m%s\033[0m", i, total, success, url)
	var resp *http.Response
	var err error
	if proxy == "" {
		client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
		if insecure {
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client = &http.Client{Transport: tr}
		}
		resp, err = client.Get("https://" + url + "/.git/")
	} else {
		proxyURL, _ := URL.Parse(proxy)
		transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		if insecure {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
		client := &http.Client{Transport: transport, Timeout: time.Duration(timeout) * time.Second}
		resp, err = client.Get("https://" + url + "/.git/")
	}
	if err != nil {
		resp.Body.Close()
		<-thread
		return
	}
	if resp.StatusCode == 200 {
		isGit := Verify(resp)
		if isGit {
			success++
			mu.Lock()
			WriteToFile(resp.Request.URL.String())
			fmt.Printf("\033[1K\rGIT FOUND: " + resp.Request.URL.String() + "\n")
			mu.Unlock()

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

func ReadTargets(input string) {
	readFile, err := os.Open(input)
	defer readFile.Close()
	if err != nil {
		fmt.Println(err)
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	i := 0
	total := LineNBR(input)
	for fileScanner.Scan() {
		thread <- struct{}{}
		i++
		wg.Add(1)
		go CheckURL(i, total, fileScanner.Text())
	}
	wg.Wait()
}

func main() {
	var threads, input, time string
	var printversion bool
	op := optionparser.NewOptionParser()
	op.Banner = "Find exposed git repos\n\nUsage:\n"
	op.On("-t", "--thread NBR", "Number of threads (default 50)", &threads)
	op.On("-o", "--output FILE", "Output file (default found_git.txt)", &output)
	op.On("-i", "--input FILE", "Input file", &input)
	op.On("-I", "--insecure", "Ignore certificate errors", &insecure)
	op.On("-t", "--timeout SEC", "Set timeout (default 5s)", &time)
	op.On("-p", "--proxy PROXY", "Use proxy (proto://ip:port)", &proxy)
	op.On("-V", "--version", "Print version and exit", &printversion)
	op.Exemple("gitXpoz -i top-alexa.txt")
	op.Exemple("gitXpoz -p socks5://127.0.0.1:9050 -o good.txt -i top-alexa.txt -t 60")
	op.Parse()
	op.Logo("gitXpoz", "smslant", false)

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

	os.Setenv("GODEBUG", "http2client=0")
	ReadTargets(input)
}
