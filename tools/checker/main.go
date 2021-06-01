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
	"time"
	"tux-lobload/store"

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
	validateLob     bool
	picFnr          adabas.Fnr
}

var timeFormat = "2006-01-02 15:04:05"

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
	var picFnrParameter int
	var limit int
	var delete bool
	var validate bool
	var verify bool
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

	flag.StringVar(&dbidParameter, "d", "23", "Map repository Database id")
	flag.IntVar(&picFnrParameter, "p", 100, "Map repository file number")
	flag.IntVar(&limit, "l", 10, "Maximum records to read (0 is all)")
	flag.BoolVar(&delete, "D", false, "Delete duplicate entries")
	flag.BoolVar(&validate, "V", false, "Validate large object entries")
	flag.BoolVar(&verify, "c", false, "Verify image content")
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
	fmt.Printf("Connect to file at  %s/%d\n", dbidParameter, picFnrParameter)

	id := adabas.NewAdabasID()
	a, err := adabas.NewAdabas(dbidParameter, id)
	if err != nil {
		fmt.Println("Adabas target generation error", err)
		return
	}
	c := &checker{adabas: a, picFnr: adabas.Fnr(picFnrParameter), limit: uint64(limit), deleteDuplikate: delete, validateLob: validate}
	err = c.analyzeDoublikats()
	if err != nil {
		fmt.Println("Error anaylzing douplikats", err)
	}
	err = c.listDuplikats()
	if err != nil {
		fmt.Printf("List duplicate error: %v\n", err)
	}
	if verify {
		fmt.Printf("%s Start verifying database picture content\n", time.Now().Format(timeFormat))
		err := store.VerifyPicture("PictureData", fmt.Sprintf("%s,%d", dbidParameter, picFnrParameter))
		if err != nil {
			fmt.Printf("%s Error during verify of database picture content: %v\n", time.Now().Format(timeFormat), err)
			return
		}
		fmt.Printf("%s finished verify of database picture content\n", time.Now().Format(timeFormat))
	}

}

func (checker *checker) analyzeDoublikats() (err error) {
	checker.conn, err = adabas.NewConnection("acj;inmap=" + checker.adabas.URL.String())
	if err != nil {
		return err
	}
	readCheck, rerr := checker.conn.CreateFileReadRequest(checker.picFnr)
	if rerr != nil {
		checker.conn.Close()
		return rerr
	}
	readCheck.Limit = 0
	rerr = readCheck.QueryFields("CP")
	if rerr != nil {
		checker.conn.Close()
		return rerr
	}
	cursor, err := readCheck.HistogramByCursoring("CP")
	if err != nil {
		fmt.Printf("Error checking descriptor quantity for ChecksumPicture: %v\n", err)
		panic("Read error " + err.Error())
	}
	counter := uint64(0)
	dupli := uint64(0)
	for cursor.HasNextRecord() && (checker.limit == 0 || counter < checker.limit) {
		record, recErr := cursor.NextRecord()
		if recErr != nil {
			panic("Read error " + recErr.Error())
		}
		if checker.validateLob {
			checker.validateData(record.HashFields["CP"].String())
		}
		counter++
		if record.Quantity != 1 {
			fmt.Printf("quantity=%03d -> %s\n", record.Quantity, record.HashFields["ChecksumPicture"])
			dupli++
		}
	}
	fmt.Printf("There are %06d duplicate of %06d\n", dupli, counter)
	return nil
}

func (checker *checker) validateData(checksum string) error {
	readCheck, rerr := checker.conn.CreateMapReadRequest("PictureBinary")
	if rerr != nil {
		checker.conn.Close()
		return rerr
	}
	rerr = readCheck.QueryFields("ChecksumPicture,PictureName,Media")
	if rerr != nil {
		checker.conn.Close()
		return rerr
	}
	readCheck.Multifetch = 1
	adatypes.Central.Log.Debugf("Read checksums records")
	cursor, err := readCheck.ReadLogicalWithCursoring("ChecksumPicture=" + checksum)
	if err != nil {
		fmt.Printf("Error checking descriptor quantity for ChecksumPicture: %v\n", err)
		panic("Read error " + err.Error())
	}
	adatypes.Central.Log.Debugf("Called and get next record")
	for cursor.HasNextRecord() {
		record, recErr := cursor.NextRecord()
		if recErr != nil {
			panic("Read error " + recErr.Error())
		}
		mv := record.HashFields["Media"]
		adatypes.Central.Log.Debugf("Length %d %#v", mv.Type().Length(), mv)
		fmt.Println("Length", record.HashFields["Media"].Type().Length())
		data := record.HashFields["Media"].Bytes()
		if len(data) == 0 {
			fmt.Printf("Empty data %s for ChecksumPicture\n", record.HashFields["PictureName"].String())
			panic("Empty media error " + record.HashFields["PictureName"].String())
		}
		fmt.Printf("  ISN=%06d %s\n", record.Isn, record.HashFields["PictureName"].String())

	}
	return nil
}

func (checker *checker) listDuplikats() error {
	readCheck, rerr := checker.conn.CreateMapReadRequest((*store.PictureMetadata)(nil), int(checker.picFnr))
	if rerr != nil {
		checker.conn.Close()
		return rerr
	}
	readCheck.Limit = 0
	rerr = readCheck.QueryFields("#PL,PL")
	if rerr != nil {
		checker.conn.Close()
		return rerr
	}
	var isnList []adatypes.Isn
	cursor, err := readCheck.ReadLogicalWithCursoring("CP>=' '")
	if err != nil {
		fmt.Printf("Error checking descriptor quantity for ChecksumPicture: %v\n", err)
		panic("Read error " + err.Error())
	}
	fmt.Println("Start checking ...")
	quantityStatistics := make(map[int]int)
	lenStatistics := make(map[int]int)
	counter := 0
	for cursor.HasNextRecord() {
		if counter%10000 == 0 {
			fmt.Printf("Working on %d\r", counter)
		}
		counter++
		data, err := cursor.NextData()
		if err != nil {
			return err
		}
		picData := data.(*store.PictureMetadata)

		checkDuplicate := make(map[string]int)
		for _, p := range picData.PictureLocation {
			q := p.PictureDirectory + "-" + p.PictureHost
			if n, ok := checkDuplicate[q]; ok {
				checkDuplicate[q] = n + 1
			} else {
				checkDuplicate[q] = 1
			}
		}
		l := len(picData.PictureLocation)
		if len(checkDuplicate) != l {
			fmt.Println("Duplicates found....", picData.Index)
			for n, v := range checkDuplicate {
				fmt.Printf("%s = %d", n, v)
			}
		}
		if quantity, ok := quantityStatistics[picData.NrPictureLocation]; ok {
			quantityStatistics[picData.NrPictureLocation] = quantity + 1
		} else {
			quantityStatistics[picData.NrPictureLocation] = 1
		}
		if ln, ok := lenStatistics[l]; ok {
			lenStatistics[l] = ln + 1
		} else {
			lenStatistics[l] = 1
		}
	}
	fmt.Printf("Have analysed %d records\n", counter)
	fmt.Printf("\nSchema quantity of picture location:\n")
	for q, c := range quantityStatistics {
		fmt.Printf("%3d - %2d\n", q, c)
	}
	fmt.Printf("\nSchema length of picture location:\n")
	for q, c := range lenStatistics {
		fmt.Printf("%3d - %2d\n", q, c)
	}
	if checker.deleteDuplikate {
		return checker.deleteIsns(isnList)
	}
	return nil
}

func (checker *checker) deleteIsns(isnList []adatypes.Isn) error {
	deleteRequest, err := checker.conn.CreateMapDeleteRequest("PictureMetadata")
	if err != nil {
		checker.conn.Close()
		return err
	}
	for _, isn := range isnList {
		err := deleteRequest.Delete(isn)
		if err != nil {
			return err
		}
		err = deleteRequest.EndTransaction()
		if err != nil {
			return err
		}
	}
	return nil
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
