package search

import (
  "gopkg.in/mgo.v2/bson"
  "hybris/db"
  "regexp"
  "sort"
  "strings"
  "sync"
)

type Community struct {
  *db.Community
  Host              string `json:"host"`
  Population        int    `json:"population"`
  FormatHost        string `json:"-"`
  FormatName        string `json:"-"`
  FormatDescription string `json:"-"`
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

func Communities(query string, sbp bool) []Community {
  query = strings.ToLower(query)
  query = regexp.MustCompile(" +").ReplaceAllString(removeSymbols(strings.ToLower(strings.TrimSpace(query))), " ")

  if len(query) < 1 {
    return []Community{}
  }

  queries := removeDuplicateQueries(strings.Split(query, " "))
  matches := Match(queries, sbp)

  return matches
}

func Match(queries []string, sbp bool) []Community {
  results := make(map[int][]Community)

  var wg sync.WaitGroup
  var mutex sync.Mutex

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
        results[total] = append(results[total], community)
        mutex.Unlock()
      }
    }(community)
  }

  wg.Wait()

  if sbp {
    return populationSortResults(results)
  }

  return relevanceSortResults(results)
}

func populationSortResults(results map[int][]Community) []Community {
  r := []Community{}
  for _, c := range results {
    for _, v := range c {
      r = append(r, v)
    }
  }
  for i := 1; i < len(r); i++ {
    v := r[i]
    j := i - 1
    for j >= 0 && r[j].Population <= v.Population {
      r[j+1] = r[j]
      j = j - 1
    }
    r[j+1] = v
  }
  return r
}

func relevanceSortResults(results map[int][]Community) []Community {
  matches := make([]int, 0)
  for match, _ := range results {
    matches = append(matches, match)
  }
  sort.Sort(sort.Reverse(sort.IntSlice(matches))) // Sorts the slice from 0..
  payload := []Community{}
  for _, match := range matches {
    for _, item := range results[match] {
      payload = append(payload, item)
    }
  }
  return payload
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
