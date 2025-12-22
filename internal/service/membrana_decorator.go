package pvz_service

type MembranaDecorator struct {
	Strategy PackagingStrategy
}

func (md *MembranaDecorator) Validate(baseCost float64) error {
	return md.Strategy.Validate(baseCost)
}

func (md *MembranaDecorator) CalculateWorth(baseCost float64) float64 {
	return md.Strategy.CalculateWorth(baseCost) + 1.0
}
