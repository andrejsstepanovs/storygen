package story

import (
	"fmt"
	"math/rand"
)

func GetAvailableProtagonists() Protagonists {
	types := []string{"human", "animal", "mythical being"}
	genders := []string{"male", "female", "girl", "boy"}
	age := []string{"child", "teenager", "adult"}
	size := []string{"small", "normal", "large"}
	voices := []string{"deep and mature", "deep and wise", "squeaky and childish", "high and soft", "high and sweet", "animated and cheerful", "animated and energetic", "animated and funny", "animated and dramatic", "animated and serious"}

	protagonists := make(Protagonists, 0)
	i := 0
	for _, s := range size {
		for _, t := range types {
			for _, g := range genders {
				for _, v := range voices {
					for _, a := range age {
						protagonists = append(protagonists, Protagonist{
							Type:   t,
							Voice:  v,
							Age:    a,
							Gender: g,
							Size:   s,
							Name:   fmt.Sprintf("John Doe %d", i),
						})
						i++
					}
				}
			}
		}
	}

	return protagonists
}

func GetRandomProtagonists(count int) Protagonists {
	entries := GetAvailableProtagonists()
	randomEntries := make(Protagonists, 0, count)

	// Ensure we don't try to select more entries than available
	if count > len(entries) {
		count = len(entries)
	}

	for i := 0; i < count; i++ {
		randomIndex := rand.Intn(len(entries))
		randomEntries = append(randomEntries, entries[randomIndex])
		entries = append(entries[:randomIndex], entries[randomIndex+1:]...)
	}

	return randomEntries
}
