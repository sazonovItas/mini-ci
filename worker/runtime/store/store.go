package store

type Store interface {
	Location(keys ...string) (path string)
	Exists(keys ...string) (exist bool, err error)
	Get(keys ...string) (content []byte, err error)
	List(keys ...string) (entries []string, err error)
	Set(content []byte, keys ...string) error
	Append(content []byte, keys ...string) error
	Delete(keys ...string) error
}

type Locker interface {
	Lock() error
	Unlock() error
	WithLock(f func() error) error
}
