package debug

import (
  "fmt"
  "sync"
  "time"
)

var mu sync.Mutex
var Debug = false

func Log(format string, params ...interface{}) {
  if !Debug {
    return
  }
  mu.Lock()
  defer mu.Unlock()
  params = append([]interface{}{time.Now().Format(time.Kitchen)}, params...)
  fmt.Printf("[%s] "+format+"\n", params...)
}
