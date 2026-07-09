package steps

import (
	"sort"
	"strconv"
	"strings"

	"github.com/oselvar/var/go/oselvar"
)

func yahtzeeScore(dice []int, category string) int {
	counts := map[int]int{}
	total := 0
	for _, d := range dice {
		counts[d]++
		total += d
	}
	sumOf := func(face int) int { return counts[face] * face }
	ofAKind := func(n int) int {
		best := 0
		for face, c := range counts {
			if c >= n && face > best {
				best = face
			}
		}
		return n * best
	}
	sorted := append([]int(nil), dice...)
	sort.Ints(sorted)
	var sb strings.Builder
	for _, d := range sorted {
		sb.WriteString(strconv.Itoa(d))
	}
	sortedDice := sb.String()

	switch category {
	case "ones":
		return sumOf(1)
	case "twos":
		return sumOf(2)
	case "threes":
		return sumOf(3)
	case "fours":
		return sumOf(4)
	case "fives":
		return sumOf(5)
	case "sixes":
		return sumOf(6)
	case "pair":
		return ofAKind(2)
	case "two pairs":
		pairs := 0
		sum := 0
		for face, c := range counts {
			if c >= 2 {
				pairs++
				sum += 2 * face
			}
		}
		if pairs >= 2 {
			return sum
		}
		return 0
	case "three of a kind":
		return ofAKind(3)
	case "four of a kind":
		return ofAKind(4)
	case "small straight":
		if sortedDice == "12345" {
			return 15
		}
		return 0
	case "large straight":
		if sortedDice == "23456" {
			return 20
		}
		return 0
	case "full house":
		cs := make([]int, 0, len(counts))
		for _, c := range counts {
			cs = append(cs, c)
		}
		sort.Ints(cs)
		if len(counts) == 2 && len(cs) == 2 && cs[0] == 2 && cs[1] == 3 {
			return total
		}
		return 0
	case "Yahtzee":
		if len(counts) == 1 {
			return 50
		}
		return 0
	case "chance":
		return total
	default:
		return 0
	}
}

func registerYahtzee(r *oselvar.Registrar[State]) {
	// Header-bound table: the paragraph names every header cell (dice, category,
	// score), so this sensor runs once per row; returning {"score": …} checks
	// that column while the others are inputs.
	r.Sensor("Examples of dice, category and score", func(_ State, row map[string]string) map[string]string {
		dice := make([]int, 0, 5)
		for _, d := range strings.Split(row["dice"], ",") {
			n, _ := strconv.Atoi(strings.TrimSpace(d))
			dice = append(dice, n)
		}
		return map[string]string{"score": strconv.Itoa(yahtzeeScore(dice, row["category"]))}
	})
}
