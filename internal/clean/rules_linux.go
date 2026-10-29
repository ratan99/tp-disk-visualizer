package clean

func init() {
	atPath(
		&rule{path: "~/.local/share/Trash", title: "Trash", safety: Review, noTrash: true, cmd: "gio trash --empty",
			why: "Items already in the Trash still use space until it is emptied."},
		&rule{path: "~/.cache/pip", title: "pip cache", safety: Safe, cmd: "pip cache purge",
			why: "Downloaded wheels. pip fetches them again when needed."},
		&rule{path: "~/.cache/go-build", title: "Go build cache", safety: Safe, cmd: "go clean -cache",
			why: "Compiled packages, rebuilt on the next go build."},
	)
}
