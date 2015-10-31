package debug

import (
  "fmt"
  "io"
  "os"
  "sync"
  "time"
)

func init() {
  if _, err := os.Stat("_logs"); os.IsNotExist(err) {
    if err := os.Mkdir("_logs", 0777); err != nil {
      panic(err)
    }
  } else if err != nil {
    panic(err)
  }
}

var logMutex sync.Mutex

func Log(format string, params ...interface{}) {
  logMutex.Lock()
  defer logMutex.Unlock()
  params = append([]interface{}{time.Now().Format(time.RFC3339)}, params...)
  payload := fmt.Sprintf("[%s] "+format+"\n", params...)
  if Debugging {
    fmt.Print(payload)
  }
  logDate := time.Now().Format("2006-01-02")
  file, err := os.OpenFile("_logs/debug-"+logDate+".log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
  if err != nil {
    return
  }
  defer file.Close()

  _, err = io.WriteString(file, payload)
  if err != nil {
    return
  }
}
