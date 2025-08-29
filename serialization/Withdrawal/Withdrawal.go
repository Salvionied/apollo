package Withdrawal

import (
	"fmt"
)

type Withdrawal map[[29]byte]int

func New() Withdrawal {
	m := make(map[[29]byte]int)
	return m
}

func (w *Withdrawal) Add(stakeAddress [29]byte, amount int) error {
	_, exists := (*w)[stakeAddress]
	if exists {
		return fmt.Errorf(
			"Withdrawal.Add: key already exists in map: %v",
			stakeAddress,
		)
	}
	(*w)[stakeAddress] = amount
	return nil
}

func (w *Withdrawal) Size() int {
	return len(*w)
}
