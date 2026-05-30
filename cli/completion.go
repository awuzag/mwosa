package cli

import (
	"fmt"

	"github.com/awuzag/mwosa/providers/builtin"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type completionShell struct {
	Name        string
	Description string
}

var supportedCompletionShells = []completionShell{
	{Name: "bash", Description: "Generate Bash completion script"},
	{Name: "zsh", Description: "Generate Zsh completion script"},
	{Name: "fish", Description: "Generate Fish completion script"},
	{Name: "powershell", Description: "Generate PowerShell completion script"},
}

func newCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion <shell>",
		Short: "Generate shell completion script",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return oops.In("cli").With("args", len(args)).New("completion requires one shell argument")
			}
			if !completionShellSupported(args[0]) {
				return oops.In("cli").With("shell", args[0]).Errorf("unsupported completion shell: %s", args[0])
			}
			return nil
		},
		ValidArgsFunction: cobra.FixedCompletions(
			completionShellChoices(),
			cobra.ShellCompDirectiveNoFileComp,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := generateCompletion(cmd, args[0]); err != nil {
				return oops.In("cli").With("shell", args[0]).Wrapf(err, "generate completion")
			}
			return nil
		},
	}
	return cmd
}

func completionShellSupported(shell string) bool {
	for _, supported := range supportedCompletionShells {
		if shell == supported.Name {
			return true
		}
	}
	return false
}

func completionShellChoices() []cobra.Completion {
	completions := make([]cobra.Completion, 0, len(supportedCompletionShells))
	for _, supported := range supportedCompletionShells {
		completions = append(completions, cobra.CompletionWithDesc(supported.Name, supported.Description))
	}
	return completions
}

func generateCompletion(cmd *cobra.Command, shell string) error {
	root := cmd.Root()
	out := cmd.OutOrStdout()
	switch shell {
	case "bash":
		return root.GenBashCompletionV2(out, true)
	case "zsh":
		return root.GenZshCompletion(out)
	case "fish":
		return root.GenFishCompletion(out, true)
	case "powershell":
		return root.GenPowerShellCompletionWithDesc(out)
	default:
		return oops.In("cli").With("shell", shell).Errorf("unsupported completion shell: %s", shell)
	}
}

func registerRootCompletions(cmd *cobra.Command) {
	mustMarkPersistentFlagFilename(cmd, "config", "json")
	mustMarkPersistentFlagFilename(cmd, "database", "db", "sqlite", "sqlite3")
	mustRegisterFlagCompletion(cmd, "output", completeOutputModes)
	mustRegisterFlagCompletion(cmd, "provider", completeProviderIDs)
	mustRegisterFlagCompletion(cmd, "prefer-provider", completeProviderIDs)
	mustRegisterFlagCompletion(cmd, "market", completeMarkets)
}

func completeOutputModes(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	completions := make([]cobra.Completion, 0, len(supportedOutputModes))
	descriptions := map[OutputMode]string{
		OutputModeTable:  "Human-readable table",
		OutputModeJSON:   "Machine-readable JSON",
		OutputModeNDJSON: "Newline-delimited JSON",
		OutputModeCSV:    "Comma-separated values",
	}
	for _, mode := range supportedOutputModes {
		completions = append(completions, cobra.CompletionWithDesc(string(mode), descriptions[mode]))
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeProviderIDs(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	builders := builtin.Builders()
	completions := make([]cobra.Completion, 0, len(builders))
	for _, builder := range builders {
		completions = append(completions, cobra.CompletionWithDesc(string(builder.ID()), "Provider ID"))
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeMarkets(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return []cobra.Completion{
		cobra.CompletionWithDesc(string(provider.MarketKRX), "Korea Exchange"),
	}, cobra.ShellCompDirectiveNoFileComp
}

func completeSecurityTypes(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return []cobra.Completion{
		cobra.CompletionWithDesc(string(provider.SecurityTypeStock), "Listed stock"),
		cobra.CompletionWithDesc(string(provider.SecurityTypeETF), "Exchange-traded fund"),
		cobra.CompletionWithDesc(string(provider.SecurityTypeETN), "Exchange-traded note"),
		cobra.CompletionWithDesc(string(provider.SecurityTypeELW), "Equity-linked warrant"),
	}, cobra.ShellCompDirectiveNoFileComp
}

func completeFinancialPeriods(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return []cobra.Completion{
		cobra.CompletionWithDesc("annual", "Annual financial statements"),
		cobra.CompletionWithDesc("quarter", "Quarterly financial statements"),
	}, cobra.ShellCompDirectiveNoFileComp
}

func completeFinancialStatementTypes(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return []cobra.Completion{
		cobra.CompletionWithDesc("summary", "Summary financial statement"),
		cobra.CompletionWithDesc("income_statement", "Income statement"),
		cobra.CompletionWithDesc("balance_sheet", "Balance sheet"),
		cobra.CompletionWithDesc("cash_flow", "Cash flow statement"),
	}, cobra.ShellCompDirectiveNoFileComp
}

func mustRegisterFlagCompletion(cmd *cobra.Command, flagName string, completion cobra.CompletionFunc) {
	if err := cmd.RegisterFlagCompletionFunc(flagName, completion); err != nil {
		panic(fmt.Sprintf("register %s completion: %v", flagName, err))
	}
}

func mustMarkFlagFilename(cmd *cobra.Command, flagName string, extensions ...string) {
	if err := cmd.MarkFlagFilename(flagName, extensions...); err != nil {
		panic(fmt.Sprintf("mark %s filename completion: %v", flagName, err))
	}
}

func mustMarkPersistentFlagFilename(cmd *cobra.Command, flagName string, extensions ...string) {
	if err := cmd.MarkPersistentFlagFilename(flagName, extensions...); err != nil {
		panic(fmt.Sprintf("mark persistent %s filename completion: %v", flagName, err))
	}
}

func skipConfigLoadForCompletion(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "completion", cobra.ShellCompRequestCmd:
			return true
		}
	}
	return false
}
