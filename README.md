# go-eurofxref-daily
Euro foreign exchange reference rates

## Installation
```
go get github.com/mrhdias/go-eurofxref-daily
```
## Example
```go
package main

import (
	"fmt"
	"log"

	eurofxref "github.com/mrhdias/go-eurofxref-daily"
)

func main() {
	cacheDir := "./eurofxref_cache"
	service := eurofxref.New(
		cacheDir, // Cache directory
		true,     // Create the cache directory if not exists
	)

	result, err := service.Query("USD")
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(result.LastUpdate, result.RateValue)
}

```
