// DOCUMENTS as a model: one row of it, and the section they make.
//
// They are their own file because PanelModel.swift is the panel's whole
// vocabulary and was over this tree's own 800 line ceiling, and because
// these two are the pair the fifth look kept adding to: the row gained the
// press it opens on and the card a rested pointer brings up, and the
// section gained the card a render holds open.
//
// Nothing here decides anything. PanelModel.documentsSection fills them in
// from the fold and the facts, and StateTests states every field of a row
// against a record.

import Foundation
import PresenceCore

/// One document's row.
public struct DocumentRowModel: Sendable, Equatable, Identifiable {
  public let doc: String
  public let title: String
  public let sub: String
  public let glyph: GlyphKind
  public let tone: GlyphTone
  /// The document is in the trash. The v1 record never says so, so this is
  /// false until M24b.
  public let gone: Bool
  /// Whether the person pinned it, which the row draws as the state of its
  /// last icon: the tack is filled on a pinned row and hollow on the rest.
  public let isPinned: Bool
  /// The document a press on the row opens: its own id where Drive could
  /// have written it, and nothing otherwise.
  ///
  /// It is the rule a NOW row already follows. An id of another shape opens
  /// nothing, because `Opener` refuses to build a URL out of it and a row
  /// that offers a press doing nothing is worse than a row that offers
  /// none: testADocumentRowOpensItselfAndCarriesFourIcons.
  public let opens: String?
  public let actions: [ActionModel]
  /// The row's own menu: Hide from the list, and nothing on a pinned row.
  public let menu: [MenuItemModel]
  /// The row uncut, which a rested pointer brings up over it. It is built
  /// for every row and drawn for the one the pointer is on, so a still of
  /// any row's card is renderable from the model alone.
  public let card: DetailCardModel

  public var id: String { doc }
}

/// DOCUMENTS: its rows, every one of which is drawn.
public struct DocumentsSectionModel: Sendable, Equatable {
  public let rows: [DocumentRowModel]
  /// The one line a panel with no documents says instead.
  public let empty: String?
  /// Whether the four labels are drawn under the first row.
  public let showsStrip: Bool
  /// The document whose card is held open with no pointer, by id. It is
  /// `UIState.forcedCard`, and only a still ever sets it.
  public let forcedCard: String?
}
