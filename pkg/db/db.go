package db

type DB interface {
	SetKey(key, name string, value string) error
	Exists(key string) (bool, error)
	GetValue(key string) (string, error)
	GetEventByValue(value string) ([]string, error)
	Exit() error
}
