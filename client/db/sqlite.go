package db

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"orange-clipboard/common/resource"
)

type ClipboardModel struct {
	ID         int
	Msg        string
	MsgType    int
	CreateTime int64
}

var DB *sql.DB

const CreateSQL = `
			CREATE TABLE IF NOT EXISTS clipboard (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				msg TEXT,
				msg_type INTEGER,
				create_time INTEGER
        )`

const (
	MsgTextType  = 1
	MsgImageType = 2
)

func InitDB() error {
	db, err := sql.Open("sqlite3", "clipboard.sqlite")
	if err != nil {
		resource.Logger.Error("初始化sqlite失败", zap.Error(err))
		return err
	}
	DB = db
	_, err = DB.Exec(CreateSQL)
	if err != nil {
		resource.Logger.Error("初始化clipboard失败", zap.Error(err))
		return err
	}
	return nil
}

func Query(limit, offset int) []ClipboardModel {
	clipboardModels := make([]ClipboardModel, 0)
	rows, err := DB.Query("SELECT * FROM clipboard ORDER BY create_time DESC LIMIT ?,?  ", limit, offset)
	if err != nil {
		resource.Logger.Error("查询历史剪贴数据失败", zap.Error(err))
		return clipboardModels
	}
	for rows.Next() {
		var (
			id         int
			msg        string
			msgType    int
			createTime int64
		)
		err := rows.Scan(&id, &msg, &msgType, &createTime)
		if err != nil {
			resource.Logger.Error("row scan error", zap.Error(err))
			return clipboardModels
		}
		clipboardModels = append(clipboardModels, ClipboardModel{
			ID:         id,
			Msg:        msg,
			MsgType:    msgType,
			CreateTime: createTime,
		})
	}
	return clipboardModels
}

func Insert(data ClipboardModel) int64 {
	result, err := DB.Exec("INSERT INTO clipboard(msg,msg_type,create_time) VALUES (?,?,?)", data.Msg, data.MsgType, data.CreateTime)
	if err != nil {
		resource.Logger.Error("insert error", zap.Error(err))
		return -1
	}
	id, _ := result.LastInsertId()
	return id
}

func InsertOrUpdate(data ClipboardModel) int64 {
	id := 0
	rows, err := DB.Query("SELECT id FROM clipboard where msg = ? ORDER BY create_time desc limit 1", data.Msg)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	if rows.Next() {
		err := rows.Scan(&id)
		if err != nil {
			return 0
		}
	}
	rows.Close()
	fmt.Println(id)
	if id == 0 {
		return Insert(data)
	} else {
		data.ID = id
		return Update(data)
	}
}

func Update(data ClipboardModel) int64 {
	result, err := DB.Exec("UPDATE clipboard set msg = ?,create_time = ? WHERE id = ?", data.Msg, data.CreateTime, data.ID)
	if err != nil {
		resource.Logger.Error("Update error", zap.Error(err))
		return -1
	}

	row, _ := result.RowsAffected()
	return row
}
