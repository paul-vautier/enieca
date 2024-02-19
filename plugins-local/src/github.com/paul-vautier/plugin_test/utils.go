package plugin_test

func argMin(slice []int, indices ...int) int {
	if len(slice) == 0 {
		return -1 // Handle empty slice
	}

	minIndex := 0
	minValue := slice[0]

	if len(indices) > 0 {
		// Iterate over the provided indices
		for _, i := range indices {
			if i < 0 || i >= len(slice) {
				continue // Skip invalid indices
			}

			value := slice[i]
			if value < minValue {
				minIndex = i
				minValue = value
			}
		}
	} else {
		// Iterate over the entire slice
		for i, value := range slice {
			if value < minValue {
				minIndex = i
				minValue = value
			}
		}
	}

	return minIndex
}

func argMax(slice []int, indices ...int) int {
	if len(slice) == 0 {
		return -1 // Handle empty slice
	}

	maxIndex := 0
	maxValue := slice[0]

	if len(indices) > 0 {
		// Iterate over the provided indices
		for _, i := range indices {
			if i < 0 || i >= len(slice) {
				continue // Skip invalid indices
			}

			value := slice[i]
			if value > maxValue {
				maxIndex = i
				maxValue = value
			}
		}
	} else {
		// Iterate over the entire slice
		for i, value := range slice {
			if value > maxValue {
				maxIndex = i
				maxValue = value
			}
		}
	}

	return maxIndex
}
