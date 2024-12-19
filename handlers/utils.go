package handlers

// findColumnIndex searches for a column by name and returns its index.
// Returns -1 if the column is not found.
func findColumnIndex(columns []string, columnName string) int {
	for i, col := range columns {
		if col == columnName {
			return i
		}
	}
	return -1
}