package cli

import (
	"os"
	"strings"

	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type strategySourceFlags struct {
	Engine   string
	Input    string
	JQ       string
	JQFile   string
	File     string
	Version  string
	SpecHash string
	Alias    string
	History  int
}

func addStrategySourceFlags(cmd *cobra.Command, flags *strategySourceFlags, includeEngine bool) {
	if includeEngine {
		cmd.Flags().StringVar(&flags.Engine, "engine", flags.Engine, "strategy engine: jq")
	}
	cmd.Flags().StringVar(&flags.Input, "input", flags.Input, "input dataset name")
	addJQFlags(cmd, flags)
}

func addJQFlags(cmd *cobra.Command, flags *strategySourceFlags) {
	cmd.Flags().StringVar(&flags.JQ, "jq", flags.JQ, "inline jq query")
	cmd.Flags().StringVar(&flags.JQFile, "jq-file", flags.JQFile, "path to a jq query file")
	mustMarkFlagFilename(cmd, "jq-file", "jq")
}

func resolveJQSource(flags strategySourceFlags) (string, error) {
	errb := oops.In("cli").With("jq_file", flags.JQFile)
	hasInline := strings.TrimSpace(flags.JQ) != ""
	hasFile := strings.TrimSpace(flags.JQFile) != ""
	if hasInline == hasFile {
		return "", errb.New("exactly one of --jq or --jq-file is required")
	}
	if hasInline {
		return flags.JQ, nil
	}
	data, err := os.ReadFile(flags.JQFile)
	if err != nil {
		return "", errb.Wrapf(err, "read jq file")
	}
	source := string(data)
	if strings.TrimSpace(source) == "" {
		return "", errb.New("jq file is empty")
	}
	return source, nil
}
