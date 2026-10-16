// swift-tools-version: 6.0

// The presence app's package: a sibling of go/, so nothing in go/boundary
// ever sees Swift and nothing here ever sees the Go module.
//
// This milestone declares the core only. The views and the executable arrive
// with the app itself, because a target declared with no source file fails
// the build from a clean clone and git keeps no empty directory.

import PackageDescription

let package = Package(
  name: "Presence",
  platforms: [.macOS(.v14)],
  targets: [
    // Foundation only: the decoding, the fold, the tail, liveness and the
    // bundle-version check. No window, no view, and nothing that could reach
    // the wire or start a program.
    .target(name: "PresenceCore"),
    .testTarget(
      name: "PresenceCoreTests",
      dependencies: ["PresenceCore"],
      // Copied rather than processed, so events-v1.jsonl in the bundle is
      // byte for byte the file the Go writer wrote.
      resources: [.copy("Fixtures")]
    ),
  ]
)
