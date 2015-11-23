package frontendaction

type Frontend interface {
  Lock()
  Unlock()
  Send([]byte)
  Terminate()
}
