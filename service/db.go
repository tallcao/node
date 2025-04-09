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

type DbPanel struct {
	Id int64
	SN string
}

const dbFile = "/home/root/node/node.db"

func DbAddConverterDevice(guid string, t int, converterSN string) error {

	db, err := sql.Open("sqlite3", dbFile)

	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("update converter set guid=?,device_type=? where sn=? ", guid, t, converterSN)
	if err != nil {
		return err
	}

	return nil
}

func DbAddSerialDevice(guid string, addr int, deviceType int) error {

	db, err := sql.Open("sqlite3", dbFile)

	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO serial(guid,addr,device_type) VALUES(?,?,?)", guid, addr, deviceType)
	if err != nil {
		return err
	}

	return nil
}

func DbDeleteSerialDevice(guid string) error {

	db, err := sql.Open("sqlite3", dbFile)

	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("delete from serial where guid=?", guid)
	if err != nil {
		return err
	}

	return nil
}

func DbDeleteConverterDevice(guid string) error {

	db, err := sql.Open("sqlite3", dbFile)

	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("update converter set guid=null,device_type=null where guid=?", guid)
	if err != nil {
		return err
	}

	return nil
}

func DbAddConverter(item *DbConverter) (*DbConverter, error) {

	result := item
	db, err := sql.Open("sqlite3", dbFile)

	if err != nil {
		return result, err
	}
	defer db.Close()

	res, err := db.Exec("INSERT INTO converter(sn,converter_type,can_no) VALUES(?,?,?)", item.SN, item.ConverterType, item.CanNo)
	if err != nil {
		return result, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return result, err
	}

	result.Id = id

	return result, nil
}

func DbGetConverters(filter string) ([]*DbConverter, error) {
	var result = make([]*DbConverter, 0, 256)
	db, err := sql.Open("sqlite3", dbFile)

	if err != nil {
		return nil, err
	}

	defer db.Close()

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

func DbGetPanels() ([]*DbPanel, error) {
	var result = make([]*DbPanel, 0, 64)
	db, err := sql.Open("sqlite3", dbFile)

	if err != nil {
		return nil, err
	}

	defer db.Close()

	query := "SELECT id,sn FROM panel order by id"

	rows, err := db.Query(query)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item DbPanel
		err = rows.Scan(&item.Id, &item.SN)
		if err != nil {
			return result, err
		}
		result = append(result, &item)

	}

	return result, nil
}

func DbAddPanel(item *DbPanel) (*DbPanel, error) {

	result := item
	db, err := sql.Open("sqlite3", dbFile)

	if err != nil {
		return result, err
	}
	defer db.Close()

	res, err := db.Exec("INSERT into panel(sn) values(?)", item.SN)
	if err != nil {
		return result, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return result, err
	}

	result.Id = id

	return result, nil
}
