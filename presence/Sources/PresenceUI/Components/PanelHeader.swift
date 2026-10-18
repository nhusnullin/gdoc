// The panel's own line: the mark, the name with its version, the account.
//
// `.panel-header` in docs/design/2026-10-17-presence-screens-v3.html is
// 36 pt tall, and until 2026-10-18 it held three things and no action. Nail
// used the panel for a morning and asked for the account and its sign-out at
// the top, where the hand goes; the version and its update followed, because
// the two things the header is about are which gdoc this is and who it is
// signed in as. Everything the two of them say under the line, the countdown
// and the changed address and the banner, is drawn here as well, so the
// account is one block and not a row at the top and a banner at the foot.
//
// It draws the model and decides nothing. The mark is a setting, ⁘ by
// default, and may be empty, so the header takes it as a string and draws
// whatever it is handed.
//
// RowTests states the height, the pieces and that they fit the line.

import SwiftUI

/// The header block: the line, and what is said under it.
public struct PanelHeader: View {
  public let model: HeaderModel
  /// The one action whose words are held open with no pointer.
  public let forced: String?
  public let actions: PanelActions

  @Environment(\.colorScheme) private var scheme

  public init(
    model: HeaderModel, forced: String? = nil, actions: PanelActions = PanelActions()
  ) {
    self.model = model
    self.forced = forced
    self.actions = actions
  }

  public var body: some View {
    VStack(alignment: .leading, spacing: 0) {
      if let question = model.question {
        // The consequence is a line of its own rather than the row's
        // sub-line: the pair of buttons takes the right-hand column, and a
        // sentence squeezed into what is left reads as half a sentence.
        ConfirmRow(
          title: question.title, danger: question.danger,
          onConfirm: { actions.perform(.signOut, nil) },
          onKeep: { actions.perform(.keep, nil) }
        )
        .frame(minHeight: Grid.headerHeight)
        .metricProbe("header.line")
        QuietLine(question.consequence)
      } else {
        line
      }
      ForEach(model.quiet, id: \.self) { words in
        QuietLine(words)
      }
      if let url = model.signInURL {
        URLLine(url: url)
      }
      if let banner = model.banner {
        SignedInBanner(account: banner.account) { actions.perform(.signOut, nil) }
      }
    }
    .padding(.bottom, Grid.headerGap)
    .metricProbe("header")
  }

  /// The line itself: gdoc on the left, the account on the right.
  ///
  /// The outer stack sets no spacing of its own. Seven pieces at 304 pt is
  /// 4 pt of room left over, and a stack that put its own gap on either
  /// side of the spacer took 16 of them: the address was squeezed by 13 pt
  /// and read "nail@example…" on every screen.
  /// testTheHeaderIs36WithTheMarkTheNameTheVersionAndTheAccount.
  private var line: some View {
    let palette = Palette.of(scheme)
    return HStack(spacing: 0) {
      if !model.mark.isEmpty {
        Text(model.mark)
          .font(Typography.mark.font)
          .foregroundStyle(palette.text.color)
          .padding(.trailing, Grid.columnGap)
          .metricProbe("header.mark")
      }
      HStack(spacing: Grid.actionGap) {
        Text(model.name)
          .font(Typography.titleEmphasis.font)
          .foregroundStyle(palette.text.color)
          .fixedSize()
          .metricProbe("header.name")
        if let version = model.version {
          Text(version)
            .font(Typography.caption.font)
            .fixedSize()
            .foregroundStyle(model.versionTone.swatch(in: palette).color)
            .metricProbe("header.version")
        }
        if model.versionWaiting {
          RingView().metricProbe("header.version.ring")
        }
        if let update = model.update {
          button(update)
        }
      }
      Spacer(minLength: Grid.columnGap)
      HStack(spacing: Grid.actionGap) {
        if model.waiting {
          RingView().metricProbe("header.ring")
        } else if model.glyph != .blank {
          StatusGlyph(model.glyph, tone: model.tone)
            .metricProbe("header.glyph")
        }
        if let account = model.account {
          Text(account)
            .font(Typography.caption.font)
            .lineLimit(1)
            .truncationMode(.tail)
            .foregroundStyle(palette.textTertiary.color)
            .metricProbe("header.account")
        }
        ForEach(model.actions) { action in
          button(action)
        }
      }
    }
    .padding(.horizontal, Grid.rowInset)
    .frame(maxWidth: .infinity, minHeight: Grid.headerHeight, alignment: .leading)
    .metricProbe("header.line")
  }

  private func button(_ action: ActionModel) -> some View {
    ActionButton(
      action.icon, tip: action.tip, state: action.state,
      forcedTooltip: action.forcedTooltip || forced == action.id,
      action: { actions.perform(action.kind, action.target) }
    )
    .metricProbe("header.\(action.kind.rawValue)")
  }
}
