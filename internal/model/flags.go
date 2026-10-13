package model

const (
	flagPointsStandard  = 10
	flagPointsAIVector  = 15
	flagPointsOSAIProof = 5
)

// DefaultFlagTypes returns a fresh default flag set for an engagement type.
func DefaultFlagTypes(t EngagementType) []FlagType {
	switch t {
	case EngagementLab, EngagementOSCP:
		return []FlagType{
			{ID: "local", Label: "Local", Points: flagPointsStandard},
			{ID: "proof", Label: "Proof", Points: flagPointsStandard},
		}
	case EngagementOSAI:
		return []FlagType{
			{ID: "ai_vector", Label: "AI vector", Points: flagPointsAIVector},
			{ID: "trad_vector", Label: "Traditional vector", Points: flagPointsStandard},
			{ID: "proof", Label: "Proof", Points: flagPointsOSAIProof},
		}
	case EngagementCustom:
		return []FlagType{}
	default:
		return []FlagType{}
	}
}
