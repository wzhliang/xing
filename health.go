package xing

import "github.com/wzhliang/xing/utils"

// HealthChecker ...
type HealthChecker interface {
	Healthy() bool
}

// DumbHealthChecker ...
type DumbHealthChecker struct {
}

// Healthy ...
func (dhc *DumbHealthChecker) Healthy() bool {
	return true
}

// DefaultHealthChecker ...
func DefaultHealthChecker() HealthChecker {
	return &DumbHealthChecker{}
}

// RandomHealthChecker ...
type RandomHealthChecker struct {
}

// Healthy ...
func (rhc *RandomHealthChecker) Healthy() bool {
	return utils.Random(1, 100) < 90
}
