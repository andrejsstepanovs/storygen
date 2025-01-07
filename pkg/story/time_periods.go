package story

import (
	"math/rand"
)

func GetAvailableTimePeriods() TimePeriods {
	timePeriods := []TimePeriod{
		{
			Name:        "Once upon a time",
			Description: "A timeless, fairy-tale setting in a faraway land, often with magical elements.",
		},
		{
			Name:        "The Stone Age",
			Description: "Early human tribes, mammoths, and survival.",
		},
		{
			Name:        "The Age of Dinosaurs",
			Description: "A world dominated by dinosaurs.",
		},
		{
			Name:        "The Middle Ages",
			Description: "Feudal systems, kings, and queens.",
		},
		{
			Name:        "The Age of Sail",
			Description: "Tall ships, naval battles, and exploration.",
		},
		{
			Name:        "The Age of Enlightenment",
			Description: "A time of scientific discovery and philosophical thought.",
		},
		{
			Name:        "The Age of Pirates",
			Description: "Treasure hunts, sea battles, and hidden maps.",
		},
		{
			Name:        "The Age of Magic",
			Description: "Wizards, witches, and enchanted creatures.",
		},
		{
			Name:        "The Age of Adventure",
			Description: "Treasure hunts, lost worlds, and daring escapades.",
		},
		{
			Name:        "The Age of Fairy Tales",
			Description: "Classic fairy tales with moral lessons.",
		},
		{
			Name:        "The Age of Folklore",
			Description: "Stories passed down through generations.",
		},
		{
			Name:        "The Age of Fantasy",
			Description: "Imaginary worlds with unique rules and creatures.",
		},
		{
			Name:        "The Age of Science Fiction",
			Description: "Futuristic technology and space exploration.",
		},
		{
			Name:        "The Age of Mystery",
			Description: "Detectives, clues, and solving puzzles.",
		},
		{
			Name:        "The Age of Animals",
			Description: "Stories where animals are the main characters.",
		},
	}

	return timePeriods
}

func GetRandomTimePeriods(count int) TimePeriods {
	entries := GetAvailableTimePeriods()
	randomEntries := make(TimePeriods, 0, count)

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

func FindTimePeriodsByName(names []string) TimePeriods {
	found := make(TimePeriods, 0)
	for _, name := range names {
		for _, morale := range GetAvailableTimePeriods() {
			if morale.Name == name {
				found = append(found, morale)
			}
		}
	}

	return found
}
