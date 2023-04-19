package db

type DB interface {
	SetKey(key string, value interface{}) error
	Exists(key string) (bool, error)
	GetValue(key string) (interface{}, error)
	GetEventByValue(value int64) ([]string, error)
	Exit() error
}
