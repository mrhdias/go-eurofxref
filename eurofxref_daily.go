//
// Copyright 2023 The GoEurofxrefDaily Authors. All rights reserved.
// Use of this source code is governed by a MIT License
// license that can be found in the LICENSE file.
// Last Modification: 2023-05-16 22:11:03
//
// References:
// https://www.ecb.europa.eu/stats/policy_and_exchange_rates/euro_reference_exchange_rates/html/index.en.html
// https://www.golangprograms.com/golang-program-to-read-xml-file-into-struct.html
// https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml
// https://github.com/miku/zek
// https://xml-to-go.github.io/
//

package eurofxref_daily

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
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

func (efr EuroFxRef) Query(currencyCode string) (*QueryResult, error) {

	if currencyCode == "" {
		return nil, errors.New("no currency code specified")
	}

	if len(currencyCode) != 3 {
		return nil, fmt.Errorf("the \"%s\" currency code is wrong",
			currencyCode)
	}

	cc := strings.ToUpper(currencyCode)

	if _, ok := efr.Currencies[cc]; !ok {
		return nil, fmt.Errorf("the currency code \"%s\" is not allowed",
			currencyCode)
	}

	req, err := http.NewRequest("GET", efr.Url, nil)
	// req.Header.Add("User-Agent", fmt.Sprintf("%s/%s", userAgent, version))

	if err != nil {
		log.Fatalf("[Fatal] %v\r\n", err)
	}

	xmlFilename := path.Base(req.URL.Path)
	xmlFilePath := filepath.Join(efr.CacheDir, xmlFilename)
	// fmt.Println(xmlFilePath)

	getFromCache := func() bool {
		if efr.CacheDir == "" {
			return false
		}
		// create the cache directory if it does not exist
		if _, err := os.Stat(efr.CacheDir); errors.Is(err, os.ErrNotExist) {
			if efr.CreateCacheDir {
				if err := os.Mkdir(efr.CacheDir, os.ModePerm); err != nil {
					log.Fatalf("[Fatal] %v\r\n", err)
				}
			}
			return false
		}

		if fileStat, err := os.Stat(xmlFilePath); err == nil {
			// fmt.Println(fileStat.ModTime())
			if (fileStat.ModTime().Local().Day() != time.Now().Local().Day()) || (fileStat.Size() == 0) {
				if err := os.Remove(xmlFilePath); err != nil {
					log.Fatalf("[Fatal] %v\r\n", err)
				}
				return false
			}
			return true
		}
		return false
	}()

	// fmt.Println("GetFromCache:", xmlFilePath, getFromCache)

	contentBytes := func() []byte {
		if getFromCache {
			data, err := os.ReadFile(xmlFilePath)
			if err != nil {
				log.Fatalf("[Fatal] %v\r\n", err)
			}
			return data
		}

		client := &http.Client{
			Timeout: time.Duration(time.Duration(efr.Timeout).Seconds()),
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("[Fatal] %v\r\n", err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Fatalf("[Fatal] The request get \"%s\" returned an error with status code %d\r\n",
				efr.Url, resp.StatusCode)
		}

		respContentBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("[Fatal] %v\r\n", err)
		}

		if efr.CacheDir != "" {
			if err := os.WriteFile(xmlFilePath, respContentBytes, 0644); err != nil {
				log.Fatalf("[Fatal] %v\r\n", err)
			}
		}

		return respContentBytes
	}()

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
		log.Fatalf("[Fatal] %v\n", err)
	}

	// fmt.Println(envelope.Cube.Cube.Time)

	for _, rate := range envelope.Cube.Cube.Cube {
		if strings.EqualFold(rate.Currency, cc) {
			rateValue, err := strconv.ParseFloat(rate.Rate, 64)
			if err != nil {
				log.Fatalf("[Fatal] %v\n", err)
			}

			cubeTime, err := time.Parse("2006-01-02", envelope.Cube.Cube.Time)
			if err != nil {
				log.Fatalf("[Fatal] %v\n", err)
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

func NewEuroFxRefDailyService(
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
