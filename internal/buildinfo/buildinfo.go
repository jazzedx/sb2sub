package buildinfo

var (
	version = "dev"
	commit  = "none"
	builtAt = "unknown"
)

type Data struct {
	Version string
	Commit  string
	BuiltAt string
}

func Info() Data {
	return Data{
		Version: version,
		Commit:  commit,
		BuiltAt: builtAt,
	}
}
