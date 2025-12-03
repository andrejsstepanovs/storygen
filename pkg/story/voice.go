package story

import "fmt"

type Voice struct {
	Provider    VoiceProvider
	Instruction VoiceInstruction
}

type VoiceProvider struct {
	Provider string
	Voice    string
	Speed    float64
}

type VoiceInstruction struct {
	Affect  string
	Tone    string
	Emotion string
	Pauses  string
	Pacing  string
	Story   Story
}

func (vi VoiceInstruction) String() string {
	protagonists := vi.Story.Protagonists.String()
	return fmt.Sprintf(
		"Voice Affect: %s\n\n"+
			"Tone: %s\n\n"+
			"Pacing: %s\n\n"+
			"Emotion: %s\n\n"+
			"Pauses: %s\n\n"+
			"Reading: Story\n\n"+
			"Story protagonists: %s\n"+
			"Story villain: %s\n"+
			"Story villain Voice: %s\n",
		vi.Affect, vi.Tone, vi.Pacing, vi.Emotion, vi.Pauses,
		protagonists, vi.Story.Villain, vi.Story.VillainVoice,
	)
}
