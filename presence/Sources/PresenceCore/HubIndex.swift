// The hub, as one map from a document id to the note paired with it.
//
// Open md needs no field in the record: what pairs a note with a document is
// the `gdoc:` block the binary writes into that note's front matter, and the
// hub is the folder the person named in SETTINGS. So the app reads that one
// block of every note, and nothing else in the file ever reaches it. This is
// the one read M23's "opens only the event files and its plist" is widened
// by, and it is widened by exactly this much.
//
// The reading stops at the closing fence. A note's body may hold a `gdoc:`
// block in a fenced example, and a front matter may hold a `document_id`
// under somebody else's key; neither is a pairing. Every id is then held to
// the Drive shape, so a path or a sentence in that field pairs nothing.
//
// The watcher says only that something changed. The index is rebuilt by the
// same synchronous scan a first open runs, so nothing about the pairing
// depends on an event's own words, and a debounce means a vault syncing a
// hundred files rescans once.

import CoreServices
import Foundation

/// Why a note cannot be opened. The panel has one sentence per case; none of
/// them is a judgement about the note.
public enum NoteFault: String, Error, Sendable, Equatable {
  /// No hub is chosen yet, so there is nowhere to look.
  case noHub
  /// No note in the hub names this document.
  case notPaired
  /// The index named a note that is not a regular file there any more. A
  /// symlink is not the note either: what is at the end of it is somewhere
  /// the person did not choose.
  case movedOrDeleted
  /// The path climbs out of the hub, or is absolute.
  case outsideTheHub
  case notMarkdown
}

/// Every document the hub's notes name, and the note that names it.
public struct HubIndex: Sendable, Equatable {
  /// The folder the person chose, resolved.
  public let hub: String
  /// A document id to the note's path relative to the hub.
  public let notes: [String: String]

  public static let empty = HubIndex(hub: "", notes: [:])

  public init(hub: String, notes: [String: String]) {
    self.hub = hub
    self.notes = notes
  }

  /// The vault Obsidian knows, which is the hub folder's own name.
  public var vault: String { URL(fileURLWithPath: hub).lastPathComponent }

  /// The note paired with a document, as a path inside the hub.
  public func note(for id: String) -> String? { notes[id] }

  // MARK: the scan

  /// How much of a note is read. Front matter is the first lines of a file;
  /// past this there is nothing the app is looking for, and a note somebody
  /// pasted a book into is not read into memory.
  public static let frontMatterBytes = 64 * 1024

  /// Every `.md` file under the hub, read for its `gdoc:` block.
  ///
  /// Two notes naming one document is somebody's own mistake rather than a
  /// refusal: the first by path wins, so the same note wins every scan.
  public static func scan(hub: String, fileManager: FileManager = .default) -> HubIndex {
    let root = URL(fileURLWithPath: hub)
    // By path and not by URL: the enumerator over paths hands back the path
    // relative to the root it was given, which is the thing the index holds.
    // The URL enumerator hands back an absolute path that has been through
    // the file system's own idea of the root, and on this machine a temporary
    // directory comes back under another prefix than the one it went in as.
    guard let walker = fileManager.enumerator(atPath: root.path) else {
      return HubIndex(hub: root.path, notes: [:])
    }

    var found: [String: String] = [:]
    for case let relative as String in walker {
      // A dot folder is somebody's own machinery: .obsidian, .git, .trash.
      // The whole folder goes, and only a folder: skipDescendants on a plain
      // file abandons the rest of the folder the file is in, and Finder
      // leaves a .DS_Store in every folder a person has opened, sorted
      // before the letters.
      if relative.split(separator: "/").contains(where: { $0.hasPrefix(".") }) {
        if (walker.fileAttributes?[.type] as? FileAttributeType) == .typeDirectory {
          walker.skipDescendants()
        }
        continue
      }
      guard relative.lowercased().hasSuffix(".md") else { continue }
      let url = root.appendingPathComponent(relative)
      let kind = try? fileManager.attributesOfItem(atPath: url.path)[.type] as? FileAttributeType
      guard kind == .typeRegular else { continue }
      for id in documentIDs(inFrontMatterOf: head(of: url)) {
        if let already = found[id], already <= relative { continue }
        found[id] = relative
      }
    }
    return HubIndex(hub: root.path, notes: found)
  }

  /// The document ids one note's front matter names, in the order they are
  /// written.
  ///
  /// Everything past the closing fence is the note's own text. Inside the
  /// fence, only the `gdoc:` block is read, and in it only `document_id` at
  /// the block's own level, which is schema 1, and `id` on an entry of
  /// `documents:`, which is schema 2. Every other key ends in `_id` or is
  /// nested deeper, and every value is held to the Drive shape.
  public static func documentIDs(inFrontMatterOf text: String) -> [String] {
    guard let fence = frontMatter(of: text) else { return [] }
    let lines = fence.map { (indent: indent(of: $0), trimmed: trimmed($0)) }
    guard let opening = lines.firstIndex(where: { $0.trimmed == "gdoc:" }) else { return [] }
    let level = lines[opening].indent

    var ids: [String] = []
    var inDocuments = false
    var documentsIndent = 0
    var entryIndent: Int?
    var entryKeyIndent: Int?

    for line in lines[(opening + 1)...] {
      if line.trimmed.isEmpty { continue }
      // The block ends where the front matter's own next key begins.
      if line.indent <= level { break }

      if inDocuments, line.indent <= documentsIndent {
        inDocuments = false
        entryIndent = nil
        entryKeyIndent = nil
      }

      if !inDocuments {
        if key(line.trimmed) == "documents", value(line.trimmed).isEmpty {
          inDocuments = true
          documentsIndent = line.indent
          continue
        }
        if key(line.trimmed) == "document_id" { take(value(line.trimmed), into: &ids) }
        continue
      }

      if line.trimmed.hasPrefix("- ") {
        // A deeper list is a list inside an entry, never another entry.
        guard entryIndent == nil || line.indent <= entryIndent! else { continue }
        entryIndent = line.indent
        entryKeyIndent = nil
        let inline = String(line.trimmed.dropFirst(2))
        if key(inline) == "id" { take(value(inline), into: &ids) }
        continue
      }

      guard let entry = entryIndent, line.indent > entry else { continue }
      if entryKeyIndent == nil { entryKeyIndent = line.indent }
      if line.indent == entryKeyIndent, key(line.trimmed) == "id" {
        take(value(line.trimmed), into: &ids)
      }
    }
    return ids
  }

  // MARK: the one URL

  /// The note's URL in Obsidian, or why there is none.
  ///
  /// The path is proven before the URL is built: inside the hub the person
  /// chose, a `.md`, and a regular file that is there now. No line parameter
  /// is promised, because nothing in the record says which line.
  public static func noteURL(
    hub: String, relative: String, fileManager: FileManager = .default
  ) -> Result<URL, NoteFault> {
    let root = URL(fileURLWithPath: hub).resolvingSymlinksInPath().path
    guard !relative.isEmpty, !relative.hasPrefix("/") else { return .failure(.outsideTheHub) }

    let candidate = URL(fileURLWithPath: root).appendingPathComponent(relative).standardized.path
    guard candidate.hasPrefix(root + "/") else { return .failure(.outsideTheHub) }
    guard URL(fileURLWithPath: candidate).pathExtension.lowercased() == "md" else {
      return .failure(.notMarkdown)
    }
    let kind = try? fileManager.attributesOfItem(atPath: candidate)[.type] as? FileAttributeType
    guard kind == .typeRegular else { return .failure(.movedOrDeleted) }

    let vault = URL(fileURLWithPath: root).lastPathComponent
    let text = "obsidian://open?vault=\(encoded(vault))&file=\(encoded(relative))"
    guard let url = URL(string: text) else { return .failure(.movedOrDeleted) }
    return .success(url)
  }

  // MARK: reading one file, and one line

  /// The first bytes of a file, as text. A note that is not UTF-8 is a note
  /// with no front matter this app can read.
  private static func head(of url: URL) -> String {
    guard let handle = try? FileHandle(forReadingFrom: url) else { return "" }
    defer { try? handle.close() }
    guard let bytes = try? handle.read(upToCount: frontMatterBytes) else { return "" }
    return String(decoding: bytes, as: UTF8.self)
  }

  /// The lines between the opening fence and the closing one, or nothing
  /// where there is no closed front matter.
  private static func frontMatter(of text: String) -> [String]? {
    let lines = text.replacingOccurrences(of: "\r\n", with: "\n").split(
      separator: "\n", omittingEmptySubsequences: false
    ).map(String.init)
    guard let first = lines.first, trimmed(first) == "---" else { return nil }
    guard
      let closing = lines[1...].firstIndex(where: {
        let word = trimmed($0)
        return word == "---" || word == "..."
      })
    else { return nil }
    return Array(lines[1..<closing])
  }

  private static func indent(of line: String) -> Int {
    line.prefix(while: { $0 == " " || $0 == "\t" }).count
  }

  private static func trimmed(_ line: String) -> String {
    line.trimmingCharacters(in: .whitespaces)
  }

  /// The key of `key: value`, or an empty string where the line is not one.
  private static func key(_ line: String) -> String {
    guard let colon = line.firstIndex(of: ":") else { return "" }
    return String(line[..<colon]).trimmingCharacters(in: .whitespaces)
  }

  private static func value(_ line: String) -> String {
    guard let colon = line.firstIndex(of: ":") else { return "" }
    var said = String(line[line.index(after: colon)...]).trimmingCharacters(in: .whitespaces)
    for quote in ["\"", "'"]
    where said.hasPrefix(quote) && said.hasSuffix(quote)
      && said.count >= 2
    {
      said = String(said.dropFirst().dropLast())
    }
    return said
  }

  private static func take(_ id: String, into ids: inout [String]) {
    guard LinkLine.isDocumentID(id), !ids.contains(id) else { return }
    ids.append(id)
  }

  /// Percent-encoded to the unreserved set, so what goes in the URL is a
  /// value a reader can predict from the path.
  private static func encoded(_ text: String) -> String {
    let unreserved = CharacterSet(charactersIn: "-._~").union(.alphanumerics)
    return text.addingPercentEncoding(withAllowedCharacters: unreserved) ?? ""
  }
}

/// A watcher over the hub that says only that something changed.
///
/// FSEvents rather than one source per file: a vault is a tree, a note is
/// added in a subfolder, and the app wants one signal for the tree. The
/// stream runs on a queue of its own and needs no run loop, so it works the
/// same in a test as in the app.
public final class HubWatcher: @unchecked Sendable {
  /// How long the watcher waits after the last change before it says so. A
  /// vault syncing a hundred files is one rescan.
  public static let debounce: TimeInterval = 0.5

  private let hub: String
  private let debounce: TimeInterval
  private let queue: DispatchQueue
  private let onChange: @Sendable () -> Void
  private let lock = NSLock()
  private var stream: FSEventStreamRef?
  private var pending: DispatchWorkItem?

  public init(
    hub: String,
    debounce: TimeInterval = HubWatcher.debounce,
    queue: DispatchQueue = DispatchQueue(label: "gdoc.presence.hub"),
    onChange: @escaping @Sendable () -> Void
  ) {
    self.hub = URL(fileURLWithPath: hub).resolvingSymlinksInPath().path
    self.debounce = debounce
    self.queue = queue
    self.onChange = onChange
  }

  public func start() {
    lock.lock()
    defer { lock.unlock() }
    guard stream == nil else { return }

    // Retained, and released by the stream: the callback arrives on a queue
    // of its own, so the stream holds the watcher up rather than trusting
    // that whoever made it is still there. `stop` is what lets it go.
    let held = Unmanaged.passRetained(self)
    var context = FSEventStreamContext(
      version: 0, info: held.toOpaque(),
      retain: nil,
      release: { info in
        guard let info else { return }
        Unmanaged<HubWatcher>.fromOpaque(info).release()
      }, copyDescription: nil)
    let callback: FSEventStreamCallback = { _, info, _, _, _, _ in
      guard let info else { return }
      Unmanaged<HubWatcher>.fromOpaque(info).takeUnretainedValue().changed()
    }
    guard
      let made = FSEventStreamCreate(
        kCFAllocatorDefault, callback, &context, [hub] as CFArray,
        FSEventStreamEventId(kFSEventStreamEventIdSinceNow), debounce / 2,
        FSEventStreamCreateFlags(
          kFSEventStreamCreateFlagFileEvents | kFSEventStreamCreateFlagNoDefer))
    else {
      // Nothing was made, so nothing will ever call the release above.
      held.release()
      return
    }
    FSEventStreamSetDispatchQueue(made, queue)
    FSEventStreamStart(made)
    stream = made
  }

  public func stop() {
    lock.lock()
    let made = stream
    stream = nil
    pending?.cancel()
    pending = nil
    lock.unlock()
    guard let made else { return }
    FSEventStreamStop(made)
    // Off the queue first, so no further callback is scheduled, and then a
    // barrier on that queue, so one already running has finished before the
    // release below drops what may be the last reference to this watcher.
    // Nothing dispatched on that queue calls back into `stop`, so the
    // barrier cannot wait on itself.
    FSEventStreamSetDispatchQueue(made, nil)
    FSEventStreamInvalidate(made)
    queue.sync {}
    FSEventStreamRelease(made)
  }

  // There is no `deinit` that stops: while a stream is up it holds this
  // watcher, so a `deinit` could only run after `stop` already ran. A
  // watcher nobody stops keeps its stream, which is a leak of one object
  // and not a callback into freed memory.

  private func changed() {
    lock.lock()
    pending?.cancel()
    let next = DispatchWorkItem { [onChange] in onChange() }
    pending = next
    lock.unlock()
    queue.asyncAfter(deadline: .now() + debounce, execute: next)
  }
}
