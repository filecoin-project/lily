package schemas

var LatestMajor = 0

func RegisterSchema(major int) {
	if major > LatestMajor {
		LatestMajor = major
	}
}

type Config struct {
	SchemaName string // name of the postgresql schema in which any database objects should be created
}
