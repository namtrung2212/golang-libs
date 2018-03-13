package Recaptcha

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var httpClient *http.Client

var lock sync.RWMutex

// InitializeHTTPClient ...
func InitializeHTTPClient(timeoutInSec int) {
	lock.Lock()
	defer lock.Unlock()

	httpClient = &http.Client{
		Timeout: time.Duration(timeoutInSec) * time.Second,
	}
}

type recaptchaResp struct {
	Success  bool
	Hostname string
}

// VerifyCaptcha verify google recaptcha
func VerifyCaptcha(secret, captcha string) error {
	lock.RLock()
	defer lock.RUnlock()

	if httpClient == nil {
		lock.RUnlock()
		InitializeHTTPClient(5)
		lock.RLock()
		defer lock.RUnlock()
	}

	data, err := httpPost(httpClient, "https://www.google.com/recaptcha/api/siteverify", url.Values{
		// "secret":   []string{""},
		"secret":   []string{secret},
		"response": []string{captcha},
	})
	if err != nil {
		return err
	}

	rep := recaptchaResp{}
	if err = json.Unmarshal(data, &rep); err != nil {
		return err
	}

	if rep.Success {
		return nil
	}

	return fmt.Errorf("Malform captcha")
}

// httpPost do http post with timeout
func httpPost(client *http.Client, uri string, data url.Values) (resp []byte, err error) {
	if data == nil || client == nil {
		return nil, nil
	}

	defer func() {
		if er := recover(); er != nil {
			err = errors.New("Client.Timeout")
			resp = nil
		}
	}()

	res, err := client.PostForm(uri, data)
	defer func() {
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
	}()

	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(res.Body)
}
