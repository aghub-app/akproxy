package cli

import (
	"io"
	"path/filepath"

	"akproxy/internal/desktop"
	"github.com/spf13/cobra"
)

// Version is the release injected at build time. Dev builds stay "dev".
var Version = "dev"

// Execute parses args with cobra and runs the command. Help is written to stdout.
func Execute(stdout io.Writer, paths desktop.Paths, args []string) error {
	root := newRoot(stdout, paths)
	root.SetOut(stdout)
	root.SetErr(stdout)
	root.SetArgs(args)
	return root.Execute()
}

func newRoot(stdout io.Writer, paths desktop.Paths) *cobra.Command {
	root := &cobra.Command{
		Use:               "akproxy",
		Short:             "把本机代理接到终端",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Version:           Version,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.AddCommand(
		modelCommand(stdout, paths),
		providerCommand(paths, "claude"),
		providerCommand(paths, "codex"),
		providerCommand(paths, "opencode"),
		providerCommand(paths, "pi"),
	)
	return root
}

func modelCommand(stdout io.Writer, paths desktop.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "model",
		Short: "从本机代理选择一个模型",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return selectModel(stdout, paths, pickWithGum)
		},
	}
}

func providerCommand(paths desktop.Paths, name string) *cobra.Command {
	command := &cobra.Command{
		Use:                name + " [参数...]",
		Short:              "用本机代理启动 " + name,
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return runProvider(paths, name, args)
		},
	}
	return command
}

func runProvider(paths desktop.Paths, name string, args []string) error {
	if err := requireApp(paths.Config); err != nil {
		return err
	}
	model, err := ReadModelID(filepath.Join(paths.Root, "cli.json"))
	if err != nil {
		return err
	}
	base, key, err := desktop.ClientCredential(paths.Config)
	if err != nil {
		return err
	}
	launch, err := BuildLaunch(name, base, key, model, args)
	if err != nil {
		return err
	}
	return start(launch)
}
