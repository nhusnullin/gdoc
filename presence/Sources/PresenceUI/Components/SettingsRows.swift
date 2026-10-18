// SETTINGS: the same rows, in the same order, on every screen.
//
// `HROW` in docs/design/2026-10-17-presence-screens-v3.html is the 28 pt
// compact row, and SETTINGS is a list of them with one rule this file holds:
// the order never changes. A row with nothing to say is left out, and the
// rest keep their places, so the switch is always the second from the
// bottom and Quit is always the last. A section whose rows move is a section
// nobody learns.
//
// The order is the order of the list in `SettingsRowKind`, which is also the
// order of what a person can do something about: the facts first, and the
// three rows that are not facts at all last.
//
// Three of the rows are not statuses. Hub is the folder somebody chose, and
// pressing it opens the system's own folder panel. Launch at login is a
// setting, so it carries the drawn switch and never a status colour. Quit is
// a row with a keystroke on the right, pressed rather than read. The group's
// name is theirs: the two facts above them are read, and the three rows
// below them are the only settings the panel has.
//
// CardTests puts every screen's rows in shuffled and reads the order back.

import SwiftUI

/// The rows, in the order SETTINGS draws them.
///
/// The account and the version were the first two until 2026-10-18, when
/// they moved into the header: the account is the thing a person checks
/// before a reply goes out under their name, and a row at the foot of a
/// panel that scrolls is not where it goes. The Thread mark was the third
/// to go, on the same morning and in Nail's words "a pointless thing": the
/// mark is drawn in the header beside the name, and a row saying it again
/// is one fact in two places.
public enum SettingsRowKind: String, Sendable, CaseIterable {
  /// Whether Google answered, which changes no icon in the menu bar.
  case network
  /// Whether Claude Desktop can have an extension at all: none was ever
  /// written on this machine, or a run of this process wrote one and could
  /// not hand it over.
  case desktop
  /// The folder the notes are in, which is the one row that opens a window
  /// of the system's rather than saying something.
  case hub
  /// The one switch in the panel.
  case launchAtLogin
  /// The last row, always.
  case quit

  /// The words on the left. They are the kind's own, so no screen can call
  /// the same row something else.
  public var label: String {
    switch self {
    case .network: return "Network"
    case .desktop: return "Claude Desktop"
    case .hub: return "Hub"
    case .launchAtLogin: return "Launch at login"
    case .quit: return "Quit"
    }
  }
}

/// One settings row: what it is, what it says, and what sits on its right.
public struct SettingsRow: Identifiable {
  public let kind: SettingsRowKind
  /// The fact on the right. Empty where the row's right-hand side is a
  /// control rather than a value.
  public let value: String
  public let glyph: GlyphKind
  public let tone: GlyphTone
  /// The small ring in the glyph's place: `<span class="ring wait">` in the
  /// screens file. The two rings the panel draws are the header's since
  /// 2026-10-18, the sign-in's and the update's, so no row here draws one
  /// today.
  public let waiting: Bool
  /// A line under the row, in the row's own words. `QUIET()` in the screens
  /// file, which is the network row's explanation.
  public let quiet: String?
  /// The icons a row carries, in the order they are drawn. One or none is
  /// what every row has today: the Claude Desktop row's Install extension,
  /// and nothing on the rest.
  public let actions: [IconAction]
  /// The switch, on the one row that is a setting.
  public let isOn: Bool?
  public let onToggle: () -> Void
  /// Pressing the row itself, which Hub, Quit and the Login Items row do.
  public let onSelect: (() -> Void)?

  public var id: String { kind.rawValue }

  public init(
    _ kind: SettingsRowKind,
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

  /// The same row with a list of icons rather than one, which is the shape
  /// the model's own list is mapped through.
  public init(
    _ kind: SettingsRowKind,
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

/// The SETTINGS list.
public struct SettingsRows: View {
  /// The keystroke on the last row, which the app does not bind: the panel
  /// is a window and ⌘Q is the application's.
  public nonisolated static let quitShortcut = "⌘Q"

  public let rows: [SettingsRow]

  public init(rows: [SettingsRow]) {
    self.rows = rows
  }

  /// The rows in the one order, whichever order they arrived in.
  public static func ordered(_ rows: [SettingsRow]) -> [SettingsRow] {
    let places = Dictionary(
      uniqueKeysWithValues: SettingsRowKind.allCases.enumerated().map { ($1, $0) })
    return rows.sorted { (places[$0.kind] ?? 0) < (places[$1.kind] ?? 0) }
  }

  public var body: some View {
    VStack(alignment: .leading, spacing: 0) {
      ForEach(Self.ordered(rows)) { row in
        SettingsRowView(row: row)
      }
    }
  }
}

/// One row and the line under it.
struct SettingsRowView: View {
  let row: SettingsRow

  var body: some View {
    VStack(alignment: .leading, spacing: 0) {
      RowView(
        .compact, title: row.kind.label, value: row.value.isEmpty ? nil : row.value,
        leading: { leading }, trailing: { trailing }
      )
      .metricProbe("settings.\(row.kind.rawValue)")
      .onTapGesture { row.onSelect?() }

      if let quiet = row.quiet {
        QuietLine(quiet)
          .metricProbe("settings.\(row.kind.rawValue).quiet")
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

  /// `.row.hasact` pulls a row's value in to make room; a row with nothing
  /// on its right keeps the content edge, which is the edge every time in
  /// TODAY is on. An empty box here is not nothing: the row's own 8 pt
  /// column gap goes with it, and every version, mark and keystroke this
  /// group drew sat 8 pt left of every other value in the panel.
  /// testAValueEndsWhereEveryOtherValueEnds.
  @ViewBuilder
  private var trailing: some View {
    if let isOn = row.isOn {
      SwitchView(isOn: isOn, action: row.onToggle)
    } else if !row.actions.isEmpty {
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
