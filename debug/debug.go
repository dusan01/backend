package debug

import (
  "fmt"
  "sync"
)

var mu sync.Mutex
var Debug = false

func Log(format string, params ...interface{}) {
  if !Debug {
    return
  }
  mu.Lock()
  defer mu.Unlock()
  fmt.Printf(format+"\n", params...)
}
