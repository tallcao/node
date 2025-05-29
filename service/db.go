package service

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DbConverter struct {
	Id            int64
	SN            string
	ConverterType int64
	CanNo         int64

	Guid       *string
	DeviceType *int64
}

type DbSerial struct {
	Id   int64
	Addr int64

	Guid       *string
	DeviceType *int64
}

type DbPanel struct {
	Id int64
	SN string
}

func DbAddConverterDevice(db *sql.DB, guid string, t int, converterSN string) error {

	_, err := db.Exec("update converter set guid=?,device_type=? where sn=? ", guid, t, converterSN)
	if err != nil {
		return err
	}

	return nil
}

func DbAddSerialDevice(db *sql.DB, guid string, addr int, deviceType int) error {

	_, err := db.Exec("INSERT INTO serial(guid,addr,device_type) VALUES(?,?,?) ON CONFLICT(addr) DO UPDATE SET guid=excluded.guid, device_type=excluded.device_type", guid, addr, deviceType)
	if err != nil {
		return err
	}

	return nil
}

func DbDeleteSerialDevice(db *sql.DB, guid string) error {

	_, err := db.Exec("delete from serial where guid=?", guid)
	if err != nil {
		return err
	}

	return nil
}

func DbDeleteConverterDevice(db *sql.DB, guid string) error {

	_, err := db.Exec("update converter set guid=null,device_type=null where guid=?", guid)
	if err != nil {
		return err
	}

	return nil
}

func DbAddConverter(db *sql.DB, item *DbConverter) (*DbConverter, error) {

	res, err := db.Exec("INSERT INTO converter(sn,converter_type,can_no) VALUES(?,?,?)", item.SN, item.ConverterType, item.CanNo)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	result := item
	result.Id = id

	return result, nil
}

func DbGetConverters(db *sql.DB, filter string) ([]*DbConverter, error) {
	var result = make([]*DbConverter, 0, 256)

	query := "SELECT id,sn,converter_type,can_no,guid,device_type FROM converter"

	switch filter {
	case "can":
		query += " where can_no > 0"
	case "lora":
		query += " where can_no = 0"
	}
	orderBy := " order by id"

	query += orderBy

	rows, err := db.Query(query)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item DbConverter
		err = rows.Scan(&item.Id, &item.SN, &item.ConverterType, &item.CanNo, &item.Guid, &item.DeviceType)
		if err != nil {
			return result, err
		}
		result = append(result, &item)

	}

	return result, nil
}

func DbGetSerials(db *sql.DB) ([]*DbSerial, error) {
	var result = make([]*DbSerial, 0, 128)

	query := "SELECT id, addr, device_type, guid FROM serial"

	rows, err := db.Query(query)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item DbSerial
		err = rows.Scan(&item.Id, &item.Addr, &item.DeviceType, &item.Guid)
		if err != nil {
			return result, err
		}
		result = append(result, &item)

	}

	return result, nil
}
