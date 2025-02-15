package dateutil

import (
	"fmt"
	"time"
)

type Date struct {
	Year  int
	Month int
	Day   int
}

func GetCurrent() Date {
	t := time.Now()
	return Date{Year: t.Year(), Month: int(t.Month()), Day: t.Day()}
}

func (d Date) String() string {
	return fmt.Sprintf("%d/%02d/%02d", d.Year, d.Month, d.Day)
}
