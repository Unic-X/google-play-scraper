package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/k3a/html2text"
	"github.com/tidwall/gjson"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.6613.137 Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36",
}

var (
	scriptRegex = regexp.MustCompile(`>AF_initDataCallback[\s\S]*?<\/script`)
	keyRegex    = regexp.MustCompile(`(ds:\d*?)'`)
	valueRegex  = regexp.MustCompile(`data:([\s\S]*?), sideChannel: {}}\);<\/`)
)

// AbsoluteURL return absolute url
func AbsoluteURL(base, path string) (string, error) {
	p, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(p).String(), nil
}

// BatchExecute for PlayStoreUi
func BatchExecute(country, language, payload string) (string, error) {
	url := "https://play.google.com/_/PlayStoreUi/data/batchexecute"

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")

	q := req.URL.Query()
	q.Add("authuser", "0")
	q.Add("bl", "boq_playuiserver_20190424.04_p0")
	q.Add("gl", country)
	q.Add("hl", language)
	q.Add("soc-app", "121")
	q.Add("soc-platform", "1")
	q.Add("soc-device", "1")
	q.Add("rpcids", "qnKhOb")
	req.URL.RawQuery = q.Encode()

	body, err := DoRequest(req)
	if err != nil {
		return "", err
	}

	var js [][]interface{}
	err = json.Unmarshal(bytes.TrimLeft(body, ")]}'"), &js)
	if err != nil {
		return "", err
	}
	if len(js) < 1 || len(js[0]) < 2 {
		return "", fmt.Errorf("invalid size of the resulting array")
	}
	if js[0][2] == nil {
		return "", nil
	}

	return js[0][2].(string), nil
}

// ExtractInitData from Google HTML
func ExtractInitData(html []byte) map[string]string {
	data := make(map[string]string)
	scripts := scriptRegex.FindAll(html, -1)
	for _, script := range scripts {
		key := keyRegex.FindSubmatch(script)
		value := valueRegex.FindSubmatch(script)
		if len(key) > 1 && len(value) > 1 {
			data[string(key[1])] = string(value[1])
		}
	}
	return data
}

// GetJSONArray by path
func GetJSONArray(data string, paths ...string) []gjson.Result {
	for _, path := range paths {
		value := gjson.Get(data, path)
		if value.Exists() && value.Type != gjson.Null {
			return value.Array()
		}
	}
	return nil
}

// GetJSONValue with multiple path
func GetJSONValue(data string, paths ...string) string {
	for _, path := range paths {
		value := gjson.Get(data, path)
		if value.Exists() && value.Type != gjson.Null {
			return value.String()
		}
	}
	return ""
}

func DoRequest(req *http.Request) ([]byte, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	}

	time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request error: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// GetInitData from Google HTML
func GetInitData(req *http.Request) (map[string]string, error) {
	html, err := DoRequest(req)
	if err != nil {
		return nil, err
	}

	return ExtractInitData(html), nil
}

// HTMLToText return plain text from HTML
func HTMLToText(html string) string {
	html2text.SetUnixLbr(true)
	return html2text.HTML2Text(html)
}
