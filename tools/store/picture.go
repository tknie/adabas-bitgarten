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
	"crypto/md5"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"regexp"
	"strings"

	"github.com/rwcarlsen/goexif/exif"

	"github.com/nfnt/resize"
	"github.com/tknie/adabas-go-api/adabas"
	"github.com/tknie/adabas-go-api/adatypes"
)

// PictureBinary definition
type PictureBinary struct {
	Index       uint64 `adabas:"#isn" json:"-"`
	FileName    string `xml:"-" json:"-"`
	MetaData    *PictureMetadata
	MaxBlobSize int64 // 50000000
	Data        *PictureData
}

// PictureMetadata definition
type PictureMetadata struct {
	Index             uint64             `adabas:"#isn" json:"-"`
	Title             string             `adabas:"::TI"`
	Fill              string             `adabas:"::FI"`
	MIMEType          string             `adabas:"::TY"`
	Option            string             `adabas:"::OP"`
	Width             uint32             `adabas:"::HE"`
	Height            uint32             `adabas:"::WI"`
	ChecksumPicture   string             `adabas:":key:CP"`
	NrPictureLocation int                `adabas:"::#PL"`
	PictureLocation   []*PictureLocation `adabas:"::PL"`
	ExifModel         string             `adabas:"::MO"`
	ExifMake          string             `adabas:"::MA"`
	ExifTaken         string             `adabas:"::TA"`
	ExifOrigTime      string             `adabas:"::OT"`
	ExifOrientation   byte               `adabas:"::OR"`
	ExifXdimension    uint32             `adabas:"::XD"`
	ExifYdimension    uint32             `adabas:"::YD"`
}

type PictureLocation struct {
	PictureName      string `adabas:"::PN"`
	PictureHash      string `adabas:"::PM"`
	PictureHost      string `adabas:"::PH"`
	PictureDirectory string `adabas:"::PD"`
}

// PictureData definition
type PictureData struct {
	Index           uint64             `adabas:":isn" json:"-"`
	ChecksumPicture string             `adabas:":key:CP"`
	PictureLocation []*PictureLocation `adabas:"::PL"`
	Media           []byte             `adabas:"::DP" xml:"-" json:"-"`
	Thumbnail       []byte             `adabas:"::DT" xml:"-" json:"-"`
	//	ChecksumThumbnail string `adabas:":key:CT"`
}

var re = regexp.MustCompile(`(?m).*/([^/]*)`)

// LoadFile load file
func (pic *PictureBinary) LoadFile() error {
	f, err := os.Open(pic.FileName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	pic.Data = &PictureData{}
	if fi.Size() > pic.MaxBlobSize {
		return fmt.Errorf("file tooo big %d>%d", fi.Size(), pic.MaxBlobSize)
	}
	pic.Data.Media = make([]byte, fi.Size())
	var n int
	n, err = f.Read(pic.Data.Media)
	adatypes.Central.Log.Debugf("Number of bytes read: %d/%d -> %v\n", n, len(pic.Data.Media), err)
	if err != nil {
		return err
	}
	pic.Data.ChecksumPicture = createMd5(pic.Data.Media)
	pic.MetaData.ChecksumPicture = pic.Data.ChecksumPicture
	adatypes.Central.Log.Debugf("PictureBinary checksum %s size=%d len=%d", pic.Data.ChecksumPicture, fi.Size(), len(pic.Data.Media))

	return nil
}

func createMd5(input []byte) string {
	return fmt.Sprintf("%X", md5.Sum(input))
}

func resizePicture(media []byte, max int) ([]byte, uint32, uint32, error) {
	var buffer bytes.Buffer
	buffer.Write(media)
	srcImage, _, err := image.Decode(&buffer)
	if err != nil {
		adatypes.Central.Log.Debugf("Decode image for thumbnail error %v", err)
		return nil, 0, 0, err
	}
	maxX := uint(0)
	maxY := uint(0)
	b := srcImage.Bounds()
	width := uint32(b.Max.X)
	height := uint32(b.Max.Y)
	if width > height {
		maxX = uint(max)
	} else {
		maxY = uint(max)
	}
	//fmt.Println("Original size: ", height, width, "to", max, "window", maxX, maxY)
	//dstImageFill := imaging.Fill(srcImage, 100, 100, imaging.Center, imaging.Lanczos)
	newImage := resize.Resize(maxX, maxY, srcImage, resize.Lanczos3)
	b = newImage.Bounds()
	width = uint32(b.Max.X)
	height = uint32(b.Max.Y)
	//fmt.Println("New size: ", height, width)
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, newImage, nil)
	if err != nil {
		// fmt.Println("Error generating thumbnail", err)
		adatypes.Central.Log.Debugf("Encode image for thumbnail error %v", err)
		return nil, 0, 0, err
	}
	return buf.Bytes(), width, height, nil
}

// ExtractExif extract EXIF data
func (pic *PictureBinary) ExtractExif() error {
	buffer := bytes.NewBuffer(pic.Data.Media)
	x, err := exif.Decode(buffer)
	if err != nil {
		// fmt.Println("Exif error: ", buffer.Len(), err)
		return err
	}
	// fmt.Println(x)
	// var p Printer
	// x.Walk(p)
	camModel, err := x.Get(exif.Model) // normally, don't ignore errors!
	if err != nil {
		adatypes.Central.Log.Infof("Error EXIF: %v", err)
	} else {
		model, _ := camModel.StringVal()
		pic.MetaData.ExifModel = model
	}

	m, merr := x.Get(exif.Make)
	if merr == nil {
		ms, _ := m.StringVal()
		pic.MetaData.ExifMake = ms
	}

	// Two convenience functions exist for date/time taken and GPS coords:
	tm, tmerr := x.DateTime()
	if tmerr == nil {
		pic.MetaData.ExifTaken = tm.String()
	}

	tmo, tmoerr := x.Get(exif.DateTimeOriginal)
	if tmoerr == nil {
		pic.MetaData.ExifOrigTime = tmo.String()
	}

	o, oerr := x.Get(exif.Orientation)
	if oerr == nil {
		v, _ := o.Int(0)
		pic.MetaData.ExifOrientation = byte(v)
	}

	xd, xderr := x.Get(exif.PixelXDimension)
	if xderr == nil {
		v, _ := xd.Int(0)
		pic.MetaData.ExifXdimension = uint32(v)
	}
	yd, yderr := x.Get(exif.PixelYDimension)
	if yderr == nil {
		v, _ := yd.Int(0)
		pic.MetaData.ExifYdimension = uint32(v)
	}
	return nil
}

// CreateThumbnail create thumbnail
func (pic *PictureBinary) CreateThumbnail() error {
	if strings.HasPrefix(pic.MetaData.MIMEType, "image") {
		thmb, w, h, err := resizePicture(pic.Data.Media, 200)
		if err != nil {
			adatypes.Central.Log.Debugf("Error generating thumbnail: %v", err)
			return err
		}
		pic.Data.Thumbnail = thmb
		pic.MetaData.Width = w
		pic.MetaData.Height = h
		// pic.Data.ChecksumThumbnail = createMd5(pic.Data.Thumbnail)
		// adatypes.Central.Log.Debugf("Thumbnail checksum", pic.Data.ChecksumThumbnail)
	} else {
		adatypes.Central.Log.Debugf("No image, skip thumbnail generation ....")
	}
	return nil

}

// ReadDatabase read picture binary from database
func (pic *PictureBinary) ReadDatabase(connection *adabas.Connection, hash, repository string) (err error) {
	request, rerr := connection.CreateMapReadRequest(PictureBinary{})
	if rerr != nil {
		fmt.Println("Error create request", rerr)
		err = rerr
		return
	}
	err = request.QueryFields("Data")
	if err != nil {
		return
	}
	result, resErr := request.ReadLogicalWith("Md5=" + hash)
	if resErr != nil {
		fmt.Println("Error reading ISN order", resErr)
		err = resErr
		return
	}
	if len(result.Data) == 0 {
		return fmt.Errorf("no data found")
	}
	resultPic := result.Data[0].(*PictureBinary)
	*pic = *resultPic
	return
}

type entry struct {
	fillType string
	imgName  string
	text     string
}

var entries []entry

func loadMovie(fileName string, ada *adabas.Adabas) error {
	fmt.Println("Load movie", fileName)
	return nil
}

func (pic *PictureBinary) storeRecord(insert bool, ps *PictureConnection) (err error) {
	fileName := pic.FileName
	suffix := fileName[strings.LastIndex(fileName, ".")+1:]
	suffix = strings.ToLower(suffix)
	switch suffix {
	case "jpg", "jpeg", "gif":
		pic.MetaData.MIMEType = "image/" + suffix
		pic.ExtractExif()
		terr := pic.CreateThumbnail()
		if terr != nil {
			adatypes.Central.Log.Debugf("Create thumbnail error %v", terr)
			return terr
		}
		if pic.MetaData.Height > pic.MetaData.Width {
			pic.MetaData.Fill = "1"
		} else {
			pic.MetaData.Fill = "2"
		}
	case "m4v", "mov":
		pic.MetaData.MIMEType = "video/mp4"
		pic.MetaData.Fill = "0"
	default:
		panic("Unknown suffix " + suffix)
	}
	adatypes.Central.Log.Debugf("Done set value to Picture, searching ...")

	if pic.MetaData.ChecksumPicture == "" {
		panic(fmt.Sprintf("Checksum picture empty: %v", pic.MetaData.PictureLocation))
	}
	fmt.Printf("Store data %s %v\n", pic.MetaData.ChecksumPicture, pic.MetaData.PictureLocation)
	if insert {
		//fmt.Println("Store record metadata ....", p.MetaData.Md5)
		err = ps.store.StoreData(pic.MetaData)
	} else {
		// fmt.Println("Update record ....", p.MetaData.Md5, "with ISN", p.MetaData.Index)
		err = ps.store.UpdateData(pic.MetaData)
	}
	if err != nil {
		fmt.Printf("Error storing record metadata: %v (%s)", err, pic.MetaData.ChecksumPicture)
		return err
	}
	fmt.Printf("Stored metadata %s into ISN=%d\n", pic.MetaData.ChecksumPicture, pic.MetaData.Index)
	pic.Data.ChecksumPicture = pic.MetaData.ChecksumPicture
	pic.Data.Index = pic.MetaData.Index
	if !ps.ChecksumRun {
		// ok, err = ps.checkPicture(pictureKey)
		// if err == nil && !ok {
		// fmt.Println("Store data storage")
		// fmt.Println("Update record data ....", p.Data.Md5, " of size ", len(p.Data.Media))
		err = ps.storeData.UpdateData(pic.Data, true)
		if err != nil {
			fmt.Println("Error updating record data:", err)
			return err
		}
		err = ps.connection.EndTransaction()
		if err != nil {
			panic("Data write: end of transaction error: " + err.Error())
		}
	}
	//}
	// fmt.Println("Update record thumbnail ....", p.Data.Md5)
	err = ps.storeThumb.UpdateData(pic.Data)
	if err != nil {
		fmt.Printf("Updating thumbnail request error %d: %v\n", pic.Data.Index, err)
		return err
	}
	adatypes.Central.Log.Debugf("Updated record into ISN=%d ChecksumPicture=%s", pic.MetaData.Index, pic.Data.ChecksumPicture)
	err = ps.store.EndTransaction()
	if err != nil {
		panic("End of transaction error: " + err.Error())
	}
	Statistics.Loaded++
	return nil
}

func (pic *PictureBinary) checkAndAddFile(ps *PictureConnection, fileName, directoryName string) (err error) {
	result, err := ps.readAddAndCheck.ReadLogicalWith("CP=" + pic.Data.ChecksumPicture)
	if err != nil {
		fmt.Printf("Error checking PictureHash=%s: %v\n", pic.Data.ChecksumPicture, err)
		panic("Read error " + err.Error())
	}
	if result.NrRecords() != 1 {
		panic("Error receiving nr records for checking")
	}
	pm := result.Data[0].(*PictureMetadata)
	ph := make(map[string]*PictureLocation)
	for _, p := range pm.PictureLocation {
		if p.PictureDirectory == directoryName && p.PictureHost == Hostname {
			Statistics.Found++
			return nil
		}

		x := p.PictureDirectory + "-" + p.PictureHost
		if _, ok := ph[x]; !ok {
			ph[x] = p
		}
	}
	location := createPictureLocation(fileName, directoryName)
	if len(pm.PictureLocation) == len(ph) {
		pm.PictureLocation = append(pm.PictureLocation, location)
	} else {
		if ps.Verbose {
			fmt.Println("Duplicate found for ", location.PictureDirectory, len(pm.PictureLocation), len(ph))
		}
		newPLList := make([]*PictureLocation, 0)
		for _, p := range ph {
			newPLList = append(newPLList, p)
		}
		newPLList = append(newPLList, location)
		for i := 0; i < len(pm.PictureLocation)-len(ph)-1; i++ {
			newPLList = append(newPLList, &PictureLocation{})
		}
		pm.PictureLocation = newPLList
		Statistics.Duplicated++
	}

	err = ps.storeEntries.UpdateData(pm)
	if err != nil {
		return err
	}
	err = ps.storeEntries.EndTransaction()
	if err != nil {
		panic("End of transaction error: " + err.Error())
	}
	Statistics.Added++

	return nil
}

func createPictureLocation(pictureName, directoryName string) *PictureLocation {
	picShortName := re.FindStringSubmatch(pictureName)[1]
	// var re = regexp.MustCompile(`(?m).*/([^/]*)/.*`)
	// d := re.FindStringSubmatch(pictureName)[1]
	// fmt.Println("Directory: ", picShortName, re.FindStringSubmatch(pictureName))
	picHash := createMd5([]byte(pictureName))
	return &PictureLocation{PictureName: picShortName, PictureHash: picHash, PictureDirectory: directoryName,
		PictureHost: Hostname}
}
