package cmd

// cliFlags holds parsed command-line flags.
type cliFlags struct {
	showVersion  bool
	showConfig   bool
	listModels   bool
	download     bool
	downloadRepo string
	modelName    string
	revision     string
	force        bool
	model        string
	device       string
	maxTokens    int
	maxTokensSet bool
}
