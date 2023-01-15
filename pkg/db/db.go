package db

import "time"

type DB interface {
	SetKey(key string, endTime time.Time) error
	Exists(key string) (bool, error)
	Exit() error
}
