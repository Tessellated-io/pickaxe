package tx

type SimulationResult struct {
	GasRecommendation uint64
}

type SigningMetadata struct {
	address       string
	accountNumber uint64
	sequence      uint64
}

func (sm *SigningMetadata) Address() string {
	return sm.address
}

func (sm *SigningMetadata) AccountNumber() uint64 {
	return sm.accountNumber
}

func (sm *SigningMetadata) Sequence() uint64 {
	return sm.sequence
}
