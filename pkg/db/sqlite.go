package db

import (
	"os"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const sqliteType = "sqlite"

type sqliteConfig struct {
	DBFile string `mapstructure:"db_file"`
}

type sqliteHandler struct {
	db   *gorm.DB
	lock sync.Mutex
}

type event struct {
	Event  string
	Status int64
}

func (*event) tableName() string {
	return "events"
}

func InitSqlite(dbFile string) (*sqliteHandler, error) {
	f, err := os.OpenFile(dbFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}
	f.Close()
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	p := &event{}
	db.Table(p.tableName()).AutoMigrate(&p)
	return &sqliteHandler{
		db: db,
	}, nil
}

func (s *sqliteHandler) SetKey(key string, value interface{}) error {
	r := &event{}
	results := s.db.Table(r.tableName()).Where("event = ?", key).First(r)
	if results.Error != nil {
		if results.Error == gorm.ErrRecordNotFound {
			results := s.db.Table(r.tableName()).Create(&event{
				Event:  key,
				Status: value.(int64),
			})
			if results.Error != nil {
				return results.Error
			}
		}
	} else {
		if err := s.db.Model(&r).Where("event = ?", key).
			UpdateColumn("status", value.(int64)).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *sqliteHandler) Exists(key string) (bool, error) {
	var exists bool
	if err := s.db.Model(&event{}).
		Select("count(*) > 0").
		Where("event = ?", key).
		Find(&exists).
		Error; err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	return true, nil
}

func (s *sqliteHandler) GetValue(key string) (interface{}, error) {
	var status int64
	if err := s.db.Model(&event{}).Where("event = ?", key).
		Select("status").Scan(&status).Error; err != nil {
		return 0, err
	}
	return status, nil
}

func (s *sqliteHandler) GetEventByValue(value int64) ([]string, error) {
	var events []string
	if err := s.db.Model(&event{}).Where("status = ?", value).
		Select("event").Scan(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (s *sqliteHandler) Exit() error {
	return nil
}
