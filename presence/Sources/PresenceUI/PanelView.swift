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
// inside itself, which is the decision table's rule, and everything under
// the header sits in a box of the same kind: four sections of an ordinary
// day are taller than 600 pt, and a panel that clipped them would lose
// HEALTH with nothing saying so. The box draws its thumb and its fade only
// when there is more under them.
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

  /// What is left for the sections once the header and the panel's own
  /// padding have taken theirs.
  public static var bodyHeight: CGFloat {
    Grid.panelMaxHeight - Grid.headerHeight - 2 * Grid.unit
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return VStack(alignment: .leading, spacing: 0) {
      PanelHeader(mark: model.mark, account: model.account)

      ScrollBox(height: Self.bodyHeight) {
        VStack(alignment: .leading, spacing: 0) {
          if let notice = model.notice {
            RowView(.plain, glyph: .exclamation, tone: .hold, title: notice)
              .metricProbe("notice")
          }
          if let empty = model.empty {
            EmptyState(mark: model.mark, words: empty)
          }
          ForEach(Array(model.sections.enumerated()), id: \.element.id) { index, section in
            SectionView(section.label, count: count(of: section), isFirst: isFirst(index)) {
              body(of: section)
            }
          }
        }
      }
    }
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
    case .health(let rows):
      HealthList(rows: rows, url: model.signInURL, forced: model.forcedTooltip, actions: actions)
    case .signedIn(let banner):
      SignedInBanner(account: banner.account) { actions.perform(.signOut, nil) }
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

/// HEALTH: the rows in their one order, and the sign-in line under the
/// Account row while a sign-in is in flight.
struct HealthList: View {
  let rows: [HealthRowModel]
  let url: String?
  let forced: String?
  let actions: PanelActions

  var body: some View {
    VStack(alignment: .leading, spacing: 0) {
      ForEach(rows) { row in
        if let question = row.question {
          // The consequence is a line of its own rather than the row's
          // sub-line: the pair of buttons takes the right-hand column, and
          // a sentence squeezed into what is left reads as half a sentence.
          ConfirmRow(
            title: question.title, danger: question.danger,
            onConfirm: { actions.perform(.signOut, nil) },
            onKeep: { actions.perform(.keep, nil) })
          QuietLine(question.consequence)
          if let quiet = row.quiet { QuietLine(quiet) }
        } else {
          HealthRows(rows: [health(row)])
        }
        if row.kind == .account, let url {
          URLLine(url: url)
        }
      }
    }
  }

  private func health(_ row: HealthRowModel) -> HealthRow {
    HealthRow(
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
