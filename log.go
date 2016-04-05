// Copyright 2016 Tim Shannon. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import "log"

func vlog(format string, v ...interface{}) {
	if verbose {
		log.Printf(format, v...)
	}
}
