// HEALTH: the same eight rows, in the same order, on every screen.
//
// `HROW` in docs/design/2026-10-17-presence-screens-v3.html is the 28 pt
// compact row, and HEALTH is a list of them with one rule this file holds:
// the order never changes. A row with nothing to say is left out, and the
// rest keep their places, so the switch is always the second from the
// bottom and Quit is always the last. A section whose rows move is a section
// nobody learns.
//
// The order is the order of the list in `HealthRowKind`, which is also the
// order of what a person can do something about: the account first, because
// nothing works without it, and the two rows that are not facts at all last.
//
// Two of the rows are not statuses. Launch at login is a setting, so it
// carries the drawn switch and never a status colour. Quit is a row with a
// keystroke on the right, pressed rather than read.
//
// CardTests puts every screen's rows in shuffled and reads the order back.

import SwiftUI

/// The eight rows, in the order HEALTH draws them.
public enum HealthRowKind: String, Sendable, CaseIterable {
  /// The signed-in address, with Sign in or Sign out beside it.
  case account
  /// Whether Google answered, which changes no icon in the menu bar.
  case network
  /// The binary's version, and the one a release named.
  case gdoc
  /// The skills, and where they came from.
  case skills
  /// Whether the Claude Desktop extension is connected.
  case desktop
  /// The mark in effect, which is a setting of the binary's.
  case mark
  /// The one switch in the panel.
  case launchAtLogin
  /// The last row, always.
  case quit

  /// The words on the left. They are the kind's own, so no screen can call
  /// the same row something else.
  public var label: String {
    switch self {
    case .account: return "Account"
    case .network: return "Network"
    case .gdoc: return "gdoc"
    case .skills: return "Skills"
    case .desktop: return "Claude Desktop"
    case .mark: return "Thread mark"
    case .launchAtLogin: return "Launch at login"
    case .quit: return "Quit"
    }
  }
}

/// One health row: what it is, what it says, and what sits on its right.
public struct HealthRow: Identifiable {
  public let kind: HealthRowKind
  /// The fact on the right. Empty where the row's right-hand side is a
  /// control rather than a value.
  public let value: String
  public let glyph: GlyphKind
  public let tone: GlyphTone
  /// The small ring in the glyph's place, which only a sign-in in flight
  /// draws: `<span class="ring wait">` in the screens file.
  public let waiting: Bool
  /// A line under the row, in the row's own words. `QUIET()` in the screens
  /// file, which is the network row's explanation.
  public let quiet: String?
  /// The icons a row carries, in the order they are drawn. Every row but
  /// the sign-in in flight has one or none; that one has Copy and Cancel.
  public let actions: [IconAction]
  /// The switch, on the one row that is a setting.
  public let isOn: Bool?
  public let onToggle: () -> Void
  /// Pressing the row itself, which only Quit does.
  public let onSelect: (() -> Void)?

  public var id: String { kind.rawValue }

  public init(
    _ kind: HealthRowKind,
    value: String,
    glyph: GlyphKind = .blank,
    tone: GlyphTone = .quiet,
    waiting: Bool = false,
    quiet: String? = nil,
    action: IconAction? = nil,
    isOn: Bool? = nil,
    onToggle: @escaping () -> Void = {},
    onSelect: (() -> Void)? = nil
  ) {
    self.kind = kind
    self.value = value
    self.glyph = glyph
    self.tone = tone
    self.waiting = waiting
    self.quiet = quiet
    self.actions = action.map { [$0] } ?? []
    self.isOn = isOn
    self.onToggle = onToggle
    self.onSelect = onSelect
  }

  /// The same row with more than one icon, which only the sign-in in
  /// flight has.
  public init(
    _ kind: HealthRowKind,
    value: String,
    glyph: GlyphKind = .blank,
    tone: GlyphTone = .quiet,
    waiting: Bool = false,
    quiet: String? = nil,
    actions: [IconAction],
    isOn: Bool? = nil,
    onToggle: @escaping () -> Void = {},
    onSelect: (() -> Void)? = nil
  ) {
    self.kind = kind
    self.value = value
    self.glyph = glyph
    self.tone = tone
    self.waiting = waiting
    self.quiet = quiet
    self.actions = actions
    self.isOn = isOn
    self.onToggle = onToggle
    self.onSelect = onSelect
  }
}

/// The HEALTH list.
public struct HealthRows: View {
  /// The keystroke on the last row, which the app does not bind: the panel
  /// is a window and ⌘Q is the application's.
  public nonisolated static let quitShortcut = "⌘Q"

  public let rows: [HealthRow]

  public init(rows: [HealthRow]) {
    self.rows = rows
  }

  /// The rows in the one order, whichever order they arrived in.
  public static func ordered(_ rows: [HealthRow]) -> [HealthRow] {
    let places = Dictionary(
      uniqueKeysWithValues: HealthRowKind.allCases.enumerated().map { ($1, $0) })
    return rows.sorted { (places[$0.kind] ?? 0) < (places[$1.kind] ?? 0) }
  }

  public var body: some View {
    VStack(alignment: .leading, spacing: 0) {
      ForEach(Self.ordered(rows)) { row in
        HealthRowView(row: row)
      }
    }
  }
}

/// One row and the line under it.
struct HealthRowView: View {
  let row: HealthRow

  var body: some View {
    VStack(alignment: .leading, spacing: 0) {
      RowView(
        .compact, title: row.kind.label, value: row.value.isEmpty ? nil : row.value,
        leading: { leading }, trailing: { trailing }
      )
      .metricProbe("health.\(row.kind.rawValue)")
      .onTapGesture { row.onSelect?() }

      if let quiet = row.quiet {
        QuietLine(quiet)
          .metricProbe("health.\(row.kind.rawValue).quiet")
      }
    }
  }

  /// The glyph, or the ring that stands in its place while Google is
  /// being waited for.
  @ViewBuilder
  private var leading: some View {
    if row.waiting {
      RingView()
    } else {
      StatusGlyph(row.glyph, tone: row.tone)
    }
  }

  @ViewBuilder
  private var trailing: some View {
    if let isOn = row.isOn {
      SwitchView(isOn: isOn, action: row.onToggle)
    } else {
      HStack(spacing: Grid.actionGap) {
        ForEach(row.actions) { action in
          ActionButton(
            action.icon, tip: action.tip, state: action.state,
            forcedTooltip: action.forcedTooltip, action: action.perform)
        }
      }
    }
  }
}
