package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/7c/aptbase/internal/config"
	"github.com/7c/aptbase/internal/render"
	"github.com/7c/aptbase/internal/ui"
	"github.com/spf13/cobra"
)

var (
	configNewForce bool
	configNewPath  string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage and inspect aptbase configuration",
	Long: `Create a starter config file or inspect the fully resolved configuration.

Configuration is layered (last wins): defaults, then /etc/aptbase/config.ini,
then ~/.config/aptbase/config.ini, then APTBASE_* env vars, then CLI flags.`,
}

var configNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Write an annotated default config.ini",
	Long: `Scaffold a commented config.ini you can edit.

By default it writes to /etc/aptbase/config.ini (which usually needs sudo). Use
--path to write somewhere else, e.g. a per-user file. Existing files are not
overwritten unless --force is given.`,
	Example: `  sudo aptbase config new
  aptbase config new --path ./aptbase.ini
  aptbase config new --path ~/.config/aptbase/config.ini --force`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := configNewPath
		if path == "" {
			path = config.SystemConfigPath
		}
		if err := config.WriteDefault(path, configNewForce); err != nil {
			if errors.Is(err, os.ErrPermission) || strings.Contains(err.Error(), "permission denied") {
				ui.Warn("permission denied writing %s — try 'sudo aptbase config new' or '--path ./aptbase.ini'", path)
			}
			return err
		}
		ui.Success("wrote default config to %s", path)
		ui.Dim("edit it, then run 'aptbase config list' to verify resolved values")
		return nil
	},
}

var configPrintCmd = &cobra.Command{
	Use:   "print",
	Short: "Print the resolved configuration as config.ini to stdout",
	Long: `Render the fully resolved configuration (config file + env vars + flags)
as a config.ini and write it to stdout, so you can review or pipe it to a file.

Unlike 'config new' (which writes a static annotated template), 'config print'
reflects the current effective settings — flags and env included — making it
easy to capture an ad-hoc invocation as a config file.`,
	Example: `  # Capture the current settings, overriding the API on the way
  aptbase --api http://aptbase:8080 config print

  # Write a system config from flags
  aptbase --api http://aptbase:8080 -d noble -d jammy --user deploy config print | sudo tee /etc/aptbase/config.ini`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(config.Render(settings))
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show every resolved setting and where it came from",
	Long: `Print each configuration setting with its resolved value and source
(default / config file path / env var / flag), reflecting config files and CLI
overrides combined.`,
	Example: `  aptbase config list
  aptbase --api http://localhost:8080 -d noble config list
  aptbase --config ./aptbase.ini config list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		type entry struct {
			Setting string `json:"setting"`
			Value   string `json:"value"`
			Source  string `json:"source"`
			Default string `json:"default"`
		}
		def := config.Defaults()
		entries := []entry{
			{config.KeyAPI, none(strings.Join(settings.APIs, " ")), settings.Source(config.KeyAPI), none(strings.Join(def.APIs, " "))},
			{config.KeyServer, none(settings.Server), sourceOrDefault(settings.Server, "flag/env"), none(def.Server)},
			{config.KeyUser, none(settings.User), settings.Source(config.KeyUser), none(def.User)},
			{config.KeyPassword, maskPassword(settings), settings.Source(config.KeyPassword), "(prompt on 401)"},
			{config.KeyDistributions, none(strings.Join(settings.Distributions, " ")), settings.Source(config.KeyDistributions), none(strings.Join(def.Distributions, " "))},
			{config.KeyRepos, none(strings.Join(settings.Repos, " ")), settings.Source(config.KeyRepos), none(strings.Join(def.Repos, " "))},
			{config.KeyPrefix, settings.Prefix, settings.Source(config.KeyPrefix), def.Prefix},
			{config.KeyInsecure, boolStr(settings.Insecure), settings.Source(config.KeyInsecure), boolStr(def.Insecure)},
			{config.KeyTimeout, settings.Timeout.String(), settings.Source(config.KeyTimeout), def.Timeout.String()},
			{config.KeyJSON, boolStr(settings.JSON), settings.Source(config.KeyJSON), boolStr(def.JSON)},
			{config.KeyNoColor, boolStr(settings.NoColor), settings.Source(config.KeyNoColor), boolStr(def.NoColor)},
			{config.KeyYes, boolStr(settings.Yes), settings.Source(config.KeyYes), boolStr(def.Yes)},
		}

		if settings.JSON {
			return render.JSON(entries)
		}

		rows := make([][]string, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, []string{e.Setting, e.Value, e.Source, e.Default})
		}
		ui.Table([]string{"SETTING", "VALUE", "SOURCE", "DEFAULT"}, rows)
		return nil
	},
}

func maskPassword(s *config.Settings) string {
	if s.HasPassword && s.Password != "" {
		return "********"
	}
	return "(prompt on 401)"
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func sourceOrDefault(val, src string) string {
	if val == "" {
		return "default"
	}
	return src
}

func init() {
	configNewCmd.Flags().BoolVar(&configNewForce, "force", false, "overwrite an existing config file")
	configNewCmd.Flags().StringVar(&configNewPath, "path", "", "destination path (default /etc/aptbase/config.ini)")
	configCmd.AddCommand(configNewCmd, configPrintCmd, configListCmd)
	rootCmd.AddCommand(configCmd)
}
