package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	timeout    = 30 * time.Second
	servername = "服务器地址"
	remote     = "http://www.google.com"
	gzip_level = 4
)

type Filter struct {
	Rules       []regexp.Regexp
	Replacement []string
	Count       int
}

func GetFilter(rules map[string]string) *Filter {
	ru := make([]regexp.Regexp, len(rules))
	rp := make([]string, len(rules))
	i := 0
	for k, v := range rules {
		ru[i] = *regexp.MustCompile(k)
		rp[i] = v
		i++
	}
	return &Filter{Rules: ru, Replacement: rp, Count: i}
}

func (this *Filter) Replace(str string) string {
	for i := 0; i < this.Count; i++ {
		str = this.Rules[i].ReplaceAllString(str, this.Replacement[i])
	}
	return str
}

var client *http.Client
var rules map[string]string = map[string]string{
	"([0-9A-Za-z.-]+\\.gstatic\\.com)":           servername + "/!$1",
	"((apis)\\.google\\.com)":                    servername + "/!$1",
	"((www)|(images))\\.google\\.[0-9a-z.]+":     servername,
	"scholar\\.google\\.[0-9a-z.]+\\\\?/scholar": servername + "/scholar",
	"scholar\\.google\\.[0-9a-z.]+":              servername + "/scholar",
	"(img\\.youtube\\.com)":                        servername + "/!$1"}
var filter *Filter
var subDomainRegexp *regexp.Regexp = regexp.MustCompile("!([^/]+)/")

func proxy(w http.ResponseWriter, req *http.Request) {
	u, _ := url.Parse(remote + req.URL.RequestURI())

	if subDomainRegexp.MatchString(req.URL.RequestURI()) {
		u, _ = url.Parse(req.URL.RequestURI()[2:])
	}
	println(u.String())
	//TODO:Implement Gzip Deflate...
	req.Header.Set("Accept-Encoding", "")
	req.Header.Set("X-Forwarded-For", req.RemoteAddr)
	req.Header.Set("X-Real-Ip", req.RemoteAddr)
	req.Header.Set("Accept-Language", "en-US;q=0.6,en;q=0.4")
	getgoogle, err := client.Do(&http.Request{
		URL:              u,
		Body:             req.Body,
		Close:            req.Close,
		TransferEncoding: req.TransferEncoding,
		Header:           req.Header,
		Method:           req.Method})
	resHead := w.Header()

	if err != nil {
		return
	}
	for key, vals := range getgoogle.Header {
		for _, val := range vals {
			resHead.Add(key, strings.Replace(val, "google.com", servername, 0))
		}
	}
	defer getgoogle.Body.Close()
	str, _ := ioutil.ReadAll(getgoogle.Body)

	fmt.Fprint(w, filter.Replace(string(str)))

}

func main() {

	client = &http.Client{
		Timeout: timeout}
	server := http.NewServeMux()
	filter = GetFilter(rules)

	//状态信息
	server.HandleFunc("/status", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Number Of Goroutines: %d", runtime.NumGoroutine())
	})
	//Proxy
	server.HandleFunc("/", proxy)

	http.TimeoutHandler(server, timeout, "Timeout")

	panic(http.ListenAndServe(":3000", server))
}
