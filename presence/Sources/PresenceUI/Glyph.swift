// The one thing the menu bar shows.
//
// Four states, chosen by priority from one glance: a hold waiting for the
// person, a finite command running, a watch sitting on a document, and
// nothing at all. A start whose pid is gone is a row in the panel and never
// a glyph here, because an icon that stays after the work stopped is an icon
// nobody believes.
//
// Every image is a template, so macOS draws it in the menu bar's own colour
// in both appearances and in both the normal and the inverted bar. There is
// no colour in the menu bar, ever.

import AppKit
import PresenceCore

/// What the menu bar is showing.
public enum GlyphState: String, Sendable, Equatable, CaseIterable {
  /// A chat write is held, waiting for a person.
  case hold
  /// A command with an end is running.
  case running
  /// A `comments --wait` is sitting on a document.
  case watching
  /// Nothing is happening that the record knows about.
  case idle
}

public enum Glyph {
  /// gdoc's own mark, which is what a live watch shows. It is a literal here
  /// and not the mark in the person's settings: the app reads the record and
  /// never the settings file.
  public static let mark = "⁘"

  /// The state one glance is in.
  ///
  /// The order is the whole rule: a hold is first because it is the one state
  /// waiting for the person reading it, and a finite job is over a watch that
  /// may sit there all day.
  public static func state(for glance: Glance) -> GlyphState {
    if !glance.holds.isEmpty { return .hold }
    let live = glance.live
    if live.contains(where: { !$0.isWait }) { return .running }
    if live.contains(where: { $0.isWait }) { return .watching }
    return .idle
  }

  /// The image for a state, and nothing for idle.
  public static func image(for state: GlyphState) -> NSImage? {
    switch state {
    case .hold: return symbol("pause.circle", fallback: "❙❙")
    case .running: return symbol("circle.dotted", fallback: "◌")
    case .watching: return drawn(mark)
    case .idle: return nil
    }
  }

  /// What the pointer resting on the glyph says. The tooltip is the one
  /// always-visible word, so it states the state and judges nothing.
  public static func tooltip(for state: GlyphState) -> String {
    switch state {
    case .hold: return "gdoc is holding a write for you"
    case .running: return "gdoc is running"
    case .watching: return "gdoc is watching a document"
    case .idle: return "gdoc"
    }
  }

  /// An SF Symbol, or the characters that mean the same thing where the
  /// system has no symbol by that name.
  private static func symbol(_ name: String, fallback: String) -> NSImage {
    guard let image = NSImage(systemSymbolName: name, accessibilityDescription: nil) else {
      return drawn(fallback)
    }
    image.isTemplate = true
    return image
  }

  /// Text as a template image, which is how the mark reaches the menu bar.
  private static func drawn(_ text: String, side: CGFloat = 16) -> NSImage {
    let font = NSFont.menuBarFont(ofSize: side - 3)
    let string = NSAttributedString(
      string: text, attributes: [.font: font, .foregroundColor: NSColor.black])
    let measured = string.size()
    let size = NSSize(width: max(ceil(measured.width), side), height: side)
    let image = NSImage(size: size, flipped: false) { rect in
      let origin = NSPoint(
        x: (rect.width - measured.width) / 2, y: (rect.height - measured.height) / 2)
      string.draw(at: origin)
      return true
    }
    image.isTemplate = true
    return image
  }
}
