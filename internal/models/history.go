package models

import "time"

type History struct {
	When   time.Time
	Names  []string
	Values []float64
}
