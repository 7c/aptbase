package cmd

import (
	"github.com/7c/aptbase/internal/target"
	"github.com/7c/aptbase/internal/ui"
	"github.com/spf13/cobra"
)

var repoAddCmd = &cobra.Command{
	Use:   "add [repo] <file.deb...>",
	Short: "Upload and add .deb files to a repository",
	Long: `Upload one or more .deb files and add them to a local repository.

If repo is omitted, the configured default repos are used (all of them). The
add runs as an async aptly task whose progress is streamed live, then the
temporary upload directory is cleaned up. To also publish, use 'repo deploy'.`,
	Example: `  aptbase repo add app-stable ./app_1.2.3_amd64.deb
  aptbase repo add ./a_1_amd64.deb ./b_1_amd64.deb        # repos from config
  aptbase --api http://prod:8080 --api http://replica:8080 repo add app-stable ./app.deb`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoArg, files, err := splitRepoAndFiles(args)
		if err != nil {
			return err
		}
		repos, err := reposFor(repoArg)
		if err != nil {
			return err
		}
		set, err := resolveTargets()
		if err != nil {
			return err
		}
		return forEachServer(set, func(srv target.Server) error {
			return addToServer(srv, repos, files)
		})
	},
}

// addToServer uploads files once and adds them to each repo on a server.
func addToServer(srv target.Server, repos, files []string) error {
	dir := newUploadDir()
	uploaded, err := srv.Client.UploadFiles(dir, files)
	if err != nil {
		return err
	}
	ui.Success("uploaded %d file(s) to %s", len(uploaded), srv.URL)
	defer func() {
		if derr := srv.Client.DeleteUploadDir(dir); derr != nil {
			ui.Warn("could not clean up upload dir %s: %s", dir, derr.Error())
		}
	}()

	for _, repo := range repos {
		task, err := srv.Client.AddPackagesFromDir(repo, dir)
		if err != nil {
			return err
		}
		if err := runTask(srv, task, "add to "+repo); err != nil {
			return err
		}
		ui.Success("added to repo %q", repo)
	}
	return nil
}

func init() {
	repoCmd.AddCommand(repoAddCmd)
}
