// Following the record while gdoc writes it.
//
// The binary appends one line per event, closes the file, and tells nobody.
// So the app watches: a DispatchSource per month file for the appends, one on
// the directory for a file that appears or goes, and a timer that brings the
// next month in at the boundary. Every signal runs the same synchronous pass,
// which is also what the tests drive, so nothing about the reading depends on
// an event arriving.
//
// Two months are open at a time, this one and the last, which is exactly what
// the binary keeps. The file is opened O_NOFOLLOW: a symlink at that path is
// something the writer refuses to write through, and the reader will not read
// through one either.

import Darwin
import Foundation

/// The file a moment belongs in.
public enum MonthFile {
  /// The name of the file an event at that moment was written to: the UTC
  /// month, so nothing is ever renamed and two machines agree.
  public static func name(for date: Date) -> String {
    var calendar = Calendar(identifier: .gregorian)
    calendar.timeZone = TimeZone(identifier: "UTC") ?? .gmt
    let parts = calendar.dateComponents([.year, .month], from: date)
    return String(format: "%04d-%02d.jsonl", parts.year ?? 0, parts.month ?? 0)
  }

  /// The two files worth following at that moment, oldest first.
  public static func names(at date: Date) -> [String] {
    var calendar = Calendar(identifier: .gregorian)
    calendar.timeZone = TimeZone(identifier: "UTC") ?? .gmt
    let before = calendar.date(byAdding: .month, value: -1, to: date) ?? date
    return [name(for: before), name(for: date)]
  }
}

/// A buffer that hands back whole lines and keeps what is not one yet.
///
/// A writer that died mid-line leaves bytes with no newline after them, and
/// the next record is written straight onto those bytes. Only a
/// newline-terminated line is a record, so the glued pair is one line that
/// decodes to nothing and the tail goes on. The buffer is bounded because a
/// file that never sees another newline would otherwise be read into memory
/// without end: past the bound the partial is let go and reading picks up at
/// the next newline.
public struct LineReader: Sendable {
  public static let defaultMaxLine = 1 << 20

  private let maxLine: Int
  private var buffer = Data()
  private var discarding = false

  /// How many oversized partials were let go. A record the app never saw is
  /// worth counting, because the alternative is a buffer that grows.
  public private(set) var dropped = 0

  public init(maxLine: Int = LineReader.defaultMaxLine) {
    self.maxLine = maxLine
  }

  /// The bytes held back, waiting for their newline.
  public var pending: Int { buffer.count }

  /// The whole lines in what arrived, in order.
  public mutating func take(_ chunk: Data) -> [Data] {
    var lines: [Data] = []
    var rest = chunk[chunk.startIndex...]

    while let newline = rest.firstIndex(of: UInt8(ascii: "\n")) {
      let head = rest[rest.startIndex..<newline]
      rest = rest[rest.index(after: newline)...]
      if discarding {
        discarding = false
        continue
      }
      var line = buffer
      line.append(contentsOf: head)
      buffer.removeAll(keepingCapacity: true)
      if !line.isEmpty { lines.append(line) }
    }

    guard !rest.isEmpty, !discarding else { return lines }
    buffer.append(contentsOf: rest)
    if buffer.count > maxLine {
      buffer = Data()
      dropped += 1
      discarding = true
    }
    return lines
  }
}

/// One month's file, open at an offset.
///
/// The file may be replaced under its own name by a person moving one in, and
/// removed by the binary's sweep when the month is two months old. Either way
/// the offset the follower held means nothing in the new file, so the device
/// and inode are compared on every pass and a file that is not the one that
/// was open is read from its first byte.
final class FileFollower {
  let url: URL

  private var descriptor: Int32 = -1
  private var identity: (device: Int32, inode: UInt64)?
  private var offset: UInt64 = 0
  private var reader: LineReader
  private let maxLine: Int

  init(url: URL, maxLine: Int = LineReader.defaultMaxLine) {
    self.url = url
    self.maxLine = maxLine
    self.reader = LineReader(maxLine: maxLine)
  }

  deinit { close() }

  /// The whole lines that arrived since the last pass.
  func drain() -> [Data] {
    guard let now = regularFile(at: url.path) else {
      close()
      return []
    }
    let here = (device: now.st_dev, inode: now.st_ino)
    if identity == nil || identity! != here {
      close()
      guard open(here) else { return [] }
    }
    let size = UInt64(max(now.st_size, 0))
    if size < offset {
      // Truncated in place. The bytes before the offset are not the bytes
      // that were there, so nothing of the old read can be trusted.
      close()
      guard open(here) else { return [] }
    }
    guard descriptor >= 0, size > offset else { return [] }

    var chunk = Data()
    var buffer = [UInt8](repeating: 0, count: 1 << 16)
    var at = offset
    while at < size {
      let wanted = Int(min(UInt64(buffer.count), size - at))
      let read = buffer.withUnsafeMutableBytes { raw in
        pread(descriptor, raw.baseAddress, wanted, off_t(at))
      }
      if read <= 0 { break }
      chunk.append(contentsOf: buffer[0..<read])
      at += UInt64(read)
    }
    offset = at
    return reader.take(chunk)
  }

  private func open(_ here: (device: Int32, inode: UInt64)) -> Bool {
    let opened = Darwin.open(url.path, O_RDONLY | O_NOFOLLOW)
    guard opened >= 0 else { return false }
    descriptor = opened
    identity = here
    offset = 0
    reader = LineReader(maxLine: maxLine)
    return true
  }

  private func close() {
    if descriptor >= 0 { Darwin.close(descriptor) }
    descriptor = -1
    identity = nil
    offset = 0
    reader = LineReader(maxLine: maxLine)
  }
}

/// What is at the path, if what is there is a regular file.
///
/// lstat rather than stat: a symlink at that name is not a regular file and
/// is read no further. The writer refuses to write through one, and the
/// reader does not read through one either.
private func regularFile(at path: String) -> stat? {
  var out = Darwin.stat()
  guard Darwin.lstat(path, &out) == 0, (out.st_mode & S_IFMT) == S_IFREG else { return nil }
  return out
}

/// The tail over the presence directory: two months, followed.
///
/// Unchecked Sendable because every piece of its state is behind the one
/// lock, which is also what lets poll() be called straight from a test while
/// the watchers run on their own queue.
public final class LogTail: @unchecked Sendable {
  private let directory: URL
  private let clock: @Sendable () -> Date
  private let rescan: TimeInterval
  private let maxLine: Int

  private let lock = NSLock()
  private let queue = DispatchQueue(label: "com.altery.gdoc.presence.tail")

  private var followers: [String: FileFollower] = [:]
  private var order: [String] = []
  private var watchers: [String: DispatchSourceFileSystemObject] = [:]
  private var directoryWatcher: DispatchSourceFileSystemObject?
  private var timer: DispatchSourceTimer?
  private var deliver: (@Sendable (Records) -> Void)?

  /// rescan is how often the month set is looked at again, which is what
  /// brings the next month's file in at the boundary. It is also the pass
  /// that catches anything a vnode event did not say.
  public init(
    directory: URL,
    clock: @escaping @Sendable () -> Date = Date.init,
    rescan: TimeInterval = 30,
    maxLine: Int = LineReader.defaultMaxLine
  ) {
    self.directory = directory
    self.clock = clock
    self.rescan = rescan
    self.maxLine = maxLine
  }

  deinit { stop() }

  /// The two files being followed, oldest first.
  public var files: [URL] {
    lock.lock()
    defer { lock.unlock() }
    sync()
    return order.compactMap { followers[$0]?.url }
  }

  /// One synchronous pass: the month set, then every line that arrived.
  public func poll() -> Records {
    lock.lock()
    defer { lock.unlock() }
    sync()
    var records = Records()
    for name in order {
      guard let follower = followers[name] else { continue }
      for line in follower.drain() {
        records.add(line)
      }
    }
    return records
  }

  /// Start watching. The first pass runs at once, so the panel has a glance
  /// before anything happens, and every later pass is a vnode event or the
  /// timer.
  public func start(onRecords: @escaping @Sendable (Records) -> Void) {
    lock.lock()
    deliver = onRecords
    lock.unlock()

    watchDirectory()
    startTimer()
    pass()
  }

  /// Stop watching. Every source is cancelled; nothing is read again.
  public func stop() {
    lock.lock()
    let sources = Array(watchers.values)
    let directorySource = directoryWatcher
    let ticker = timer
    watchers = [:]
    directoryWatcher = nil
    timer = nil
    deliver = nil
    lock.unlock()

    for source in sources { source.cancel() }
    directorySource?.cancel()
    ticker?.cancel()
  }

  /// One pass, with whatever it found handed to the callback. An empty pass
  /// says nothing: the panel is not redrawn because a file was touched.
  private func pass() {
    let records = poll()
    lock.lock()
    let onRecords = deliver
    lock.unlock()
    guard !records.isEmpty else { return }
    onRecords?(records)
  }

  /// The month set, and a follower and a watcher for each file in it. Called
  /// under the lock on every pass, so the boundary needs no special case.
  private func sync() {
    let wanted = MonthFile.names(at: clock())
    guard wanted != order else {
      watchFiles()
      return
    }
    for name in followers.keys where !wanted.contains(name) {
      followers.removeValue(forKey: name)
      watchers.removeValue(forKey: name)?.cancel()
    }
    for name in wanted where followers[name] == nil {
      followers[name] = FileFollower(
        url: directory.appendingPathComponent(name), maxLine: maxLine)
    }
    order = wanted
    watchFiles()
  }

  /// A vnode source per file that exists, for the appends. A file that is
  /// written to, removed or renamed signals, and the source on a file that
  /// went away is cancelled: the next pass opens whatever is at that name now.
  private func watchFiles() {
    guard deliver != nil else { return }
    for name in order where watchers[name] == nil {
      let path = directory.appendingPathComponent(name).path
      let descriptor = Darwin.open(path, O_EVTONLY | O_NOFOLLOW)
      guard descriptor >= 0 else { continue }
      let source = DispatchSource.makeFileSystemObjectSource(
        fileDescriptor: descriptor, eventMask: [.write, .extend, .delete, .rename, .link],
        queue: queue)
      source.setEventHandler { [weak self] in
        guard let self else { return }
        let data = source.data
        if data.contains(.delete) || data.contains(.rename) {
          self.forget(name)
        }
        self.pass()
      }
      source.setCancelHandler { Darwin.close(descriptor) }
      watchers[name] = source
      source.resume()
    }
  }

  /// One source on the directory, for a file that appears or goes under a
  /// name no file had when the last pass ran.
  private func watchDirectory() {
    let descriptor = Darwin.open(directory.path, O_EVTONLY)
    guard descriptor >= 0 else { return }
    let source = DispatchSource.makeFileSystemObjectSource(
      fileDescriptor: descriptor, eventMask: [.write, .delete, .rename], queue: queue)
    source.setEventHandler { [weak self] in self?.pass() }
    source.setCancelHandler { Darwin.close(descriptor) }
    lock.lock()
    directoryWatcher = source
    lock.unlock()
    source.resume()
  }

  private func startTimer() {
    let ticker = DispatchSource.makeTimerSource(queue: queue)
    ticker.schedule(deadline: .now() + rescan, repeating: rescan)
    ticker.setEventHandler { [weak self] in self?.pass() }
    lock.lock()
    timer = ticker
    lock.unlock()
    ticker.resume()
  }

  /// Drop the watcher on a file that was removed or renamed, so the next pass
  /// installs one on whatever takes its place.
  private func forget(_ name: String) {
    lock.lock()
    let source = watchers.removeValue(forKey: name)
    lock.unlock()
    source?.cancel()
  }
}
