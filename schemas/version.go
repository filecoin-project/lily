package schemas

var LatestMajor = 0

func RegisterSchema(major int) {
	if major > LatestMajor {
		LatestMajor = major
	}
}
