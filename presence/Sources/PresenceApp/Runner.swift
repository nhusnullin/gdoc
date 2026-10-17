// The one room that starts a program.
//
// M23 said the app launches nothing at all. The panel reverses that one
// sentence and no other: six argv lists, at an absolute path the app resolved
// itself, with the null device for standard input and four variables in the
// environment. There is no seventh list and no way to add one from outside
// this file: the set is a closed enum, a source scan in PresenceCoreTests
// holds the word Process to this file alone, and a second scan holds the enum
// at six cases.
//
// Never from PATH. A PATH is a thing a shell, a launch agent or an installer
// writes, so a gdoc found through one is a program somebody else chose; the
// two places the app looks are the one `gdoc update` installs into and the one
// beside its own bundle, and both are named in full before anything runs.
//
// Nothing here knows why a call is made. The schedule is the pollers', the
// sign-in's URL is SignIn's, and every word a person reads is the panel's.

import Darwin
import Foundation
import PresenceCore

/// The six argv lists, and nothing else this app can run.
///
/// Each is a command a person can type, with no flag the app invented and no
/// setting of its own: what the app does to somebody's documents is what they
/// would have done at a terminal.
public enum Call: String, CaseIterable, Sendable {
  case help
  case authStatus
  case authLogin
  case authLogout
  case update
  case updateDesktop

  public var arguments: [String] {
    switch self {
    case .help: ["help", "--json"]
    case .authStatus: ["auth", "status"]
    case .authLogin: ["auth", "login"]
    case .authLogout: ["auth", "logout"]
    case .update: ["update"]
    case .updateDesktop: ["update", "--desktop"]
    }
  }
}

/// Where the binary is, and where it was looked for.
public struct Location: Sendable, Equatable {
  /// The resolved absolute path, or nothing where neither place held one.
  public let path: String?
  /// The places tried, in order, in full.
  public let candidates: [String]
  /// The same places for a person to read, the home directory as a tilde. It
  /// is the HEALTH row's "looked for it at ..." and nothing else.
  public let tried: String

  public init(path: String?, candidates: [String], tried: String) {
    self.path = path
    self.candidates = candidates
    self.tried = tried
  }

  /// Whether gdoc was there to be asked. An app that cannot find the binary
  /// can do nothing at all, which is why HEALTH says so and the menu bar
  /// counts it as stuck.
  public var found: Bool { path != nil }

  /// Where `gdoc update --desktop` leaves the Claude Desktop extension: a
  /// file called `gdoc.mcpb` beside the binary. Nothing where no binary was
  /// found, because then there is no "beside" to look in.
  ///
  /// It is a path and not a reading: whether the file is there is the
  /// session's question, and the only thing the answer proves is the
  /// absence. A machine with no such file never had `update --desktop` run
  /// on it, so Claude Desktop cannot have the extension; a machine with one
  /// says nothing about what Desktop loaded.
  public var extensionFile: String? {
    path.map {
      URL(fileURLWithPath: $0)
        .deletingLastPathComponent()
        .appendingPathComponent(Location.extensionName).path
    }
  }

  /// The name `go/cmd/gdoc/desktop.go` writes it under, as a literal.
  public static let extensionName = "gdoc.mcpb"
}

/// Which strict reading of the one printed object did not hold. The panel
/// shows none of these words; a test names the rule it is holding.
public enum EnvelopeFault: String, Sendable, Equatable {
  case notJSON
  case notAnObject
  case noOK
  case badOK
  case badError
  case badWarnings
  case badVersion
  case badData
}

/// Why no answer came back.
public enum RunFailure: Error, Sendable, Equatable {
  /// Neither place held a gdoc. The string is the places, as HEALTH says them.
  case notFound(tried: String)
  /// A child of that kind is already up.
  case busy(Call)
  case launchFailed(String)
  case timedOut(after: TimeInterval)
  case envelope(EnvelopeFault)
  /// The outcome of a child is taken once, by whoever started it.
  case outcomeAlreadyTaken
}

/// The one JSON object a command prints, as the app reads it.
///
/// `data` stays bytes. What is inside it differs per command and belongs to
/// whoever asked, so this type carries the envelope's own four fields and
/// hands the rest on unread.
public struct Envelope: Sendable, Equatable {
  public let ok: Bool
  public let data: Data?
  public let error: String?
  public let warnings: [String]
  public let version: String?

  public init(
    ok: Bool,
    data: Data? = nil,
    error: String? = nil,
    warnings: [String] = [],
    version: String? = nil
  ) {
    self.ok = ok
    self.data = data
    self.error = error
    self.warnings = warnings
    self.version = version
  }

  /// One object, nothing before it and nothing after it, with `ok` a true
  /// boolean and every other field the type the contract says.
  ///
  /// A field this app does not know is read past rather than refused, as the
  /// record's own decoder reads past one: the alternative is an app that stops
  /// working on the release after this one. A field it does know and that
  /// holds the wrong kind of value is refused by name, because something
  /// other than gdoc wrote it.
  public static func decode(_ bytes: Data) throws -> Envelope {
    guard let top = try? JSONSerialization.jsonObject(with: bytes, options: []) else {
      throw RunFailure.envelope(.notJSON)
    }
    guard let object = top as? [String: Any] else { throw RunFailure.envelope(.notAnObject) }

    guard let stated = object["ok"] else { throw RunFailure.envelope(.noOK) }
    guard let number = stated as? NSNumber, CFGetTypeID(number) == CFBooleanGetTypeID() else {
      throw RunFailure.envelope(.badOK)
    }

    var error: String?
    if let said = present(object["error"]) {
      guard let text = said as? String else { throw RunFailure.envelope(.badError) }
      error = text
    }

    var warnings: [String] = []
    if let said = present(object["warnings"]) {
      guard let list = said as? [String] else { throw RunFailure.envelope(.badWarnings) }
      warnings = list
    }

    var version: String?
    if let said = present(object["version"]) {
      guard let text = said as? String else { throw RunFailure.envelope(.badVersion) }
      version = text
    }

    var data: Data?
    if let said = present(object["data"]) {
      guard let inside = said as? [String: Any],
        let bytes = try? JSONSerialization.data(withJSONObject: inside)
      else { throw RunFailure.envelope(.badData) }
      data = bytes
    }

    return Envelope(
      ok: number.boolValue, data: data, error: error, warnings: warnings, version: version)
  }

  /// A field that is there and is not null. The contract leaves a field out
  /// rather than printing an empty one, so null is absent either way.
  private static func present(_ value: Any?) -> Any? {
    guard let value, !(value is NSNull) else { return nil }
    return value
  }
}

/// What one child said.
public struct Outcome: Sendable, Equatable {
  public let call: Call
  public let envelope: Envelope
  /// Everything the child wrote to standard error. A login's link is in here,
  /// and so is a screen a person would have read at a terminal.
  public let standardError: String
  public let status: Int32

  public init(call: Call, envelope: Envelope, standardError: String, status: Int32) {
    self.call = call
    self.envelope = envelope
    self.standardError = standardError
    self.status = status
  }
}

/// The runner: one binary, one child per kind, nothing else.
///
/// Unchecked Sendable because the set of kinds in flight is behind the one
/// lock, which is what lets the panel start a child from the main thread while
/// another one is being waited on.
public final class Runner: @unchecked Sendable {
  /// Long enough for an update over a slow line, short enough that a child
  /// which will never answer does not hold its kind for the session.
  public static let defaultTimeout: TimeInterval = 120

  /// Where `gdoc update` installs, under the home directory.
  public static let installedAt = ".local/bin/gdoc"

  public let location: Location
  public let timeout: TimeInterval

  private let parentEnvironment: [String: String]
  private let lock = NSLock()
  private var inFlight: Set<Call> = []

  public init(
    location: Location,
    timeout: TimeInterval = Runner.defaultTimeout,
    environment: [String: String] = ProcessInfo.processInfo.environment
  ) {
    self.location = location
    self.timeout = timeout
    self.parentEnvironment = environment
  }

  /// The two places, in order, and the first one that holds an executable.
  ///
  /// `exists` and `resolve` are injected so the decision is a pure function of
  /// its arguments. The resolution is the Stopper's own, so the path the app
  /// launches is the path the kernel will name for the child.
  public static func locate(
    home: String,
    bundle: URL?,
    exists: (String) -> Bool = Runner.isExecutable,
    resolve: (String) -> String = ProcessCheck.resolved
  ) -> Location {
    var candidates = [URL(fileURLWithPath: home).appendingPathComponent(installedAt).path]
    if let bundle {
      candidates.append(bundle.deletingLastPathComponent().appendingPathComponent("gdoc").path)
    }
    let found = candidates.first(where: exists).map(resolve)
    let tried = candidates.map { abbreviate($0, home: home) }.joined(separator: " or ")
    return Location(path: found, candidates: candidates, tried: tried)
  }

  public static func isExecutable(_ path: String) -> Bool {
    FileManager.default.isExecutableFile(atPath: path)
  }

  /// The four variables a child gets, and no fifth.
  ///
  /// HOME, PATH and LANG are the parent's where the parent has them; NO_COLOR
  /// is always on, because the app reads JSON and a colour code in it is a
  /// parse failure. Everything else is dropped, GDOC_CONFIG_DIR included: the
  /// app asks about the token the person's own terminal uses, and a variable
  /// the app inherited from a launch agent would point the child somewhere
  /// else without saying so.
  public static func childEnvironment(from parent: [String: String]) -> [String: String] {
    var child = ["NO_COLOR": "1"]
    for name in ["HOME", "PATH", "LANG"] {
      if let value = parent[name] { child[name] = value }
    }
    return child
  }

  public func isRunning(_ call: Call) -> Bool {
    lock.lock()
    defer { lock.unlock() }
    return inFlight.contains(call)
  }

  /// Launch one child. The kind is reserved before the launch, in one locked
  /// step, so two callers cannot both pass the one-child rule.
  ///
  /// The deadline is the runner's own unless the caller names another. A
  /// sign-in is the one call that waits longer than the runner's default,
  /// because the thing it waits for is a person opening a browser.
  public func start(_ call: Call, timeout: TimeInterval? = nil) throws -> Child {
    guard let gdoc = location.path else { throw RunFailure.notFound(tried: location.tried) }

    lock.lock()
    if inFlight.contains(call) {
      lock.unlock()
      throw RunFailure.busy(call)
    }
    inFlight.insert(call)
    lock.unlock()

    do {
      return try Child(
        call: call, executable: gdoc,
        environment: Runner.childEnvironment(from: parentEnvironment),
        timeout: timeout ?? self.timeout,
        release: { [weak self] in self?.free(call) })
    } catch {
      free(call)
      throw error
    }
  }

  /// Launch one child and wait for its object.
  public func run(_ call: Call) async throws -> Outcome {
    try await start(call).outcome()
  }

  private func free(_ call: Call) {
    lock.lock()
    inFlight.remove(call)
    lock.unlock()
  }

  /// A path with the home directory as a tilde, for a row somebody reads.
  private static func abbreviate(_ path: String, home: String) -> String {
    guard path.hasPrefix(home + "/") else { return path }
    return "~" + path.dropFirst(home.count)
  }
}

/// One child, up or finished.
///
/// Unchecked Sendable for the same reason as the runner: the two flags are
/// behind its lock, and the waiting happens off the main thread.
public final class Child: @unchecked Sendable {
  public let call: Call
  public let executable: String
  public let arguments: [String]
  public let environment: [String: String]

  private let process: Process
  private let output: Pipe
  private let errors: Pipe
  private let timeout: TimeInterval
  private let release: @Sendable () -> Void
  /// Signalled by the child's own termination handler.
  ///
  /// Not waitUntilExit: that runs a run loop on the waiting thread and the
  /// end is delivered to the one the launch happened on, so a wait off the
  /// main thread while something else holds the main run loop in a mode of
  /// its own never sees the child end. The handler is a dispatch callback and
  /// needs no run loop at all.
  private let exited = DispatchSemaphore(value: 0)
  private let lock = NSLock()
  private var expired = false
  private var taken = false
  private var errorLines: (@Sendable (String) -> Void)?

  init(
    call: Call,
    executable: String,
    environment: [String: String],
    timeout: TimeInterval,
    release: @escaping @Sendable () -> Void
  ) throws {
    self.call = call
    self.executable = executable
    self.arguments = call.arguments
    self.environment = environment
    self.timeout = timeout
    self.release = release
    self.process = Process()
    self.output = Pipe()
    self.errors = Pipe()

    process.executableURL = URL(fileURLWithPath: executable)
    process.arguments = arguments
    process.environment = environment
    // Nothing is ever asked of a person through a child of this app, so the
    // child is handed a standard input that is already at its end.
    process.standardInput = FileHandle.nullDevice
    process.standardOutput = output
    process.standardError = errors
    let end = exited
    process.terminationHandler = { _ in end.signal() }
    do {
      try process.run()
    } catch {
      throw RunFailure.launchFailed(String(describing: error))
    }
  }

  /// How long after the deadline's terminate the end is still waited for.
  public static let grace: TimeInterval = 5

  /// Hear each line of standard error as it is written, rather than at the
  /// end.
  ///
  /// One caller: a sign-in, which needs the link while the child is still
  /// waiting for the browser. Set it before the outcome is taken; a sink
  /// registered after that reading has begun hears nothing, which is why
  /// this is a method on the child the starter holds and not a notice
  /// anybody can subscribe to.
  public func onStandardErrorLine(_ sink: @escaping @Sendable (String) -> Void) {
    lock.lock()
    errorLines = sink
    lock.unlock()
  }

  public var isRunning: Bool { process.isRunning }

  /// Send the child a terminate. It is what Cancel does to a sign-in, and
  /// what the deadline does to a child that stopped answering. The kind stays
  /// reserved until the outcome is taken, because until then the child is
  /// still dying.
  public func terminate() {
    guard process.isRunning else { return }
    process.terminate()
  }

  /// Wait for the child, read what it wrote, and read its one object.
  ///
  /// Taken once. Both pipes are drained on their own threads, so a child that
  /// writes more than a pipe holds is never waiting on a reader that is
  /// waiting on it.
  public func outcome() async throws -> Outcome {
    try await withCheckedThrowingContinuation { continuation in
      DispatchQueue.global().async { [self] in
        do {
          continuation.resume(returning: try waited())
        } catch {
          continuation.resume(throwing: error)
        }
      }
    }
  }

  private func waited() throws -> Outcome {
    lock.lock()
    if taken {
      lock.unlock()
      throw RunFailure.outcomeAlreadyTaken
    }
    taken = true
    lock.unlock()
    defer { release() }

    lock.lock()
    let sink = errorLines
    lock.unlock()
    let out = Collector(output.fileHandleForReading.fileDescriptor)
    let said = Collector(errors.fileHandleForReading.fileDescriptor, line: sink)
    let deadline = DispatchWorkItem { [weak self] in self?.ranOut() }
    DispatchQueue.global().asyncAfter(deadline: .now() + timeout, execute: deadline)
    // The grace is for the end to arrive after the terminate the deadline
    // sends. A child that is still there after it is one the app stops
    // waiting on rather than one it signals again.
    let arrived = exited.wait(timeout: .now() + timeout + Child.grace)
    deadline.cancel()

    lock.lock()
    let late = expired || arrived == .timedOut
    lock.unlock()
    // The pipes are not waited on here: a child that was terminated may have
    // left a program of its own holding the write end, and the deadline has
    // already passed once.
    if late { throw RunFailure.timedOut(after: timeout) }

    return Outcome(
      call: call, envelope: try Envelope.decode(out.bytes()), standardError: said.text(),
      status: process.terminationStatus)
  }

  private func ranOut() {
    lock.lock()
    expired = true
    lock.unlock()
    terminate()
  }
}

/// One pipe, drained on its own thread.
///
/// The descriptor rather than the handle, so nothing that is not safe to pass
/// between threads is passed between them; the pipe itself stays alive as long
/// as the child does.
final class Collector: @unchecked Sendable {
  private let group = DispatchGroup()
  private let lock = NSLock()
  private var collected = Data()

  init(_ descriptor: Int32, line: (@Sendable (String) -> Void)? = nil) {
    group.enter()
    DispatchQueue.global().async { [self] in
      var buffer = [UInt8](repeating: 0, count: 1 << 16)
      var all = Data()
      var partial = Data()
      while true {
        let read = buffer.withUnsafeMutableBytes { raw in
          Darwin.read(descriptor, raw.baseAddress, raw.count)
        }
        if read <= 0 { break }
        all.append(contentsOf: buffer[0..<read])
        guard let line else { continue }
        partial.append(contentsOf: buffer[0..<read])
        while let newline = partial.firstIndex(of: 0x0A) {
          let whole = Data(partial[partial.startIndex..<newline])
          partial = Data(partial[partial.index(after: newline)...])
          line(String(decoding: whole, as: UTF8.self))
        }
      }
      // A child that died mid-line leaves bytes with no newline after them.
      // They are still what it said, so the last of them is a line too.
      if let line, !partial.isEmpty { line(String(decoding: partial, as: UTF8.self)) }
      lock.lock()
      collected = all
      lock.unlock()
      group.leave()
    }
  }

  func bytes() -> Data {
    group.wait()
    lock.lock()
    defer { lock.unlock() }
    return collected
  }

  func text() -> String { String(decoding: bytes(), as: UTF8.self) }
}
