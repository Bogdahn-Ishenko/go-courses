package main

import "fmt"

func main() {
	result := twoSum([]int{-1, -2, -3}, -5)
	fmt.Println(result)
}

func twoSum(nums []int, target int) []int {
	var numMap = map[int]int{}

	for key, num := range nums {
		var need = target - num
		value, ok := numMap[need]
		if ok {
			return []int{key, value}
		}
		numMap[num] = key
	}

	return []int{}
}
