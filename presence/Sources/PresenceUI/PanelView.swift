// The panel, drawn.
//
// Plain native presentation in this milestone: the system's own background,
// the system's own text colours, one accent nowhere. The two palettes as
// drawn in the design files are M24, after Nail has used this one.
//
// It draws the model and decides nothing. Every word it shows came out of
// PanelModel, so a row that reads wrong is a failing test in RenderTests and
// never a screenshot somebody has to squint at.

import SwiftUI

/// The sizes the panel is laid out on: the 4/8 pt grid of the design
/// principles, written out once so no view holds a number of its own.
enum Metrics {
  static let width: CGFloat = 320
  static let inset: CGFloat = 8
  static let radius: CGFloat = 12
  static let row: CGFloat = 44
  static let thinRow: CGFloat = 28
  static let label: CGFloat = 11
}

/// NOW, TODAY, the versions and Quit.
public struct PanelView: View {
  private let model: PanelModel
  private let onQuit: (() -> Void)?

  public init(model: PanelModel, onQuit: (() -> Void)? = nil) {
    self.model = model
    self.onQuit = onQuit
  }

  public var body: some View {
    VStack(alignment: .leading, spacing: Metrics.inset) {
      header

      if let notice = model.notice {
        RowView(row: notice)
      }

      if model.showsNow {
        RowSection(label: "NOW", rows: model.now)
      }

      if model.today.isEmpty {
        Text("Nothing ran today.")
          .font(.system(size: 12))
          .foregroundStyle(.secondary)
      } else {
        RowSection(label: "TODAY", rows: model.today)
      }

      Divider()

      VStack(alignment: .leading, spacing: 0) {
        ForEach(Array(model.versions.enumerated()), id: \.offset) { _, row in
          RowView(row: row)
        }
      }

      if let onQuit {
        Divider()
        Button(action: onQuit) {
          Text("Quit")
            .font(.system(size: 12))
            .frame(maxWidth: .infinity, alignment: .leading)
            .frame(height: Metrics.thinRow)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
      }
    }
    .padding(Metrics.inset)
    .frame(width: Metrics.width, alignment: .leading)
    .background(Color(nsColor: .windowBackgroundColor))
  }

  private var header: some View {
    HStack(spacing: 6) {
      Text("gdoc")
        .font(.system(size: 13, weight: .semibold))
      Spacer()
    }
    .frame(height: Metrics.thinRow)
  }
}

/// One section: its label, then its rows.
private struct RowSection: View {
  let label: String
  let rows: [Row]

  var body: some View {
    VStack(alignment: .leading, spacing: 2) {
      Text(label)
        .font(.system(size: Metrics.label, weight: .semibold))
        .foregroundStyle(.tertiary)
      ForEach(Array(rows.enumerated()), id: \.offset) { _, row in
        RowView(row: row)
      }
    }
  }
}

/// What a NOW row is, in the same four shapes the menu bar uses, so the panel
/// and the glyph never say different things about the same run. The other
/// sections keep the space and draw nothing in it, so every leading word in
/// the panel starts at the same place.
private struct KindGlyph: View {
  let kind: RowKind

  var body: some View {
    Group {
      switch kind {
      case .hold: Image(systemName: "pause.circle")
      case .running: Image(systemName: "circle.dotted")
      case .watching: Text(Glyph.mark)
      case .stopped: Image(systemName: "circle.slash")
      case .done, .version, .notice: Color.clear
      }
    }
    .font(.system(size: 11))
    .foregroundStyle(.secondary)
    .frame(width: 13, alignment: .leading)
  }
}

/// One row: a leading word, a sentence under it, a value on the right, and
/// the one symbol a finished row carries.
private struct RowView: View {
  let row: Row

  var body: some View {
    HStack(alignment: .firstTextBaseline, spacing: 6) {
      KindGlyph(kind: row.kind)
      VStack(alignment: .leading, spacing: 1) {
        Text(row.leading)
          .font(.system(size: 12, weight: weight))
          .foregroundStyle(.primary)
          .lineLimit(1)
          .truncationMode(.middle)
        if let detail = row.detail {
          Text(detail)
            .font(.system(size: 11))
            .foregroundStyle(.secondary)
            .lineLimit(1)
            .truncationMode(.middle)
        }
      }
      Spacer(minLength: 6)
      if let symbol {
        Image(systemName: symbol)
          .font(.system(size: 11))
          .foregroundStyle(.secondary)
      }
      if let value = row.value {
        Text(value)
          .font(.system(size: 11).monospacedDigit())
          .foregroundStyle(.secondary)
      }
    }
    .frame(minHeight: height, alignment: .leading)
  }

  private var weight: Font.Weight {
    switch row.kind {
    case .hold, .notice: return .semibold
    default: return .regular
    }
  }

  private var height: CGFloat {
    row.detail == nil ? Metrics.thinRow : Metrics.row * 0.7
  }

  /// The symbols, from the facts the record carried. A hollow check is
  /// `verified: false`, which the design principles say is not a failure.
  private var symbol: String? {
    switch row.mark {
    case .none: return nil
    case .ok: return "checkmark"
    case .failed: return "exclamationmark.triangle"
    case .verified: return "checkmark.seal.fill"
    case .unverified: return "checkmark.seal"
    }
  }
}
