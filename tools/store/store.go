package store

import (
	"fmt"
	"os"
	"strings"

	"github.com/SoftwareAG/adabas-go-api/adabas"
	"github.com/SoftwareAG/adabas-go-api/adatypes"
)

const PictureNameSN = "PN"

type DatabaseReference struct {
	Dbid        string
	PictureFile adabas.Fnr
	AlbumFile   adabas.Fnr
}

// InitStorePictureBinary init store picture connection
func InitStorePictureBinary(shortenName bool, dbReference *DatabaseReference, connection *adabas.Connection) (ps *PictureConnection, err error) {
	ps = &PictureConnection{ShortenName: shortenName, ChecksumRun: false}
	ps.dbReference = dbReference
	ps.connection = connection
	ps.store, err = connection.CreateMapStoreRequest((*PictureMetadata)(nil))
	if err != nil {
		connection.Close()
		return nil, err
	}
	err = ps.store.StoreFields("*")
	if err != nil {
		return nil, err
	}
	ps.storeData, err = ps.connection.CreateMapStoreRequest((*PictureData)(nil))
	if err != nil {
		ps.connection.Close()
		return nil, err
	}
	err = ps.storeData.StoreFields("M5,DP")
	if err != nil {
		return nil, err
	}
	ps.storeThumb, err = ps.connection.CreateMapStoreRequest((*PictureData)(nil))
	if err != nil {
		ps.connection.Close()
		return nil, err
	}
	err = ps.storeThumb.StoreFields("M5,CP,DT")
	// "Md5,ChecksumPicture,ChecksumThumbnail,Thumbnail")
	if err != nil {
		return nil, err
	}
	ps.storeEntries, err = ps.connection.CreateMapStoreRequest((*PictureMetadata)(nil))
	if err != nil {
		ps.connection.Close()
		return nil, err
	}
	err = ps.storeEntries.StoreFields("PL")
	if err != nil {
		return nil, err
	}
	ps.readFileNameCheck, err = ps.connection.CreateMapReadRequest((*PictureMetadata)(nil))
	if err != nil {
		ps.connection.Close()
		return nil, err
	}
	err = ps.readFileNameCheck.QueryFields("PM")
	if err != nil {
		ps.connection.Close()
		return nil, err
	}
	ps.readAddAndCheck, err = ps.connection.CreateMapReadRequest((*PictureMetadata)(nil))
	if err != nil {
		ps.connection.Close()
		return nil, err
	}
	err = ps.readAddAndCheck.QueryFields("CP,PL")
	if err != nil {
		ps.connection.Close()
		return nil, err
	}
	ps.readMediaCheck, err = ps.connection.CreateMapReadRequest((*PictureData)(nil))
	if err != nil {
		ps.connection.Close()
		return nil, err
	}
	err = ps.readMediaCheck.QueryFields("CP")
	if err != nil {
		ps.connection.Close()
		return nil, err
	}
	ps.histCheck, err = ps.connection.CreateMapReadRequest((*PictureMetadata)(nil))
	if err != nil {
		ps.connection.Close()
		return nil, err
	}
	return
}

// LoadPicture load picture data into database
func (ps *PictureConnection) LoadPicture(insert bool, fileName string) error {
	fs := strings.Split(fileName, string(os.PathSeparator))
	pictureName := fileName
	directoryName := fileName
	if !ps.ShortenName {
		if fs[len(fs)-2] == "img" {
			pictureName = fs[len(fs)-3] + "/" + fs[len(fs)-1]
		} else {
			pictureName = fs[len(fs)-2] + "/" + fs[len(fs)-1]
		}
		fmt.Printf("Shorten name from %s to %s\n", fileName, pictureName)
	}
	pictureKey := createMd5([]byte(pictureName))
	var err error
	var ok bool
	ok, err = ps.pictureFileAvailable(pictureKey)
	if err != nil {
		adatypes.Central.Log.Debugf("Availability check error %v", err)
		return err
	}
	empty := checkEmpty(fileName)
	if empty {
		adatypes.Central.Log.Debugf(pictureName, "-> picture file empty")
		Statistics.Empty++
		if ok {
			fmt.Printf("Remove empty file from database: %s(%s)\n", fileName, pictureKey)
			ps.DeleteMd5(pictureKey)
		}
		return nil
	}
	Statistics.Checked++
	if ok && insert {
		adatypes.Central.Log.Debugf("%s -> picture name already loaded", pictureName)
		Statistics.Found++
		return nil
	}
	pictureLocation := createPictureLocation(pictureName, directoryName)
	p := PictureBinary{FileName: fileName,
		MetaData: &PictureMetadata{Md5: pictureKey}, MaxBlobSize: ps.MaxBlobSize}
	p.MetaData.PictureLocation = append(p.MetaData.PictureLocation, pictureLocation)
	err = p.LoadFile()
	if err != nil {
		adatypes.Central.Log.Debugf("Load file error %v", err)
		return err
	}

	mediaAvailable, merr := ps.pictureMediaAvailable(p.Data.ChecksumPicture)
	if merr != nil {
		adatypes.Central.Log.Debugf("Availability data check error %v", merr)
		return merr
	}
	if !mediaAvailable {
		info := "Loading"
		if !insert {
			info = "Updating"
		}
		fmt.Printf("%s picture ... %s\r", info, fileName)
		p.storeRecord(insert, ps)
	} else {
		fmt.Printf("Skipping picture ... %s [%s]\r", fileName, p.Data.ChecksumPicture)
		p.checkAndAddFile(ps, fileName, directoryName)
	}
	return nil
}

// DeleteMd5 delete picture key
func (psx *PictureConnection) DeleteMd5(key string) error {
	result, err := psx.readFileNameCheck.ReadLogicalWith("Md5=" + key)
	if err != nil {
		fmt.Printf("Error checking Md5=%s: %v\n", key, err)
		panic("Read error " + err.Error())
		//		return false, err
	}

	deleteRequest, err := psx.connection.CreateDeleteRequest(psx.dbReference.PictureFile)
	defer deleteRequest.BackoutTransaction()
	if err != nil {
		return err
	}
	for _, r := range result.Values {
		deleteRequest.Delete(r.Isn)
	}
	return nil
}

// DeleteIsn delete image Isn
func (psx *PictureConnection) DeleteIsn(isn adatypes.Isn) error {
	fmt.Printf("Delete image with ISN=%d\n", isn)
	deleteRequest, err := psx.connection.CreateDeleteRequest(psx.dbReference.PictureFile)
	defer deleteRequest.BackoutTransaction()
	if err != nil {
		return err
	}
	err = deleteRequest.Delete(isn)
	if err != nil {
		return err
	}
	err = deleteRequest.EndTransaction()
	return err
}

// DeletePath delete image given with path
func (psx *PictureConnection) DeletePath(path string) error {
	if path == "" {
		return nil
	}
	fmt.Printf("Delete image with path=%s\n", path)
	readRequest, err := psx.connection.CreateFileReadRequest(psx.dbReference.PictureFile)
	if err != nil {
		return err
	}
	readRequest.QueryFields("")
	result, resErr := readRequest.ReadLogicalWith(PictureNameSN + "=" + path)
	if resErr != nil {
		return resErr
	}
	if result.NrRecords() != 1 {
		fmt.Printf("Found more then one or no record: %d\n", result.NrRecords())
		return fmt.Errorf("Found more then one record")
	}
	for _, record := range result.Values {
		psx.DeleteIsn(record.Isn)
	}
	return nil
}
