package main

import (
	"fmt"
	"testing"
	"time"
)

func TestBoardGeneration(t *testing.T) {
	for _, diff := range []Difficulty{Easy, Medium, Hard} {
		t.Run(diff.String(), func(t *testing.T) {
			start := time.Now()
			b, opt := NewRandomBoard(diff)
			elapsed := time.Since(start)
			lo, hi := difficultyRange(diff)

			fmt.Printf("  %s: optimal=%d moves, generated in %v\n", diff, opt, elapsed)

			if b == nil {
				t.Fatal("board is nil")
			}
			if opt < lo || opt >= hi {
				t.Fatalf("optimal %d outside range [%d, %d)", opt, lo, hi)
			}

			// Verify piece counts.
			smalls, mediums, larges := 0, 0, 0
			for _, p := range b.Pieces {
				switch p.Kind {
				case Small:
					smalls++
				case Vertical, Horizontal:
					mediums++
				case Large:
					larges++
				}
			}
			if smalls != 4 {
				t.Fatalf("expected 4 small pieces, got %d", smalls)
			}
			if mediums != 5 {
				t.Fatalf("expected 5 medium pieces, got %d", mediums)
			}
			if larges != 1 {
				t.Fatalf("expected 1 large piece, got %d", larges)
			}

			// Verify no overlaps — count occupied cells.
			var occupied [BoardW][BoardH]bool
			cells := 0
			for _, p := range b.Pieces {
				for _, c := range p.Cells() {
					if occupied[c[0]][c[1]] {
						t.Fatalf("overlap at (%d, %d)", c[0], c[1])
					}
					occupied[c[0]][c[1]] = true
					cells++
				}
			}
			if cells != 18 {
				t.Fatalf("expected 18 occupied cells, got %d", cells)
			}
		})
	}
}
