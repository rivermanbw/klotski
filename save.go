package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// PlayerData holds one player's league progress.
type PlayerData struct {
	Scores map[int]int `json:"scores"` // puzzle index -> best score
}

// SaveData is the top-level persistent state written to disk.
type SaveData struct {
	LastPlayer string                 `json:"last_player"`
	Players    map[string]*PlayerData `json:"players"`
}

func saveDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".klotski-puzzle")
}

func savePath() string {
	return filepath.Join(saveDir(), "save.json")
}

func loadSave() *SaveData {
	data, err := os.ReadFile(savePath())
	if err != nil {
		return &SaveData{Players: map[string]*PlayerData{}}
	}
	var s SaveData
	if err := json.Unmarshal(data, &s); err != nil {
		return &SaveData{Players: map[string]*PlayerData{}}
	}
	if s.Players == nil {
		s.Players = map[string]*PlayerData{}
	}
	// Ensure all player score maps are initialised.
	for _, p := range s.Players {
		if p.Scores == nil {
			p.Scores = map[int]int{}
		}
	}
	return &s
}

func (s *SaveData) save() error {
	if err := os.MkdirAll(saveDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(savePath(), data, 0644)
}

func (s *SaveData) player(name string) *PlayerData {
	if p, ok := s.Players[name]; ok {
		return p
	}
	p := &PlayerData{Scores: map[int]int{}}
	s.Players[name] = p
	return p
}

// totalScore returns the sum of best scores across all completed puzzles.
func (p *PlayerData) totalScore() int {
	total := 0
	for _, s := range p.Scores {
		total += s
	}
	return total
}

// completed returns the number of puzzles with a score.
func (p *PlayerData) completed() int {
	return len(p.Scores)
}

// highestUnlocked returns the index of the highest puzzle the player may attempt.
// Puzzle 0 is always unlocked. Completing puzzle N unlocks N+1.
func (p *PlayerData) highestUnlocked() int {
	idx := 0
	for idx < len(presets)-1 {
		if _, ok := p.Scores[idx]; !ok {
			break
		}
		idx++
	}
	return idx
}

// leaderboardEntry is one row in the leaderboard view.
type leaderboardEntry struct {
	Name      string
	Total     int
	Completed int
}

func (s *SaveData) leaderboard() []leaderboardEntry {
	var entries []leaderboardEntry
	for name, p := range s.Players {
		entries = append(entries, leaderboardEntry{
			Name:      name,
			Total:     p.totalScore(),
			Completed: p.completed(),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Total != entries[j].Total {
			return entries[i].Total > entries[j].Total
		}
		return entries[i].Name < entries[j].Name
	})
	return entries
}
