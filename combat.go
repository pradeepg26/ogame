package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
)

type UnitType int

type Resources struct {
	Metal     int
	Crystal   int
	Deuterium int
}

func min(inp ...int) int {
	min := inp[0]
	for _, i := range inp {
		if i < min {
			min = i
		}
	}
	return min
}

func (r *Resources) Add(other Resources) {
	r.Metal += other.Metal
	r.Crystal += other.Crystal
	r.Deuterium += other.Crystal
}

func (r *Resources) Total() int {
	return int(float64(r.Metal) + 1.505520505*float64(r.Crystal) + 2.666201117*float64(r.Deuterium))
}

func (r *Resources) AllocateN(other Resources, n int) {
	r.Metal -= n * other.Metal
	r.Crystal -= n * other.Crystal
	r.Deuterium -= n * other.Deuterium
	if r.Metal < 0 || r.Crystal < 0 || r.Deuterium < 0 {
		panic("over allocated on resources")
	}
}

func (r Resources) MaxAllocation(other Resources) int {
	maxMetal := math.MaxInt32
	maxCrystal := math.MaxInt32
	maxDeut := math.MaxInt32
	if other.Metal > 0 {
		maxMetal = int(math.Floor(float64(r.Metal) / float64(other.Metal)))
	}
	if other.Crystal > 0 {
		maxCrystal = int(math.Floor(float64(r.Crystal) / float64(other.Crystal)))
	}
	if other.Deuterium > 0 {
		maxDeut = int(math.Floor(float64(r.Deuterium) / float64(other.Deuterium)))
	}
	return min(maxMetal, maxCrystal, maxDeut)
}

const (
	LightFighter UnitType = iota
	HeavyFighter
	Cruiser
	Battleship
	RocketLauncher
	LightLaser
	HeavyLaser
	GaussCannon
	IonCannon
)

func (t UnitType) String() string {
	switch t {
	case LightFighter:
		return "LightFighter"
	case HeavyFighter:
		return "HeavyFighter"
	case Cruiser:
		return "Cruiser"
	case Battleship:
		return "Battleship"
	case RocketLauncher:
		return "RocketLauncher"
	case LightLaser:
		return "LightLaser"
	case HeavyLaser:
		return "HeavyLaser"
	case GaussCannon:
		return "GaussCannon"
	case IonCannon:
		return "IonCannon"
	default:
		return "Unknown"
	}
}

type Composition []UnitType

var AttackerComposition = Composition{
	LightFighter, HeavyFighter, Cruiser, Battleship,
}

var DefenderComposition = Composition{
	RocketLauncher, LightLaser, HeavyLaser, IonCannon, GaussCannon,
}

var rapidFire = map[UnitType]map[UnitType]float64{
	Cruiser: map[UnitType]float64{
		LightFighter:   0.833333333,
		RocketLauncher: 0.9,
	},
}

var UnitCosts = map[UnitType]Resources{
	// Offensive
	LightFighter: Resources{3000, 1000, 0},
	HeavyFighter: Resources{6000, 4000, 0},
	Cruiser:      Resources{20000, 7000, 2000},
	Battleship:   Resources{45000, 15000, 0},
	// Defensive
	RocketLauncher: Resources{2000, 0, 0},
	LightLaser:     Resources{1500, 500, 0},
	HeavyLaser:     Resources{6000, 2000, 0},
	GaussCannon:    Resources{20000, 15000, 2000},
	IonCannon:      Resources{2000, 6000, 0},
}

var initial_hull = map[UnitType]int{
	LightFighter:   800,
	HeavyFighter:   2000,
	Cruiser:        5400,
	Battleship:     12000,
	RocketLauncher: 400,
	LightLaser:     400,
	HeavyLaser:     1600,
	GaussCannon:    7000,
	IonCannon:      1600,
}

var shields = map[UnitType]int{
	LightFighter:   20,
	HeavyFighter:   50,
	Cruiser:        100,
	Battleship:     400,
	RocketLauncher: 40,
	LightLaser:     50,
	HeavyLaser:     200,
	GaussCannon:    400,
	IonCannon:      1000,
}

var weaponry = map[UnitType]int{
	LightFighter:   100,
	HeavyFighter:   300,
	Cruiser:        800,
	Battleship:     2000,
	RocketLauncher: 160,
	LightLaser:     200,
	HeavyLaser:     500,
	GaussCannon:    2200,
	IonCannon:      300,
}

type Unit struct {
	Hull     int
	Shields  int
	Weaponry int
	Type     UnitType
	Targets  []int
}

type Fleet struct {
	Units []*Unit
	Lost  Resources
}

func (f *Fleet) PrintSummary() {
	comp := map[UnitType]int{}
	for _, u := range f.Units {
		comp[u.Type]++
	}
	for ut, count := range comp {
		fmt.Printf("\t%s = %d\n", ut.String(), count)
	}
}

func PickAnotherAttacker(unit, target *Unit) bool {
	return rand.Float64() < rapidFire[unit.Type][target.Type]
}

func PickTargets(attacker, defender *Fleet) {
	num_defenders := len(defender.Units)
	for _, u := range attacker.Units {
		targets := make([]int, 0)
		// pick at least 1 target per unit
		// if the unit has rapid fire against the target,
		// roll the dice and see if you get to pick another target
		// keep picking targets until the roll fails
		// do-while loop
		for ok := true; ok; ok = PickAnotherAttacker(u, defender.Units[len(defender.Units)-1]) {
			t := rand.Intn(num_defenders)
			targets = append(targets, t)
		}
		u.Targets = targets
	}
}

func AttackTargets(attacker, defender *Fleet) {
	for _, u := range attacker.Units {
		for _, t := range u.Targets {
			target := defender.Units[t]
			if target.Hull > 0 && u.Weaponry >= target.Shields/100 {
				// shoot only if the target hasn't already been destroyed
				// shot only counts if attackers weaponry > 1% of targets shielding
				if u.Weaponry < target.Shields {
					// weapons don't penetrate shields
					target.Shields = target.Shields - u.Weaponry
				} else {
					// weapons penetrated the shields
					target.Hull = target.Hull - (u.Weaponry - target.Shields)
					target.Shields = 0
				}
				hullPct := float64(target.Hull) / float64(initial_hull[target.Type])
				if hullPct < 0.7 {
					// if the ship's hull is below 70% health,
					// there is a chance that the ship can explode
					// probability of explosion = 1 - % hull remaining
					if rand.Float64() < (1 - hullPct) {
						// ship exploded
						target.Hull = 0
					}
				}
			}
		}
	}
}

func Process(attacker, defender *Fleet) {
	PickTargets(attacker, defender)
	AttackTargets(attacker, defender)
}

func RemoveDeadUnits(fleet *Fleet) {
	units := make([]*Unit, 0, len(fleet.Units)/2)
	for _, u := range fleet.Units {
		if u.Hull > 0 {
			units = append(units, u)
			// restore shields
			u.Shields = shields[u.Type]
		} else {
			unitCost := UnitCosts[u.Type]
			fleet.Lost.Add(unitCost)
		}
	}
	fleet.Units = units
}

func MakeMultipleUnits(unitType UnitType, num int) []*Unit {
	units := make([]*Unit, num)
	for i := 0; i < num; i++ {
		units[i] = MakeUnit(unitType)
	}
	return units
}

func MakeUnit(unitType UnitType) *Unit {
	return &Unit{
		Hull:     initial_hull[unitType],
		Shields:  shields[unitType],
		Weaponry: weaponry[unitType],
		Type:     unitType,
	}
}
func SimulateCombat(attacker, defender *Fleet) string {
	for i := 0; i < 6; i++ {
		Process(attacker, defender)
		Process(defender, attacker)
		RemoveDeadUnits(attacker)
		RemoveDeadUnits(defender)
		// Combat ends if one of the players has no units
		if len(attacker.Units) == 0 && len(defender.Units) == 0 {
			return fmt.Sprintf("DRAW-BOTH-LOSE(%d)", i)
		} else if len(attacker.Units) == 0 && len(defender.Units) > 0 {
			return fmt.Sprintf("LOSS(%d)", i)
		} else if len(defender.Units) == 0 && len(attacker.Units) > 0 {
			return fmt.Sprintf("WIN(%d)", i)
		}
	}
	// automatic draw after 6 rounds of combat
	return "DRAW-TIMEOUT"
}

func MakeFleetByAlloc(resources *Resources, composition Composition, allocation []float64) *Fleet {
	units := make([]*Unit, 0)
	for i, a := range allocation {
		unitType := composition[i]
		unitCost := UnitCosts[unitType]
		maxUnits := resources.MaxAllocation(unitCost)
		numUnits := int(math.Floor(float64(maxUnits) * a))
		if numUnits > 0 {
			units = append(units, MakeMultipleUnits(unitType, numUnits)...)
			resources.AllocateN(unitCost, numUnits)
		}
	}
	return &Fleet{units, Resources{0, 0, 0}}
}

func SimulateFight(attacker, defender *Fleet) float64 {
	for i := 0; i < 6; i++ {
		Process(attacker, defender)
		Process(defender, attacker)
		RemoveDeadUnits(attacker)
		RemoveDeadUnits(defender)
		if len(attacker.Units) == 0 || len(defender.Units) == 0 {
			break
		}
	}
	return float64(defender.Lost.Total()) / float64(attacker.Lost.Total())
}

func avg(left, right []float64) []float64 {
	if len(left) != len(right) {
		panic("must have equal lengths")
	}
	out := make([]float64, len(left))
	for i := 0; i < len(left); i++ {
		// 0.1% mutation chance of super strong
		// 0.1% mutation chance of super weak
		mc := rand.Float64()
		if mc < 0.001 {
			// super weak
			out[i] = 0.001
		} else if mc > 0.999 {
			// super strong
			out[i] = 0.999
		} else {
			p := (rand.Float64() + rand.Float64()) / 2
			out[i] = p*left[i] + (1.0-p)*right[i]
		}
	}
	return out
}

var AttackerResources = Resources{22449840, 14911680, 8420160}
var DefenderResources = Resources{2244984, 1491168, 842016}

type AttackerAlloc [4]float64

func NewAttackerAlloc(f []float64) AttackerAlloc {
	var out AttackerAlloc
	if len(f) != len(out) {
		panic("bad input")
	}
	for i, a := range f {
		out[i] = a
	}
	return out
}

type DefenderAlloc [5]float64

func NewDefenderAlloc(f []float64) DefenderAlloc {
	var out DefenderAlloc
	if len(f) != len(out) {
		panic("bad input")
	}
	for i, a := range f {
		out[i] = a
	}
	return out
}

type Scores struct {
	sort.Float64Slice
	idx []int
}

func NewScores(data []float64) *Scores {
	s := &Scores{Float64Slice: data, idx: make([]int, len(data))}
	for i := range s.idx {
		s.idx[i] = i
	}
	return s
}

func (s Scores) Swap(i, j int) {
	s.Float64Slice.Swap(i, j)
	s.idx[i], s.idx[j] = s.idx[j], s.idx[i]
}

func DoCombat(atkComp AttackerAlloc, defComp DefenderAlloc) float64 {
	ares := AttackerResources
	attacker := MakeFleetByAlloc(
		&ares,
		AttackerComposition,
		atkComp[:],
	)
	dres := DefenderResources
	defender := MakeFleetByAlloc(
		&dres,
		DefenderComposition,
		defComp[:],
	)
	return SimulateFight(attacker, defender)
}

func step(atkComp [100]AttackerAlloc, defComp [100]DefenderAlloc) (*Scores, *Scores) {
	attackerScores := make([]float64, 100)
	defenderScores := make([]float64, 100)

	type task struct {
		i, j int
	}
	type result struct {
		i, j  int
		score float64
	}
	taskChan := make(chan task)
	resultChan := make(chan result)
	var rwg sync.WaitGroup
	rwg.Add(1)
	go func() {
		defer rwg.Done()
		for r := range resultChan {
			attackerScores[r.i] += r.score
			defenderScores[r.j] += r.score
		}
	}()
	go func() {
		// coordinator thread for workers
		var twg sync.WaitGroup
		twg.Add(8)
		for i := 0; i < 8; i++ {
			// 4 threads to perform simulations
			go func() {
				defer twg.Done()
				for t := range taskChan {
					s := DoCombat(atkComp[t.i], defComp[t.j])
					resultChan <- result{t.i, t.j, s}
				}
			}()
		}
		twg.Wait()
		close(resultChan)
	}()

	for i := 0; i < 100; i++ {
		for j := 0; j < 100; j++ {
			taskChan <- task{i, j}
		}
	}
	close(taskChan)
	rwg.Wait() // wait for all results

	// sort the attacker/defender scores
	AS := NewScores(attackerScores)
	sort.Sort(AS)
	DS := NewScores(defenderScores)
	sort.Sort(DS)
	return AS, DS
}

func reproduceAtk(atkComp [100]AttackerAlloc, scores *Scores) [100]AttackerAlloc {
	// highest scorers are the best attackers
	// take the top 20 and reproduce randomly
	var newAlloc [100]AttackerAlloc
	for i := 0; i < 100; i++ {
		p, q := rand.Intn(20), rand.Intn(20)
		dist := avg([]float64(atkComp[99-p][:]), []float64(atkComp[99-q][:]))
		newAlloc[i] = NewAttackerAlloc(dist)
	}
	// // shuffle the array
	// idx := rand.Perm(100)
	// for i, j := range idx {
	// 	newAlloc[i], newAlloc[j] = newAlloc[j], newAlloc[i]
	// }
	return newAlloc
}

func reproduceDef(defComp [100]DefenderAlloc, scores *Scores) [100]DefenderAlloc {
	// lowest scores are the best defenders
	// take the bottom 20 and reproduce randomly
	var newAlloc [100]DefenderAlloc
	for i := 0; i < 100; i++ {
		p, q := rand.Intn(20), rand.Intn(20)
		dist := avg([]float64(defComp[p][:]), []float64(defComp[q][:]))
		newAlloc[i] = NewDefenderAlloc(dist)
	}
	// // shuffle the array
	// idx := rand.Perm(100)
	// for i, j := range idx {
	// 	newAlloc[i], newAlloc[j] = newAlloc[j], newAlloc[i]
	// }
	return newAlloc
}

func main() {
	var atkComp [100]AttackerAlloc
	var defComp [100]DefenderAlloc
	// initialize random attackers and defenders
	for i := 0; i < 100; i++ {
		atkComp[i] = AttackerAlloc{rand.Float64(), rand.Float64(), rand.Float64(), 1.0}
		defComp[i] = DefenderAlloc{rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64(), 1.0}
	}
	for i := 0; i < 10; i++ {
		atkScores, defScores := step(atkComp, defComp)
		atkComp = reproduceAtk(atkComp, atkScores)
		defComp = reproduceDef(defComp, defScores)
		fmt.Printf("[%f, %f] (%v, %v)\n", defScores.Float64Slice[0], atkScores.Float64Slice[99], defComp[0], atkComp[99])
	}

	fmt.Printf("def: [%v]\n", defComp[:5])
	fmt.Printf("atk: [%v]\n", atkComp[:5])
	ares := AttackerResources
	dres := DefenderResources
	attacker := MakeFleetByAlloc(
		&ares,
		AttackerComposition,
		atkComp[99][:],
	)
	defender := MakeFleetByAlloc(
		&dres,
		DefenderComposition,
		defComp[0][:],
	)
	fmt.Println("--------")
	fmt.Println("Attacker")
	fmt.Println("--------")
	attacker.PrintSummary()
	fmt.Println("--------")
	fmt.Println("Defender")
	fmt.Println("--------")
	defender.PrintSummary()
	fmt.Println("--------")

	fmt.Println(SimulateCombat(attacker, defender))

	fmt.Println("--------")
	fmt.Println("Attacker")
	fmt.Println("--------")
	attacker.PrintSummary()
	fmt.Printf("Lost = %d\n", attacker.Lost.Total())
	fmt.Println("--------")
	fmt.Println("Defender")
	fmt.Println("--------")
	defender.PrintSummary()
	fmt.Printf("Lost = %d\n", defender.Lost.Total())
	fmt.Println("--------")
}
