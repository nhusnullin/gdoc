// swift-tools-version: 6.0

// The presence app's package: a sibling of go/, so nothing in go/boundary
// ever sees Swift and nothing here ever sees the Go module.
//
// Four targets, in one direction: the core knows nothing of a window, the
// views know nothing of a status item, the app library is what the shell is
// made of, and the executable is the only room that holds a run loop. Nothing
// here knows about the Go module beside it.
//
// The app library exists because an executable target cannot be imported by a
// test target: anything the shell does that is worth a test lives in
// PresenceApp, and Sources/Presence holds the wiring and nothing else.
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
    // The decoding, the folds, the tail, liveness, the hub index and the
    // bundle-version check. Foundation, and CoreServices for the one
    // FSEvents stream the hub is watched with. No window, no view, and
    // nothing that could reach the wire or start a program.
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
      dependencies: ["PresenceUI"],
      // One record per scene of the screens file, copied rather than
      // processed so a fixture in the bundle is the file somebody can read.
      resources: [.copy("Scenes")]
    ),
    // What the shell does, where a test can reach it: the runner that starts
    // the six argv lists, the polls over them, and the app's own memory. It
    // depends on PresenceUI because the facts it gathers are the panel's
    // `Facts`, which is where the views read them.
    .target(name: "PresenceApp", dependencies: ["PresenceCore", "PresenceUI"]),
    .testTarget(
      name: "PresenceAppTests",
      dependencies: ["PresenceApp"],
      // Fixtures holds fake-gdoc.sh, which the tests run at its own path in
      // the checkout rather than out of a bundle: a copy into a bundle is a
      // copy whose mode bits are not the ones git holds.
      exclude: ["Fixtures"]
    ),
    // The shell: the status item, the panel's window, the tail's delivery and
    // the watcher on its own bundle. It holds no reading and no wording.
    .executableTarget(name: "Presence", dependencies: ["PresenceApp", "PresenceUI", "PresenceCore"]),
  ]
)
