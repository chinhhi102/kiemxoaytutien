// Command sim runs one room headless with N bots to check balance (spec §15):
// average time to reach each realm, average sword count, deaths.
//
//	go run ./cmd/sim --bots 40 --minutes 10
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	"kiemxoay/server/internal/bot"
	"kiemxoay/server/internal/config"
	"kiemxoay/server/internal/game"
	"kiemxoay/server/internal/world"
)

type botStats struct {
	spawnedAt  float64
	reachedAt  map[int]float64 // realm → first time reached in the current life
	peakRealm  int
	deaths     int
	lastDeadAt float64
}

func main() {
	bots := flag.Int("bots", 40, "number of bots")
	minutes := flag.Float64("minutes", 10, "simulated minutes")
	mapID := flag.String("map", "", "map id from maps/ (default: random)")
	seed := flag.Int64("seed", time.Now().UnixNano(), "random seed")
	monsters := flag.Int("monsters", -1, "override room.monster_count (-1 = config)")
	cell := flag.Float64("orb-cell", 0, "override exp_orbs.cell (0 = config)")
	farm := flag.Bool("farm", false, "nobody can be hit: measures pure farming pace (spec §6.1 target 60–90s to Trúc Cơ)")
	flag.Parse()

	cfg := config.MustLoadDefault()
	if *farm {
		cfg.Player.InvulnOnSpawn = 1e9
	}
	if *monsters >= 0 {
		cfg.Room.MonsterCount = *monsters
	}
	if *cell > 0 {
		cfg.ExpOrbs.Cell = *cell
	}
	m := pickMap(cfg, *mapID, *seed)
	g := game.New(cfg, m, *seed)
	rng := rand.New(rand.NewSource(*seed))
	brains, stats := spawnBots(g, cfg, rng, *bots)

	dt := 1 / float64(cfg.Net.TickHz)
	ticks := int(*minutes * 60 / dt)
	var tickTime time.Duration
	start := time.Now()
	for i := 0; i < ticks; i++ {
		for _, b := range brains {
			b.Update(g)
		}
		t0 := time.Now()
		g.Tick(dt)
		tickTime += time.Since(t0)
		g.DrainEvents()
		track(g, stats, cfg)
	}
	report(g, stats, ticks, tickTime, time.Since(start), m.ID)
}

func pickMap(cfg *config.Config, id string, seed int64) *world.Map {
	maps, err := world.LoadDir(config.FindRepoFile("maps"), world.LoadOptions{HP: cfg.Obstacles.HP, FootprintW: cfg.Obstacles.FootprintW, FootprintH: cfg.Obstacles.FootprintH})
	if err != nil || len(maps) == 0 {
		log.Printf("no maps loaded (%v); using empty map", err)
		return world.EmptyMap(cfg.Room.MapSize)
	}
	for _, m := range maps {
		if m.ID == id {
			return m
		}
	}
	return maps[rand.New(rand.NewSource(seed)).Intn(len(maps))]
}

// Sim bots all start at Luyện Khí 1 so the realm timings measure real progression.
func spawnBots(g *game.Game, cfg *config.Config, rng *rand.Rand, n int) (map[uint32]*bot.Brain, map[uint32]*botStats) {
	brains := map[uint32]*bot.Brain{}
	stats := map[uint32]*botStats{}
	for i := 0; i < n; i++ {
		p := g.AddPlayer(game.SpawnOptions{Name: fmt.Sprintf("bot%d", i), Char: cfg.Characters[i%len(cfg.Characters)], IsBot: true, Swords: 1})
		brains[p.ID] = bot.New(p.ID, rng.Int63(), g)
		stats[p.ID] = &botStats{reachedAt: map[int]float64{0: 0}}
	}
	return brains, stats
}

func track(g *game.Game, stats map[uint32]*botStats, cfg *config.Config) {
	for id, s := range stats {
		p := g.Players[id]
		if p.Dead {
			if p.DiedAt != s.lastDeadAt {
				s.lastDeadAt = p.DiedAt
				s.deaths++
			}
			if g.Time-p.DiedAt >= cfg.Bots.RespawnSecs {
				g.Respawn(p, 0, 1)
				s.spawnedAt = g.Time
				s.reachedAt = map[int]float64{0: 0}
			}
			continue
		}
		r := g.Prog.Realm(p.Level)
		if _, ok := s.reachedAt[r]; !ok {
			s.reachedAt[r] = g.Time - s.spawnedAt
		}
		s.peakRealm = max(s.peakRealm, r)
	}
}

func report(g *game.Game, stats map[uint32]*botStats, ticks int, tickTime, wall time.Duration, mapID string) {
	fmt.Printf("map %s · %d bots · %.1f simulated minutes (wall %s)\n", mapID, len(stats), g.Time/60, wall.Round(time.Millisecond))
	fmt.Printf("avg tick %.3f ms (target < 3 ms)\n", float64(tickTime.Microseconds())/float64(ticks)/1000)
	reach := map[int][]float64{}
	deaths, swords, alive := 0, 0, 0
	for id, s := range stats {
		deaths += s.deaths
		for r, t := range s.reachedAt {
			if r > 0 {
				reach[r] = append(reach[r], t)
			}
		}
		if p := g.Players[id]; !p.Dead {
			swords += p.Ring.Count
			alive++
		}
	}
	realms := make([]int, 0, len(reach))
	for r := range reach {
		realms = append(realms, r)
	}
	sort.Ints(realms)
	fmt.Println("realm reached (first time in a life): bots · avg time")
	for _, r := range realms {
		sum := 0.0
		for _, t := range reach[r] {
			sum += t
		}
		fmt.Printf("  %-12s %3d · %6.1fs\n", g.Prog.RealmName(r), len(reach[r]), sum/float64(len(reach[r])))
	}
	if alive > 0 {
		fmt.Printf("avg swords (alive) %.1f · total deaths %d · deaths/bot %.2f\n", float64(swords)/float64(alive), deaths, float64(deaths)/float64(len(stats)))
	}
	top := g.Ranking()
	if len(top) > 0 {
		fmt.Printf("top: %s %s tầng %d (%.0f EXP, %d kills)\n", top[0].Name, g.Prog.RealmName(g.Prog.Realm(top[0].Level)), g.Prog.Tier(top[0].Level)+1, top[0].TotalExp, top[0].Kills)
	}
}
