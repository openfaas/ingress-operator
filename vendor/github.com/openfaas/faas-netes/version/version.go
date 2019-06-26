package version

var (
	//Version release version of the provider
	Version string
	//GitCommit SHA of the last git commit
	GitCommit string
	//DevVerison string for the development version
	DevVerison = "dev"
)

//BuildVersion returns current version of the provider
func BuildVersion() string {
	if len(Version) == 0 {
		return DevVerison
	}
	return Version
}
