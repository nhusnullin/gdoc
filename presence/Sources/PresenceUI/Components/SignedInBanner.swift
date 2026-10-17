// The address, once, at the size a glance catches.
//
// Scene e8's lower half: for five seconds after a sign-in the panel says the
// account in 15 pt with "Not you?" under it. It is the one place the address
// is said at that size, and it is there for one reason: a reply carries the
// signed-in person's name into somebody else's document, so catching a
// personal account in the first five seconds is cheaper than catching it in
// a comment thread.
//
// It is not a gate. Whose account it is decides nothing in gdoc; the mark
// decides. The banner only says what the token says.
//
// `.bigmail` is the type: 15 pt at the medium weight with the quieter line
// under it. The row is the scene's own, 48 pt tall with the sign-out icon on
// the right.
//
// CardTests states the five seconds and the words.

import Foundation
import SwiftUI

/// The banner after a sign-in.
public struct SignedInBanner: View {
  /// How long it stays, which is the same five seconds as the menu bar's
  /// check: long enough for one glance, short enough that it is not a row.
  public nonisolated static let life: TimeInterval = Timing.checkFlash
  /// The sentence under the address.
  public static let words = "Signed in just now. Not you? Sign out and sign in again."
  /// The words on the one icon.
  public static let signOutTip = ButtonWord.signOut.rawValue
  /// The scene's own `min-height:48px`: the address is a line taller than a
  /// title, so the row is taller than 44.
  public static let minHeight: CGFloat = 48

  public let account: String
  public let onSignOut: () -> Void

  @Environment(\.colorScheme) private var scheme

  public init(account: String, onSignOut: @escaping () -> Void = {}) {
    self.account = account
    self.onSignOut = onSignOut
  }

  /// Whether the banner is still on the screen. The clock is the caller's,
  /// so the life is a value a test states rather than a timer it waits for.
  public nonisolated static func isShowing(signedInAt: Date?, now: Date) -> Bool {
    guard let signedInAt else { return false }
    return now.timeIntervalSince(signedInAt) < life
  }

  public var body: some View {
    let palette = Palette.of(scheme)
    return HStack(alignment: .top, spacing: Grid.columnGap) {
      StatusGlyph(.check, tone: .ok)
        .frame(width: Grid.glyphSize, height: Typography.bigMail.lineHeight)
      VStack(alignment: .leading, spacing: 2) {
        Text(account)
          .font(Typography.bigMail.font)
          .tracking(Typography.bigMail.letterSpacing)
          .lineLimit(1)
          .truncationMode(.tail)
          .foregroundStyle(palette.text.color)
        Text(Self.words)
          .font(Typography.caption.font)
          .fixedSize(horizontal: false, vertical: true)
          .foregroundStyle(palette.textSecondary.color)
      }
      .frame(maxWidth: .infinity, alignment: .leading)
      ActionButton(.signout, tip: Self.signOutTip, action: onSignOut)
        .metricProbe("banner.sign-out")
    }
    .padding(.horizontal, Grid.rowInset)
    .padding(.vertical, 6)
    .frame(maxWidth: .infinity, minHeight: Self.minHeight, alignment: .leading)
    .metricProbe("banner")
  }
}
