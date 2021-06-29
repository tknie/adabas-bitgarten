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
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"tux-lobload/store"

	"github.com/SoftwareAG/adabas-go-api/adabas"
	"github.com/SoftwareAG/adabas-go-api/adatypes"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type reader struct {
	mapName    string
	repository string
}

func init() {
	level := zapcore.ErrorLevel
	ed := os.Getenv("ENABLE_DEBUG")
	switch ed {
	case "1":
		level = zapcore.DebugLevel
		adatypes.Central.SetDebugLevel(true)
	case "2":
		level = zapcore.InfoLevel
	}

	err := initLogLevelWithFile("reader.log", level)
	if err != nil {
		panic(err)
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
		panic(err)
	}
	cfg.Level.SetLevel(level)
	cfg.OutputPaths = []string{name}
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()

	sugar.Infof("Start logging with level", level)
	adatypes.Central.Log = sugar

	return
}

func main() {
	r := &reader{}
	var verify bool
	var compare bool
	var fileName string
	var hash string
	flag.StringVar(&r.mapName, "m", "", "Adabas map name")
	flag.StringVar(&r.repository, "r", "", "repository location of Adabas maps")
	flag.StringVar(&fileName, "l", "", "Load file into media data")
	flag.StringVar(&hash, "h", "", "Hash value the data should be load at")
	flag.BoolVar(&verify, "v", false, "Verify data")
	flag.BoolVar(&compare, "c", false, "Compare data")
	flag.Parse()

	connection, err := adabas.NewConnection("acj;map" + r.mapName + ";config=[" + r.repository + "]")
	if err != nil {
		fmt.Println(err.Error())
		panic("Error repository:" + err.Error())
	}

	if fileName != "" && hash != "" {
		if compare {
			compareMedia(connection, r, fileName, hash)
			return
		}
		loadMedia(r, fileName, hash)
		return
	}

	if !verify && r.mapName == "" {
		fmt.Println("Adabas Map option is required")
		return
	}

	if r.repository != "" {
		err := adabas.AddGlobalMapRepositoryReference(r.repository)
		if err != nil {
			fmt.Println(err.Error())
			panic("Error repository:" + err.Error())
		}
	}
	if verify {
		verifyLargeObjects(r)
		return
	}
	readTitle(r)
}

func createChecksum(b []byte) string {
	m := md5.New()
	m.Write(b)
	ms := m.Sum(nil)
	return fmt.Sprintf("%X", ms)
}

func receiveInterface(data interface{}, x interface{}) error {
	p := data.(*store.PictureData)
	ckSum := createChecksum(p.Media)
	chkSav := strings.Trim(p.ChecksumPicture, " ")
	if ckSum != chkSav {
		fmt.Println("Received Media data not valid")
		fmt.Println(ckSum, " -> ", chkSav, "=", len(p.Media))
	}
	/*if strings.Trim(p.Data.ChecksumThumbnail, " ") != "" {
		ckSum = createChecksum(p.Data.Thumbnail)
		chkSav = strings.Trim(p.Data.ChecksumThumbnail, " ")
		if ckSum != chkSav {
			fmt.Println("Received Thumbnail data not valid")
			fmt.Println(ckSum, " -> ", chkSav, "=", len(p.Data.Thumbnail))
		}
	}*/
	return nil
}

func compareMedia(connection *adabas.Connection, r *reader, loadFile, hash string) (err error) {
	fmt.Println("Compare file", loadFile, "with data in", hash)
	p := &store.PictureBinary{MetaData: &store.PictureMetadata{}, FileName: loadFile}
	err = p.LoadFile()
	if err != nil {
		return
	}
	p2 := &store.PictureBinary{}
	err = p2.ReadDatabase(connection, hash, r.repository)
	if err != nil {
		return err
	}
	if len(p.Data.Media) != len(p2.Data.Media) {
		fmt.Printf("Different media length %d != %d\n", p.Data.Media, p2.Data.Media)
	}
	for i := 0; i < len(p.Data.Media); i++ {
		if p.Data.Media[i] != p2.Data.Media[i] {
			fmt.Printf("Error difference offset at %d\n", i)
			fmt.Println(adatypes.FormatByteBuffer("File at offset", p.Data.Media[i-10:i+100]))
			fmt.Println(adatypes.FormatByteBuffer("Database at offset", p2.Data.Media[i-10:i+100]))
			break
		}
	}

	return nil
}

func loadMedia(r *reader, loadFile, hash string) (*store.PictureBinary, error) {
	fmt.Println("Load file", loadFile, "into", hash)
	p := &store.PictureBinary{MetaData: &store.PictureMetadata{}, FileName: loadFile}
	p.LoadFile()

	connection, err := adabas.NewConnection("acj;map;config=[" + r.repository + "]")
	if err != nil {
		return nil, err
	}
	defer connection.Close()

	request, serr := connection.CreateMapStoreRequest(store.PictureData{})
	if serr != nil {
		fmt.Println("Error create request", serr)
		return nil, serr
	}
	serr = request.StoreFields("DP")
	if serr != nil {
		fmt.Println("Error define fields", serr)
		return nil, serr
	}
	serr = request.StoreData(p.Data)
	if serr != nil {
		fmt.Println("Error store data", serr)
		return nil, serr
	}
	serr = request.EndTransaction()
	if serr != nil {
		fmt.Println("Error finalize transaction", serr)
		return nil, serr
	}
	return p, nil
}

func readTitle(r *reader) {
	connection, err := adabas.NewConnection("acj;map;config=[" + r.repository + "]")
	if err != nil {
		return
	}
	defer connection.Close()

	request, rerr := connection.CreateMapReadRequest(store.Album{})
	if rerr != nil {
		fmt.Println("Error create request", rerr)
		return
	}
	err = request.QueryFields("Title")
	if err != nil {
		return
	}
	var result *adabas.Response
	result, err = request.ReadPhysicalSequence()
	if err != nil {
		fmt.Println("Error reading ISN order", err)
		return
	}
	for _, x := range result.Data {
		fmt.Println(x.(*store.Album).Title)
	}

}

func verifyLargeObjects(r *reader) {
	connection, err := adabas.NewConnection("acj;map;config=[" + r.repository + "]")
	if err != nil {
		return
	}
	defer connection.Close()

	request, rerr := connection.CreateMapReadRequest(store.PictureData{})
	if rerr != nil {
		fmt.Println("Error create request", rerr)
		return
	}
	_, err = request.ReadPhysicalInterface(receiveInterface, nil)
	if err != nil {
		fmt.Println("Error reading ISN order", err)
		return
	}

}
