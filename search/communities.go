package search

import (
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
  "hybris/debug"
  "regexp"
  "strings"
  "sync"
  "time"
)

type Community struct {
  *db.Community
  Host              string `json:"host"`
  Population        int    `json:"population"`
  FormatHost        string `json:"-"`
  FormatName        string `json:"-"`
  FormatDescription string `json:"-"`
}

type Result struct {
  Community Community
  Ranking   int
}

var communities = map[bson.ObjectId]Community{}

func UpsertCommunity(community *db.Community, population int) error {
  user, err := db.GetUser(bson.M{"_id": community.HostId})
  if err != nil {
    return err
  }
  communities[community.Id] = Community{
    community,
    user.DisplayName,
    population,
    strings.ToLower(removeSymbols(user.DisplayName)),
    strings.ToLower(removeSymbols(community.Name)),
    strings.ToLower(removeSymbols(community.Description)),
  }
  return nil
}

func Communities(query string, sbp bool) []Result {
  if len(query) <= 0 {
    results := []Result{}
    for _, c := range communities {
      results = append(results, Result{c, 0})
    }
    return populationSortResults(results)
  }

  query = strings.ToLower(query)
  query = regexp.MustCompile(" +").ReplaceAllString(removeSymbols(strings.ToLower(strings.TrimSpace(query))), " ")

  if len(query) < 1 {
    return []Result{}
  }

  queries := removeDuplicateQueries(strings.Split(query, " "))
  matches := Match(queries, sbp)

  return matches
}

func Match(queries []string, sbp bool) []Result {
  results := []Result{}

  var wg sync.WaitGroup
  var mutex sync.Mutex

  s := time.Now()
  for _, community := range communities {
    wg.Add(1)
    go func(community Community) {
      defer wg.Done()
      total := 0
      for i, query := range queries {
        t := 0
        t += 3 * (len(strings.Split(community.FormatName, query)) - 1)
        t += 2 * (len(strings.Split(community.FormatHost, query)) - 1)
        t += (len(strings.Split(community.FormatDescription, query)) - 1)
        if t > 1 {
          total += t - i
        }
      }
      if total > 0 && total >= len(queries)-1 {
        mutex.Lock()
        results = append(results, Result{community, total})
        mutex.Unlock()
      }
    }(community)
  }

  debug.Log("Took %s to iterate through communities", time.Since(s))

  wg.Wait()

  if sbp {
    return populationSortResults(results)
  }

  return relevanceSortResults(results)
}

func populationSortResults(results []Result) []Result {
  h := 1
  for h < len(results) {
    h = 3*h + 1
  }
  for h >= 1 {
    for i := h; i < len(results); i++ {
      for j := i; j >= h && results[j].Community.Population < results[j-h].Community.Population; j = j - h {
        results[j], results[j-1] = results[j-1], results[j]
      }
    }
    h = h / 3
  }
  return results
}

func relevanceSortResults(results []Result) []Result {
  h := 1
  for h < len(results) {
    h = 3*h + 1
  }
  for h >= 1 {
    for i := h; i < len(results); i++ {
      for j := i; j >= h && results[j].Ranking < results[j-h].Ranking; j = j - h {
        results[j], results[j-1] = results[j-1], results[j]
      }
    }
    h = h / 3
  }
  return results
}

func removeDuplicateQueries(queries []string) []string {
  payload := make([]string, 0)
  for _, val := range queries {
    if len(val) <= 0 {
      continue
    }
    found := false
    for _, d := range payload {
      if val == d {
        found = true
        break
      }
    }
    if !found {
      payload = append(payload, val)
    }
  }
  return payload
}

func removeSymbols(in string) string {
  var payload string
  characters := []rune{'.', '/', '\\', '_', '-', '(', ')', '[', ']', '=', '+', ':', ';', ',', '{', '}', '!', '"', '£',
    '$', '%', '^', '&', '*', '\'', '#', '~', '@', '?', '<', '>', '`', '¬'}
  for _, val := range in {
    found := false
    for _, character := range characters {
      if val == character {
        found = true
        break
      }
    }
    if !found {
      payload += string(val)
    }
  }
  return payload
}
