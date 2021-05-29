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
	"fmt"
	"os"
	"strings"

	"github.com/SoftwareAG/adabas-go-api/adabas"
	"github.com/SoftwareAG/adabas-go-api/adatypes"
)

// PictureConnection picture connection handle
type PictureConnection struct {
	dbReference       *DatabaseReference
	store             *adabas.StoreRequest
	storeData         *adabas.StoreRequest
	storeThumb        *adabas.StoreRequest
	storeEntries      *adabas.StoreRequest
	readFileNameCheck *adabas.ReadRequest
	readMediaCheck    *adabas.ReadRequest
	readAddAndCheck   *adabas.ReadRequest
	histCheck         *adabas.ReadRequest
	ShortenName       bool
	ChecksumRun       bool
	Found             uint64
	Empty             uint64
	Loaded            uint64
	Added             uint64
	Checked           uint64
	ToBig             uint64
	Errors            map[string]uint64
	Filter            []string
	NrErrors          uint64
	NrDeleted         uint64
	Ignored           uint64
	MaxBlobSize       int64
}

// Hostname of this host
var Hostname = "Unknown"

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

func (ps *PictureConnection) checkPicture(key string) (bool, error) {
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
}

// Close connection
func (ps *PictureConnection) Close() {
	if ps != nil && ps.dbReference.Connection != nil {
		ps.dbReference.Connection.Close()
	}
}

func verifyPictureRecord(record *adabas.Record, x interface{}) error {
	f, ferr := record.SearchValue("PictureName")
	if ferr != nil {
		return ferr
	}
	fileName := f.String()
	v, xerr := record.SearchValue("Media")
	if xerr != nil {
		return xerr
	}
	vLen := len(v.Bytes())
	md := createMd5(v.Bytes())
	v, xerr = record.SearchValue("ChecksumPicture")
	if xerr != nil {
		return xerr
	}
	smd := strings.Trim(v.String(), " ")
	fmt.Printf("ISN=%d. name=%s len=%d\n", record.Isn, fileName, vLen)
	if md != smd {
		fmt.Printf("MD5 data=<%s> expected=<%s>\n", md, smd)
		fmt.Println("Record checksum error", record.Isn)
		return fmt.Errorf("Record checksum error")
	}
	return nil
}

// VerifyPicture verify pictures
func VerifyPicture(mapName, ref string) error {
	connection, cerr := adabas.NewConnection("acj;map;config=[" + ref + "]")
	if cerr != nil {
		return cerr
	}
	defer connection.Close()
	request, rerr := connection.CreateMapReadRequest(mapName)
	if rerr != nil {
		fmt.Println("Error create request", rerr)
		return rerr
	}
	err := request.QueryFields("Media,ChecksumPicture,PictureName")
	if err != nil {
		return err
	}
	request.Limit = 0
	request.Multifetch = 1
	result, rErr := request.ReadPhysicalSequenceStream(verifyPictureRecord, nil)
	if rErr != nil {
		return rErr
	}
	fmt.Println(result)
	return nil
}
