package storage

import sync "sync"

var DataMap = map[string]*Ticker{}

var DataMapMutex sync.Mutex
