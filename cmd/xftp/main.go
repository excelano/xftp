// Command xftp gives SharePoint document libraries an FTP-client feel over
// Microsoft Graph: connect to a site, then list/get/put/mkdir/rm/mv files.
//
// This is the scaffold: it authenticates, binds a library, and lists the root
// to prove the auth + read path. The interactive FTP-style REPL and the
// mutating verbs (get/put/mkdir/rm/mv) come next — see internal/drive.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/excelano/xftp/internal/drive"
	"github.com/excelano/xftp/internal/spauth"
)

func configDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "xftp")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".xftp"
	}
	return filepath.Join(home, ".config", "xftp")
}

func main() {
	os.Exit(run())
}

func run() int {
	fs := flag.NewFlagSet("xftp", flag.ContinueOnError)
	site := fs.String("site", "", "SharePoint site URL, e.g. https://contoso.sharepoint.com/sites/Marketing (required)")
	library := fs.String("library", "", "Document library display name (default: the site's default library)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: xftp --site <site-url> [--library <name>]")
		fmt.Fprintln(os.Stderr)
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Authentication is device-code via Microsoft Graph; refresh tokens are")
		fmt.Fprintln(os.Stderr, "cached at "+filepath.Join(configDir(), "sp-token.json")+".")
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if *site == "" {
		fmt.Fprintln(os.Stderr, "Error: --site is required")
		fs.Usage()
		return 2
	}

	ctx := context.Background()
	tokenCachePath := filepath.Join(configDir(), "sp-token.json")

	client, err := spauth.NewPublicClient(tokenCachePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup error: %v\n", err)
		return 1
	}

	result, err := spauth.Authenticate(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Authentication failed: %v%s\n", err, spauth.HintForAuthError(err))
		return 1
	}

	graph := spauth.NewGraphClient(client, result.Account)

	d, err := drive.ResolveDrive(ctx, graph, *site, *library)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to bind library: %v\n", err)
		return 1
	}

	fmt.Fprintf(os.Stderr, "Authenticated as: %s\n", result.Account.PreferredUsername)
	fmt.Fprintf(os.Stderr, "Connected to: %s / %s\n\n", d.Hostname, d.Name)

	items, err := d.List(ctx, graph, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "List failed: %v\n", err)
		return 1
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	for _, it := range items {
		kind := "-"
		if it.IsFolder {
			kind = "d"
		}
		fmt.Fprintf(tw, "%s\t%10d\t%s\t%s\n", kind, it.Size, it.LastModified.Format("2006-01-02 15:04"), it.Name)
	}
	tw.Flush()
	return 0
}
