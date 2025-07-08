package migrator

type Migration struct {
	version  int
	name     string
	filename string
	up       string
	down     string
}
