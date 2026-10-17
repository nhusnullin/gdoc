// What a render test does with a view: draw it, count its ink, write it out.
//
// Four test files draw the same way, so the renderer, the pixel count, the
// measurements and the directory the PNGs go to live here once. The directory is outside the
// repository, always: a render is something a person looks at once, and a
// committed screenshot is a file nobody compares against the code that drew
// it.

import AppKit
import SwiftUI
import XCTest

@testable import PresenceUI

/// One view through the renderer at 2x, in the appearance named.
@MainActor
func png<V: View>(_ view: V, scheme: ColorScheme) throws -> Data {
  let renderer = ImageRenderer(content: view.environment(\.colorScheme, scheme))
  renderer.scale = 2
  var data: Data?
  let appearance = NSAppearance(named: scheme == .dark ? .darkAqua : .aqua)
  appearance?.performAsCurrentDrawingAppearance {
    guard let image = renderer.cgImage else { return }
    data = NSBitmapImageRep(cgImage: image).representation(using: .png, properties: [:])
  }
  return try XCTUnwrap(data, "the renderer drew nothing in \(scheme)")
}

/// How many pixels carry ink. The view is laid out in a square of the size
/// named over nothing, so a transparent pixel is a pixel nothing drew on.
@MainActor
func inkedPixels<V: View>(_ view: V, scheme: ColorScheme, side: CGFloat) throws -> Int {
  var counted = 0
  try eachPixel(view, scheme: scheme, side: side) { alpha in
    if alpha > 0.05 { counted += 1 }
  }
  return counted
}

/// How much ink, rather than how many pixels: the alphas added up, so a view
/// drawn at a lower opacity weighs less than the same view at full.
@MainActor
func inkWeight<V: View>(_ view: V, scheme: ColorScheme, side: CGFloat) throws -> Double {
  var total = 0.0
  try eachPixel(view, scheme: scheme, side: side) { total += $0 }
  return total
}

@MainActor
private func eachPixel<V: View>(
  _ view: V, scheme: ColorScheme, side: CGFloat, _ body: (Double) -> Void
) throws {
  let framed = view.frame(width: side, height: side)
  let renderer = ImageRenderer(content: framed.environment(\.colorScheme, scheme))
  renderer.scale = 2
  var failure: String?
  let appearance = NSAppearance(named: scheme == .dark ? .darkAqua : .aqua)
  appearance?.performAsCurrentDrawingAppearance {
    guard let image = renderer.cgImage else {
      failure = "the renderer drew nothing"
      return
    }
    let rep = NSBitmapImageRep(cgImage: image)
    for x in 0..<rep.pixelsWide {
      for y in 0..<rep.pixelsHigh {
        body(Double(rep.colorAt(x: x, y: y)?.alphaComponent ?? 0))
      }
    }
  }
  if let failure { XCTFail(failure) }
}

/// Where the PNGs go: outside the repository, named by PRESENCE_SHOTS or a
/// directory in the temporary one.
func shotsDirectory() throws -> URL {
  let named = ProcessInfo.processInfo.environment["PRESENCE_SHOTS"]
  let temporary = URL(fileURLWithPath: NSTemporaryDirectory())
    .appendingPathComponent("gdoc-presence-shots")
  let base = named.map { URL(fileURLWithPath: $0) } ?? temporary
  let repository = presencePackageRoot().deletingLastPathComponent().standardizedFileURL.path
  XCTAssertFalse(
    base.standardizedFileURL.path.hasPrefix(repository),
    "PRESENCE_SHOTS names \(base.path), which is inside the repository")
  try FileManager.default.createDirectory(at: base, withIntermediateDirectories: true)
  return base
}

/// This file is Tests/PresenceUITests/Shots.swift, so the package is three
/// directories up from it. The path is the file's own and never the working
/// directory, which swift test does not promise.
func presencePackageRoot() -> URL {
  URL(fileURLWithPath: #filePath)
    .deletingLastPathComponent()
    .deletingLastPathComponent()
    .deletingLastPathComponent()
}

/// What a view measured, by name: the `metricProbe` names under it against
/// the boxes they were drawn in.
///
/// A size in this package is a literal from the screens file, and a literal
/// is worth nothing until the drawing is shown to honour it, so the tests
/// read the numbers back through the preference key rather than off the
/// constant they came from.
@MainActor
func probes<V: View>(_ view: V, scheme: ColorScheme, width: CGFloat = Grid.panelWidth) throws
  -> [String: CGSize]
{
  var byName: [String: CGSize] = [:]
  for probe in try probeList(view, scheme: scheme, width: width) where byName[probe.name] == nil {
    byName[probe.name] = probe.size
  }
  return byName
}

/// The same measurements in the order they were drawn, so a test can tell
/// two reports of one name apart: a box inside a box is two `scrollbox`
/// probes, and which of them carries a thumb is the thing under test.
@MainActor
func probeList<V: View>(_ view: V, scheme: ColorScheme, width: CGFloat = Grid.panelWidth) throws
  -> [MetricProbe]
{
  let collector = ProbeCollector()
  let framed =
    view
    .frame(width: width)
    .environment(\.colorScheme, scheme)
    .onPreferenceChange(MetricProbes.self) { reported in
      MainActor.assumeIsolated { collector.reported = reported }
    }
  let renderer = ImageRenderer(content: framed)
  renderer.scale = 2
  let appearance = NSAppearance(named: scheme == .dark ? .darkAqua : .aqua)
  appearance?.performAsCurrentDrawingAppearance {
    _ = renderer.cgImage
  }
  XCTAssertFalse(collector.reported.isEmpty, "nothing reported a size in \(scheme)")
  return collector.reported
}

/// The box a render writes its measurements into.
@MainActor
final class ProbeCollector {
  var reported: [MetricProbe] = []
}

/// One pixel of a view, by its point in the square the view was laid out in.
///
/// The render is at 2x, so a point is two pixels and the sample is the first
/// of them. A test that asks for a colour asks for it where it drew
/// something, so there is no searching here: the caller names the place.
@MainActor
func pixel<V: View>(
  _ view: V, scheme: ColorScheme, side: CGFloat, x: CGFloat, y: CGFloat
) throws -> NSColor {
  let framed = view.frame(width: side, height: side)
  let renderer = ImageRenderer(content: framed.environment(\.colorScheme, scheme))
  renderer.scale = 2
  var sampled: NSColor?
  let appearance = NSAppearance(named: scheme == .dark ? .darkAqua : .aqua)
  appearance?.performAsCurrentDrawingAppearance {
    guard let image = renderer.cgImage else { return }
    let rep = NSBitmapImageRep(cgImage: image)
    sampled = rep.colorAt(x: Int(x * 2), y: Int(y * 2))?.usingColorSpace(.sRGB)
  }
  return try XCTUnwrap(sampled, "the renderer drew nothing in \(scheme)")
}

/// Where a view's ink ends, as a point in the width it was laid out in.
///
/// The view is drawn over nothing, so a pixel with no alpha is a pixel
/// nothing drew on: what comes back is the right edge of the last thing
/// drawn. It is the one way to ask whether two rows in two sections line
/// their values up, because the answer is a layout's and not a constant's.
@MainActor
func inkEnds<V: View>(_ view: V, scheme: ColorScheme, width: CGFloat = Grid.panelWidth) throws
  -> CGFloat
{
  let renderer = ImageRenderer(
    content: view.frame(width: width).environment(\.colorScheme, scheme))
  renderer.scale = 2
  var found: CGFloat?
  let appearance = NSAppearance(named: scheme == .dark ? .darkAqua : .aqua)
  appearance?.performAsCurrentDrawingAppearance {
    guard let image = renderer.cgImage else { return }
    let rep = NSBitmapImageRep(cgImage: image)
    for x in stride(from: rep.pixelsWide - 1, through: 0, by: -1) {
      for y in 0..<rep.pixelsHigh {
        guard (rep.colorAt(x: x, y: y)?.alphaComponent ?? 0) > 0.05 else { continue }
        found = CGFloat(x + 1) / 2
        return
      }
    }
  }
  return try XCTUnwrap(found, "the view drew nothing in \(scheme)")
}

/// Where the ink lies vertically inside one column of a view, as points in
/// the box it was laid out in.
///
/// Two pieces of one row are two bands: the words on the left and whatever
/// sits at the trailing edge. A row reads as one row when the two bands have
/// the same middle, which is a question only the drawing can answer.
@MainActor
func inkBand<V: View>(
  _ view: V, scheme: ColorScheme, width: CGFloat = Grid.panelWidth,
  from: CGFloat, to: CGFloat
) throws -> ClosedRange<CGFloat> {
  let renderer = ImageRenderer(
    content: view.frame(width: width).environment(\.colorScheme, scheme))
  renderer.scale = 2
  var top: Int?
  var bottom: Int?
  let appearance = NSAppearance(named: scheme == .dark ? .darkAqua : .aqua)
  appearance?.performAsCurrentDrawingAppearance {
    guard let image = renderer.cgImage else { return }
    let rep = NSBitmapImageRep(cgImage: image)
    for y in 0..<rep.pixelsHigh {
      for x in Int(from * 2)..<min(Int(to * 2), rep.pixelsWide) {
        guard (rep.colorAt(x: x, y: y)?.alphaComponent ?? 0) > 0.05 else { continue }
        if top == nil { top = y }
        bottom = y
        break
      }
    }
  }
  let first = try XCTUnwrap(top, "nothing is drawn between \(from) and \(to) in \(scheme)")
  let last = try XCTUnwrap(bottom)
  return (CGFloat(first) / 2)...(CGFloat(last + 1) / 2)
}

/// How many lines a sentence takes in the panel's own font at a width,
/// which is the one thing a clamp cannot be read off the code for.
///
/// Three test files ask it, so it lives here with the renderer and the
/// probes rather than three times over.
func lines(_ text: String, role: TypeRole, width: CGFloat) -> Int {
  let font = NSFont.systemFont(ofSize: role.size, weight: role.weight.appKit)
  let box = NSAttributedString(string: text, attributes: [.font: font]).boundingRect(
    with: CGSize(width: width, height: .greatestFiniteMagnitude),
    options: [.usesLineFragmentOrigin, .usesFontLeading])
  return Int((box.height / role.lineHeight).rounded(.up))
}

// MARK: the sheets

/// The pieces the two sheets draw, taken out of the scenes.
///
/// A sheet is a page of one component after another, so it is the one page
/// in this package with no record behind it: its words would be written by
/// hand, and a hand-written word is the one way a string the app never
/// draws reaches a screen Nail reads. Every piece below is a model a scene
/// built, so a sheet says what the panel says and nothing else.
///
/// RenderTests holds that: testEverySheetWordIsAWordAPanelDraws over the
/// words, and testTheSheetsSayNothingNoSceneSays over the two files' own
/// source.
@MainActor
enum SheetPieces {
  /// One scene by its id.
  static func scene(_ id: String) throws -> Scene {
    try XCTUnwrap(Scenes.all.first { $0.id == id }, "there is no scene \(id)")
  }

  /// Scene r6: the hold in the chat over the watch.
  static func hold() throws -> HoldCardModel {
    let cards = try scene("r6").model().now.compactMap { item -> HoldCardModel? in
      if case .hold(_, let card, _) = item { return card }
      return nil
    }
    return try XCTUnwrap(cards.first, "scene r6 drew no hold card")
  }

  /// Scene r2: the publish that stopped, and the icons beside it.
  static func stuck() throws -> (card: StuckCardModel, actions: [ActionModel]) {
    let found = try scene("r2").model().now.compactMap {
      item -> (StuckCardModel, [ActionModel])? in
      if case .stuck(_, let card, let actions) = item { return (card, actions) }
      return nil
    }
    let first = try XCTUnwrap(found.first, "scene r2 drew no stuck card")
    return (first.0, first.1)
  }

  /// Scene r4: the watch that has been going since the afternoon.
  static func live() throws -> LiveRowModel {
    let rows = try scene("r4").model().now.compactMap { item -> LiveRowModel? in
      if case .live(_, let row, _, _) = item { return row }
      return nil
    }
    return try XCTUnwrap(rows.first, "scene r4 drew no live row")
  }

  /// Scene r1: the publish on its second step.
  static func job() throws -> JobCardModel {
    let cards = try scene("r1").model().now.compactMap { item -> JobCardModel? in
      if case .job(_, let card) = item { return card }
      return nil
    }
    return try XCTUnwrap(cards.first, "scene r1 drew no job card")
  }

  /// HEALTH with every row that has a fact behind it, which no one screen
  /// has: r8 is the network stop and the update, e5 is the machine with no
  /// extension and no token. One of each kind, in `HealthRowKind`'s order,
  /// which is the order HEALTH is drawn in.
  static func health() throws -> [HealthRowModel] {
    var byKind: [HealthRowKind: HealthRowModel] = [:]
    for id in ["r8", "e5"] {
      for row in try scene(id).model().health where byKind[row.kind] == nil {
        byKind[row.kind] = row
      }
    }
    return HealthRowKind.allCases.compactMap { byKind[$0] }
  }

  /// Scene e6's documents: the paired row the strip sits under, and the
  /// row whose note icon is dim because no note is paired with it.
  static func documentRows() throws -> [DocumentRowModel] {
    let rows = try XCTUnwrap(try scene("e6").model().documents, "e6 drew no documents").rows
    XCTAssertGreaterThanOrEqual(rows.count, 2, "the sheet draws two document rows")
    return Array(rows.prefix(2))
  }

  /// Scene r3's first log row: a write that landed and was read back.
  static func logRow() throws -> LogRowModel {
    let today = try XCTUnwrap(try scene("r3").model().today, "r3 drew no TODAY")
    return try XCTUnwrap(today.rows.first, "r3's TODAY is empty")
  }

  /// Scene e1's DOCUMENTS: its header count, and the chevron's own words
  /// for what is under it.
  static func documents() throws -> (count: String, more: String) {
    let model = try XCTUnwrap(try scene("e1").model().documents, "e1 drew no documents")
    let count = try XCTUnwrap(model.count, "e1's DOCUMENTS has no count")
    let shown = ShowMore(total: model.rows.count, collapsed: model.collapsed, isOpen: false)
    return (count, try XCTUnwrap(shown.title, "e1 hides nothing, so there is no word"))
  }

  /// The two labels the rows sheet draws, out of the model that names
  /// them: scene r4 carries NOW and DOCUMENTS both.
  static func sectionLabels() throws -> (now: String, documents: String) {
    var now: String?
    var documents: String?
    for section in try scene("r4").model().sections {
      if case .now = section { now = section.label }
      if case .documents = section { documents = section.label }
    }
    return (
      try XCTUnwrap(now, "r4 drew no NOW"), try XCTUnwrap(documents, "r4 drew no DOCUMENTS")
    )
  }

  /// Scene r1's quiet line: a machine that has published nothing yet.
  static func quiet() throws -> String {
    let model = try XCTUnwrap(try scene("r1").model().documents, "r1 drew no documents")
    return try XCTUnwrap(model.empty, "r1's DOCUMENTS says nothing about being empty")
  }

  /// Scene e10's question: the words, what follows from them, and the loud
  /// word. The controls sheet draws the one inline confirm the panel has
  /// beside the Account row, so the sheet's question is that row's own.
  static func signOut() throws -> QuestionModel {
    let health = try scene("e10").model().health
    let asked = health.compactMap(\.question)
    return try XCTUnwrap(asked.first, "scene e10 asked nothing")
  }

  /// The words on two of a document row's icons, one live and one with
  /// nothing to open: scene e6's first two rows, which is where the panel
  /// holds a tooltip open of its own accord.
  static func tips() throws -> (live: String, dead: (words: String, state: ActionState)) {
    let rows = try documentRows()
    let open = try XCTUnwrap(
      rows[0].actions.first { $0.kind == .openDoc }, "no row opens a document")
    let note = try XCTUnwrap(
      rows[1].actions.first { $0.kind == .openNote }, "no row opens a note")
    guard case .off(let reason) = note.state else {
      throw XCTSkip("scene e6's second row has a note after all")
    }
    return (open.tip, (reason, note.state))
  }

  /// Every string the three sheets draw, which is what the vocabulary test
  /// reads. A piece that gains a line gains it here too, or the test is
  /// checking a page that is no longer the page.
  static func words() throws -> [String] {
    let hold = try self.hold()
    let stuck = try self.stuck()
    let live = try self.live()
    let job = try self.job()
    let documents = try self.documents()

    let labels = try sectionLabels()
    var words: [String] = [
      Scenes.account, "⁘",
      labels.now, labels.documents,
      hold.title, hold.sub, hold.quiet, hold.question, hold.consequence,
      stuck.card.title, stuck.card.reason,
      live.title, live.sub,
      job.title,
      documents.count, documents.more,
      try quiet(),
      SignedInBanner.words, Scenes.signInURL,
    ]
    let question = try signOut()
    words += [question.title, question.consequence, question.danger.rawValue]
    let tips = try self.tips()
    words += [tips.live, tips.dead.words]
    words += stuck.actions.map(\.tip)
    words += [job.sub].compactMap { $0 }
    words += job.steps.flatMap { [$0.title] + [$0.value].compactMap { $0 } }
    words += try health().flatMap { row -> [String] in
      [row.kind.label, row.value] + [row.quiet].compactMap { $0 } + row.actions.map(\.tip)
    }
    words += try documentRows().flatMap { row -> [String] in
      [row.title, row.sub] + row.actions.map(\.tip)
    }
    let log = try logRow()
    words += [log.title, log.value] + [log.sub].compactMap { $0 }
    words += ButtonWord.allCases.map(\.rawValue)
    words += StripLabel.allCases.map(\.rawValue)
    return words.filter { !$0.isEmpty }
  }
}

/// Every string the scenes draw, as one set.
///
/// This is the vocabulary: the panel's own words, each of them out of a
/// record and a set of facts rather than out of somebody's typing. A string
/// a sheet draws that is not in here is a string the app has no way of
/// printing.
@MainActor
func panelVocabulary() throws -> Set<String> {
  var words: Set<String> = []
  for scene in Scenes.all {
    let model = try scene.model()
    words.formUnion([model.mark])
    words.formUnion([model.account, model.empty, model.notice, model.signInURL].compactMap { $0 })
    for item in model.now { words.formUnion(nowWords(item)) }
    for section in model.sections {
      words.insert(section.label)
      switch section {
      case .now: break
      case .documents(let documents):
        words.formUnion([documents.count, documents.empty].compactMap { $0 })
        words.formUnion(
          ShowMore(total: documents.rows.count, collapsed: documents.collapsed, isOpen: false)
            .title.map { [$0] } ?? [])
        for row in documents.rows {
          words.formUnion([row.title, row.sub])
          words.formUnion(row.actions.flatMap(actionWords))
        }
      case .today(let log):
        words.formUnion([log.empty].compactMap { $0 })
        words.formUnion(
          ShowMore(total: log.rows.count, collapsed: log.collapsed, isOpen: false)
            .title.map { [$0] } ?? [])
        for row in log.rows { words.formUnion(logWords(row)) }
      case .yesterday(let yesterday):
        words.insert(yesterday.summary)
        for row in yesterday.rows { words.formUnion(logWords(row)) }
      case .health(let rows):
        for row in rows {
          words.formUnion([row.kind.label, row.value])
          words.formUnion([row.quiet].compactMap { $0 })
          words.formUnion(row.actions.map(\.tip))
          if let question = row.question {
            words.formUnion([question.title, question.consequence, question.danger.rawValue])
          }
        }
      case .signedIn(let banner):
        words.formUnion([banner.account, SignedInBanner.words, SignedInBanner.signOutTip])
      }
    }
  }
  // The words that belong to no row and no record: the two buttons of the
  // hold card, the four labels of the strip once, and the one sentence a
  // machine with no token is met with.
  words.formUnion(ButtonWord.allCases.map(\.rawValue))
  words.formUnion(StripLabel.allCases.map(\.rawValue))
  words.insert(EmptyState.firstRun)
  return words.filter { !$0.isEmpty }
}

/// One action's own strings: the words under the pointer, and the reason an
/// icon with nothing to open gives instead. Both are drawn, so both are the
/// panel's words.
private func actionWords(_ action: ActionModel) -> [String] {
  if case .off(let reason) = action.state { return [action.tip, reason] }
  return [action.tip]
}

/// One NOW item's own strings.
private func nowWords(_ item: NowItem) -> [String] {
  switch item {
  case .hold(_, let card, _):
    return [card.title, card.sub, card.quiet, card.question, card.consequence]
  case .stuck(_, let card, let actions):
    return [card.title, card.reason] + actions.map(\.tip)
  case .job(_, let card):
    return [card.title] + [card.sub].compactMap { $0 }
      + card.steps.flatMap { [$0.title] + [$0.value].compactMap { $0 } }
  case .live(_, let row, let stop, _):
    return [row.title, row.sub] + (stop.map { [$0.tip] } ?? [])
  }
}

/// One log row's own strings.
private func logWords(_ row: LogRowModel) -> [String] {
  [row.title, row.value] + [row.sub].compactMap { $0 }
}
