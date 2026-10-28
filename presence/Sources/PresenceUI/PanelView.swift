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
// Nothing in it scrolls, and it has no cap. Everything under the header was
// one box capped at 600 pt until 2026-10-18, with NOW capped inside it and
// a Show more on each list, and Nail's screenshot of that evening had three
// scroll thumbs on one panel. The decision of that day is one panel as tall
// as its content: a panel taller than the screen is cut at the screen's
// edge, which he chose over any scroll. So the sections are a plain stack
// in the order the model put them in, each at its own height.
//
// A document row carries its own menu, with Hide from the list, on a
// secondary click and on a press held half a second. Pin was in it until
// the third look and is the last of the row's four icons now, because Nail
// had pins and never found the menu. A pinned row offers nothing, so it
// opens no menu. A click on the row opens the document it names, which is
// the fifth look: the Google Docs icon went with it.
//
// RenderTests draws every scene of the screens file through this view, in
// both appearances, and reads its sizes back through the metric probes.

import SwiftUI

/// The whole panel.
public struct PanelView: View {
  public let model: PanelModel
  public let actions: PanelActions

  @Environment(\.colorScheme) private var scheme

  public init(model: PanelModel, actions: PanelActions = PanelActions()) {
    self.model = model
    self.actions = actions
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return VStack(alignment: .leading, spacing: 0) {
      // At its own height, like every section under it. A row states a
      // minimum and not a height, so a header inside a stack that had room
      // to spare took it: on the sign-out question it drew "Sign out?" in
      // the middle of a 190 pt hole. Nothing has room to spare now, because
      // the panel is the sum of what is in it, and the header still says
      // what its own height is:
      // testTheQuestionLeavesTheHeaderAtThirtySix.
      PanelHeader(model: model.header, forced: model.forcedTooltip, actions: actions)
        .fixedSize(horizontal: false, vertical: true)

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
        ForEach(placed) { placed in
          section(placed)
            .fixedSize(horizontal: false, vertical: true)
            .metricProbe("panel.\(placed.section.id.lowercased())")
        }
      }
      .metricProbe("panel.body")
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
    .tooltipLayer()
    .metricProbe("panel")
  }

  /// The sections with the place each of them had in the model's own list,
  /// which is what decides the hairline over it.
  private var placed: [PlacedSection] {
    model.sections.enumerated().map { PlacedSection(index: $0, section: $1) }
  }

  /// One section, with the hairline rule its place in the list decides.
  private func section(_ placed: PlacedSection) -> some View {
    SectionView(placed.section.label, isFirst: isFirst(placed.index)) {
      body(of: placed.section)
    }
  }

  /// `.section:first-of-type`: the one under the header carries no hairline.
  /// The notice and the empty state stand above every section, so a panel
  /// with either of them has no first section at all.
  private func isFirst(_ index: Int) -> Bool {
    index == 0 && model.notice == nil && model.empty == nil
  }

  @ViewBuilder
  private func body(of section: PanelSection) -> some View {
    switch section {
    case .now(let items):
      NowList(items: items, forced: model.forcedTooltip, actions: actions)
    case .documents(let documents):
      DocumentsList(model: documents, forced: model.forcedTooltip, actions: actions)
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

/// What a press on a NOW row means, which is one thing: open the document
/// the row names.
///
/// Every row in NOW is about a document, so the row itself is the click
/// target and not an icon on it, which is Nail's own answer at the fourth
/// look (docs/v2/DECISIONS.md, 2026-10-26). The press leaves as the kind a
/// DOCUMENTS row's first icon sends, so `Session.perform` reaches
/// `Opener.openDocument` and nothing new asks the system.
///
/// The controls keep their own press: Stop is a button inside the row and
/// takes the click that lands on it, so a hand reaching for Stop never
/// opens a browser tab instead: testTheRowOpensAndOnlyStopStops.
enum NowPress {
  /// The words the title area carries, which are the Google Docs icon's:
  /// the press is that icon's press, moved to the row.
  static let words = Tips.openDoc

  /// The document this row's press opens, or nothing where it has no press.
  ///
  /// A hold that is asking its question opens nothing: the card is two
  /// buttons and a sentence about what follows from them, and a click
  /// anywhere on it that went to a browser would be an answer nobody gave.
  static func opens(_ item: NowItem) -> String? {
    if case .hold(_, _, let asking, _) = item, asking { return nil }
    return item.opens
  }

  /// The row's own press, where it has one.
  static func open(_ item: NowItem, actions: PanelActions) -> (() -> Void)? {
    guard let doc = opens(item) else { return nil }
    return { actions.perform(.openDoc, doc) }
  }

  /// The words a row's title area carries, where the row is a press. A row
  /// that opens nothing says nothing: a tooltip over a title that does not
  /// click is the panel offering something it does not have.
  static func tip(for item: NowItem) -> String? {
    opens(item) == nil ? nil : words
  }

  /// The one control a live row carries, which is its own press and never
  /// the row's.
  static func stop(_ stop: ActionModel?, actions: PanelActions) -> () -> Void {
    { actions.perform(.stopLive, stop?.target) }
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
        row(item)
          .opensDocument(NowPress.open(item, actions: actions))
      }
    }
  }

  @ViewBuilder
  private func row(_ item: NowItem) -> some View {
    let tip = NowPress.tip(for: item)
    switch item {
    case .hold(let id, let card, let asking, _):
      HoldCard(
        model: card, asking: asking, titleTip: tip,
        onOpenChat: { actions.perform(.openChat, id) },
        onDismiss: { actions.perform(.dismiss, id) },
        onConfirm: { actions.perform(.dismiss, id) },
        onKeep: { actions.perform(.keep, id) })
    case .stuck(_, let card, let icons, _):
      StuckCard(
        model: card, actions: icons.map { $0.icon(forced: forced, actions) }, titleTip: tip)
    case .job(_, let card, _):
      JobCard(model: card, titleTip: tip)
    case .live(_, let row, let stop, let pending, _):
      LiveRow(
        model: row, stop: stop?.state, pending: pending, titleTip: tip,
        onStop: NowPress.stop(stop, actions: actions))
    }
  }
}

/// DOCUMENTS: the rows, and the strip under the first of them.
///
/// Every row the model carries, because the list is short by the time it
/// gets here: the fold decides how many documents there are and the view
/// draws them. There is no Show more and nothing to open.
struct DocumentsList: View {
  let model: DocumentsSectionModel
  let forced: String?
  let actions: PanelActions

  var body: some View {
    if let empty = model.empty {
      QuietLine(empty)
    } else {
      VStack(alignment: .leading, spacing: 0) {
        ForEach(model.rows) { row in
          // One value for both ways into the row's menu, so the words the
          // secondary click draws and the words the held press pops up are
          // the same list and the anchor is the row's own.
          let menu = RowMenu(items: row.menu, actions: actions)
          // The row's own press, in the two places it lands: the title
          // line, where it sits with the held press so one gesture
          // decides between them, and the row's box for the sub-line and
          // the gaps. The icons are drawn over it and take their own
          // clicks.
          let press = DocumentPress.open(row, actions: actions)
          VStack(alignment: .leading, spacing: 0) {
            RowView(
              .doc, glyph: row.glyph, tone: row.tone, title: row.title,
              titleHeld: menu.held, titleTap: press, sub: row.sub,
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
              }
            )
            .opensDocument(press)
            .rowMenu(menu)
            if model.showsStrip, row.id == model.rows.first?.id {
              IconStrip()
            }
          }
        }
      }
    }
  }
}

extension View {
  /// The whole row or card as one press, which opens the document it names.
  ///
  /// The shape is the row's own box, so the quiet parts of it press too: a
  /// sub-line of facts and the gap beside a title are what a hand lands on
  /// first. A row that names no document carries no press at all and no
  /// shape either, so it stays what it was.
  func opensDocument(_ press: (() -> Void)?) -> some View {
    modifier(OpensDocument(press: press))
  }

  /// A document row's own menu: the secondary click, and the anchor the
  /// held press opens it from.
  ///
  /// Two ways in, because the menu is the one place a hide is and a menu
  /// bar's panel is reached by both hands: a right-click is what the
  /// platform does, and a held press is what a trackpad under one finger
  /// does. The secondary click is SwiftUI's `.contextMenu`, which tracks an
  /// `NSMenu`; the held press is `RowMenu.held`, on the row's title area,
  /// which pops up an `NSMenu` of the same words from the anchor behind
  /// that title. Neither is a window, which is the fourth look's answer to
  /// the popover that could have closed the panel under itself.
  ///
  /// It decides nothing: the words and the kinds are the model's, and the
  /// press leaves through `PanelActions` like every other press.
  func rowMenu(_ menu: RowMenu) -> some View {
    modifier(menu)
  }
}

/// The menu of one row, in its two ways in.
struct RowMenu: ViewModifier {
  let items: [MenuItemModel]
  let actions: PanelActions
  let present: MenuPresenter
  /// Where the system's menu is measured from, filled in by the tree.
  let anchor = MenuAnchorBox()
  /// A render has no menu to open, and `ImageRenderer` cannot draw the
  /// anchor's `NSView`: testARenderDrawsNoMenuAnchor.
  @Environment(\.renderMode) private var mode

  init(
    items: [MenuItemModel], actions: PanelActions, present: MenuPresenter = .system()
  ) {
    self.items = items
    self.actions = actions
    self.present = present
  }

  func body(content: Content) -> some View {
    // A row with nothing to offer opens nothing. A pinned row is that row,
    // and an empty menu under the pointer is a panel that answered a press
    // with a grey rectangle: testTheRowMenuIsHideAndAPinnedRowHasNone.
    if items.isEmpty {
      content
    } else {
      content
        .contextMenu { words }
        .background {
          // Behind the title area, which is the row less its icon column:
          // the menu opens under the words the press was held on.
          if mode == .live {
            MenuAnchor(box: anchor)
              .frame(maxWidth: .infinity, maxHeight: .infinity)
              .padding(.trailing, Grid.actionColumn)
          }
        }
    }
  }

  /// The held press of this row's title, where the row has a menu at all.
  var held: (() -> Void)? {
    guard !items.isEmpty else { return nil }
    let entries = entries
    let present = present
    let anchor = anchor
    let press: @MainActor () -> Void = { present.open(entries, anchor.view) }
    return press
  }

  /// The words as the system draws them, each wired to the press the
  /// `.contextMenu`'s own button sends.
  var entries: [MenuEntry] {
    items.map { item in
      MenuEntry(word: item.word, press: { actions.perform(item.kind, item.target) })
    }
  }

  @ViewBuilder private var words: some View {
    ForEach(items) { item in
      Button(item.word) {
        actions.perform(item.kind, item.target)
      }
    }
  }
}

/// The row's own press: the one gesture, over the row's whole box.
///
/// It is a tap and not a button, because a button around a row with buttons
/// in it is two controls one inside the other, and the inner one stops
/// reporting its own state. The icons on the row are drawn over this and
/// take the clicks that land on them, which is the platform's own rule
/// about a child and the view behind it.
private struct OpensDocument: ViewModifier {
  let press: (() -> Void)?

  func body(content: Content) -> some View {
    if let press {
      content
        .contentShape(Rectangle())
        .onTapGesture(perform: press)
    } else {
      content
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
