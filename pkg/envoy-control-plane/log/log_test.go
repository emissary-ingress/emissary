// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package log

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ExampleLoggerFuncs() {
	logger := log.Logger{}

	xdsLogger := LoggerFuncs{
		DebugFunc: logger.Printf,
		InfoFunc:  logger.Printf,
		WarnFunc:  logger.Printf,
		ErrorFunc: logger.Printf,
	}

	xdsLogger.Debugf("debug")
	xdsLogger.Infof("info")
	xdsLogger.Warnf("warn")
	xdsLogger.Errorf("error")
}

func TestLoggerFuncs(t *testing.T) {
	debug := 0
	info := 0
	warn := 0
	error := 0

	xdsLogger := LoggerFuncs{
		DebugFunc: func(string, ...interface{}) { debug++ },
		InfoFunc:  func(string, ...interface{}) { info++ },
		WarnFunc:  func(string, ...interface{}) { warn++ },
		ErrorFunc: func(string, ...interface{}) { error++ },
	}

	xdsLogger.Debugf("debug")
	xdsLogger.Infof("info")
	xdsLogger.Warnf("warn")
	xdsLogger.Errorf("error")

	assert.Equal(t, debug, 1)
	assert.Equal(t, info, 1)
	assert.Equal(t, warn, 1)
	assert.Equal(t, error, 1)
}

func TestNilLoggerFuncs(t *testing.T) {
	xdsLogger := LoggerFuncs{}

	// Just verifying that nothing panics.
	xdsLogger.Debugf("debug")
	xdsLogger.Infof("info")
	xdsLogger.Warnf("warn")
	xdsLogger.Errorf("error")
}
