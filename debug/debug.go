package debug

import (
  "flag"
)

var Debugging = false

func init() {
  flag.BoolVar(&Debugging, "debug", false, "Specifies whether or not logging is in debug mode")
}
