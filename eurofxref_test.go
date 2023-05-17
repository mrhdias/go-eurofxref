//
// Copyright 2023 The GoEurofxref Authors. All rights reserved.
// Use of this source code is governed by a MIT License
// license that can be found in the LICENSE file.
// Last Modification: 2023-05-17 22:15:25
//

package eurofxref

import (
	"os"
	"testing"
)

func TestEuroFxRef(t *testing.T) {

	cacheDir := "./eurofxref_cache"
	query := New(cacheDir, true)

	if err := query.ValidateCurrencyCode("USD"); err != nil {
		t.Fatal(err)
	}

	if _, err := query.Daily("USD"); err != nil {
		t.Fatal(err)
	}

	want := 1.00
	got, err := query.Daily("EUR")
	if err != nil {
		t.Fatal(err)
	}
	if got.RateValue != want {
		t.Errorf("got = %.2f, want %.2f", got.RateValue, want)
	}

	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		if err = os.RemoveAll(cacheDir); err != nil {
			t.Fatal(err)
		}
	}

}
