package main

import (
	"errors"
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
	timeout    = 10 * time.Second
	servername = "服务器的地址"
	remote     = "www.google.com"
	scheme     = "http"
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
	"https": "http",
	"([0-9A-Za-z.-]+\\.gstatic\\.com)":       servername + "/!$1",
	"((apis)\\.google\\.com)":                servername + "/!$1",
	"((www)|(images))\\.google\\.[0-9a-z.]+": servername,
	"(img\\.youtube\\.com)":                  servername + "/!$1"}
var filter *Filter
var NotFollowRedirect error

func proxy(w http.ResponseWriter, req *http.Request) {
	var u *url.URL

	if strings.HasPrefix(req.URL.RequestURI(), "/!") {
		u, _ = url.Parse(scheme + "://" + req.URL.RequestURI()[2:])
	} else {
		u, _ = url.Parse(scheme + "://" + remote + req.URL.RequestURI())
	}
	println(u.String())
	reqHead := make(http.Header)
	reqHead.Add("Accept:text/html", "application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	//TODO:Implement Gzip Deflate...
	reqHead.Add("Accept-Encoding", "")
	reqHead.Add("Accept-Language", "en-US")
	reqHead.Add("Cache-Control", "no-cache")
	reqHead.Add("Pragma", "no-cache")
	reqHead.Add("User-Agent", req.Header.Get("User-Agent"))
	reqHead.Add("Cookie", "NID=76=KJwxX-SN2oOpNEkWTjdLvPLOp-sIhTAxIG-aOBIcxDqAxmjmNmg1q4Wfw_chL0GUSTeaRLfEbu6RqCvVOOq5-RZHmSQaRNOSBaHUjVYTwK_dqnYnTDXWmyrEW24aJrrz; expires=Sun, 14-Aug-2026 03:37:23 GMT; path=/; domain=.google.com; HttpOnly")
	getgoogle, _ := client.Do(&http.Request{
		URL:              u,
		Body:             req.Body,
		Close:            false,
		TransferEncoding: []string{"chunked"},
		Header:           reqHead,
		Method:           req.Method})

	defer getgoogle.Body.Close()
	if getgoogle.StatusCode == 302 || getgoogle.StatusCode == 301 {
		uu, _ := url.Parse(getgoogle.Header.Get("Location"))
		uu.Scheme = "http"
		uu.Host = filter.Replace(uu.Host)
		w.Header().Add("Location", uu.String())
	}
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("Content-Type", getgoogle.Header.Get("Content-Type"))
	w.WriteHeader(getgoogle.StatusCode)

	str, _ := ioutil.ReadAll(getgoogle.Body)
	fmt.Fprint(w, filter.Replace(string(str)))

}

func main() {
	NotFollowRedirect = errors.New("Not Follow Redirect")

	client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return NotFollowRedirect },
		Timeout:       timeout}
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
