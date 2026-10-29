package clean

// Rules that apply on every OS. Platform-specific locations live in
// rules_<os>.go.
func init() {
	js := []string{"package.json"}
	gradle := []string{"build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"}

	// Project folders, recognised by name plus a marker file.
	byName(&rule{title: "node_modules", safety: Rebuild, markers: js,
		why: "JavaScript dependencies. Reinstall any time with npm / yarn / pnpm install."},
		"node_modules")
	byName(&rule{title: "JS framework build caches", safety: Safe, markers: js,
		why: "Build output and caches (.next, .turbo, .parcel-cache, …). Regenerated on the next build or dev run."},
		".next", ".nuxt", ".svelte-kit", ".turbo", ".parcel-cache", ".vite", ".expo", ".docusaurus", ".angular", ".cache")
	byName(&rule{title: "Python caches", safety: Safe,
		why: "Bytecode and tool caches. Python and the tools recreate them automatically."},
		"__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache", ".hypothesis")
	byName(&rule{title: "Python virtualenvs", safety: Rebuild, inside: []string{"pyvenv.cfg"},
		why: "Isolated Python installs. Recreate with python -m venv .venv and pip install -r requirements.txt."},
		".venv", "venv", "env", "virtualenv")
	byName(&rule{title: "tox / nox environments", safety: Safe, markers: []string{"tox.ini", "noxfile.py"},
		why: "Test environments, recreated on the next tox / nox run."},
		".tox", ".nox")
	byName(&rule{title: "Rust build output", safety: Rebuild, markers: []string{"Cargo.toml"}, cmd: "cargo clean",
		why: "Compiled artifacts in target/. Rebuilt by cargo build."},
		"target")
	byName(&rule{title: "Maven build output", safety: Rebuild, markers: []string{"pom.xml"}, cmd: "mvn clean",
		why: "Compiled artifacts in target/. Rebuilt by mvn package."},
		"target")
	byName(&rule{title: "Gradle build output", safety: Rebuild, markers: gradle, cmd: "./gradlew clean",
		why: "Compiled artifacts in build/. Rebuilt by the next Gradle build."},
		"build")
	byName(&rule{title: "Gradle project caches", safety: Safe, markers: gradle,
		why: "Per-project Gradle state, recreated on the next build."},
		".gradle")
	byName(&rule{title: "Terraform providers", safety: Rebuild, markers: []string{"*.tf", "*.tf.json"}, cmd: "terraform init",
		why: "Downloaded providers and modules (often 500 MB+ each). terraform init downloads them again."},
		".terraform")
	byName(&rule{title: "CocoaPods", safety: Rebuild, markers: []string{"Podfile"}, cmd: "pod install",
		why: "iOS dependencies in Pods/. Restored by pod install."},
		"Pods")
	byName(&rule{title: "Dart / Flutter caches", safety: Safe, markers: []string{"pubspec.yaml"}, cmd: "flutter clean",
		why: "Tool caches, recreated on the next build."},
		".dart_tool")
	byName(&rule{title: "Zig caches", safety: Safe, markers: []string{"build.zig"},
		why: "Build cache, recreated on the next zig build."},
		".zig-cache", "zig-cache")

	// Package-manager and toolchain caches in the home folder.
	atPath(
		&rule{path: "~/.npm/_cacache", title: "npm cache", safety: Safe, cmd: "npm cache clean --force",
			why: "Downloaded packages. npm fetches them again when needed."},
		&rule{path: "~/.npm/_npx", title: "npx cache", safety: Safe,
			why: "Packages downloaded by npx. Fetched again on the next npx run."},
		&rule{path: "~/.yarn/berry/cache", title: "Yarn cache", safety: Safe, cmd: "yarn cache clean --all",
			why: "Downloaded packages. Yarn fetches them again when needed."},
		&rule{path: "~/.pnpm-store", title: "pnpm store", safety: Rebuild, cmd: "pnpm store prune",
			why: "Shared package store. Prune removes only packages no project uses."},
		&rule{path: "~/.local/share/pnpm/store", title: "pnpm store", safety: Rebuild, cmd: "pnpm store prune",
			why: "Shared package store. Prune removes only packages no project uses."},
		&rule{path: "~/.bun/install/cache", title: "Bun cache", safety: Safe, cmd: "bun pm cache rm",
			why: "Downloaded packages. Bun fetches them again when needed."},
		&rule{path: "~/.gradle/caches", title: "Gradle caches", safety: Rebuild,
			why: "Downloaded dependencies and build caches. Re-downloaded on the next build."},
		&rule{path: "~/.gradle/wrapper/dists", title: "Gradle distributions", safety: Rebuild,
			why: "Gradle versions downloaded by the wrapper. Old versions are rarely needed."},
		&rule{path: "~/.m2/repository", title: "Maven repository", safety: Rebuild,
			why: "Downloaded dependencies. Maven re-downloads what a build needs."},
		&rule{path: "~/go/pkg/mod", title: "Go module cache", safety: Rebuild, cmd: "go clean -modcache", noTrash: true,
			why: "Downloaded modules (read-only files). Use the command; Go re-downloads on the next build."},
		&rule{path: "~/.cargo/registry", title: "Cargo registry", safety: Rebuild,
			why: "Downloaded crates. Cargo fetches them again when needed."},
		&rule{path: "~/.cache/huggingface", title: "Hugging Face models", safety: Rebuild,
			why: "Downloaded models and datasets. Re-downloaded when a script uses them."},
		&rule{path: "~/.cache", title: "User caches (~/.cache)", safety: Rebuild, children: true,
			why: "Tool caches. Each tool re-creates or re-downloads what it needs."},
		&rule{path: "~/.rustup/toolchains", title: "Rust toolchains", safety: Review, children: true, cmd: "rustup toolchain list",
			why: "Installed Rust versions. Remove old ones with rustup toolchain uninstall <name>."},
		&rule{path: "~/.nvm/versions/node", title: "Node versions (nvm)", safety: Review, children: true, cmd: "nvm ls",
			why: "Installed Node.js versions. Remove unused ones with nvm uninstall <version>."},
		&rule{path: "~/.pyenv/versions", title: "Python versions (pyenv)", safety: Review, children: true, cmd: "pyenv versions",
			why: "Installed Python versions. Remove unused ones with pyenv uninstall <version>."},
		&rule{path: "~/.ollama/models", title: "Ollama models", safety: Review, noTrash: true, cmd: "ollama list",
			why: "Local LLM weights. Remove models you don't use with ollama rm <model>."},
		&rule{path: "~/.android/avd", title: "Android emulators", safety: Review, children: true,
			why: "Emulator images. Delete unused ones in Android Studio → Device Manager."},
		&rule{path: "~/Downloads", title: "Installers & archives in Downloads", safety: Review,
			exts: []string{"dmg", "pkg", "mpkg", "iso", "xip", "zip", "tgz", "gz", "7z", "rar", "apk", "ipa", "msi", "exe"}, minFile: 50 << 20,
			why: "Usually not needed after installing or extracting."},
	)
}
