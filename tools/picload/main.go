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
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
	"tux-lobload/store"

	"github.com/tknie/adabas-go-api/adabas"
	"github.com/tknie/adabas-go-api/adatypes"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var hostname string
var timeFormat = "2006-01-02 15:04:05"
var wg sync.WaitGroup

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

	err := initLogLevelWithFile("picload.log", level)
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

func main() {
	var pictureDirectory string
	var dbidParameter string
	var picFnrParameter int
	var filter string
	var deleteIsn int
	var binarySize int
	var verify bool
	var verbose bool
	var update bool
	var checksumRun bool
	var shortenName bool
	var query string
	var interval int
	var nrThreads int
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
	dbReference := &store.DatabaseReference{}

	flag.StringVar(&pictureDirectory, "D", "", "Directory of picture to be imported")
	flag.StringVar(&dbidParameter, "d", "23", "Map repository Database id")
	flag.StringVar(&filter, "F", "@eadir", "Comma-separated list of parts which may excluded")
	flag.StringVar(&query, "q", ".*/@eaDir/.*", "Ignore paths using this regexp")
	flag.IntVar(&picFnrParameter, "p", 4, "Picture file number")
	flag.IntVar(&nrThreads, "t", 2, "Nr of parallel storage threads")
	flag.BoolVar(&verify, "V", false, "Verify data")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.BoolVar(&update, "u", false, "Update data")
	flag.BoolVar(&shortenName, "s", false, "Shorten directory name")
	flag.IntVar(&interval, "I", 60, "Interval for the statistics output")
	flag.BoolVar(&checksumRun, "c", false, "Checksum run, no data load")
	flag.IntVar(&deleteIsn, "r", -1, "Delete ISN image")
	flag.IntVar(&binarySize, "b", 1550000000, "Maximum binary blob size")
	flag.Parse()
	dbReference.Dbid = dbidParameter
	dbReference.PictureFile = adabas.Fnr(picFnrParameter)

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

	if !verify && (pictureDirectory == "" && deleteIsn == -1) {
		fmt.Println("Picture directory option is required")
		flag.Usage()
		return
	}
	fmt.Printf("Connect to map repository %s/%d\n", dbidParameter, picFnrParameter)

	if deleteIsn > 0 {
		ps := createPictureStore(dbReference, shortenName)
		defer ps.Close()

		ps.ChecksumRun = checksumRun
		ps.MaxBlobSize = int64(binarySize)
		ps.Update = update
		ps.Verbose = verbose
		ps.Filter = strings.Split(filter, ",")
		err := ps.DeleteIsn(adatypes.Isn(deleteIsn))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting Isn=%d: %v", deleteIsn, err)
		} else {
			fmt.Printf("Isn=%d successfull deleted ....\n", deleteIsn)
		}
		return
	}

	if pictureDirectory != "" {
		c := 0
		lastChecked := uint64(0)
		psList := make([]*store.PictureConnection, 0)
		output := func() {
			fmt.Print(store.Statistics.String())
			c++
			if lastChecked != store.Statistics.Checked {
				c = 0
			} else {
				if c > 25 {
					for i, ps := range psList {
						fmt.Println(i, ". works on ", ps.CurrentFile)
					}
					panic("Multiple loop found")
				}
			}
			lastChecked = store.Statistics.Checked
		}
		queries := strings.Split(query, ",")
		reg := make([]*regexp.Regexp, 0)
		for _, q := range queries {
			r, err := regexp.Compile(q)
			if err != nil {
				fmt.Println("Query error regexp:", err)
				return
			}
			reg = append(reg, r)
		}
		if verbose {
			fmt.Printf("%s Loading path %s\n", time.Now().Format(timeFormat), pictureDirectory)
		}
		stop := schedule(output, 60*time.Second)
		pathChan := make(chan string, nrThreads)
		stopThread := make(chan bool, nrThreads)
		wg.Add(nrThreads)
		for i := 0; i < nrThreads; i++ {

			ps := createPictureStore(dbReference, shortenName)
			psList = append(psList, ps)
			ps.ChecksumRun = checksumRun
			ps.MaxBlobSize = int64(binarySize)
			ps.Update = update
			ps.Verbose = verbose
			ps.Filter = strings.Split(filter, ",")
			go processImage(ps, pathChan, stopThread)
		}
		_ = filepath.Walk(pictureDirectory, func(path string, info os.FileInfo, err error) error {
			if info == nil || info.IsDir() {
				adatypes.Central.Log.Infof("Info empty or dir: %s", path)
				return nil
			}
			suffix := path[strings.LastIndex(path, ".")+1:]
			suffix = strings.ToLower(suffix)
			switch suffix {
			case "jpg", "jpeg", "gif", "m4v", "mov":
				adatypes.Central.Log.Debugf("Checking picture file: %s", path)
				add := true
				if query != "" {
					for _, r := range reg {
						add = checkQueryPath(r, path)
						if !add {
							break
						}
					}
				}
				if add {
					pathChan <- path
				} else {
					store.Statistics.Ignored++
				}
			default:
			}
			return nil
		})
		stop <- true
		output()
		fmt.Printf("%s Done\n",
			time.Now().Format(timeFormat))
		for e, n := range store.Statistics.Errors {
			fmt.Println(e, ":", n)
		}
		for i := 0; i < nrThreads; i++ {
			stopThread <- true
		}
		wg.Wait()
	}
	if verify {
		output := func() {
			fmt.Printf("%s Verified=%d NotFound=%d DiffData=%d DiffSize=%d OtherHost=%d\n", time.Now().Format(timeFormat),
				store.Statistics.Verified, store.Statistics.NotFound, store.Statistics.DiffFound,
				store.Statistics.SizeDiffFound, store.Statistics.OtherHost)
			list := make([]string, 0)
			store.Statistics.HostsFound.Range(func(key, value interface{}) bool {
				list = append(list, key.(string))
				return true
			})
			fmt.Printf("%s hosts -> %v\n", time.Now().Format(timeFormat), list)
		}
		stop := schedule(output, time.Duration(interval)*time.Second)
		fmt.Printf("%s Start verifying database picture content\n", time.Now().Format(timeFormat))
		err := store.VerifyPicture(dbidParameter, adabas.Fnr(picFnrParameter), nrThreads)
		if err != nil {
			fmt.Printf("%s Error during verify of database picture content: %v\n", time.Now().Format(timeFormat), err)
			return
		}
		stop <- true
		output()
		fmt.Printf("%s finished verify of database picture content\n", time.Now().Format(timeFormat))
	}

}

func createPictureStore(dbReference *store.DatabaseReference, shortenName bool) *store.PictureConnection {
	connection, err := adabas.NewConnection(fmt.Sprintf("acj;inmap=%s,%d", dbReference.Dbid, dbReference.PictureFile))
	if err != nil {
		fmt.Println("Adabas connection error", err)
		panic("Adabas communication error")
	}

	ps, perr := store.InitStorePictureBinary(!shortenName, dbReference, connection)
	if perr != nil {
		fmt.Println("Adabas connection error", perr)
		panic("Adabas communication error")
	}
	return ps
}

func processImage(ps *store.PictureConnection, pathChan chan string, stopThread chan bool) {
	defer ps.Close()
	defer wg.Done()
	for {
		select {
		case <-stopThread:
			if ps.Verbose {
				fmt.Println("Close processing thread")
			}
			return
		case path := <-pathChan:
			for _, f := range ps.Filter {
				if strings.Contains(path, f) {
					err := ps.DeletePath(path)
					if err == nil {
						store.Statistics.NrDeleted++
					}
				}
			}

			ps.CurrentFile = path
			err := ps.LoadPicture(!ps.Update, path)
			if err != nil {
				adatypes.Central.Log.Debugf("Loaded %s with error=%v", ps, err)
				fmt.Fprintln(os.Stderr, "Error loading picture", path, ":", err)
				if strings.HasPrefix(err.Error(), "file tooo big") {
					store.Statistics.ToBig++
				} else {
					if n, ok := store.Statistics.Errors[err.Error()]; ok {
						store.Statistics.Errors[err.Error()] = n + 1
					} else {
						store.Statistics.Errors[err.Error()] = 1
					}
					store.Statistics.NrErrors++
				}
			}
		}
	}
}

func checkQueryPath(reg *regexp.Regexp, path string) bool {
	return !reg.MatchString(path)
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
