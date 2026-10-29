#!/usr/bin/env python3
"""Create a fake home folder for screenshots and the demo report.

The tree looks like a developer's Mac (projects, media, caches, Xcode,
Docker) but every file is filler, so screenshots show no real data.
Files are written for real (not sparse) because tpdv measures allocated
size. Total is about 3.5 GB; delete the folder when done.

usage: scripts/make_demo.py /tmp/tpdv-demo
"""
import os
import shutil
import sys
import time

MB = 1 << 20
ZERO = bytes(MB)
DAY = 86400


def write(path, mb=0.0, kb=0):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    size = int(mb * MB) + kb * 1024
    with open(path, "wb") as f:
        while size > 0:
            n = min(size, MB)
            f.write(ZERO[:n])
            size -= n


def touch(path, age_days=0, text="{}"):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write(text)
    t = time.time() - age_days * DAY
    os.utime(path, (t, t))


def many(dir, count, kb, ext, prefix="f"):
    for i in range(count):
        write(os.path.join(dir, f"{prefix}{i:03d}.{ext}"), kb=kb)


def node_modules(root, heavy):
    """Lots of small files like a real install, plus a few heavy packages."""
    for i in range(36):
        many(os.path.join(root, f"pkg-{i:02d}", "dist"), 24, 48, "js")
        touch(os.path.join(root, f"pkg-{i:02d}", "package.json"))
    for name, mb in heavy.items():
        write(os.path.join(root, name, "bin", "binary.node"), mb)


def main(base):
    home = os.path.join(base, "alex")
    if os.path.exists(base):
        shutil.rmtree(base)
    h = lambda *p: os.path.join(home, *p)

    # Media
    write(h("Movies", "Family trip 2025.mov"), 420)
    write(h("Movies", "Drone - coastline.mp4"), 260)
    write(h("Movies", "Screen Recording 2026-03-02.mov"), 90)
    for i in range(40):
        write(h("Pictures", "Photos Library.photoslibrary", "originals", f"{i % 16:X}", f"IMG_{4100 + i}.HEIC"), 3)
    for i in range(4):
        write(h("Pictures", "Wallpapers", f"wallpaper-{i}.png"), 6)
    for i in range(18):
        write(h("Music", "Albums", "Lo-fi Sessions", f"{i + 1:02d} Track.m4a"), 7)

    # Downloads and documents
    write(h("Downloads", "Xcode_16.2.xip"), 380)
    write(h("Downloads", "Docker.dmg"), 180)
    write(h("Downloads", "ollama-darwin.zip"), 95)
    write(h("Downloads", "dataset-2025.csv"), 60)
    write(h("Downloads", "invoice-0420.pdf"), 2)
    for y in range(2021, 2026):
        write(h("Documents", "Taxes", f"{y}-return.pdf"), 4)
    write(h("Documents", "Thesis.pages"), 35)
    write(h("Documents", "Budget.numbers"), 8)
    many(h("Documents", "Notes"), 120, 6, "md", "note-")

    # Projects
    p = lambda *x: h("Projects", *x)
    touch(p("acme-web", "package.json"), 3)
    many(p("acme-web", "src", "components"), 160, 8, "tsx")
    node_modules(p("acme-web", "node_modules"),
                 {"@next/swc-darwin-arm64": 120, "typescript": 22, "sharp": 30, "esbuild": 9})
    for i in range(10):
        write(p("acme-web", ".next", "cache", "webpack", f"client-{i}.pack"), 12)
    many(p("acme-web", ".next", "server"), 80, 16, "js")

    touch(p("mobile-app", "package.json"), 60)
    touch(p("mobile-app", "ios", "Podfile"), 60, "platform :ios")
    node_modules(p("mobile-app", "node_modules"), {"react-native": 60, "hermes-engine": 45})
    write(p("mobile-app", "ios", "Pods", "Firebase", "Firebase.xcframework", "binary"), 85)
    write(p("mobile-app", "ios", "Pods", "GoogleMaps", "GoogleMaps.framework", "binary"), 35)

    touch(p("ml-experiments", ".venv", "pyvenv.cfg"), 20, "home = /usr/bin")
    write(p("ml-experiments", ".venv", "lib", "python3.12", "site-packages", "torch", "lib", "libtorch_cpu.dylib"), 240)
    write(p("ml-experiments", ".venv", "lib", "python3.12", "site-packages", "numpy", "core.so"), 35)
    write(p("ml-experiments", "data", "train.parquet"), 180)
    write(p("ml-experiments", "data", "eval.parquet"), 40)
    write(p("ml-experiments", "models", "checkpoint-final.safetensors"), 240)
    many(p("ml-experiments", "notebooks"), 12, 300, "ipynb", "exp-")
    many(p("ml-experiments", "src", "__pycache__"), 30, 90, "pyc")

    touch(p("rust-cli", "Cargo.toml"), 150, "[package]")
    write(p("rust-cli", "target", "debug", "deps", "librust_cli.rlib"), 120)
    write(p("rust-cli", "target", "debug", "rust-cli"), 40)
    write(p("rust-cli", "target", "release", "rust-cli"), 18)
    many(p("rust-cli", "src"), 20, 12, "rs")

    touch(p("infra", "main.tf"), 380, 'provider "aws" {}')
    write(p("infra", ".terraform", "providers", "registry.terraform.io", "hashicorp", "aws", "terraform-provider-aws"), 210)

    touch(p("old-landing-page", "package.json"), 520)
    node_modules(p("old-landing-page", "node_modules"), {"gatsby": 35})
    many(p("old-landing-page", "public"), 40, 120, "jpg", "img-")

    # Library: caches, Xcode, apps
    L = lambda *x: h("Library", *x)
    write(L("Caches", "com.spotify.client", "Data", "storage.bin"), 140)
    write(L("Caches", "Google", "Chrome", "Default", "Cache", "data_1"), 85)
    write(L("Caches", "Homebrew", "downloads", "node--22.11.0.bottle.tar.gz"), 70)
    write(L("Caches", "Homebrew", "downloads", "python@3.13.bottle.tar.gz"), 40)
    write(L("Caches", "pip", "http", "wheels.bin"), 45)
    write(L("Caches", "go-build", "trim.bin"), 70)
    write(L("Caches", "com.apple.Safari", "WebKitCache"), 30)
    write(L("Developer", "Xcode", "DerivedData", "MyApp-bxqzt", "Build", "Products.bin"), 230)
    write(L("Developer", "Xcode", "Archives", "2026-01-12", "MyApp 1.4.xcarchive", "Products.bin"), 90)
    write(L("Logs", "DiagnosticReports", "MyApp-2026-02-11.ips"), 20)
    write(L("Application Support", "Slack", "Cache", "Cache_Data", "data_0"), 60)
    write(L("Containers", "com.docker.docker", "Data", "vms", "0", "data", "Docker.raw"), 520)

    # Package-manager and toolchain caches
    write(h(".npm", "_cacache", "content-v2", "sha512", "blob"), 75)
    write(h(".cache", "huggingface", "hub", "models--bert-base-uncased", "model.safetensors"), 200)
    write(h(".cache", "puppeteer", "chrome", "chrome-mac.zip"), 55)
    for v, mb in (("v18.20.4", 60), ("v20.18.0", 65), ("v22.11.0", 70)):
        write(h(".nvm", "versions", "node", v, "bin", "node"), mb)

    print(home)


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "/tmp/tpdv-demo")
