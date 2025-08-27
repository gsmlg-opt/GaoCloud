package querylog

var digitMap []string = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}

func Format2DigitNumber(num int) string {
	return digitMap[num/10] + digitMap[num%10]
}

func Format3DigitNumber(num int) string {
	return digitMap[num/100] + digitMap[num%100/10] + digitMap[num%10]
}

func Format4DigitNumber(num int) string {
	return digitMap[num/1000] + digitMap[num%1000/100] + digitMap[num%100/10] + digitMap[num%10]
}
