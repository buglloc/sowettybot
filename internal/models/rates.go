package models

import "time"

type Rate struct {
	Name string
	When time.Time
	Rate float64
}

type Rates []Rate
