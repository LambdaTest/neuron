package core

// SyncMap provides the wrapper functions available in sync.Map with defined key-value pair type
type SyncMap interface {
	// Add store the key-value pair in threadsafe env
	Set(key string, value chan struct{}) (chan struct{}, bool)
	// Get gives the value corresponding to the key in threadsafe env
	Get(key string) (chan struct{}, bool)
	// CleanUp deletes the key from map
	CleanUp(key string)
}
