// A finite job that is running: the large ring, what it is, and its steps.
//
// `.now-job` in docs/design/2026-10-17-presence-screens-v3.html is a 28 pt
// ring, the job's title in the heavier weight, and one quiet line of facts.
// Under it, in scene r1, are the steps as compact rows: what is done with
// what it produced, what is running with how far, and what has not happened
// yet in the quietest grey.
//
// The steps come from the record's `step` events, which M24b adds. A job
// whose steps are not known draws none of them: four grey rows the app
// invented would say that gdoc is somewhere in a sequence the app is only
// guessing at, and the ring with no fraction already says the one true thing,
// which is that something is running.
//
// Nothing here predicts a time. A ring without a fraction turns and says
// nothing about how long.
//
// CardTests states the two cases and the ring's two sizes, and RowTests
// holds that the title's verb takes its own room.

import SwiftUI

/// Where one step of a job stands.
public enum JobStepState: Equatable, Sendable {
  /// `.glyph-check.glyph-ok`: it happened, and the row says what came out.
  case done
  /// The small ring, with the fraction where the record carries one.
  case running(Double?)
  /// `.glyph-empty` and a tertiary title: it has not happened.
  case waiting
}

/// One step row.
public struct JobStep: Equatable, Sendable {
  public let title: String
  /// What came out of it, on the right.
  public let value: String?
  public let state: JobStepState

  public init(title: String, value: String? = nil, state: JobStepState) {
    self.title = title
    self.value = value
    self.state = state
  }
}

/// What a job card draws.
public struct JobCardModel: Equatable, Sendable {
  /// The verb in front of the name, `Publishing`, drawn whole. Nothing
  /// where the record names no document: then the verb is the whole title
  /// and there is no name to keep room for.
  public let prefix: String?
  /// The document's own name, `Chapter 4`, or the verb alone where the
  /// record names no document.
  public let name: String
  /// `step 2 of 4 · started 09:03 · Falcon / Drafts`, or nothing.
  public let sub: String?
  /// How far along, where that is known. Nothing is an indeterminate ring.
  public let progress: Double?
  /// The steps, or none at all.
  public let steps: [JobStep]

  public init(
    prefix: String? = nil, name: String, sub: String? = nil, progress: Double? = nil,
    steps: [JobStep] = []
  ) {
    self.prefix = prefix
    self.name = name
    self.sub = sub
    self.progress = progress
    self.steps = steps
  }

  /// `Publishing · Chapter 4`: the line a reader sees.
  public var title: String { MiddleTitle.whole(prefix: prefix, name: name) }
}

/// One running job.
public struct JobCard: View {
  /// `.now-job{column-gap:12px;padding:4px 12px 8px}`.
  public static let columnGap: CGFloat = 12
  public static let topPadding: CGFloat = 4
  public static let bottomPadding: CGFloat = Grid.unit

  /// The room the job's title has: 360 less the panel's 8 either side, the
  /// card's 12 either side, the 28 pt ring and the 12 pt gap after it.
  /// 280 pt.
  public static let titleColumn: CGFloat =
    Grid.panelWidth - 2 * Grid.unit - 2 * Grid.rowInset - Grid.ringLarge - columnGap

  public let model: JobCardModel
  /// The words the title carries where the card itself opens the document.
  public let titleTip: String?

  @Environment(\.colorScheme) private var scheme

  public init(model: JobCardModel, titleTip: String? = nil) {
    self.model = model
    self.titleTip = titleTip
  }

  /// Whether the step rows are drawn at all.
  public static func showsSteps(_ model: JobCardModel) -> Bool {
    !model.steps.isEmpty
  }

  /// The large ring's fraction, which is the job's own and never a step's.
  public static func ringProgress(_ model: JobCardModel) -> Double? {
    model.progress
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return VStack(alignment: .leading, spacing: 0) {
      HStack(spacing: Self.columnGap) {
        RingView(progress: Self.ringProgress(model), size: .large)
          .metricProbe("job.ring")
        VStack(alignment: .leading, spacing: 2) {
          // The name is cut in the middle like every other name in the
          // panel. Left to the drawing it was cut at its tail, so a publish
          // of "Ch 5. Commercial Model v0.1" lost the one word that tells
          // it from the draft beside it.
          MiddleTitle(
            model.name, prefix: model.prefix, role: Typography.titleEmphasis,
            column: Self.titleColumn
          )
          .metricProbe("job.title")
          .tooltip(titleTip ?? "", align: .leading)
          if let sub = model.sub {
            Text(sub)
              .font(Typography.caption.font)
              .fixedSize(horizontal: false, vertical: true)
              .foregroundStyle(palette.textTertiary.color)
          }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
      }
      .padding(.horizontal, Grid.rowInset)
      .padding(.top, Self.topPadding)
      .padding(.bottom, Self.bottomPadding)
      .metricProbe("job.card")

      if Self.showsSteps(model) {
        ForEach(Array(model.steps.enumerated()), id: \.offset) { index, step in
          JobStepRow(step: step)
            .metricProbe("job.step.\(index)")
        }
      }
    }
  }
}

/// One step, as the compact row of scene r1.
///
/// It is drawn here rather than through `RowView(.compact)` for one reason:
/// a step that has not happened carries a tertiary title, which is the
/// inline span the screens file writes around it, and the shared row has no
/// ink of its own to hand over.
struct JobStepRow: View {
  let step: JobStep

  @Environment(\.colorScheme) private var scheme

  var body: some View {
    let palette = Palette.of(scheme)
    return HStack(alignment: .firstTextBaseline, spacing: Grid.columnGap) {
      leading
        .frame(width: Grid.glyphSize, height: Typography.rowTitle.lineHeight)
      Text(step.title)
        .font(Typography.rowTitle.font)
        .tracking(Typography.rowTitle.letterSpacing)
        .lineLimit(1)
        .foregroundStyle(ink(in: palette).color)
      Spacer(minLength: 0)
      if let value = step.value {
        Text(value)
          .font(Typography.rowValue.font)
          .monospacedDigit()
          .lineLimit(1)
          .foregroundStyle(palette.textSecondary.color)
      }
    }
    .padding(.horizontal, Grid.rowInset)
    .frame(maxWidth: .infinity, minHeight: Grid.compactRowHeight, alignment: .leading)
  }

  @ViewBuilder
  private var leading: some View {
    switch step.state {
    case .done:
      StatusGlyph(.check, tone: .ok)
    case .running(let fraction):
      RingView(progress: fraction)
    case .waiting:
      StatusGlyph(.empty, tone: .dim)
    }
  }

  /// A step that has not happened is as quiet as what it will say.
  private func ink(in palette: Palette) -> Swatch {
    step.state == .waiting ? palette.textTertiary : palette.text
  }
}
