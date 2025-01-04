package story

import (
	"math/rand"
)

func GetAvailableStoryMorales() []Morale {
	morales := []Morale{
		{
			Name:        "Kindness and Compassion",
			Description: "Treat others with care and empathy, as your actions can make a big difference in someone's life.",
		},
		{
			Name:        "Honesty and Integrity",
			Description: "Telling the truth and doing the right thing, even when it’s hard, builds trust and respect.",
		},
		{
			Name:        "Perseverance and Hard Work",
			Description: "Never give up, even when things are tough; success comes through effort and determination.",
		},
		{
			Name:        "Friendship and Teamwork",
			Description: "Working together and supporting friends can help overcome challenges and make life more enjoyable.",
		},
		{
			Name:        "Courage and Facing Fears",
			Description: "Being brave and standing up to fears or challenges helps you grow and achieve great things.",
		},
		{
			Name:        "Respect for Others",
			Description: "Treat everyone with respect, regardless of their differences, and value their feelings and opinions.",
		},
		{
			Name:        "Sharing and Generosity",
			Description: "Sharing what you have with others brings happiness and strengthens relationships.",
		},
		{
			Name:        "Responsibility and Accountability",
			Description: "Take responsibility for your actions and understand that your choices have consequences.",
		},
		{
			Name:        "Self-Acceptance and Confidence",
			Description: "Be proud of who you are, embrace your uniqueness, and believe in your abilities.",
		},
		{
			Name:        "Forgiveness and Letting Go",
			Description: "Forgiving others and moving past mistakes helps heal relationships and brings inner peace.",
		},
		{
			Name:        "Curiosity and Learning",
			Description: "Asking questions, exploring, and learning new things leads to growth and understanding.",
		},
		{
			Name:        "Environmental Awareness and Care",
			Description: "Taking care of the planet and its resources ensures a better future for everyone.",
		},
		{
			Name:        "Fairness and Justice",
			Description: "Treating everyone equally and standing up for what is right creates a better world for all.",
		},
		{
			Name:        "Empathy and Understanding",
			Description: "Putting yourself in someone else’s shoes helps you understand their feelings and build stronger connections.",
		},
		{
			Name:        "Creativity and Imagination",
			Description: "Using your imagination and thinking creatively can solve problems and make life more fun.",
		},
		{
			Name:        "Optimism and Positivity",
			Description: "Looking on the bright side and staying positive helps you overcome challenges and find joy in life.",
		},
		{
			Name:        "Resilience and Adaptability",
			Description: "Bouncing back from setbacks and adapting to change makes you stronger and more capable.",
		},
		{
			Name:        "Self-Discipline and Focus",
			Description: "Staying focused and disciplined helps you achieve your goals and do your best in everything.",
		},
		{
			Name:        "Inclusivity and Diversity",
			Description: "Celebrating differences and including everyone makes the world a richer and more harmonious place.",
		},
		{
			Name:        "Generosity of Spirit",
			Description: "Giving your time, energy, and kindness to others without expecting anything in return enriches your life and theirs.",
		},
		{
			Name:        "Problem-Solving and Critical Thinking",
			Description: "Thinking carefully and finding solutions to problems helps you overcome obstacles and learn new skills.",
		},
		{
			Name:        "Self-Care and Well-Being",
			Description: "Taking care of your body, mind, and emotions ensures you stay healthy and happy.",
		},
		{
			Name:        "Community and Helping Others",
			Description: "Contributing to your community and helping those in need makes the world a better place.",
		},
		{
			Name:        "Independence and Self-Reliance",
			Description: "Learning to do things on your own builds confidence and prepares you for life’s challenges.",
		},
		{
			Name:        "Open-Mindedness and Flexibility",
			Description: "Being open to new ideas and willing to change your mind helps you learn and grow.",
		},
		{
			Name:        "Leadership and Initiative",
			Description: "Taking the lead and showing initiative inspires others and helps you achieve great things.",
		},
		{
			Name:        "Communication and Listening",
			Description: "Expressing yourself clearly and listening to others strengthens relationships and resolves conflicts.",
		},
		{
			Name:        "Celebration of Uniqueness",
			Description: "Embracing what makes you different and celebrating others’ uniqueness fosters a world of acceptance.",
		},
		{
			Name:        "Celebration of Effort Over Perfection",
			Description: "Focusing on effort rather than perfection encourages growth and reduces fear of failure.",
		},
		{
			Name:        "Celebration of Curiosity",
			Description: "Encouraging curiosity and a love for learning leads to discovery and innovation.",
		},
		{
			Name:        "Celebration of Team Spirit",
			Description: "Working together and celebrating team achievements fosters unity and shared success.",
		},
		{
			Name:        "Celebration of Effort and Progress",
			Description: "Recognizing progress, no matter how small, keeps you motivated and focused on your goals.",
		},
		{
			Name:        "Celebration of Learning from Mistakes",
			Description: "Viewing mistakes as opportunities to learn and grow builds resilience and wisdom.",
		},
		{
			Name:        "Celebration of Individuality",
			Description: "Embracing what makes each person unique fosters a culture of acceptance and creativity.",
		},
		{
			Name:        "Celebration of Teamwork and Collaboration",
			Description: "Working together and valuing each team member’s contribution leads to shared success.",
		},
		{
			Name:        "Respect for the Future",
			Description: "Thinking about the consequences of your actions helps create a better future for everyone.",
		},
		{
			Name:        "Celebration of Courage to Be Different",
			Description: "Embracing your uniqueness and standing out inspires others to do the same.",
		},
		{
			Name:        "Respect for the Power of Actions",
			Description: "Understanding that your actions speak louder than words helps you make a positive impact.",
		},
		{
			Name:        "Respect for the Power of Forgiveness",
			Description: "Understanding that forgiveness can heal wounds and restore relationships.",
		},
		{
			Name:        "Celebration of the Power of Sharing",
			Description: "Recognizing that sharing brings joy and strengthens relationships.",
		},
		{
			Name:        "Respect for the Power of Responsibility",
			Description: "Understanding that responsibility builds character and trust.",
		},
		{
			Name:        "Respect for the Power of Problem-Solving",
			Description: "Understanding that problem-solving helps you overcome challenges.",
		},
	}

	return morales
}

func GetRandomMorales(count int, morales Morales) Morales {
	randomMorales := make([]Morale, 0, count)

	// Ensure we don't try to select more morales than available
	if count > len(morales) {
		count = len(morales)
	}

	for i := 0; i < count; i++ {
		randomIndex := rand.Intn(len(morales))
		randomMorales = append(randomMorales, morales[randomIndex])
		// Remove the selected morale from the list to avoid duplicates
		morales = append(morales[:randomIndex], morales[randomIndex+1:]...)
	}

	return randomMorales
}

func FindMoralesByName(names []string) Morales {
	found := make(Morales, 0)
	for _, name := range names {
		for _, morale := range GetAvailableStoryMorales() {
			if morale.Name == name {
				found = append(found, morale)
			}
		}
	}

	return found
}
