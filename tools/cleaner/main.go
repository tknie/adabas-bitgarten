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
	"regexp"
	"runtime"
	"runtime/pprof"

	"github.com/SoftwareAG/adabas-go-api/adabas"
	"github.com/SoftwareAG/adabas-go-api/adatypes"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var hostname string

type deleter struct {
	re            *regexp.Regexp
	deleteRequest *adabas.DeleteRequest
	test          bool
	counter       uint64
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

	err := initLogLevelWithFile("cleaner.log", level)
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
	var test bool
	var query string
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

	flag.StringVar(&dbidParameter, "d", "23", "Map repository Database id")
	flag.IntVar(&mapFnrParameter, "f", 4, "Map repository file number")
	flag.IntVar(&limit, "l", 10, "Maximum records to read (0 is all)")
	flag.BoolVar(&test, "t", false, "Dry run, don't change")
	flag.StringVar(&query, "q", "", "Filter for")
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

	fmt.Printf("Connect to map repository %s/%d\n", dbidParameter, mapFnrParameter)
	d := &deleter{test: test}
	re, err := regexp.Compile(query)
	if err != nil {
		fmt.Println("Query error regexp:", err)
		return
	}
	d.re = re
	id := adabas.NewAdabasID()
	a, err := adabas.NewAdabas(dbidParameter, id)
	if err != nil {
		fmt.Println("Adabas target generation error", err)
		return
	}
	adabas.AddGlobalMapRepository(a.URL, adabas.Fnr(mapFnrParameter))
	defer adabas.DelGlobalMapRepository(a.URL, adabas.Fnr(mapFnrParameter))

	err = removeQueries(a, d, uint64(limit))
	if err != nil {
		fmt.Println("Error anaylzing douplikats", err)
	}
}

func removeQuery(record *adabas.Record, x interface{}) error {
	fn := record.HashFields["PictureName"].String()
	de := x.(*deleter)
	found := de.re.MatchString(fn)
	if found {
		fmt.Println(record.HashFields["PictureName"].String())
		if !de.test {
			de.deleteRequest.Delete(record.Isn)
			de.counter++
			if de.counter%100 == 0 {
				err := de.deleteRequest.EndTransaction()
				return err
			}
		}
	}
	return nil
}

func removeQueries(a *adabas.Adabas, de *deleter, limit uint64) error {
	conn, err := adabas.NewConnection("acj;map")
	if err != nil {
		return err
	}
	defer conn.Close()
	readCheck, rerr := conn.CreateMapReadRequest("PictureMetadata")
	if rerr != nil {
		conn.Close()
		return rerr
	}
	de.deleteRequest, err = conn.CreateMapDeleteRequest("PictureMetadata")
	if err != nil {
		conn.Close()
		return err
	}
	readCheck.Limit = limit
	rerr = readCheck.QueryFields("PictureName")
	if rerr != nil {
		conn.Close()
		return rerr
	}
	_, err = readCheck.ReadPhysicalSequenceStream(removeQuery, de)
	if err != nil {
		fmt.Printf("Error checking descriptor quantity for ChecksumPicture: %v\n", err)
		de.deleteRequest.BackoutTransaction()
		panic("Read error " + err.Error())
	}
	err = de.deleteRequest.EndTransaction()
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
