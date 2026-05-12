//go:build spike

// spike-spotted is the Phase 1 pre-merge harness for the m_bSpottedByMask
// reliability check (see .claude/plans/timeline-contact-moments/phase-1/
// 04-tests-spike.md §4). Subscribes to PlayerSpottersChanged and prints
// `tick / spotted / spotters` triples so the operator can eyeball a recent
// MM/Faceit demo and confirm the bitfield reads land on real engagements,
// not garbage.
//
// Build with `-tags spike` so this is not part of the default binary.
//
//	go build -tags spike -o /tmp/spike-spotted ./cmd/spike-spotted
//	/tmp/spike-spotted /path/to/recent.dem | head -200
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: spike-spotted <demo.dem>")
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	p := demoinfocs.NewParser(f)
	defer func() { _ = p.Close() }()

	p.RegisterEventHandler(func(e events.PlayerSpottersChanged) {
		if e.Spotted == nil {
			return
		}
		gs := p.GameState()
		var spotters []string
		for _, other := range gs.Participants().Playing() {
			if other == nil || other.SteamID64 == e.Spotted.SteamID64 {
				continue
			}
			if e.Spotted.IsSpottedBy(other) {
				spotters = append(spotters, fmt.Sprintf("%s(%d)", other.Name, other.SteamID64))
			}
		}
		fmt.Printf("tick=%d spotted=%s(%d) spotters=%v\n",
			gs.IngameTick(), e.Spotted.Name, e.Spotted.SteamID64, spotters)
	})

	if err := p.ParseToEnd(); err != nil {
		log.Fatal(err)
	}
}
