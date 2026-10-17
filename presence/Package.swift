// swift-tools-version: 6.0

// The presence app's package: a sibling of go/, so nothing in go/boundary
// ever sees Swift and nothing here ever sees the Go module.
//
// Three targets, in one direction: the core knows nothing of a window, the
// views know nothing of a status item, and the executable is the only room
// that holds a run loop. Nothing here knows about the Go module beside it.
//
// The product is GdocPresence so the executable inside the bundle is at
// Contents/MacOS/GdocPresence, which is the path the release packs and the
// updater checks.

import PackageDescription

let package = Package(
  name: "Presence",
  platforms: [.macOS(.v14)],
  products: [
    .executable(name: "GdocPresence", targets: ["Presence"])
  ],
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
    // AppKit and SwiftUI, and no run loop: the glyph, the rows and the panel
    // as a function of one glance, so every state a person can see renders
    // from a fixture.
    .target(name: "PresenceUI", dependencies: ["PresenceCore"]),
    .testTarget(
      name: "PresenceUITests",
      dependencies: ["PresenceUI"]
    ),
    // The shell: the status item, the panel's window, the tail's delivery and
    // the watcher on its own bundle. It holds no reading and no wording.
    .executableTarget(name: "Presence", dependencies: ["PresenceUI", "PresenceCore"]),
  ]
)
