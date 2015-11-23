package debug

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var (
	logMutex   sync.Mutex
	logChannel chan log
)

type log struct {
	Format string
	Params []interface{}
}

func init() {
	if _, err := os.Stat("_logs"); os.IsNotExist(err) {
		if err := os.Mkdir("_logs", 0777); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}

	logChannel = make(chan log, 1000)
	go logListener(logChannel)
}

func Log(format string, params ...interface{}) {
	logChannel <- log{format, params}
}

func logListener(channel chan log) {
	for data := range logChannel {
		t := time.Now()
		data.Params = append([]interface{}{t.Format(time.RFC3339)}, data.Params...)
		message := fmt.Sprintf("[%s] "+data.Format+"\n", data.Params...)
		if Debugging {
			fmt.Printf(message)
		}

		file, err := os.OpenFile("_logs/debug-"+t.Format("2006-01-02")+".log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
		if err != nil {
			fmt.Println("[WARNING] FAILED TO SAVE DEBUG LOG")
			continue
		}

		io.WriteString(file, message)
		file.Close()
	}
}
