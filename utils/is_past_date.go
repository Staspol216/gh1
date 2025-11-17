package utils

import "time"

func IsPastDate(date time.Time) bool {
	nowDate := time.Now()
	res := nowDate.Compare(date)

	return res == 1
}
