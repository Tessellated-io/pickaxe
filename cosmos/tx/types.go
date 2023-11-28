package tx

type SimulationResult struct {
	GasRecommendation uint64
}

type SigningMetadata struct {
	Address       string
	AccountNumber uint64
	Sequence      uint64
}
