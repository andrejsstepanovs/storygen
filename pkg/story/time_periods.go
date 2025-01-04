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
			Name:        "The Future",
			Description: "Space exploration, advanced technology, and futuristic societies. Examples include 'The Little Prince' and 'Wall-E'.",
		},
		{
			Name:        "The Stone Age",
			Description: "Early human tribes, mammoths, and survival. Examples include 'The First Dog' by Jan Brett and 'Stone Age Boy' by Satoshi Kitamura.",
		},
		{
			Name:        "The Age of Dinosaurs",
			Description: "A world dominated by dinosaurs. Examples include 'Dinotopia' and 'The Magic School Bus: In the Time of the Dinosaurs'.",
		},
		{
			Name:        "The Middle Ages",
			Description: "Feudal systems, kings, and queens. Examples include 'The Door in the Wall' and 'Catherine, Called Birdy'.",
		},
		{
			Name:        "The Age of Sail",
			Description: "Tall ships, naval battles, and exploration. Examples include 'Carry On, Mr. Bowditch' and 'The True Confessions of Charlotte Doyle'.",
		},
		{
			Name:        "The Age of Enlightenment",
			Description: "A time of scientific discovery and philosophical thought. Examples include 'Ben and Me' by Robert Lawson and 'The Wright Brothers' by Quentin Reynolds.",
		},
		{
			Name:        "The Age of Pirates",
			Description: "Treasure hunts, sea battles, and hidden maps. Examples include 'Peter Pan' and 'Pirateology'.",
		},
		{
			Name:        "The Age of Magic",
			Description: "Wizards, witches, and enchanted creatures. Examples include 'Harry Potter' and 'The Chronicles of Narnia'.",
		},
		{
			Name:        "The Age of Adventure",
			Description: "Treasure hunts, lost worlds, and daring escapades. Examples include 'The Adventures of Tintin' and 'Swiss Family Robinson'.",
		},
		{
			Name:        "The Age of Fairy Tales",
			Description: "Classic fairy tales with moral lessons. Examples include 'Grimm's Fairy Tales' and 'Hans Christian Andersen's Fairy Tales'.",
		},
		{
			Name:        "The Age of Folklore",
			Description: "Stories passed down through generations. Examples include 'Anansi the Spider' and 'The People Could Fly'.",
		},
		{
			Name:        "The Age of Fantasy",
			Description: "Imaginary worlds with unique rules and creatures. Examples include 'The Hobbit' and 'The Golden Compass'.",
		},
		{
			Name:        "The Age of Science Fiction",
			Description: "Futuristic technology and space exploration. Examples include 'A Wrinkle in Time' and 'Ender's Game'.",
		},
		{
			Name:        "The Age of Mystery",
			Description: "Detectives, clues, and solving puzzles. Examples include 'Nancy Drew' and 'The Hardy Boys'.",
		},
		{
			Name:        "The Age of Animals",
			Description: "Stories where animals are the main characters. Examples include 'Charlotte's Web' and 'The Tale of Peter Rabbit'.",
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
