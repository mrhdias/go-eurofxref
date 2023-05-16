//
// Copyright 2023 The GoEurofxrefDaily Authors. All rights reserved.
// Use of this source code is governed by a MIT License
// license that can be found in the LICENSE file.
// Last Modification: 2023-05-16 19:35:29
//

package eurofxref_daily

import (
	"os"
	"testing"
)

func TestEuroFxRefDaily(t *testing.T) {

	cacheDir := "./eurofxref_cache"
	service := New(cacheDir, true)

	if _, err := service.Query("USD"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		if err = os.RemoveAll(cacheDir); err != nil {
			t.Fatal(err)
		}
	}

}
