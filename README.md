# go-eurofxref
Euro foreign exchange reference rates

## Installation
```
go get github.com/mrhdias/go-eurofxref
```
## Example
```go
package main

import (
	"fmt"
	"log"

	eurofxref "github.com/mrhdias/go-eurofxref"
)

func main() {

	cacheDir := "./eurofxref_cache"

	query := eurofxref.New(
		cacheDir, // Cache directory
		true,     // Create the cache directory if not exists
	)

	if err := query.ValidateCurrencyCode("USD"); err != nil {
		log.Fatalln(err)
	}

	result, err := query.Daily("USD")
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(result.LastUpdate, result.RateValue)
}
```
