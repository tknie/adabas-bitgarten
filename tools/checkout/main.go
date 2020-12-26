/*
* Copyright © 2018-2019 private, Darmstadt, Germany and/or its licensors
*
* SPDX-License-Identifier: Apache-2.0
*
*   Licensed under the Apache License, Version 2.0 (the "License");
*   you may not use this file except in compliance with the License.
*   You may obtain a copy of the License at
*
*       http://www.apache.org/licenses/LICENSE-2.0
*
*   Unless required by applicable law or agreed to in writing, software
*   distributed under the License is distributed on an "AS IS" BASIS,
*   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*   See the License for the specific language governing permissions and
*   limitations under the License.
*
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/SoftwareAG/adabas-go-api/adabas"
	"github.com/SoftwareAG/adabas-go-api/adatypes"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var hostname string

type checker struct {
	conn            *adabas.Connection
	adabas          *adabas.Adabas
	limit           uint64
	deleteDuplikate bool
}

func init() {
	hostname, _ = os.Hostname()
	level := zapcore.ErrorLevel
	ed := os.Getenv("ENABLE_DEBUG")
	switch ed {
	case "1":
		level = zapcore.DebugLevel
		adatypes.Central.SetDebugLevel(true)
	case "2":
		level = zapcore.InfoLevel
	}

	err := initLogLevelWithFile("checker.log", level)
	if err != nil {
		fmt.Println("Error initialize logging")
		os.Exit(255)
	}
}

func initLogLevelWithFile(fileName string, level zapcore.Level) (err error) {
	p := os.Getenv("LOGPATH")
	if p == "" {
		p = "."
	}
	name := p + string(os.PathSeparator) + fileName

	rawJSON := []byte(`{
		"level": "error",
		"encoding": "console",
		"outputPaths": [ "loadpicture.log"],
		"errorOutputPaths": ["stderr"],
		"encoderConfig": {
		  "messageKey": "message",
		  "levelKey": "level",
		  "levelEncoder": "lowercase"
		}
	  }`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		fmt.Println("Error initialize logging (json)")
		os.Exit(255)
	}
	cfg.Level.SetLevel(level)
	cfg.OutputPaths = []string{name}
	logger, err := cfg.Build()
	if err != nil {
		fmt.Println("Error initialize logging (build)")
		os.Exit(255)
	}
	defer logger.Sync()

	sugar := logger.Sugar()

	sugar.Infof("Start logging with level", level)
	adatypes.Central.Log = sugar

	return
}

func main() {
	var dbidParameter string
	var mapFnrParameter int
	var limit int
	var delete bool
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

	flag.StringVar(&dbidParameter, "d", "23", "Map repository Database id")
	flag.IntVar(&mapFnrParameter, "f", 4, "Map repository file number")
	flag.IntVar(&limit, "l", 10, "Maximum records to read (0 is all)")
	flag.BoolVar(&delete, "D", false, "Delete duplicate entries")
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic("could not create CPU profile: " + err.Error())
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			panic("could not start CPU profile: " + err.Error())
		}
		defer pprof.StopCPUProfile()
	}
	defer writeMemProfile(*memprofile)

	// if  {
	// 	fmt.Println("File name option is required")
	// 	flag.Usage()
	// 	return
	// }
	fmt.Printf("Connect to map repository %s/%d\n", dbidParameter, mapFnrParameter)

	id := adabas.NewAdabasID()
	a, err := adabas.NewAdabas(dbidParameter, id)
	if err != nil {
		fmt.Println("Adabas target generation error", err)
		return
	}
	adabas.AddGlobalMapRepository(a.URL, adabas.Fnr(mapFnrParameter))
	defer adabas.DelGlobalMapRepository(a.URL, adabas.Fnr(mapFnrParameter))
	c := &checker{adabas: a, limit: uint64(limit), deleteDuplikate: delete}
	err = c.analyzeDoublikats()
	if err != nil {
		fmt.Println("Error anaylzing douplikats", err)
	}
}

func (checker *checker) analyzeDoublikats() (err error) {
	checker.conn, err = adabas.NewConnection("acj;map")
	if err != nil {
		return err
	}
	readCheck, rerr := checker.conn.CreateMapReadRequest("PictureData")
	if rerr != nil {
		checker.conn.Close()
		return rerr
	}
	readCheck.Limit = 0
	rerr = readCheck.QueryFields("ChecksumPicture")
	if rerr != nil {
		checker.conn.Close()
		return rerr
	}
	cursor, err := readCheck.HistogramByCursoring("ChecksumPicture")
	if err != nil {
		fmt.Printf("Error checking descriptor quantity for ChecksumPicture: %v\n", err)
		panic("Read error " + err.Error())
	}
	counter := uint64(0)
	for cursor.HasNextRecord() && (checker.limit == 0 || counter < checker.limit) {
		record, recErr := cursor.NextRecord()
		if recErr != nil {
			panic("Read error " + recErr.Error())
		}
		counter++
		fmt.Printf("quantity=%03d -> %s\n", record.Quantity, record.HashFields["ChecksumPicture"])
		err = checker.listDuplikats(record.HashFields["ChecksumPicture"].String())
		if err != nil {
			return err
		}
	}
	fmt.Printf("There are %06d records\n", counter)
	return nil
}

func (checker *checker) listDuplikats(checksum string) error {
	readCheck, rerr := checker.conn.CreateMapReadRequest("PictureMetadata")
	if rerr != nil {
		checker.conn.Close()
		return rerr
	}
	rerr = readCheck.QueryFields("PictureName,Option")
	if rerr != nil {
		checker.conn.Close()
		return rerr
	}
	cursor, err := readCheck.ReadLogicalWithCursoring("ChecksumPicture=" + checksum)
	if err != nil {
		fmt.Printf("Error checking descriptor quantity for ChecksumPicture: %v\n", err)
		panic("Read error " + err.Error())
	}
	first := true
	for cursor.HasNextRecord() {
		record, recErr := cursor.NextRecord()
		if recErr != nil {
			panic("Read error " + recErr.Error())
		}
		currentOption := strings.Trim(record.HashFields["Option"].String(), " ")
		if first {
			switch currentOption {
			case "":
				err = checker.updateOption(record, "original")
			case "original":
			default:
				fmt.Println(currentOption, "should be original")
			}
			first = false
		} else {
			switch currentOption {
			case "":
				err = checker.updateOption(record, "duplicate")
			case "duplicate":
			default:
				fmt.Println(currentOption, "should be original")
			}
		}
		if err != nil {
			return err
		}
		fmt.Printf("  ISN=%06d %s -> %s\n", record.Isn, record.HashFields["PictureName"].String(), record.HashFields["Option"])
	}
	return checker.conn.EndTransaction()
}

func (checker *checker) updateOption(record *adabas.Record, option string) error {
	fmt.Println(record.HashFields["PictureName"], record.HashFields["Option"], option)
	vErr := record.SetValue("Option", option)
	if vErr != nil {
		return vErr
	}
	sReq, err := checker.conn.CreateMapStoreRequest("PictureMetadata")
	if err != nil {
		return err
	}
	err = sReq.Store(record)
	if err != nil {
		return err
	}
	err = sReq.EndTransaction()
	return err
}

func writeMemProfile(file string) {
	if file != "" {
		f, err := os.Create(file)
		if err != nil {
			panic("could not create memory profile: " + err.Error())
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			panic("could not write memory profile: " + err.Error())
		}
		defer f.Close()
		fmt.Println("Memory profile written")
	}

}
