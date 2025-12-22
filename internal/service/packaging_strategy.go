package pvz_service

import "errors"

type PackagingStrategy interface {
	Validate(weight float64) error
	CalculateWorth(baseCost float64) float64
}

type PackagingBagStrategy struct{}
type PackagingMembranaStrategy struct{}
type PackagingBoxStrategy struct{}

func (pbs *PackagingBagStrategy) Validate(weight float64) error {
	if weight > 10.00 {
		return errors.New("order should be less than 10kg with bag package")
	}
	return nil
}
func (pbs *PackagingBagStrategy) CalculateWorth(baseCost float64) float64 {
	return baseCost + 5
}

func (pbs *PackagingMembranaStrategy) Validate(weight float64) error {
	return nil
}
func (pbs *PackagingMembranaStrategy) CalculateWorth(baseCost float64) float64 {
	return baseCost + 1
}

func (pbs *PackagingBoxStrategy) Validate(weight float64) error {
	if weight > 20.00 {
		return errors.New("order should be less than 20kg with box package")
	}
	return nil
}
func (pbs *PackagingBoxStrategy) CalculateWorth(baseCost float64) float64 {
	return baseCost + 20
}
