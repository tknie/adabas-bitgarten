package store

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/SoftwareAG/adabas-go-api/adabas"
	"github.com/SoftwareAG/adabas-go-api/adatypes"
)

// PictureConnection picture connection handle
type PictureConnection struct {
	conn       *adabas.Connection
	store      *adabas.StoreRequest
	storeData  *adabas.StoreRequest
	storeThumb *adabas.StoreRequest
	readCheck  *adabas.ReadRequest
}

// InitStorePictureBinary init store picture connection
func InitStorePictureBinary() (ps *PictureConnection, err error) {
	ps = &PictureConnection{}
	ps.conn, err = adabas.NewConnection("acj;map")
	if err != nil {
		return nil, err
	}
	ps.store, err = ps.conn.CreateMapStoreRequest((*PictureMetadata)(nil))
	if err != nil {
		ps.conn.Close()
		return nil, err
	}
	err = ps.store.StoreFields("*")
	if err != nil {
		return nil, err
	}
	ps.storeData, err = ps.conn.CreateMapStoreRequest((*PictureData)(nil))
	if err != nil {
		ps.conn.Close()
		return nil, err
	}
	err = ps.storeData.StoreFields("Md5,ChecksumPicture,Media")
	if err != nil {
		return nil, err
	}
	ps.storeThumb, err = ps.conn.CreateMapStoreRequest((*PictureData)(nil))
	if err != nil {
		ps.conn.Close()
		return nil, err
	}
	err = ps.storeThumb.StoreFields("Md5,ChecksumThumbnail,Thumbnail")
	if err != nil {
		return nil, err
	}
	ps.readCheck, err = ps.conn.CreateMapReadRequest("PictureMetadata")
	if err != nil {
		ps.conn.Close()
		return nil, err
	}
	err = ps.readCheck.QueryFields("Md5")
	if err != nil {
		ps.conn.Close()
		return nil, err
	}
	return
}

func (ps *PictureConnection) LoadPicture(insert bool, fileName string, ada *adabas.Adabas) error {
	fs := strings.Split(fileName, string(os.PathSeparator))
	pictureName := ""
	if fs[len(fs)-2] == "img" {
		pictureName = fs[len(fs)-3] + "/" + fs[len(fs)-1]
	} else {
		pictureName = fs[len(fs)-2] + "/" + fs[len(fs)-1]
	}
	pictureKey := createMd5([]byte(pictureName))
	var err error
	var ok bool
	ok, err = ps.available(pictureKey)
	if err != nil {
		return err
	}
	if ok && insert {
		fmt.Println(pictureName, "-> picture name already loaded")
		return nil
	}
	info := "Loading"
	if !insert {
		info = "Updating"
	}
	fmt.Printf("%s picture ... %s\n", info, fileName)
	fmt.Println("-> load picture name ...", pictureName, "Md5=", pictureKey)
	var re = regexp.MustCompile(`(?m)([^/]*)/.*`)
	d := re.FindStringSubmatch(pictureName)[1]
	fmt.Println("Directory: ", d)
	p := PictureBinary{FileName: fileName, MetaData: &PictureMetadata{PictureName: pictureName, Directory: d, Md5: pictureKey}}
	err = p.LoadFile()
	if err != nil {
		return err
	}

	suffix := fileName[strings.LastIndex(fileName, ".")+1:]
	switch suffix {
	case "jpg", "jpeg", "gif":
		p.MetaData.MIMEType = "image/" + suffix
		fmt.Println("Len", len(p.Data.Media))
		terr := p.CreateThumbnail()
		if terr != nil {
			return terr
		}
		fmt.Println("Len", len(p.Data.Media))
		if p.MetaData.Height > p.MetaData.Width {
			p.MetaData.Fill = "1"
		} else {
			p.MetaData.Fill = "2"
		}
	case "m4v", "mov":
		p.MetaData.MIMEType = "video/mp4"
		p.MetaData.Fill = "0"
	default:
		panic("Unknown suffix " + suffix)
	}
	adatypes.Central.Log.Debugf("Done set value to Picture, searching ...")

	if insert {
		fmt.Println("Store record metadata ....", p.MetaData.Md5)
		err = ps.store.StoreData(p.MetaData)
	} else {
		fmt.Println("Update record ....", p.MetaData.Md5, "with ISN", p.MetaData.Index)
		err = ps.store.UpdateData(p.MetaData)
	}
	fmt.Println("Stored metadata into ISN=", p.MetaData.Index)
	if err != nil {
		fmt.Println("Error storing record metadata:", err)
		return err
	}
	p.Data.Md5 = p.MetaData.Md5
	p.Data.Index = p.MetaData.Index
	fmt.Println("Update record data ....", p.Data.Md5, " of size ", len(p.Data.Media))
	err = ps.storeData.UpdateData(p.Data, true)
	if err != nil {
		fmt.Println("Error storing record data:", err)
		return err
	}
	ps.conn.EndTransaction()
	fmt.Println("Update record thumbnail ....", p.Data.Md5)
	err = ps.storeThumb.UpdateData(p.Data)
	if err != nil {
		fmt.Printf("Store request error %v\n", err)
		return err
	}
	fmt.Println("Updated record into ISN=", p.MetaData.Index)
	err = ps.store.EndTransaction()
	if err != nil {
		panic("End of transaction error: " + err.Error())
	}
	validateUsingMap(ada, adatypes.Isn(p.MetaData.Index))
	return nil
}

func (ps *PictureConnection) available(key string) (bool, error) {
	//fmt.Println("Check Md5=" + key)
	result, err := ps.readCheck.HistogramWith("Md5=" + key)
	if err != nil {
		fmt.Printf("Error checking Md5=%s: %v\n", key, err)
		panic("Read error " + err.Error())
		//		return false, err
	}
	// result.DumpValues()
	if len(result.Values) > 0 || len(result.Data) > 0 {
		fmt.Printf("Md5=%s is available\n", key)
		return true, nil
	}
	fmt.Printf("Md5=%s is not loaded\n", key)
	return false, nil
}

// Close connection
func (ps *PictureConnection) Close() {
	if ps != nil && ps.conn != nil {
		ps.conn.Close()
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
	fmt.Printf("ISN=%d. name=%s len=%d\n     MD5 data=<%s> expected=<%s>\n", record.Isn, fileName, vLen, md, smd)
	if md != smd {
		fmt.Println("Record checksum error", record.Isn)
		os.Exit(255)
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
