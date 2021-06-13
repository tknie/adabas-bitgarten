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
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
	"tux-lobload/store"

	"github.com/SoftwareAG/adabas-go-api/adabas"
	"github.com/SoftwareAG/adabas-go-api/adatypes"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var hostname string

type deleter struct {
	re                    []*regexp.Regexp
	deleteRequest         *adabas.DeleteRequest
	readDirectoryRequest  *adabas.ReadRequest
	storeDirectoryRequest *adabas.StoreRequest
	connection            *adabas.Connection
	test                  bool
	picFnr                adabas.Fnr
	found                 uint64
	deleted               uint64
	transactions          uint64
	counter               uint64
}

var timeFormat = "2006-01-02 15:04:05"

type elementCounter struct {
	counter uint64
}

type validater struct {
	conn            *adabas.Connection
	read            *adabas.ReadRequest
	delete          *adabas.DeleteRequest
	list            *adabas.ReadRequest
	limit           uint64
	elementMap      map[int]*elementCounter
	checkedPicture  uint64
	okPictures      uint64
	failurePictures uint64
	emptyPictures   uint64
	unique          uint64
	deleteDuplikate uint64
	deleteEmpty     uint64
	test            bool
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
	var validate bool
	var query string
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

	flag.StringVar(&dbidParameter, "d", "23", "Database id")
	flag.IntVar(&mapFnrParameter, "p", 100, "Picture file number")
	flag.IntVar(&limit, "l", 10, "Maximum records to read (0 is all)")
	flag.BoolVar(&test, "t", false, "Dry run, don't change")
	flag.BoolVar(&validate, "v", false, "Validate uniquness of media content")
	flag.StringVar(&query, "q", "", "Filter for regexp query used to clean up")
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

	if query == "" && !validate {
		fmt.Println("Need to give exclude mask or enable validation!!!")
		return
	}

	if test {
		fmt.Println("Test mode ENABLED")
	}

	fmt.Printf("Connect to %s/%d\n", dbidParameter, mapFnrParameter)
	if query != "" {
		d := &deleter{test: test, picFnr: adabas.Fnr(mapFnrParameter)}
		fmt.Println("Clear using exclude mask with: " + query)
		queries := strings.Split(query, ",")
		for _, q := range queries {
			re, err := regexp.Compile(q)
			if err != nil {
				fmt.Println("Query error regexp:", err)
				return
			}
			d.re = append(d.re, re)

		}
		connection, err := adabas.NewConnection(fmt.Sprintf("acj;inmap=%s,%d", dbidParameter, mapFnrParameter))
		if err != nil {
			fmt.Println("Error getting connection")
			return
		}
		connection.Close()
		d.connection = connection
		err = removeQueries(connection, d, uint64(limit))
		if err != nil {
			fmt.Println("Error anaylzing douplikats", err)
		}
	}
	if validate {
		val := &validater{limit: uint64(limit), test: test, elementMap: make(map[int]*elementCounter)}
		val.analyzeDoublikats()
	}
}

func removeQuery(record *adabas.Record, x interface{}) error {
	v := record.HashFields["PL"].(*adatypes.StructureValue)
	//fmt.Printf("%d %d\n", v.NrElements(), v.NrValues(1))
	found := 0
	fnMap := make(map[string]bool)
	de := x.(*deleter)
	for _, e := range v.Elements {
		for _, sv := range e.Values {
			fn := sv.String()
			//			fmt.Printf("%s %T -> %s\n", sv.Type().Name(), sv, sv.String())
			for _, re := range de.re {
				if re.MatchString(fn) {
					fnMap[fn] = true
					found++
					break
				} else {
					fnMap[fn] = false
				}
			}
		}
	}
	switch {
	case found == v.NrElements():
		fmt.Println("Found all, could delete ISN:", record.Isn)
		if !de.test {
			err := de.deleteRequest.Delete(record.Isn)
			if err != nil {
				return err
			}
			de.deleted++
			if de.counter%100 == 0 {
				err := de.deleteRequest.EndTransaction()
				if err != nil {
					return err
				}
				de.transactions++
			}
		}
		de.found++
	case found > 0:
		fmt.Println("Found parts, could delete parts of ISN:", record.Isn)
		de.filterDirectories(record.Isn, fnMap)
		// for v, b := range fnMap {
		// 	fmt.Println(v, b)
		// }
	default:
		//	fmt.Println("Ignore :" + fn)

	}
	de.counter++
	return nil
}

func (de *deleter) filterDirectories(isn adatypes.Isn, fnMap map[string]bool) {
	result, err := de.readDirectoryRequest.ReadISN(isn)
	if err != nil {
		panic("Error reading ISN: " + err.Error())
	}
	metadata := result.Data[0].(*store.PictureMetadata)
	fmt.Println("Read ISN:", metadata.Index)

	pnList := make([]*store.PictureLocation, 0)
	extra := 0
	for _, pd := range metadata.PictureLocation {
		if reduce, ok := fnMap[pd.PictureDirectory]; ok {
			if reduce {
				fmt.Println("Reduce", pd.PictureDirectory)
				extra++
			} else {
				fmt.Println("Stay", pd.PictureDirectory)
				pnList = append(pnList, pd)
			}
		} else {
			fmt.Println("Unknown", pd.PictureDirectory)
		}
	}
	for i := 0; i < extra; i++ {
		pnList = append(pnList, &store.PictureLocation{})
	}

	metadata.PictureLocation = pnList
	if !de.test {
		fmt.Println("Update ISN:", metadata.Index)
		err = de.storeDirectoryRequest.UpdateData(metadata)
		if err != nil {
			panic("Error storing ISN: " + err.Error())
		}
		err = de.storeDirectoryRequest.EndTransaction()
		if err != nil {
			panic("Error end transaction of ISN: " + err.Error())
		}
	}
}

func removeQueries(conn *adabas.Connection, de *deleter, limit uint64) error {
	readCheck, err := conn.CreateFileReadRequest(de.picFnr)
	if err != nil {
		conn.Close()
		return err
	}
	readCheck.Limit = limit
	err = readCheck.QueryFields("PD")
	if err != nil {
		conn.Close()
		return err
	}
	de.deleteRequest, err = conn.CreateDeleteRequest(de.picFnr)
	if err != nil {
		conn.Close()
		return err
	}
	de.readDirectoryRequest, err = conn.CreateMapReadRequest((*store.PictureMetadata)(nil))
	if err != nil {
		conn.Close()
		return err
	}
	err = de.readDirectoryRequest.QueryFields("PL")
	if err != nil {
		fmt.Printf("Error defining field query: %v\n", err)
		de.deleteRequest.BackoutTransaction()
		panic("Read error " + err.Error())
	}
	de.storeDirectoryRequest, err = conn.CreateMapStoreRequest((*store.PictureMetadata)(nil))
	if err != nil {
		conn.Close()
		return err
	}
	err = de.storeDirectoryRequest.StoreFields("PL")
	if err != nil {
		fmt.Printf("Error defining store fields: %v\n", err)
		de.deleteRequest.BackoutTransaction()
		panic("Read error " + err.Error())
	}
	_, err = readCheck.ReadPhysicalSequenceStream(removeQuery, de)
	if err != nil {
		fmt.Printf("Error reading physical sequence stream: %v\n", err)
		de.deleteRequest.BackoutTransaction()
		panic("Read error " + err.Error())
	}
	fmt.Printf("Check %d records, found=%d, deleted=%d,transactions=%d\n", de.counter, de.found, de.deleted, de.transactions)
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

func (validater *validater) analyzeDoublikats() (err error) {
	validater.conn, err = adabas.NewConnection("acj;map")
	if err != nil {
		return err
	}
	defer validater.conn.Close()
	if validater.read == nil {
		validater.read, err = validater.conn.CreateMapReadRequest("PictureMetadata")
		if err != nil {
			validater.conn.Close()
			return err
		}
		validater.read.Limit = validater.limit
		err = validater.read.QueryFields("ChecksumPicture,PictureName")
		if err != nil {
			validater.conn.Close()
			return err
		}
	}
	counter := uint64(0)
	output := func() {
		fmt.Printf("%s Picture counter=%d checked=%d ok=%d unique=%d failure=%d empty=%d del Dupli=%d del Empty=%d\n",
			time.Now().Format(timeFormat), counter, validater.checkedPicture,
			validater.okPictures, validater.unique, validater.failurePictures,
			validater.emptyPictures, validater.deleteDuplikate, validater.deleteEmpty)
	}
	stop := schedule(output, 15*time.Second)
	cursor, err := validater.read.HistogramByCursoring("ChecksumPicture")
	// result, err := validater.read.ReadLogicalByStream("ChecksumPicture", func(record *adabas.Record, x interface{}) error {
	// 	// fmt.Printf("quantity=%03d -> %s\n", record.Quantity, record.HashFields["ChecksumPicture"])
	// 	err = validater.listDuplikats(record.HashFields["ChecksumPicture"].String())
	// 	if err != nil {
	// 		return err
	// 	}
	// 	counter++
	// 	return nil
	// }, nil)
	if err != nil {
		fmt.Printf("Error histogram descriptor quantity for ChecksumPicture: %v\n", err)
		panic("Read error " + err.Error())
	}
	for cursor.HasNextRecord() {
		counter++
		record, err := cursor.NextRecord()
		if err != nil {
			fmt.Printf("Error getting next record cursor: %v\n", err)
			panic("Cursor error " + err.Error())
		}
		// fmt.Println("Quantity: ", record.Quantity)
		if record.Quantity > 1 {
			err = validater.listDuplikats(record.HashFields["ChecksumPicture"].String())
			if err != nil {
				fmt.Printf("Error cursor list duplicates: %v\n", err)
				panic("Duplicate error " + err.Error())
			}
		}
		if validater.limit != 0 && counter >= validater.limit {
			break
		}
	}
	stop <- true
	fmt.Printf("%s Picture counter=%d checked=%d ok=%d unique=%d failure=%d empty=%d del Dupli=%d del Empty=%d\n",
		time.Now().Format(timeFormat), counter, validater.checkedPicture,
		validater.okPictures, validater.unique, validater.failurePictures,
		validater.emptyPictures, validater.deleteDuplikate, validater.deleteEmpty)
	fmt.Printf("There are %06d unique records\n", counter)
	for c, ce := range validater.elementMap {
		fmt.Println("Elements of ", c, " = ", ce.counter, "occurance")
	}
	return nil
}

func (validater *validater) listDuplikats(checksum string) (err error) {
	if validater.list == nil {
		validater.list, err = validater.conn.CreateMapReadRequest(&store.PictureData{})
		if err != nil {
			validater.conn.Close()
			return
		}
		err = validater.list.QueryFields("Media")
		if err != nil {
			validater.conn.Close()
			return
		}
		validater.list.Multifetch = 1
		validater.list.Limit = 1
	}
	cursor, err := validater.list.ReadLogicalWithCursoring("ChecksumPicture=" + checksum)
	if err != nil {
		fmt.Printf("Error checking descriptor quantity for ChecksumPicture: %v (%s)\n", err, checksum)
		panic("Read error " + err.Error())
	}
	validater.unique++
	first := true
	var data []byte
	var baseIsn uint64
	counter := 0
	for cursor.HasNextRecord() {
		validater.checkedPicture++
		counter++
		record, recErr := cursor.NextData()
		if recErr != nil {
			panic("Read error " + recErr.Error())
		}
		curPicture := record.(*store.PictureData)
		if first {
			data = curPicture.Media
			if len(data) == 0 {
				fmt.Println("Main record media is empty", checksum)
				validater.emptyPictures++
			} else {
				validater.okPictures++
			}
			baseIsn = curPicture.Index
			first = false
		} else {
			if data != nil {
				if len(curPicture.Media) == 0 {
					fmt.Println("Second record media is empty", checksum)
					validater.emptyPictures++
					fmt.Println("Delete empty ISN:", curPicture.Index, " of ", baseIsn)
					err = validater.Delete(curPicture.Index)
					if err != nil {
						return err
					}
					validater.deleteEmpty++
				} else if bytes.Equal(data, curPicture.Media) {
					fmt.Println("Record entry differ to first", checksum)
					validater.failurePictures++
				} else {
					validater.okPictures++
					fmt.Println("Delete duplikate ISN:", curPicture.Index, " of ", baseIsn)
					err = validater.Delete(curPicture.Index)
					if err != nil {
						return err
					}
					validater.deleteDuplikate++
				}
			} else {
				fmt.Println("First record is empty")
			}
		}
		if err != nil {
			return err
		}
		// fmt.Printf("  ISN=%06d %s -> %s\n", record.Isn, record.HashFields["PictureName"].String(), record.HashFields["Option"])
	}
	if c, ok := validater.elementMap[counter]; ok {
		c.counter++
	} else {
		validater.elementMap[counter] = &elementCounter{counter: 1}
	}
	if !validater.test {
		return validater.conn.EndTransaction()
	}
	return nil
}

func (validater *validater) Delete(isn uint64) (err error) {
	if !validater.test {
		if validater.delete == nil {
			validater.delete, err = validater.conn.CreateMapDeleteRequest("PictureMetadata")
			if err != nil {
				validater.conn.Close()
				return
			}
		}

		validater.delete.Delete(adatypes.Isn(isn))
	}
	return nil
}

func schedule(what func(), delay time.Duration) chan bool {
	stop := make(chan bool)

	go func() {
		for {
			what()
			select {
			case <-time.After(delay):
			case <-stop:
				return
			}
		}
	}()

	return stop
}
