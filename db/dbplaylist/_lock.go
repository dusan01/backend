package dbplaylist

// import (
//   "hybris/db"
//   "sync"
// )

// var lockMutexes = map[db.Id]*sync.Mutex{}

// func Lock(id db.Id) {
//   if _, ok := lockMutexes[id]; !ok {
//     lockMutexes[id] = &sync.Mutex{}
//   }

//   lockMutexes[id].Lock()
// }

// func Unlock(id db.Id) {
//   if _, ok := lockMutexes[id]; !ok {
//     return
//   }
//   lockMutexes[id].Unlock()
// }

// func LockGet(id db.Id) (*Playlist, error) {
//   Lock(id)
//   return GetId(id)
// }
