package clean

func init() {
	atPath(
		// App and tool caches.
		&rule{path: "~/Library/Caches", title: "App caches", safety: Safe, children: true,
			why: "Apps rebuild their caches. Quit the app first for best results."},
		&rule{path: "~/Library/Caches/Homebrew", title: "Homebrew downloads", safety: Safe, cmd: "brew cleanup --prune=all",
			why: "Bottles and installers already installed."},
		&rule{path: "~/Library/Caches/pip", title: "pip cache", safety: Safe, cmd: "pip cache purge",
			why: "Downloaded wheels. pip fetches them again when needed."},
		&rule{path: "~/Library/Caches/go-build", title: "Go build cache", safety: Safe, cmd: "go clean -cache",
			why: "Compiled packages, rebuilt on the next go build."},
		&rule{path: "~/Library/Caches/Yarn", title: "Yarn cache", safety: Safe, cmd: "yarn cache clean",
			why: "Downloaded packages. Yarn fetches them again when needed."},
		&rule{path: "~/Library/Caches/CocoaPods", title: "CocoaPods cache", safety: Safe, cmd: "pod cache clean --all",
			why: "Downloaded pods, fetched again on the next pod install."},
		&rule{path: "~/Library/Caches/ms-playwright", title: "Playwright browsers", safety: Rebuild, cmd: "npx playwright install",
			why: "Test browsers, re-downloaded by playwright install."},
		&rule{path: "~/Library/pnpm/store", title: "pnpm store", safety: Rebuild, cmd: "pnpm store prune",
			why: "Shared package store. Prune removes only packages no project uses."},
		&rule{path: "~/Library/Logs", title: "Log files", safety: Safe, children: true,
			why: "Diagnostic logs. Apps start new ones as needed."},

		// Xcode and simulators.
		&rule{path: "~/Library/Developer/Xcode/DerivedData", title: "Xcode DerivedData", safety: Safe,
			why: "Build products and indexes. Xcode rebuilds them on the next build."},
		&rule{path: "~/Library/Developer/Xcode/iOS DeviceSupport", title: "iOS device support", safety: Rebuild, children: true,
			why: "Debug symbols per iOS version. Copied again when that device connects."},
		&rule{path: "~/Library/Developer/Xcode/watchOS DeviceSupport", title: "watchOS device support", safety: Rebuild, children: true,
			why: "Debug symbols per watchOS version. Copied again when that device connects."},
		&rule{path: "~/Library/Developer/CoreSimulator/Caches", title: "Simulator caches", safety: Safe,
			why: "Simulator dyld and runtime caches, regenerated on demand."},
		&rule{path: "~/Library/Developer/Xcode/Archives", title: "Xcode archives", safety: Review,
			why: "App builds you archived. Keep the releases you need for crash symbolication."},
		&rule{path: "~/Library/Developer/CoreSimulator/Devices", title: "iOS simulators", safety: Review, noTrash: true,
			cmd: "xcrun simctl delete unavailable",
			why: "Simulator devices and their data. Delete old ones with simctl or Xcode → Devices and Simulators."},
		&rule{path: "~/Library/Android/sdk/system-images", title: "Android system images", safety: Review, children: true,
			why: "Emulator OS images. Remove unused ones in Android Studio → SDK Manager."},

		// Editors and chat apps.
		&rule{path: "~/Library/Application Support/Code/Cache", title: "VS Code caches", safety: Safe,
			why: "Electron web cache. Recreated when VS Code starts."},
		&rule{path: "~/Library/Application Support/Code/CachedData", title: "VS Code caches", safety: Safe,
			why: "Compiled code cache for older VS Code versions."},
		&rule{path: "~/Library/Application Support/Code/CachedExtensionVSIXs", title: "VS Code caches", safety: Safe,
			why: "Extension installers already installed."},
		&rule{path: "~/Library/Application Support/Code/User/workspaceStorage", title: "VS Code workspace storage", safety: Review,
			why: "Per-project editor state; entries for deleted projects are dead weight."},
		&rule{path: "~/Library/Application Support/Slack/Cache", title: "Slack cache", safety: Safe,
			why: "Downloaded media and web cache. Quit Slack first."},
		&rule{path: "~/Library/Application Support/Slack/Service Worker/CacheStorage", title: "Slack cache", safety: Safe,
			why: "Offline web cache. Quit Slack first."},
		&rule{path: "~/Library/Application Support/Google/Chrome/Default/Service Worker/CacheStorage", title: "Chrome site caches", safety: Safe,
			why: "Offline caches of websites. Quit Chrome first; sites re-download them."},

		// Big things that need a human decision.
		&rule{path: "~/Library/Containers/com.docker.docker", title: "Docker data", safety: Review, noTrash: true,
			cmd: "docker system prune -a --volumes",
			why: "Docker images, containers and volumes. Prune unused ones from Docker; deleting the folder resets Docker."},
		&rule{path: "~/Library/Application Support/MobileSync/Backup", title: "iPhone / iPad backups", safety: Review, noTrash: true,
			why: "Device backups. Delete old ones in Finder → your device → Manage Backups."},
		&rule{path: "~/.Trash", title: "Trash", safety: Review, noTrash: true,
			why: "Items already in the Trash still use space. Empty it from Finder."},
	)
}
