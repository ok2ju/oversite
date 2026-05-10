package main

import (
	"encoding/json"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/store"
)

// buildPlayerStatsInputs converts the database query results into the typed
// shape ComputePlayerMatchStats expects: parsed extras (*KillExtra,
// *PlayerHurtExtra), per-round rosters, and a round-number-keyed loadouts map.
//
// We intentionally rebuild the typed extras here rather than reading the
// promoted columns directly into demo.GameEvent without the JSON decode —
// stats.go and player_stats.go already consume *KillExtra / *PlayerHurtExtra
// (the same shape produced at parse time), so reconstructing the typed
// extras keeps them as the single source of truth for round/clutch/trade
// detection.
func buildPlayerStatsInputs(
	storeRounds []store.Round,
	storeEvents []store.GameEvent,
	loadoutRows []store.GetRoundLoadoutsByDemoIDRow,
	rosterRows []store.GetRostersByDemoIDRow,
) ([]demo.RoundData, []demo.GameEvent, map[int][]demo.RoundLoadoutEntry) {
	// Per-round rosters keyed by round number for quick attach.
	roster := make(map[int][]demo.RoundParticipant)
	for _, r := range rosterRows {
		roster[int(r.RoundNumber)] = append(roster[int(r.RoundNumber)], demo.RoundParticipant{
			SteamID:    r.SteamID,
			PlayerName: r.PlayerName,
			TeamSide:   r.TeamSide,
		})
	}

	rounds := make([]demo.RoundData, len(storeRounds))
	for i, r := range storeRounds {
		rounds[i] = demo.RoundData{
			Number:        int(r.RoundNumber),
			StartTick:     int(r.StartTick),
			FreezeEndTick: int(r.FreezeEndTick),
			EndTick:       int(r.EndTick),
			WinnerSide:    r.WinnerSide,
			WinReason:     r.WinReason,
			CTScore:       int(r.CtScore),
			TScore:        int(r.TScore),
			IsOvertime:    r.IsOvertime != 0,
			CTTeamName:    r.CtTeamName,
			TTeamName:     r.TTeamName,
			Roster:        roster[int(r.RoundNumber)],
		}
	}

	// To map round_id → round_number for events.
	roundNumberByID := make(map[int64]int, len(storeRounds))
	for _, r := range storeRounds {
		roundNumberByID[r.ID] = int(r.RoundNumber)
	}

	events := make([]demo.GameEvent, len(storeEvents))
	for i, e := range storeEvents {
		ev := demo.GameEvent{
			Tick:        int(e.Tick),
			RoundNumber: roundNumberByID[e.RoundID],
			Type:        e.EventType,
			X:           e.X,
			Y:           e.Y,
			Z:           e.Z,
		}
		if e.AttackerSteamID.Valid {
			ev.AttackerSteamID = e.AttackerSteamID.String
		}
		if e.VictimSteamID.Valid {
			ev.VictimSteamID = e.VictimSteamID.String
		}
		if e.Weapon.Valid {
			ev.Weapon = e.Weapon.String
		}

		switch e.EventType {
		case "kill":
			extra := &demo.KillExtra{
				Headshot:     e.Headshot != 0,
				AttackerName: e.AttackerName,
				AttackerTeam: e.AttackerTeam,
				VictimName:   e.VictimName,
				VictimTeam:   e.VictimTeam,
			}
			if e.AssisterSteamID.Valid {
				extra.AssisterSteamID = e.AssisterSteamID.String
			}
			if e.ExtraData != "" {
				_ = json.Unmarshal([]byte(e.ExtraData), extra)
			}
			ev.ExtraData = extra
		case "player_hurt":
			extra := &demo.PlayerHurtExtra{
				HealthDamage: int(e.HealthDamage),
				AttackerName: e.AttackerName,
				AttackerTeam: e.AttackerTeam,
				VictimName:   e.VictimName,
				VictimTeam:   e.VictimTeam,
			}
			if e.ExtraData != "" {
				_ = json.Unmarshal([]byte(e.ExtraData), extra)
			}
			ev.ExtraData = extra
		case "player_flashed":
			extra := &demo.PlayerFlashedExtra{
				AttackerName: e.AttackerName,
				AttackerTeam: e.AttackerTeam,
				VictimName:   e.VictimName,
				VictimTeam:   e.VictimTeam,
			}
			if e.ExtraData != "" {
				_ = json.Unmarshal([]byte(e.ExtraData), extra)
			}
			ev.ExtraData = extra
		case "grenade_throw":
			extra := &demo.GrenadeThrowExtra{}
			if e.ExtraData != "" {
				_ = json.Unmarshal([]byte(e.ExtraData), extra)
			}
			ev.ExtraData = extra
		}
		events[i] = ev
	}

	loadouts := make(map[int][]demo.RoundLoadoutEntry)
	for _, l := range loadoutRows {
		loadouts[int(l.RoundNumber)] = append(loadouts[int(l.RoundNumber)], demo.RoundLoadoutEntry{
			SteamID:   l.SteamID,
			Inventory: l.Inventory,
		})
	}
	return rounds, events, loadouts
}

// computedPlayerMatchStatsToBinding converts the package-internal stats type
// to the JSON-tagged DTO exposed via Wails.
func computedPlayerMatchStatsToBinding(s demo.PlayerMatchStats) PlayerMatchStats {
	out := PlayerMatchStats{
		SteamID:       s.SteamID,
		PlayerName:    s.PlayerName,
		TeamSide:      s.TeamSide,
		RoundsPlayed:  s.RoundsPlayed,
		Kills:         s.Kills,
		Deaths:        s.Deaths,
		Assists:       s.Assists,
		Damage:        s.Damage,
		HSKills:       s.HeadshotKills,
		ClutchKills:   s.ClutchKills,
		FirstKills:    s.FirstKills,
		FirstDeaths:   s.FirstDeaths,
		OpeningWins:   s.OpeningWins,
		OpeningLosses: s.OpeningLosses,
		TradeKills:    s.TradeKills,
		HSPercent:     s.HSPercent,
		ADR:           s.ADR,
		Movement: MovementStats{
			DistanceUnits:   s.Movement.DistanceUnits,
			AvgSpeedUps:     s.Movement.AvgSpeedUps,
			MaxSpeedUps:     s.Movement.MaxSpeedUps,
			StrafePercent:   s.Movement.StrafePercent,
			StationaryRatio: s.Movement.StationaryRatio,
			WalkingRatio:    s.Movement.WalkingRatio,
			RunningRatio:    s.Movement.RunningRatio,
		},
		Timings: TimingStats{
			AvgTimeToFirstContactSecs: s.Timings.AvgTimeToFirstContactSecs,
			AvgAliveDurationSecs:      s.Timings.AvgAliveDurationSecs,
			TimeOnSiteASecs:           s.Timings.TimeOnSiteASecs,
			TimeOnSiteBSecs:           s.Timings.TimeOnSiteBSecs,
		},
		Utility: UtilityStats{
			FlashesThrown:          s.Utility.FlashesThrown,
			SmokesThrown:           s.Utility.SmokesThrown,
			HEsThrown:              s.Utility.HEsThrown,
			MolotovsThrown:         s.Utility.MolotovsThrown,
			DecoysThrown:           s.Utility.DecoysThrown,
			FlashAssists:           s.Utility.FlashAssists,
			BlindTimeInflictedSecs: s.Utility.BlindTimeInflictedSecs,
			EnemiesFlashed:         s.Utility.EnemiesFlashed,
		},
	}
	out.HitGroups = make([]HitGroupBreakdown, len(s.HitGroups))
	for i, hg := range s.HitGroups {
		out.HitGroups[i] = HitGroupBreakdown{
			HitGroup: hg.HitGroup,
			Label:    hg.Label,
			Damage:   hg.Damage,
			Hits:     hg.Hits,
		}
	}
	out.DamageByWeapon = make([]DamageByWeapon, len(s.DamageByWeapon))
	for i, d := range s.DamageByWeapon {
		out.DamageByWeapon[i] = DamageByWeapon{Weapon: d.Weapon, Damage: d.Damage}
	}
	out.DamageByOpponent = make([]DamageByOpponent, len(s.DamageByOpponent))
	for i, d := range s.DamageByOpponent {
		out.DamageByOpponent[i] = DamageByOpponent{
			SteamID:    d.SteamID,
			PlayerName: d.PlayerName,
			TeamSide:   d.TeamSide,
			Damage:     d.Damage,
		}
	}
	out.Rounds = make([]PlayerRoundDetail, len(s.Rounds))
	for i, r := range s.Rounds {
		out.Rounds[i] = PlayerRoundDetail{
			RoundNumber:           r.RoundNumber,
			TeamSide:              r.TeamSide,
			Kills:                 r.Kills,
			Deaths:                r.Deaths,
			Assists:               r.Assists,
			Damage:                r.Damage,
			HSKills:               r.HeadshotKills,
			ClutchKills:           r.ClutchKills,
			FirstKill:             r.FirstKill,
			FirstDeath:            r.FirstDeath,
			TradeKill:             r.TradeKill,
			LoadoutValue:          r.LoadoutValue,
			DistanceUnits:         r.DistanceUnits,
			AliveDurationSecs:     r.AliveDurationSecs,
			TimeToFirstContactSec: r.TimeToFirstContactSec,
		}
	}
	return out
}

// tickDataToSamples converts the store-shaped TickDatum slice into the
// position/yaw subset the demo aggregator needs. The store rows are ordered
// by tick (custom query), so the slice we return preserves that order.
func tickDataToSamples(rows []store.TickDatum) []demo.PlayerTickSample {
	out := make([]demo.PlayerTickSample, len(rows))
	for i, r := range rows {
		out[i] = demo.PlayerTickSample{
			Tick:    int(r.Tick),
			X:       r.X,
			Y:       r.Y,
			Yaw:     r.Yaw,
			IsAlive: r.IsAlive != 0,
		}
	}
	return out
}
