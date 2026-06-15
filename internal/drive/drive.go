// Package drive is xftp's SharePoint document-library client: it resolves a
// site URL to a Graph drive, then exposes FTP-shaped operations over that drive
// (list, download, upload, mkdir, remove, move).
//
// STATUS: stub. ResolveDrive and List are implemented so the auth + read path
// can be exercised end to end. The mutating operations (Download, Upload,
// Mkdir, Remove, Move) are stubbed with their exact Graph endpoints documented
// inline; fill them in next.
package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/excelano/xftp/internal/spauth"
)

// ErrNotImplemented marks the drive operations still to be written.
var ErrNotImplemented = fmt.Errorf("not implemented yet")

// Drive is a resolved SharePoint document library the session operates on.
// SourceURL is kept so a future REPL "refresh"/reconnect can re-bind without
// re-prompting.
type Drive struct {
	SiteID    string
	DriveID   string
	Name      string
	Hostname  string
	SitePath  string
	SourceURL string
}

// Item is one entry in a drive folder listing — the unit an FTP-style "ls"
// prints and "cd" descends into.
type Item struct {
	Name         string
	ID           string
	IsFolder     bool
	Size         int64
	ChildCount   int
	LastModified time.Time
}

// parseSiteURL extracts the hostname and server-relative site path from a
// SharePoint URL such as https://contoso.sharepoint.com/sites/Marketing or a
// deep link into a library/folder under it. Anything at or below the document
// library is treated as site path for resolution; library/folder selection is
// handled separately by ResolveDrive + List.
//
// Adapted from xql's parseListURL, minus the /Lists/ requirement.
func parseSiteURL(rawURL string) (hostname, sitePath string, err error) {
	u, perr := url.Parse(rawURL)
	if perr != nil {
		return "", "", fmt.Errorf("parsing site URL: %w", perr)
	}
	if u.Host == "" {
		return "", "", fmt.Errorf("site URL has no host: %s", rawURL)
	}
	hostname = u.Host

	path := strings.Trim(u.Path, "/")
	if path == "" {
		return hostname, "", nil
	}
	parts := strings.Split(path, "/")

	// A canonical site URL is /sites/{name} or /teams/{name}; keep just that
	// prefix as the site path so deep links into libraries still resolve.
	if len(parts) >= 2 && (strings.EqualFold(parts[0], "sites") || strings.EqualFold(parts[0], "teams")) {
		sitePath = "/" + parts[0] + "/" + parts[1]
		return hostname, sitePath, nil
	}

	// Root site (no /sites/ segment).
	return hostname, "", nil
}

// ResolveDrive resolves a SharePoint site URL to a Graph drive. With library
// empty it binds the site's default document library ("Documents"/"Shared
// Documents"); otherwise it matches a library by display name.
func ResolveDrive(ctx context.Context, g *spauth.GraphClient, siteURL, library string) (*Drive, error) {
	hostname, sitePath, err := parseSiteURL(siteURL)
	if err != nil {
		return nil, err
	}

	siteID, err := resolveSiteID(ctx, g, hostname, sitePath)
	if err != nil {
		return nil, fmt.Errorf("resolving site: %w", err)
	}

	driveID, name, err := resolveDriveID(ctx, g, siteID, library)
	if err != nil {
		return nil, fmt.Errorf("resolving library: %w", err)
	}

	return &Drive{
		SiteID:    siteID,
		DriveID:   driveID,
		Name:      name,
		Hostname:  hostname,
		SitePath:  sitePath,
		SourceURL: siteURL,
	}, nil
}

// resolveSiteID mirrors xql's resolveSiteID: GET /sites/{host}:{path}.
func resolveSiteID(ctx context.Context, g *spauth.GraphClient, hostname, sitePath string) (string, error) {
	var path string
	if sitePath == "" {
		path = fmt.Sprintf("/sites/%s", hostname)
	} else {
		path = fmt.Sprintf("/sites/%s:%s", hostname, sitePath)
	}
	body, err := g.Get(ctx, path, nil)
	if err != nil {
		return "", err
	}
	var site struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &site); err != nil {
		return "", fmt.Errorf("decoding site response: %w", err)
	}
	if site.ID == "" {
		return "", fmt.Errorf("site response missing id")
	}
	return site.ID, nil
}

// resolveDriveID returns the default drive when library is empty, else the
// drive whose display name matches (case-insensitive).
func resolveDriveID(ctx context.Context, g *spauth.GraphClient, siteID, library string) (id, name string, err error) {
	if library == "" {
		body, err := g.Get(ctx, fmt.Sprintf("/sites/%s/drive", siteID), nil)
		if err != nil {
			return "", "", err
		}
		var d struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &d); err != nil {
			return "", "", fmt.Errorf("decoding default drive: %w", err)
		}
		return d.ID, d.Name, nil
	}

	raws, err := g.GetAll(ctx, fmt.Sprintf("/sites/%s/drives", siteID), url.Values{
		"$select": {"id,name"},
	})
	if err != nil {
		return "", "", err
	}
	var names []string
	for _, raw := range raws {
		var d struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return "", "", fmt.Errorf("decoding drive entry: %w", err)
		}
		names = append(names, d.Name)
		if strings.EqualFold(d.Name, library) {
			return d.ID, d.Name, nil
		}
	}
	return "", "", fmt.Errorf("no library named %q (found: %s)", library, strings.Join(names, ", "))
}

// itemRef builds the Graph drive-item addressing segment for a library-relative
// path. "" or "/" addresses the drive root; otherwise each segment is
// percent-escaped and wrapped in the root:/<path>: path-addressing form.
func itemRef(path string) string {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return "/root"
	}
	segs := strings.Split(clean, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return "/root:/" + strings.Join(segs, "/") + ":"
}

// driveItemJSON is the subset of a Graph driveItem xftp reads.
type driveItemJSON struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Size                 int64  `json:"size"`
	LastModifiedDateTime string `json:"lastModifiedDateTime"`
	Folder               *struct {
		ChildCount int `json:"childCount"`
	} `json:"folder"`
	File *json.RawMessage `json:"file"`
}

func (j driveItemJSON) toItem() Item {
	it := Item{ID: j.ID, Name: j.Name, Size: j.Size}
	if j.Folder != nil {
		it.IsFolder = true
		it.ChildCount = j.Folder.ChildCount
	}
	if t, err := time.Parse(time.RFC3339, j.LastModifiedDateTime); err == nil {
		it.LastModified = t
	}
	return it
}

// List returns the children of a library-relative folder path ("" or "/" for
// the library root). This is the read primitive behind an FTP "ls".
func (d *Drive) List(ctx context.Context, g *spauth.GraphClient, path string) ([]Item, error) {
	endpoint := fmt.Sprintf("/drives/%s%s/children", d.DriveID, itemRef(path))
	raws, err := g.GetAll(ctx, endpoint, url.Values{
		"$select": {"id,name,size,lastModifiedDateTime,folder,file"},
	})
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(raws))
	for _, raw := range raws {
		var j driveItemJSON
		if err := json.Unmarshal(raw, &j); err != nil {
			return nil, fmt.Errorf("decoding drive item: %w", err)
		}
		items = append(items, j.toItem())
	}
	return items, nil
}

// Download streams the content of a remote file at the library-relative path
// into w. (FTP "get".)
//
// TODO: GET /drives/{driveID}/root:/{path}:/content — returns a 302 to a
// pre-authed URL that spauth.GraphClient.Get already follows. Stream the body
// to w instead of buffering for large files.
func (d *Drive) Download(ctx context.Context, g *spauth.GraphClient, path string, w io.Writer) error {
	return ErrNotImplemented
}

// Upload writes local content to the library-relative remote path. (FTP "put".)
//
// TODO: simple upload is PUT /drives/{driveID}/root:/{path}:/content via
// spauth.GraphClient.PutRaw for files up to 250MB. Above that, create an upload
// session: POST .../createUploadSession then PUT byte ranges to its uploadUrl.
func (d *Drive) Upload(ctx context.Context, g *spauth.GraphClient, path, contentType string, data []byte) error {
	return ErrNotImplemented
}

// Mkdir creates a folder at the library-relative path. (FTP "mkdir".)
//
// TODO: POST /drives/{driveID}{itemRef(parent)}/children with body
// {"name": <leaf>, "folder": {}, "@microsoft.graph.conflictBehavior": "fail"}.
// For a top-level folder the parent ref is "/root".
func (d *Drive) Mkdir(ctx context.Context, g *spauth.GraphClient, path string) error {
	return ErrNotImplemented
}

// Remove deletes the file or folder at the library-relative path. (FTP
// "delete"/"rmdir".) Caller should confirm before calling — folder deletes are
// recursive in Graph.
//
// TODO: DELETE /drives/{driveID}/root:/{path}: via spauth.GraphClient.Delete.
func (d *Drive) Remove(ctx context.Context, g *spauth.GraphClient, path string) error {
	return ErrNotImplemented
}

// Move renames or relocates an item from src to dst (both library-relative).
// (FTP "rename".)
//
// TODO: PATCH /drives/{driveID}/root:/{src}: with body
// {"parentReference": {"path": "/drive/root:/<dstParent>"}, "name": <dstLeaf>}.
func (d *Drive) Move(ctx context.Context, g *spauth.GraphClient, src, dst string) error {
	return ErrNotImplemented
}
