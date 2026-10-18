// One line of the record, and what this app refuses to read.
//
// The shape is gdoc's, written by go/internal/presence, and the one fixture
// both languages decode is held byte for byte equal by a Go test. Unknown
// fields are ignored, because a newer gdoc may add one; an unknown schema is
// not, because a field that changed meaning cannot be noticed from here.

import Foundation

/// The eight things the record says happened. A command has a start and an
/// end, and the one command long enough to have parts says which part it is
/// on between them; a chat hold is held and then released, expired, landed or
/// dropped, exactly once. Dropped is the sign-in it was held under changing,
/// which gdoc treats as the card being gone.
public enum EventKind: String, Sendable, CaseIterable {
  case start
  case end
  case step
  case held
  case released
  case expired
  case landed
  case dropped
}

/// How a hold stopped waiting. The record's own words, with no reading on
/// top, and each of them is `cmd/gdoc`'s own: released is the person
/// pressing the card, written before the write it approved goes out
/// (`mcprelease.go` takes the hold out and then sends); landed is that write
/// reaching the document another way, which is the model sending it again
/// with no card (`mcphold.go`'s `landed`); expired is the hold's own life
/// running out; and dropped is the sign-in it was held under stopping being
/// the sign-in on the computer.
///
/// No line says a person refused one. A card nobody answers expires, which
/// is why the panel never words any of these as a no.
public enum HoldEnding: String, Sendable, Equatable, CaseIterable {
  case released
  case expired
  case landed
  case dropped

  /// The ending an event kind is, and nothing for the kinds that are not one.
  public init?(_ kind: EventKind) {
    guard let ending = HoldEnding(rawValue: kind.rawValue) else { return nil }
    self = ending
  }
}

/// Why a line is not a record. A newer schema is the one fault the app shows
/// a person, as "gdoc is newer than this app"; everything else is a line that
/// is skipped and never guessed at.
public enum RecordFault: Error, Equatable, Sendable {
  case newerSchema(Int)
  case undecodable(String)
}

/// The events of a stretch of the record, with what could not be read.
///
/// The counts are what the app may say about the file: how many lines it let
/// go, and whether one of them came from a gdoc it does not understand. The
/// lines themselves are never kept, because a line nobody could decode is
/// also a line nobody should display.
public struct Records: Sendable, Equatable {
  public var events: [Event] = []
  public var skipped = 0
  public var newerSchema = false

  public init() {}

  public var isEmpty: Bool { events.isEmpty && skipped == 0 && !newerSchema }
}

/// One event, as the app reads it: the decoded form of one line.
public struct Event: Sendable, Equatable {
  /// The one schema this app knows. A line at any other is refused.
  public static let knownSchema = 1

  public let schema: Int
  public let at: Date
  public let gdoc: String?
  public let pid: Int32
  public let run: String
  public let call: Int64?
  public let kind: EventKind
  public let command: String?
  public let via: String?
  /// Claude Code's own id for the session a run came from, on the start of a
  /// `via: code` run and nowhere else. A terminal run and a chat call carry
  /// none, so this is nil for every line but those starts.
  public let session: String?
  public let doc: String?
  public let wait: Bool
  /// Whether a person stopped this wait, on the end of a `comments --wait` and
  /// nowhere else. A wait that ran to its deadline carries nothing, so this is
  /// false for every line that says nothing about stopping.
  public let interrupted: Bool
  /// Which part of a run is beginning, on a step line and nowhere else. It is
  /// one of gdoc's four fixed words for the parts of a publish, and a word
  /// this version has not heard of is still the word gdoc wrote.
  public let step: String?
  /// That part's place in the list of parts, counted from one, and how many
  /// parts there are. They are positions and never a measure of work done:
  /// gdoc writes no percentage and no estimate.
  public let index: Int?
  public let count: Int?
  public let title: String?
  /// The document a run created, which only a publish does. Every other
  /// command was handed its document, and `doc` names that one.
  public let made: String?
  public let ok: Bool?
  public let verified: Bool?
  /// Why a failed run stopped, in one of gdoc's seven fixed words. It is on a
  /// failed end alone, and this app keeps it as the word gdoc wrote: a word
  /// this version has not heard of is still a word, and nothing here guesses
  /// at what it means.
  public let stop: String?
  /// The line of the note the cause sits on, where the cause has one. Today
  /// that is the picture the house style cannot carry, and nothing else.
  public let line: Int?
  public let ms: Int64?
  public let hold: String?
  public let rule: String?
  public let until: Date?

  public init(
    schema: Int, at: Date, gdoc: String?, pid: Int32, run: String, call: Int64?, kind: EventKind,
    command: String?, via: String?, session: String? = nil, doc: String?, wait: Bool,
    interrupted: Bool = false,
    step: String? = nil, index: Int? = nil, count: Int? = nil, title: String?,
    made: String? = nil, ok: Bool?,
    verified: Bool?, stop: String? = nil, line: Int? = nil,
    ms: Int64?, hold: String?, rule: String?, until: Date?
  ) {
    self.schema = schema
    self.at = at
    self.gdoc = gdoc
    self.pid = pid
    self.run = run
    self.call = call
    self.kind = kind
    self.command = command
    self.via = via
    self.session = session
    self.doc = doc
    self.wait = wait
    self.interrupted = interrupted
    self.step = step
    self.index = index
    self.count = count
    self.title = title
    self.made = made
    self.ok = ok
    self.verified = verified
    self.stop = stop
    self.line = line
    self.ms = ms
    self.hold = hold
    self.rule = rule
    self.until = until
  }

  /// A start or an end whose run and call pair it with its other half. A hold
  /// has neither, and pairs by its own id.
  public var pairing: String { "\(run)|\(call.map(String.init) ?? "-")" }
}

extension Event {
  /// One line into one event, or the reason it is not one.
  public static func decode(_ line: Data) throws -> Event {
    let wire: Wire
    do {
      wire = try JSONDecoder().decode(Wire.self, from: line)
    } catch {
      throw RecordFault.undecodable("not one JSON object")
    }
    return try wire.decoded()
  }

  public static func decode(_ line: String) throws -> Event {
    try decode(Data(line.utf8))
  }

  /// Every newline-terminated line of a stretch of the file, decoded. Bytes
  /// after the last newline are not a record yet and are left alone: the
  /// writer may be mid-line, and half a line is what a reader would have had
  /// to guess at.
  public static func scan(_ data: Data) -> Records {
    var reader = LineReader()
    var records = Records()
    for line in reader.take(data) {
      records.add(line)
    }
    return records
  }
}

extension Records {
  /// One line, decoded into this set or counted as a fault.
  mutating func add(_ line: Data) {
    do {
      events.append(try Event.decode(line))
    } catch let fault as RecordFault {
      if case .newerSchema = fault {
        newerSchema = true
      } else {
        skipped += 1
      }
    } catch {
      skipped += 1
    }
  }
}

/// The line as it is on the disk: every field optional, so a missing one is a
/// refusal this file words rather than a decoding error nobody can read.
/// Unknown keys are ignored here, which is how a newer gdoc may add a field
/// without this app refusing the line.
private struct Wire: Decodable {
  var v: Int?
  var at: String?
  var gdoc: String?
  var pid: Int32?
  var run: String?
  var call: Int64?
  var event: String?
  var command: String?
  var via: String?
  var session: String?
  var doc: String?
  var wait: Bool?
  var interrupted: Bool?
  var step: String?
  var index: Int?
  var count: Int?
  var title: String?
  var made: String?
  var ok: Bool?
  var verified: Bool?
  var stop: String?
  var line: Int?
  var ms: Int64?
  var hold: String?
  var rule: String?
  var until: String?

  /// The schema is read before anything else, because a line from a newer
  /// gdoc may hold a field whose meaning changed, and the app has no way to
  /// notice that from here. It says so and reads none of the rest. A schema
  /// under the known one is not a newer gdoc and never says it is: no gdoc
  /// wrote one, so it is a line this app skips like any other.
  func decoded() throws -> Event {
    guard let v else { throw RecordFault.undecodable("no v") }
    guard v <= Event.knownSchema else { throw RecordFault.newerSchema(v) }
    guard v == Event.knownSchema else { throw RecordFault.undecodable("unknown v") }

    guard let at, let when = moment(at) else { throw RecordFault.undecodable("no at") }
    guard let pid, pid > 0 else { throw RecordFault.undecodable("no pid") }
    guard let run, !run.isEmpty else { throw RecordFault.undecodable("no run") }
    guard let event, let kind = EventKind(rawValue: event) else {
      throw RecordFault.undecodable("no event")
    }

    switch kind {
    case .start, .end, .step:
      guard let command, !command.isEmpty else { throw RecordFault.undecodable("no command") }
    case .held, .released, .expired, .landed, .dropped:
      guard let hold, !hold.isEmpty else { throw RecordFault.undecodable("no hold") }
    }

    var ran: Date?
    if let until {
      guard let parsed = moment(until) else { throw RecordFault.undecodable("no until") }
      ran = parsed
    }

    return Event(
      schema: v, at: when, gdoc: gdoc, pid: pid, run: run, call: call, kind: kind,
      command: command, via: via, session: session, doc: doc, wait: wait ?? false,
      interrupted: interrupted ?? false,
      step: step, index: index, count: count, title: title, made: made,
      ok: ok, verified: verified, stop: stop, line: line, ms: ms, hold: hold, rule: rule,
      until: ran)
  }
}

/// The moment a field names, in the one format gdoc writes: RFC 3339 in UTC,
/// with or without a fraction of a second.
///
/// The formatter is built per call rather than held in a global, because an
/// ISO8601DateFormatter promises nothing about two threads and the record is
/// read from more than one. A few hundred lines a day costs nothing.
private func moment(_ text: String) -> Date? {
  let plain = ISO8601DateFormatter()
  plain.formatOptions = [.withInternetDateTime]
  if let date = plain.date(from: text) { return date }
  let fractional = ISO8601DateFormatter()
  fractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
  return fractional.date(from: text)
}
