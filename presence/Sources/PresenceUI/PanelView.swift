// The panel, drawn.
//
// `.panel` in docs/design/2026-10-17-presence-screens-v3.html: 320 pt wide,
// 8 pt of padding, a 12 pt corner, a half-pixel edge and two shadows. Inside
// it the header, then the sections in the order the model put them in.
//
// It draws the model and decides nothing. Every word on the screen came out
// of PanelModel and every press leaves through PanelActions, so a row that
// reads wrong is a failing test over a value and never a screenshot somebody
// has to squint at. That is also why Tasks 16 to 24 add no code here: they
// produce facts and perform actions, and this file already draws both.
//
// The panel never grows past 600 pt. A section a person opens scrolls
// inside itself, which is the decision table's rule.
//
// What scrolls under the header changed on 2026-10-18. Everything under it
// used to be one box, and Nail's first sentence after a morning with the
// panel was that the scrolling is wrong: NOW went away upwards and Quit
// went off the bottom, so the two parts a hand reaches for were the two
// that moved. The panel's edges are pinned now. The order is the header,
// NOW, the middle and SETTINGS; the header, NOW and SETTINGS never scroll,
// and the one box is the middle: DOCUMENTS, TODAY and Yesterday. The middle
// takes what is left of the 600 pt, which is how SETTINGS keeps its place
// whatever the day held.
//
// NOW has a cap of its own, half the body, because four holds and a watch
// are taller than the whole panel. Past that it scrolls inside itself
// rather than pushing Quit off the bottom.
//
// RenderTests draws every scene of the screens file through this view, in
// both appearances, and reads its sizes back through the metric probes.

import SwiftUI

/// The whole panel.
public struct PanelView: View {
  public let model: PanelModel
  public let actions: PanelActions

  @Environment(\.colorScheme) private var scheme
  @Environment(\.renderMode) private var renderMode

  public init(model: PanelModel, actions: PanelActions = PanelActions()) {
    self.model = model
    self.actions = actions
  }

  /// What is left for the sections once the header, the gap under it and
  /// the panel's own padding have taken theirs.
  ///
  /// Every part of the panel is in this sum, because the cap is a frame with
  /// a clip under it: a body that left the header's gap out would read 600
  /// on the probe and lose 4 pt of the bottom edge, its corner and the
  /// scroll fade off the screen.
  public static var bodyHeight: CGFloat {
    Grid.panelMaxHeight - Grid.headerHeight - Grid.headerGap - 2 * Grid.unit
  }

  /// The most NOW takes before it scrolls inside itself: half the body.
  ///
  /// Half, because the two things under it are a list of documents and a
  /// group of settings, and a NOW that took more than its share would leave
  /// one row of each. A day with four holds in it is the day this matters
  /// on, and on that day the person still reaches Quit.
  public static var nowHeight: CGFloat { (Self.bodyHeight / 2).rounded(.down) }

  public var body: some View {
    let palette = Palette.of(scheme)
    let parts = PanelParts(model.sections)
    return VStack(alignment: .leading, spacing: 0) {
      PanelHeader(model: model.header, forced: model.forcedTooltip, actions: actions)

      VStack(alignment: .leading, spacing: 0) {
        if let notice = model.notice {
          RowView(.plain, glyph: .exclamation, tone: .hold, title: notice)
            .fixedSize(horizontal: false, vertical: true)
            .metricProbe("notice")
        }
        if let empty = model.empty {
          EmptyState(mark: model.mark, words: empty)
            .fixedSize(horizontal: false, vertical: true)
        }
        ForEach(parts.above) { pinned($0) }
        ScrollBox(.given) {
          VStack(alignment: .leading, spacing: 0) {
            ForEach(parts.middle) { section($0) }
          }
        }
        .metricProbe("panel.middle")
        ForEach(parts.below) { pinned($0) }
      }
      .metricProbe("panel.body")
    }
    // The cap is here and not on the body, because the header is not always
    // 36 pt: a sign-out question, a countdown and a banner are drawn in it,
    // and a body capped at a number of its own would push the foot of a
    // panel with a tall header off the screen. The header takes its own
    // height, the two pinned parts take theirs, and the middle is what
    // gives way.
    .frame(
      maxHeight: renderMode == .whole ? .infinity : Grid.panelMaxHeight - 2 * Grid.unit,
      alignment: .top
    )
    .padding(Grid.unit)
    .frame(width: Grid.panelWidth, alignment: .leading)
    .background(
      RoundedRectangle(cornerRadius: Grid.panelRadius, style: .continuous)
        .fill(palette.surface.color)
    )
    .overlay(
      RoundedRectangle(cornerRadius: Grid.panelRadius, style: .continuous)
        .strokeBorder(palette.panelEdge.color, lineWidth: Grid.hairline)
    )
    .shadows(palette.panelShadows)
    // A whole render is the reviewer's copy, where nothing may be hidden.
    // Every other drawing is capped, which is what the person sees.
    .frame(
      maxHeight: renderMode == .whole ? .infinity : Grid.panelMaxHeight, alignment: .top
    )
    .clipped()
    .tooltipLayer()
    .metricProbe("panel")
  }

  /// One of the panel's pinned parts: NOW under its own cap, or SETTINGS at
  /// its own height.
  @ViewBuilder private func pinned(_ placed: PlacedSection) -> some View {
    switch placed.section {
    case .now:
      ScrollBox(height: Self.nowHeight) { section(placed) }
        .metricProbe("panel.now")
    default:
      // SETTINGS, and nothing else: PanelParts puts the other three in the
      // middle.
      //
      // At its own height, and not at the height a stack felt like giving
      // it: a row states a minimum rather than a height, so a group of rows
      // stretches into whatever it is offered, and the middle and SETTINGS
      // would split the panel evenly between them.
      section(placed)
        .fixedSize(horizontal: false, vertical: true)
        .metricProbe("panel.settings")
    }
  }

  /// One section, with the hairline rule its place in the list decides.
  private func section(_ placed: PlacedSection) -> some View {
    SectionView(
      placed.section.label, count: count(of: placed.section),
      isFirst: isFirst(placed.index)
    ) {
      body(of: placed.section)
    }
  }

  /// `.section:first-of-type`: the one under the header carries no hairline.
  /// The notice and the empty state stand above every section, so a panel
  /// with either of them has no first section at all.
  private func isFirst(_ index: Int) -> Bool {
    index == 0 && model.notice == nil && model.empty == nil
  }

  private func count(of section: PanelSection) -> String? {
    if case .documents(let documents) = section { return documents.count }
    return nil
  }

  @ViewBuilder
  private func body(of section: PanelSection) -> some View {
    switch section {
    case .now(let items):
      NowList(items: items, forced: model.forcedTooltip, actions: actions)
    case .documents(let documents):
      DocumentsList(model: documents, forced: model.forcedTooltip, actions: actions)
    case .today(let log):
      LogList(model: log, list: .today, actions: actions)
    case .yesterday(let yesterday):
      YesterdayList(model: yesterday, actions: actions)
    case .settings(let rows):
      SettingsList(rows: rows, forced: model.forcedTooltip, actions: actions)
    }
  }
}

/// One section with the place it had in the model's own list, which is what
/// decides the hairline over it.
struct PlacedSection: Identifiable {
  let index: Int
  let section: PanelSection

  var id: String { section.id }
}

/// The sections sorted into the three parts of the panel: what is pinned
/// above the box, what is in it, and what is pinned under it.
///
/// Which part a section belongs to is the section's own answer. Where it
/// goes is the model's order: SETTINGS is above everything on the two
/// screens the model pins it on, and under everything on the rest, and this
/// reads that order rather than holding a rule of its own.
struct PanelParts {
  var above: [PlacedSection] = []
  var middle: [PlacedSection] = []
  var below: [PlacedSection] = []

  init(_ sections: [PanelSection]) {
    for (index, section) in sections.enumerated() {
      let placed = PlacedSection(index: index, section: section)
      if section.scrolls {
        middle.append(placed)
      } else if middle.isEmpty {
        above.append(placed)
      } else {
        below.append(placed)
      }
    }
  }
}

extension PanelSection {
  /// Whether this section is in the box that scrolls.
  ///
  /// NOW and SETTINGS are the panel's pinned edges since the second
  /// morning: the first is what gdoc is doing and the second is where Quit
  /// is, and neither is a thing a person should have to scroll to. The
  /// three in the middle are a log, and a log is what scrolling is for.
  var scrolls: Bool {
    switch self {
    case .now, .settings: return false
    case .documents, .today, .yesterday: return true
    }
  }
}

/// NOW: the cards and the live rows, in the order the fold put them in.
struct NowList: View {
  let items: [NowItem]
  let forced: String?
  let actions: PanelActions

  var body: some View {
    VStack(alignment: .leading, spacing: 0) {
      ForEach(items) { item in
        switch item {
        case .hold(let id, let card, let asking):
          HoldCard(
            model: card, asking: asking,
            onOpenChat: { actions.perform(.openChat, id) },
            onDismiss: { actions.perform(.dismiss, id) },
            onConfirm: { actions.perform(.dismiss, id) },
            onKeep: { actions.perform(.keep, id) })
        case .stuck(_, let card, let icons):
          StuckCard(model: card, actions: icons.map { $0.icon(forced: forced, actions) })
        case .job(_, let card):
          JobCard(model: card)
        case .live(_, let row, let stop, let pending):
          LiveRow(
            model: row, stop: stop?.state, pending: pending,
            onStop: { actions.perform(.stopLive, stop?.target) })
        }
      }
    }
  }
}

/// DOCUMENTS: the rows, the strip under the first of them, and Show more.
struct DocumentsList: View {
  let model: DocumentsSectionModel
  let forced: String?
  let actions: PanelActions

  var body: some View {
    if let empty = model.empty {
      QuietLine(empty)
    } else {
      ShowMoreList(
        model.rows, collapsed: model.collapsed, isOpen: model.isOpen,
        toggle: { actions.toggle(.documents) }
      ) { row in
        VStack(alignment: .leading, spacing: 0) {
          RowView(
            .doc, glyph: row.glyph, tone: row.tone, title: row.title, sub: row.sub,
            gone: row.gone,
            trailing: {
              HStack(spacing: Grid.actionGap) {
                ForEach(row.actions) { action in
                  ActionButton(
                    action.icon, tip: action.tip, state: action.state,
                    forcedTooltip: action.forcedTooltip || forced == action.id,
                    action: { actions.perform(action.kind, action.target) })
                }
              }
            })
          if model.showsStrip, row.id == model.rows.first?.id {
            IconStrip()
          }
        }
      }
    }
  }
}

/// TODAY: the log, newest first, with the rest behind Show more.
struct LogList: View {
  let model: LogSectionModel
  let list: PanelList
  let actions: PanelActions

  var body: some View {
    if let empty = model.empty {
      QuietLine(empty)
    } else {
      ShowMoreList(
        model.rows, collapsed: model.collapsed, isOpen: model.isOpen,
        toggle: { actions.toggle(list) }
      ) { row in
        RowView(
          row.sub == nil ? .plain : .two, glyph: row.glyph, tone: row.tone, title: row.title,
          value: row.value, sub: row.sub)
      }
    }
  }
}

/// Yesterday: one chevron row that opens in place.
struct YesterdayList: View {
  let model: YesterdayModel
  let actions: PanelActions

  var body: some View {
    VStack(alignment: .leading, spacing: 0) {
      ChevronRow(model.summary, isOpen: model.isOpen) { actions.toggle(.yesterday) }
      if model.isOpen {
        ForEach(model.rows) { row in
          RowView(
            row.sub == nil ? .plain : .two, glyph: row.glyph, tone: row.tone,
            title: row.title, value: row.value, sub: row.sub)
        }
      }
    }
  }
}

/// SETTINGS: the rows in their one order.
struct SettingsList: View {
  let rows: [SettingsRowModel]
  let forced: String?
  let actions: PanelActions

  var body: some View {
    SettingsRows(rows: rows.map(settings))
  }

  private func settings(_ row: SettingsRowModel) -> SettingsRow {
    SettingsRow(
      row.kind, value: row.value, glyph: row.glyph, tone: row.tone, waiting: row.waiting,
      quiet: row.quiet, actions: row.actions.map { $0.icon(forced: forced, actions) },
      isOn: row.isOn,
      onToggle: { actions.perform(.toggleLoginItem, nil) },
      onSelect: row.select.map { kind in { actions.perform(kind, nil) } })
  }
}

extension ActionModel {
  /// The same action with its closure, which is the one thing the model
  /// cannot carry.
  func icon(forced: String?, _ actions: PanelActions) -> IconAction {
    IconAction(
      icon, tip: tip, state: state, forcedTooltip: forcedTooltip || forced == id,
      perform: { actions.perform(kind, target) })
  }
}

extension View {
  /// A CSS shadow stack, in the order the stylesheet writes it.
  ///
  /// The tree is flattened first. SwiftUI's `shadow` is not a box-shadow: it
  /// reaches every layer under it, so a panel that asked for its two shadows
  /// without compositing drew one under every row, every glyph and every
  /// line of text, which on white is a grey smear around each word. The
  /// screens file puts no shadow inside `.panel`.
  func shadows(_ stack: [Shadow]) -> some View {
    stack.reduce(AnyView(self.compositingGroup())) { view, shadow in
      AnyView(
        view.shadow(
          color: shadow.swatch.color, radius: shadow.radius, x: shadow.x, y: shadow.y))
    }
  }
}
