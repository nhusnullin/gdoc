// The note paired with a document, found rather than configured.
//
// The hub was one folder for a thing that differs by document: Nail's notes
// live under several roots, and a single setting could only ever name one of
// them. So the app asks Spotlight, which has already read every file on the
// machine, for the Markdown files whose text holds the document id, and then
// reads the `gdoc:` front-matter block of each of those few candidates
// exactly as the hub scan did. A file that merely mentions the id in a
// sentence is a candidate that check throws away.
//
// Two notes naming one document is a fault with its own sentence rather than
// a pick. The hub scan took the first by path, because it was walking a
// folder somebody chose; here there is no folder and no order worth
// trusting, and a click that opened one of two notes at random would be
// worse than a click that says what is wrong.
//
// Nothing here reaches the wire and nothing here starts a program: a
// metadata query is a question to the indexer that is already running. The
// query is held to a document id of the Drive shape before it is written
// into a query string, for the reason every other id in this app is: a line
// of the record is forgeable.
//
// The search is a protocol, so no test on this machine runs a real query:
// the tests hand over a finder that answers from a literal.

import Foundation
import PresenceCore

/// What Spotlight is asked, as one question.
///
/// The handler is how a live query says the answer went stale: it is called
/// with the document id when one of the files that query named changes, and
/// the finder drops that id so the next ask looks again.
public protocol NoteSearch: Sendable {
  /// The Markdown files under the home folder whose text holds this id, or
  /// nothing at all where the index did not answer inside the ceiling.
  ///
  /// The two are not the same answer. An empty list is no note, and it is
  /// written down; nothing at all is no answer, and the draw that asked
  /// gives up while the next ask runs the query again.
  func candidates(
    naming id: String, changed: @escaping @Sendable (String) -> Void
  ) async -> [String]?
}

/// The note that names a document, and the answers already given.
///
/// An actor because the lookups come off the main actor and the cache is one
/// map. It holds no query of its own against a second ask: an actor is
/// reentrant, so a document asked about twice while the first query is out
/// would run two. Nothing in the app asks twice, because the session keeps
/// the one `looking` set and asks about a document once at a time.
public actor NoteFinder {
  /// How long a query may take before the draw gives up on it. Two seconds
  /// is far above the measurement in `docs/v2/MEASURED.md`, and it is a
  /// ceiling rather than a budget: nothing waits on it but one greyed icon,
  /// and the next ask runs the query again. When that next ask comes is the
  /// session's business, not the finder's: it holds the clock off that
  /// document for a minute, and asks at once when the panel opens.
  public static let ceiling: TimeInterval = 2

  private let search: NoteSearch
  private let read: @Sendable (String) -> String
  private let forgot: @Sendable (String) -> Void

  /// The answer per document, and the number of times that answer has been
  /// thrown away. The count is what keeps a query that was already running
  /// when the notice arrived from writing its stale answer back.
  private var answers: [String: Result<String, NoteFault>] = [:]
  private var asked: [String: Int] = [:]

  public init(
    search: NoteSearch = SpotlightSearch(),
    read: @escaping @Sendable (String) -> String = {
      HubIndex.head(of: URL(fileURLWithPath: $0))
    },
    forgot: @escaping @Sendable (String) -> Void = { _ in }
  ) {
    self.search = search
    self.read = read
    self.forgot = forgot
  }

  /// The note whose front matter names this document, or why there is none,
  /// or nothing where the query did not answer inside the ceiling.
  ///
  /// That third answer is not "no note". A query that hit the ceiling left
  /// no live query behind, so no Spotlight notice can ever come for that
  /// id, and writing "no note" down would grey one icon for the life of the
  /// app over a note that is sitting there. The draw gives up, and the
  /// session asks again a minute later, or at once when the panel opens:
  /// testAQueryThatHitTheCeilingIsNotWrittenDown.
  public func note(for id: String) async -> Result<String, NoteFault>? {
    if let already = answers[id] { return already }

    let generation = asked[id] ?? 0
    let answered = await search.candidates(naming: id) { [weak self] changed in
      Task { await self?.forget(changed) }
    }
    guard let candidates = answered else { return nil }
    let naming = candidates.filter { path in
      HubIndex.documentIDs(inFrontMatterOf: read(path)).contains(id)
    }
    let answer: Result<String, NoteFault> =
      switch naming.count {
      case 0: .failure(.notPaired)
      case 1: .success(naming[0])
      default: .failure(.twoNotes)
      }

    // A notice that arrived while the query was out has already thrown this
    // answer away, so it is handed back once and never written down.
    guard (asked[id] ?? 0) == generation else { return answer }
    answers[id] = answer
    return answer
  }

  /// Throw one answer away, which is what a Spotlight notice does. The next
  /// ask runs the query again.
  ///
  /// Whoever holds a copy is told whether or not there was an answer here to
  /// drop. A notice can land in the moment between the live query going on,
  /// which is inside the search, and the answer being written down: the
  /// lookup out in that moment is one the count above throws away, and a
  /// silence here would leave its stale answer in the session with the
  /// watcher that would have taken it away already stopped:
  /// testANoticeWithNothingWrittenDownYetIsStillForwarded.
  public func forget(_ id: String) {
    asked[id] = (asked[id] ?? 0) + 1
    answers.removeValue(forKey: id)
    forgot(id)
  }
}

/// Spotlight itself: one metadata query per document, kept live so the one
/// notice that matters drops the answer.
public final class SpotlightSearch: NoteSearch, @unchecked Sendable {
  /// The query, as the string Spotlight's own parser reads.
  ///
  /// The word form (`cdw`) and not a `*id*` substring: measured on
  /// 2026-10-18 the word form answers in a quarter of a second and the
  /// substring form in a second and a half, over the same index and with the
  /// same three files coming back.
  static func queryString(for id: String) -> String {
    "kMDItemTextContent = \"\(id)\"cdw && kMDItemFSName = \"*.md\"c"
  }

  private let home: String
  private let ceiling: TimeInterval
  private let lock = NSLock()
  /// The live query per document, dropped when its notice arrives or when
  /// the same document is asked about again.
  private var live: [String: LiveQuery] = [:]

  public init(
    home: String = FileManager.default.homeDirectoryForCurrentUser.path,
    ceiling: TimeInterval = NoteFinder.ceiling
  ) {
    self.home = home
    self.ceiling = ceiling
  }

  deinit {
    for (_, watcher) in live { watcher.stop() }
  }

  public func candidates(
    naming id: String, changed: @escaping @Sendable (String) -> Void
  ) async -> [String]? {
    guard LinkLine.isDocumentID(id),
      let predicate = NSPredicate(fromMetadataQueryString: Self.queryString(for: id))
    else { return [] }

    let query = NSMetadataQuery()
    query.predicate = predicate
    query.searchScopes = [home]
    let queue = OperationQueue()
    queue.maxConcurrentOperationCount = 1
    query.operationQueue = queue
    let held = QueryBox(query, on: queue)

    let gathered: [String]? = await withCheckedContinuation { continuation in
      let gather = Gather(held, ceiling: ceiling) { paths in
        continuation.resume(returning: paths)
      }
      gather.start()
    }
    guard let paths = gathered else {
      held.stop()
      return nil
    }

    watch(held, for: id, changed: changed)
    return paths
  }

  /// Keep this query running under the document's name, letting go of
  /// whatever was running under it before.
  private func watch(
    _ held: QueryBox, for id: String, changed: @escaping @Sendable (String) -> Void
  ) {
    let watcher = LiveQuery(held, id: id) { [weak self] named in
      self?.drop(named)
      changed(named)
    }
    lock.lock()
    let replaced = live.updateValue(watcher, forKey: id)
    lock.unlock()
    replaced?.stop()
    watcher.start()
  }

  private func drop(_ id: String) {
    lock.lock()
    let going = live.removeValue(forKey: id)
    lock.unlock()
    going?.stop()
  }
}

/// One metadata query and the queue it answers on.
///
/// `NSMetadataQuery` is not `Sendable`, and every use of it here is on that
/// one queue, which is what this box says: nothing outside reaches the query
/// except through these three calls.
private final class QueryBox: @unchecked Sendable {
  let query: NSMetadataQuery
  let queue: OperationQueue

  init(_ query: NSMetadataQuery, on queue: OperationQueue) {
    self.query = query
    self.queue = queue
  }

  func start() {
    queue.addOperation { [self] in _ = query.start() }
  }

  func keepUpdating() {
    queue.addOperation { [self] in query.enableUpdates() }
  }

  func stop() {
    queue.addOperation { [self] in query.stop() }
  }

  /// The paths of everything the query has found.
  func paths() -> [String] {
    (0..<query.resultCount).compactMap {
      (query.result(at: $0) as? NSMetadataItem)?
        .value(forAttribute: kMDItemPath as String) as? String
    }
  }

  /// Watch one of the query's own notices, on its own queue.
  func observe(
    _ name: Notification.Name, _ body: @escaping @Sendable () -> Void
  ) -> NSObjectProtocol {
    NotificationCenter.default.addObserver(forName: name, object: query, queue: queue) { _ in
      body()
    }
  }
}

/// One query's first pass: the gathering, or the ceiling, whichever comes
/// first, and never both.
///
/// It is an object because both ends capture it: the notification centre
/// holds it until the observer is taken away, and the delayed work holds it
/// until the ceiling is up, so nobody else has to keep it alive.
private final class Gather: @unchecked Sendable {
  private let lock = NSLock()
  private var answered = false
  private var token: NSObjectProtocol?
  private let held: QueryBox
  private let ceiling: TimeInterval
  private let answer: @Sendable ([String]?) -> Void

  init(
    _ held: QueryBox, ceiling: TimeInterval, answer: @escaping @Sendable ([String]?) -> Void
  ) {
    self.held = held
    self.ceiling = ceiling
    self.answer = answer
  }

  func start() {
    let watching = held.observe(.NSMetadataQueryDidFinishGathering) { [self] in
      done(held.paths())
    }
    lock.lock()
    token = watching
    lock.unlock()
    held.start()
    DispatchQueue.global(qos: .utility).asyncAfter(deadline: .now() + ceiling) { [self] in
      done(nil)
    }
  }

  /// Nothing where the ceiling came first, which the caller reads as a query
  /// that said nothing rather than as an empty answer.
  private func done(_ paths: [String]?) {
    lock.lock()
    guard !answered else {
      lock.unlock()
      return
    }
    answered = true
    let taken = token
    token = nil
    lock.unlock()
    if let taken { NotificationCenter.default.removeObserver(taken) }
    answer(paths)
  }
}

/// A gathered query left running, so the one notice that the files it named
/// have changed reaches the finder. It fires once and then lets go.
private final class LiveQuery: @unchecked Sendable {
  private let lock = NSLock()
  private var token: NSObjectProtocol?
  private let held: QueryBox
  private let id: String
  private let changed: @Sendable (String) -> Void

  init(_ held: QueryBox, id: String, changed: @escaping @Sendable (String) -> Void) {
    self.held = held
    self.id = id
    self.changed = changed
  }

  func start() {
    let watching = held.observe(.NSMetadataQueryDidUpdate) { [self] in
      guard let taken = take() else { return }
      NotificationCenter.default.removeObserver(taken)
      held.stop()
      changed(id)
    }
    lock.lock()
    token = watching
    lock.unlock()
    held.keepUpdating()
  }

  func stop() {
    guard let taken = take() else { return }
    NotificationCenter.default.removeObserver(taken)
    held.stop()
  }

  /// The observer, taken away, or nothing where somebody else already took
  /// it: the notice and a second query for the same document race.
  private func take() -> NSObjectProtocol? {
    lock.lock()
    defer { lock.unlock() }
    let taken = token
    token = nil
    return taken
  }
}

/// The one wire from the finder back to whoever is holding its answers.
///
/// The finder is built in an initialiser, before the object that wants the
/// notice exists, so the handler is set a line later. It is a box with a
/// lock rather than a captured closure for that reason, and because the
/// notice arrives on whatever queue Spotlight answered on.
public final class NoteForgotten: @unchecked Sendable {
  private let lock = NSLock()
  private var handler: (@Sendable (String) -> Void)?

  public init() {}

  /// Who to tell. The last caller wins, and there is one.
  public func answer(_ handler: @escaping @Sendable (String) -> Void) {
    lock.lock()
    self.handler = handler
    lock.unlock()
  }

  /// One document's answer was thrown away.
  public func tell(_ id: String) {
    lock.lock()
    let telling = handler
    lock.unlock()
    telling?(id)
  }
}
