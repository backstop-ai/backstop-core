package testdata

func InsecureQuery(userID string) string {
	q := "SELECT * FROM users WHERE id = " + userID
	return q
}
