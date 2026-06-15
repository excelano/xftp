# xftp

xftp gives a SharePoint document library the feel of an FTP session. You connect to a site, land at an interactive prompt, and move files around with the verbs your fingers already know: `ls`, `cd`, `get`, `put`, `mkdir`, `rm`, `mv`. There is no SSH, FTP, or SCP server behind SharePoint to connect to — those protocols simply don't exist there — so xftp recreates the experience on top of the Microsoft Graph drive API and hides the Graph plumbing entirely.

It is a single static Go binary with no daemon and no mounted filesystem. Authentication is device-code OAuth: the first time you connect, xftp prints a short code and a URL, you sign in once in a browser, and the refresh token is cached under `~/.config/xftp` so later runs are silent.

## Connecting

Point xftp at a SharePoint site URL. With no library named, it binds the site's default document library; pass `--library` to pick another by its display name.

```
xftp --site https://contoso.sharepoint.com/sites/Marketing
xftp --site https://contoso.sharepoint.com/sites/Marketing --library "Project Files"
```

Once connected you're at the prompt, which shows your position in the library:

```
xftp:/> ls
xftp:/> cd "Shared Documents"
xftp:/Shared Documents> get "Q1 Plan.xlsx"
xftp:/Shared Documents> put report.pdf Archive/report.pdf
```

Paths may be relative to the current folder or absolute with a leading `/`, and `.`/`..` work as you'd expect. Names containing spaces are double-quoted.

## Commands

| Command | What it does |
|---|---|
| `ls [path]` | List a remote folder. Defaults to the current folder. |
| `cd [path]` | Change remote folder. With no argument, prints the current folder. |
| `pwd` | Print the current remote folder. |
| `get <remote> [local]` | Download a file. Defaults the local name to the remote's. |
| `put <local> [remote]` | Upload a file (up to 250 MB). Defaults the remote name to the local's. |
| `mkdir <path>` | Create a remote folder. |
| `rm <path>` | Delete a file. Folders are recursive, so they prompt for confirmation first. |
| `mv <src> <dst>` | Move or rename a remote item. |
| `lcd [dir]` | Change the local working folder for `get`/`put`. With no argument, prints it. |
| `lpwd` | Print the local working folder. |
| `lls [dir]` | List a local folder. |
| `help` | Show the command list. |
| `quit` | Exit. |

Deleting a single file goes straight through, since SharePoint routes it to the recycle bin and it can be recovered there. Deleting a folder is recursive and irreversible from xftp's side, so it asks first.

## Authentication and tenants

xftp authenticates through a multi-tenant Azure app registration ("Excelano SharePoint tools"), shared with its sibling tool [xql](https://github.com/excelano/xql), so consenting once covers both. Pointing xftp at another organization's site uses that same registration — nobody sets up their own. The first connection to a new tenant raises a one-time consent prompt; depending on that tenant's policy, either the user or an administrator clears it, after which everyone in the tenant is covered. The single scope requested is `Sites.ReadWrite.All`.

To use your own app registration instead, change `defaultClientID` in `internal/spauth/auth.go` and rebuild.

## Building

```
go build -o xftp ./cmd/xftp
```

## Limitations

Uploads use a single request, which Graph caps at 250 MB; larger files are rejected rather than truncated. Resumable upload sessions for big files are the planned next step.

---

Built by David M. Anderson, with AI assistance.
