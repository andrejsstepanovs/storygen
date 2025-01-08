package story

import (
	"math/rand"
)

func GetAvailableStoryStructures() []Structure {
	structures := []Structure{
		{
			Name:        "Three-Act Structure",
			Description: "Act I: Setup (20% of story). Act II: Primary action (60% of story). Act III: Resolution (20% of story)",
		},
		{
			Name:        "Basic Plot Structure",
			Description: "The traditional structure follows a main character who faces a specific problem, makes multiple attempts to solve it (typically failing three times), experiences a moment of despair, and finally succeeds in an unexpected way.",
		},
		{
			Name:        "In Media Res",
			Description: "In media res is a narrative structure that starts in the middle of the action, rather than at the beginning.",
		},
		{
			Name:        "Circular Structure",
			Description: "The story ends where it began, creating a sense of closure and often emphasizing themes of growth or cyclical nature.",
		},
		{
			Name:        "Parallel Structure",
			Description: "Two or more storylines run simultaneously, often converging at the end to resolve the plot.",
		},
		{
			Name:        "Quest Structure",
			Description: "The protagonist sets out on a journey to achieve a specific goal, encountering obstacles and allies along the way.",
		},
		{
			Name:        "Problem-Solution Structure",
			Description: "The story presents a clear problem early on, and the narrative focuses on the protagonist's attempts to solve it, often with a satisfying resolution.",
		},
		{
			Name:        "Flashback Structure",
			Description: "The story is told through a series of flashbacks, revealing key information about the characters or plot as the narrative progresses.",
		},
		{
			Name:        "Frame Story",
			Description: "A story within a story, where a main narrative serves as a framework for one or more secondary tales.",
		},
		//{
		//	Name:        "Repetitive Structure",
		//	Description: "A pattern of repetition is used to reinforce themes, build anticipation, or create a rhythmic flow (e.g., 'Brown Bear, Brown Bear, What Do You See?').",
		//},
		{
			Name:        "Rags to Riches",
			Description: "The protagonist starts in a lowly or unfortunate state and, through perseverance or luck, achieves success or happiness.",
		},
		{
			Name:        "Overcoming the Monster",
			Description: "The protagonist faces and defeats a formidable antagonist or force, often symbolizing a larger theme like fear or injustice.",
		},
		{
			Name:        "Voyage and Return",
			Description: "The protagonist travels to a strange or unfamiliar world, faces challenges, and returns home with new wisdom or perspective.",
		},
		{
			Name:        "Fractured Fairy Tale",
			Description: "A traditional fairy tale is retold with a twist, often subverting expectations or adding humor and modern elements.",
		},
		{
			Name:        "Mystery Structure",
			Description: "The story revolves around solving a mystery, with clues and red herrings leading to a satisfying reveal.",
		},
	}

	return structures
}

func GetRandomStoryStructure() Structure {
	structures := GetAvailableStoryStructures()
	return structures[rand.Intn(len(structures))]
}
