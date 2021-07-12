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

package store

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SoftwareAG/adabas-go-api/adabas"
	"github.com/SoftwareAG/adabas-go-api/adatypes"
)

// PictureConnection picture connection handle
type PictureConnection struct {
	dbReference       *DatabaseReference
	connection        *adabas.Connection
	store             *adabas.StoreRequest
	storeData         *adabas.StoreRequest
	storeThumb        *adabas.StoreRequest
	storeEntries      *adabas.StoreRequest
	readFileNameCheck *adabas.ReadRequest
	readMediaCheck    *adabas.ReadRequest
	readAddAndCheck   *adabas.ReadRequest
	histCheck         *adabas.ReadRequest
	ShortenName       bool
	Update            bool
	ChecksumRun       bool
	Verbose           bool
	Filter            []string
	MaxBlobSize       int64
	CurrentFile       string
}

type PictureStatistic struct {
	Found         uint64
	Empty         uint64
	Loaded        uint64
	Added         uint64
	Checked       uint64
	ToBig         uint64
	NrErrors      uint64
	Duplicated    uint64
	Errors        map[string]uint64
	NrDeleted     uint64
	Ignored       uint64
	Verified      uint64
	SizeDiffFound uint64
	DiffFound     uint64
	NotFound      uint64
	OtherHost     uint64
	HostsFound    sync.Map
}

var Statistics = &PictureStatistic{Errors: make(map[string]uint64)}

// Hostname of this host
var Hostname = "Unknown"
var timeFormat = "2006-01-02 15:04:05"

func init() {
	host, err := os.Hostname()
	if err == nil {
		Hostname = host
	}
}

func checkEmpty(fileName string) bool {
	st, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		// file is not exists similar to empty
		return true
	}
	if st.Size() == 0 {
		return true
	}
	return false
}

func (stat *PictureStatistic) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%s Picture directory checked=%d loaded=%d found=%d too big=%d errors=%d deleted=%d\n",
		time.Now().Format(timeFormat), stat.Checked, stat.Loaded, stat.Found, stat.ToBig, stat.NrErrors, stat.NrDeleted))
	buffer.WriteString(fmt.Sprintf("%s Picture directory added=%d empty=%d ignored=%d duplicated=%d\n",
		time.Now().Format(timeFormat), stat.Added, stat.Empty, stat.Ignored, stat.Duplicated))

	return buffer.String()
}

func (ps *PictureConnection) pictureFileAvailable(key string) (bool, error) {
	//fmt.Println("Check Md5=" + key)
	result, err := ps.readFileNameCheck.HistogramWith("PM=" + key)
	if err != nil {
		fmt.Printf("Error checking PictureHash=%s: %v\n", key, err)
		panic("Read error " + err.Error())
		//		return false, err
	}
	// result.DumpValues()
	if len(result.Values) > 0 || len(result.Data) > 0 {
		adatypes.Central.Log.Debugf("PM=%s is available\n", key)
		return true, nil
	}
	adatypes.Central.Log.Debugf("PM=%s is not loaded\n", key)
	return false, nil
}

func (ps *PictureConnection) pictureMediaAvailable(key string) (bool, error) {
	//fmt.Println("Check Md5=" + key)
	result, err := ps.readMediaCheck.HistogramWith("CP=" + key)
	if err != nil {
		fmt.Printf("Error checking PictureHash=%s: %v\n", key, err)
		panic("Read error " + err.Error())
		//		return false, err
	}
	// result.DumpValues()
	if len(result.Values) > 0 || len(result.Data) > 0 {
		adatypes.Central.Log.Debugf("CP=%s is available\n", key)
		return true, nil
	}
	adatypes.Central.Log.Debugf("CP=%s is not loaded\n", key)
	return false, nil
}

/* func (ps *PictureConnection) checkPicture(key string) (bool, error) {
	//fmt.Println("Check Md5=" + key)
	result, err := ps.histCheck.HistogramWith("CP=" + key)
	if err != nil {
		fmt.Printf("Error checking ChecksumPicture=%s: %v\n", key, err)
		panic("Read error " + err.Error())
		//		return false, err
	}
	// result.DumpValues()
	if len(result.Values) > 0 || len(result.Data) > 0 {
		adatypes.Central.Log.Debugf("ChecksumPicture=%s is available\n", key)
		return true, nil
	}
	adatypes.Central.Log.Debugf("ChecksumPicture=%s is not loaded\n", key)
	return false, nil
}*/

// Close connection
func (ps *PictureConnection) Close() {
	if ps != nil && ps.connection != nil {
		ps.connection.Close()
	}
}

func verifyPictureRecord(cursor *adabas.Cursoring, nrThreads int) error {
	pictureDataChan := make(chan *PictureData, nrThreads)
	stopThread := make(chan bool, nrThreads)
	var wg sync.WaitGroup
	wg.Add(nrThreads)
	for i := 0; i < nrThreads; i++ {
		go VerifyPictureData(&wg, stopThread, pictureDataChan)
	}
	fmt.Printf("%s Start reading records ... \n", time.Now().Format(timeFormat))
	for cursor.HasNextRecord() {
		data, err := cursor.NextData()
		if err != nil {
			return err
		}
		pm := data.(*PictureData)
		pictureDataChan <- pm
		//fmt.Printf("ISN=%d. Checksum=%s len=%d\n", pm.Index, pm.ChecksumPicture, len(pm.PictureLocation))
		// for _, p := range pm.PictureLocation {
		// 	//	fmt.Println(p.PictureHost, p.PictureDirectory)
		// 	if p.PictureHost == Hostname {
		// 		pm.compareMedia(p.PictureDirectory)
		// 	} else {
		// 		Statistics.OtherHost++
		// 	}
		// }
	}
	fmt.Printf("%s Stop all threads verifying read records ...\n", time.Now().Format(timeFormat))
	for i := 0; i < nrThreads; i++ {
		stopThread <- true
	}
	wg.Wait()
	fmt.Printf("%s Got all threads\n", time.Now().Format(timeFormat))
	return nil
}

func VerifyPictureData(wg *sync.WaitGroup, stopThread chan bool, pictureDataChan chan *PictureData) {
	for {
		select {
		case <-stopThread:
			wg.Done()
			return
		case pm := <-pictureDataChan:
			for _, p := range pm.PictureLocation {
				//	fmt.Println(p.PictureHost, p.PictureDirectory)
				if p.PictureHost == Hostname {
					pm.compareMedia(p.PictureDirectory)
				} else {
					Statistics.HostsFound.Store(p.PictureHost, true)
					Statistics.OtherHost++
				}
			}
		}
	}
}

// VerifyPicture verify pictures
func VerifyPicture(target string, file adabas.Fnr, nrThreads int) error {
	connection, err := adabas.NewConnection(fmt.Sprintf("acj;inmap=%s,%d", target, file))
	if err != nil {
		fmt.Println("Adabas connection error", err)
		panic("Adabas communication error in verify")
	}
	defer connection.Close()
	request, rerr := connection.CreateMapReadRequest((*PictureData)(nil))
	if rerr != nil {
		fmt.Println("Error creating request", rerr)
		return rerr
	}
	err = request.QueryFields("DP,CP,PL")
	if err != nil {
		fmt.Println("Error query fields", err)
		return err
	}
	request.Limit = 0
	request.Multifetch = 1

	// cursor, rErr := request.ReadPhysicalWithCursoring()
	fmt.Println(time.Now().Format(timeFormat), "Read all pictures from host", Hostname)
	cursor, rErr := request.ReadLogicalWithCursoring("PH=" + Hostname)
	// request.ReadPhysicalSequenceStream(verifyPictureRecord, nil)
	if rErr != nil {
		fmt.Println("Error read physical cursor start", rErr)
		return rErr
	}
	return verifyPictureRecord(cursor, nrThreads)
}

func (pic *PictureData) compareMedia(loadFile string) (err error) {
	// fmt.Println("Compare file", loadFile, "with data in", pic.ChecksumPicture)
	f, err := os.Open(loadFile)
	if err != nil {
		fmt.Printf("Error loading file [%d]: %v\n", pic.Index, loadFile)
		Statistics.NotFound++
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if fi.Size() > int64(len(pic.Media)) {
		return fmt.Errorf("file tooo big %d>%d", fi.Size(), len(pic.Media))
	}
	fileData := make([]byte, fi.Size())
	var n int
	n, err = f.Read(fileData)
	adatypes.Central.Log.Debugf("Number of bytes read: %d/%d -> %v\n", n, len(pic.Media), err)
	if err != nil {
		fmt.Printf("Error reading file: %v", err)
		return err
	}
	md := createMd5(fileData)
	if strings.Trim(pic.ChecksumPicture, " ") != md {
		fmt.Printf("Checksum mismatch <%s> <%s> of %s[%d]\n", md, pic.ChecksumPicture, loadFile, pic.Index)
	}
	if len(pic.Media) != len(fileData) {
		fmt.Printf("Different media length %d != %d of %s[%d]\n", len(pic.Media), len(fileData), loadFile, pic.Index)
		Statistics.SizeDiffFound++
		return fmt.Errorf("size difference found")
	}
	for i := 0; i < len(pic.Media); i++ {
		if pic.Media[i] != fileData[i] {
			fmt.Printf("Error difference offset at %d[%d]\n", i, pic.Index)
			fmt.Println(adatypes.FormatByteBuffer("Database at offset", pic.Media[i-10:i+100]))
			fmt.Println(adatypes.FormatByteBuffer("File     at offset", fileData[i-10:i+100]))
			Statistics.DiffFound++
			return fmt.Errorf("data difference found")
		}
	}
	Statistics.Verified++
	return nil
}
