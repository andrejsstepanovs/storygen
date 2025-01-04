package story

import (
	"math/rand"
)

func GetAvailableProtagonists() Protagonists {
	types := []string{"human", "animal", "mythical being"}
	genders := []string{"male", "female", "girl", "boy"}
	age := []string{"child", "teenager", "adult"}
	size := []string{"small", "normal", "large"}

	protagonists := make(Protagonists, 0)
	for _, s := range size {
		for _, t := range types {
			for _, g := range genders {
				for _, a := range age {
					protagonists = append(protagonists, Protagonist{
						Type:   t,
						Age:    a,
						Gender: g,
						Size:   s,
					})
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
		// Remove the selected morale from the list to avoid duplicates
		entries = append(entries[:randomIndex], entries[randomIndex+1:]...)
	}

	return randomEntries
}
