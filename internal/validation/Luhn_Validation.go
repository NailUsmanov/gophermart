package validation

import "strconv"

type OrderValidation interface {
	IsValidLuhn(number string) bool
}

type LuhnValidation struct{}

func (l *LuhnValidation) IsValidLuhn(number string) bool {
	valNumb := make([]int, 0)
	for _, v := range number {
		vInt, err := strconv.Atoi(string(v))
		if err != nil || vInt < 0 || vInt > 9 {
			return false
		}
		valNumb = append(valNumb, vInt)
	}
	sum := 0
	for i := len(valNumb) - 1; i >= 0; i-- {
		n := valNumb[i]
		if (len(valNumb)-1-i)%2 == 1 {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
	}
	return sum%10 == 0
}
