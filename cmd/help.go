package cmd

import (
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Colors used in help output. fatih/color automatically disables ANSI codes
// when stdout is not a terminal or NO_COLOR is set.
var (
	cHeader  = color.New(color.Bold, color.FgCyan)
	cCommand = color.New(color.FgGreen)
	cFlag    = color.New(color.FgYellow)
	cComment = color.New(color.Faint)
)

var flagTokenRE = regexp.MustCompile(`(^|\s)(--?[A-Za-z][A-Za-z0-9-]*)`)

// colorFlags highlights flag tokens (e.g. --api, -d) within a flag-usage block.
func colorFlags(s string) string {
	return flagTokenRE.ReplaceAllStringFunc(s, func(m string) string {
		sub := flagTokenRE.FindStringSubmatch(m)
		return sub[1] + cFlag.Sprint(sub[2])
	})
}

// colorExamples dims comment lines (starting with #) in an example block.
func colorExamples(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			lines[i] = cComment.Sprint(line)
		}
	}
	return strings.Join(lines, "\n")
}

func init() {
	// Help short-circuits PersistentPreRunE, so honor --no-color here too.
	for _, a := range os.Args[1:] {
		if a == "--no-color" {
			color.NoColor = true
			break
		}
	}

	cobra.AddTemplateFunc("hdr", func(s string) string { return cHeader.Sprint(s) })
	cobra.AddTemplateFunc("cmdName", func(s string) string { return cCommand.Sprint(s) })
	cobra.AddTemplateFunc("flagStyle", colorFlags)
	cobra.AddTemplateFunc("examples", colorExamples)

	rootCmd.SetUsageTemplate(usageTemplate)
}

// usageTemplate mirrors cobra's default usage layout with color applied to
// section headers, command names, flags, and example comments.
const usageTemplate = `{{hdr "Usage:"}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

{{hdr "Aliases:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{hdr "Examples:"}}
{{.Example | examples}}{{end}}{{if .HasAvailableSubCommands}}

{{hdr "Available Commands:"}}{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding | cmdName}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{hdr "Flags:"}}
{{.LocalFlags.FlagUsages | flagStyle | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{hdr "Global Flags:"}}
{{.InheritedFlags.FlagUsages | flagStyle | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

{{hdr "Additional help topics:"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding | cmdName}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
