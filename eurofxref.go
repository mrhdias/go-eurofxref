//
// Copyright 2023 The GoEurofxref Authors. All rights reserved.
// Use of this source code is governed by a MIT License
// license that can be found in the LICENSE file.
// Last Modification: 2023-05-17 21:00:07
//
// References:
// https://www.ecb.europa.eu/stats/policy_and_exchange_rates/euro_reference_exchange_rates/html/index.en.html
// https://www.golangprograms.com/golang-program-to-read-xml-file-into-struct.html
// https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml
// https://github.com/miku/zek
// https://xml-to-go.github.io/
//

package eurofxref

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type void struct{}

type EuroFxRef struct {
	Url            string
	Timeout        int
	CacheDir       string
	CreateCacheDir bool
	Currencies     map[string]void
	Debug          bool
}

type QueryResult struct {
	LastUpdate time.Time
	RateValue  float64
}

func (efr EuroFxRef) ValidateCurrencyCode(currencyCode string) error {

	if currencyCode == "" {
		return errors.New("no currency code specified")
	}

	if len(currencyCode) != 3 {
		return fmt.Errorf("the \"%s\" currency code has a wrong number of characters",
			currencyCode)
	}

	cc := strings.ToUpper(currencyCode)
	if _, ok := efr.Currencies[cc]; !ok {
		if strings.EqualFold(cc, "EUR") {
			return errors.New("all currencies quoted against the euro (base currency)")
		}
		return fmt.Errorf("the currency code \"%s\" is not part of the reference list",
			currencyCode)
	}

	return nil
}

func (efr EuroFxRef) Daily(currencyCode string) (*QueryResult, error) {

	if err := efr.ValidateCurrencyCode(currencyCode); err != nil {
		if strings.EqualFold(strings.ToUpper(currencyCode), "EUR") {
			return &QueryResult{
				LastUpdate: time.Now(),
				RateValue:  1.00,
			}, nil
		}

		return nil, err
	}

	req, err := http.NewRequest("GET", efr.Url, nil)
	// req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", userAgent, version))

	if err != nil {
		// log.Fatalf("[Fatal] %v\r\n", err)
		return nil, fmt.Errorf("client could not create request: %v", err)
	}

	xmlFilename := path.Base(req.URL.Path)
	xmlFilePath := filepath.Join(efr.CacheDir, xmlFilename)
	// fmt.Println(xmlFilePath)

	getFromCache, err := func() (bool, error) {
		if efr.CacheDir == "" {
			return false, nil
		}

		// create the cache directory if it does not exist
		if _, err := os.Stat(efr.CacheDir); errors.Is(err, os.ErrNotExist) {
			if efr.CreateCacheDir {
				if err := os.Mkdir(efr.CacheDir, os.ModePerm); err != nil {
					return false, fmt.Errorf("error creating cache directory: %v", err)
				}
			}
			return false, nil
		}

		if fileStat, err := os.Stat(xmlFilePath); err == nil {
			// fmt.Println(fileStat.ModTime())
			if (fileStat.ModTime().Local().Day() != time.Now().Local().Day()) || (fileStat.Size() == 0) {
				if err := os.Remove(xmlFilePath); err != nil {
					return false, fmt.Errorf("error removing cached xml file: %v", err)
				}
				return false, nil
			}
			return true, nil
		}
		return false, nil
	}()
	if err != nil {
		return nil, err
	}

	// fmt.Println("GetFromCache:", xmlFilePath, getFromCache)

	contentBytes, err := func() ([]byte, error) {
		if getFromCache {
			data, err := os.ReadFile(xmlFilePath)
			if err != nil {
				return nil, fmt.Errorf("error reading the cached xml file: %v", err)
			}
			return data, nil
		}

		client := &http.Client{
			Timeout: time.Duration(time.Duration(efr.Timeout).Seconds()),
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making http request: %v", err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("the request get \"%s\" returned an error with status code %d",
				efr.Url, resp.StatusCode)
		}

		respContentBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("client could not read response body: %v", err)
		}

		if efr.CacheDir != "" {
			if err := os.WriteFile(xmlFilePath, respContentBytes, 0644); err != nil {
				return nil, fmt.Errorf("error writing the cached xml file: %v", err)
			}
		}

		return respContentBytes, nil
	}()
	if err != nil {
		return nil, err
	}

	if efr.Debug {
		fmt.Println(string(contentBytes))
	}

	type CubeElement struct {
		Text     string `xml:",chardata"`
		Currency string `xml:"currency,attr"`
		Rate     string `xml:"rate,attr"`
	}

	type Envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Text    string   `xml:",chardata"`
		Gesmes  string   `xml:"gesmes,attr"`
		Xmlns   string   `xml:"xmlns,attr"`
		Subject string   `xml:"subject"`
		Sender  struct {
			Text string `xml:",chardata"`
			Name string `xml:"name"`
		} `xml:"Sender"`
		Cube struct {
			Text string `xml:",chardata"`
			Cube struct {
				Text string        `xml:",chardata"`
				Time string        `xml:"time,attr"`
				Cube []CubeElement `xml:"Cube"`
			} `xml:"Cube"`
		} `xml:"Cube"`
	}

	var envelope Envelope

	if err := xml.Unmarshal(contentBytes, &envelope); err != nil {
		return nil, fmt.Errorf("error when unmarshal parses the XML-encoded data: %v", err)
	}

	// fmt.Println(envelope.Cube.Cube.Time)

	for _, rate := range envelope.Cube.Cube.Cube {
		if strings.EqualFold(rate.Currency, strings.ToUpper(currencyCode)) {
			rateValue, err := strconv.ParseFloat(rate.Rate, 64)
			if err != nil {
				return nil, fmt.Errorf("error when convert rate string from envelope to float: %v", err)
			}

			cubeTime, err := time.Parse("2006-01-02", envelope.Cube.Cube.Time)
			if err != nil {
				return nil, fmt.Errorf("error when convert time string from envelope to float: %v", err)
			}

			return &QueryResult{
				LastUpdate: cubeTime,
				RateValue:  rateValue,
			}, nil
		}
	}

	return nil, fmt.Errorf("no conversion rate value was returned for \"%s\" currency code",
		currencyCode)
}

func New(
	cacheDir string,
	createCacheDir bool,
	debugOption ...bool) EuroFxRef {

	debug := false
	if len(debugOption) == 1 {
		debug = debugOption[0]
	}

	eurofxref := new(EuroFxRef)

	eurofxref.Currencies = map[string]void{
		"USD": {}, "JPY": {}, "BGN": {}, "CZK": {}, "DKK": {},
		"GBP": {}, "HUF": {}, "PLN": {}, "RON": {}, "SEK": {},
		"CHF": {}, "ISK": {}, "NOK": {}, "TRY": {}, "AUD": {},
		"BRL": {}, "CAD": {}, "CNY": {}, "HKD": {}, "IDR": {},
		"ILS": {}, "INR": {}, "KRW": {}, "MXN": {}, "MYR": {},
		"NZD": {}, "PHP": {}, "SGD": {}, "THB": {}, "ZAR": {},
	}

	eurofxref.Url = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
	eurofxref.Timeout = 60
	// cache xml file only 24 hours
	eurofxref.CacheDir = cacheDir
	eurofxref.CreateCacheDir = createCacheDir
	eurofxref.Debug = debug

	return *eurofxref
}
