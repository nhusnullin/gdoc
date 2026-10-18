// The events a fold test states by hand, where a fixture would say less than
// four lines of Swift.
//
// Every builder takes its moment as the string the record writes, so a test
// reads as the record reads and no test is a function of this machine's
// clock. The pid is the same in every builder: whether a process is alive is
// an injected answer, never a real pid.

import Foundation

@testable import PresenceCore

/// The one pid every built event carries. Liveness is injected, so the number
/// matters nowhere.
let builtPid: Int32 = 6000

/// A start, with no end. It carries a title where gdoc read the note before
/// it began, which is what a publish does: it is handed no document, so its
/// own lines are the only name it has.
func start(
  at text: String, command: String, run: String, doc: String?, call: Int64 = 1,
  via: String = "terminal", wait: Bool = false, title: String? = nil, session: String? = nil
) -> Event {
  built(
    at: text, kind: .start, command: command, run: run, call: call, via: via, doc: doc,
    wait: wait, title: title, session: session)
}

/// An end, with the facts an end carries: how it went, what a publish
/// created, why it stopped and whether a person stopped the wait.
func end(
  at text: String, command: String, run: String, doc: String?, call: Int64 = 1,
  via: String = "terminal", wait: Bool = false, ok: Bool = true, verified: Bool? = nil,
  title: String? = nil, ms: Int64? = 1200, made: String? = nil, stop: String? = nil,
  line: Int? = nil, interrupted: Bool = false
) -> Event {
  built(
    at: text, kind: .end, command: command, run: run, call: call, via: via, doc: doc, wait: wait,
    ok: ok, verified: verified, title: title, ms: ms, made: made, stop: stop, line: line,
    interrupted: interrupted)
}

/// One part of a publish beginning, with its place in the fixed list.
func step(
  at text: String, name: String, index: Int, run: String, count: Int = 4,
  command: String = "publish", call: Int64 = 1, title: String? = nil
) -> Event {
  built(
    at: text, kind: .step, command: command, run: run, call: call, via: "terminal", doc: nil,
    title: title, step: name, index: index, count: count)
}

/// One call that ran and ended: the start, and its end twenty seconds later.
func pair(
  at text: String, command: String, run: String, doc: String?, call: Int64 = 1,
  via: String = "terminal", wait: Bool = false, ok: Bool = true, verified: Bool? = nil,
  title: String? = nil
) -> [Event] {
  [
    start(at: text, command: command, run: run, doc: doc, call: call, via: via, wait: wait),
    end(
      at: text.secondsLater(20), command: command, run: run, doc: doc, call: call, via: via,
      wait: wait, ok: ok, verified: verified, title: title),
  ]
}

/// A chat write held for a person.
func hold(
  at text: String, run: String, doc: String?, id: String, command: String = "reply",
  rule: String = "dictated", until: String? = nil
) -> Event {
  built(
    at: text, kind: .held, command: command, run: run, call: nil, via: "chat", doc: doc,
    hold: id, rule: rule, until: until)
}

/// A hold that stopped waiting: landed, released or expired.
func holdEnd(
  at text: String, kind: EventKind, run: String, doc: String?, id: String,
  command: String = "reply"
) -> Event {
  built(
    at: text, kind: kind, command: command, run: run, call: nil, via: "chat", doc: doc, hold: id)
}

private func built(
  at text: String, kind: EventKind, command: String?, run: String, call: Int64?, via: String,
  doc: String?, wait: Bool = false, ok: Bool? = nil, verified: Bool? = nil, title: String? = nil,
  ms: Int64? = nil, hold: String? = nil, rule: String? = nil, until: String? = nil,
  made: String? = nil, stop: String? = nil, line: Int? = nil, interrupted: Bool = false,
  step: String? = nil, index: Int? = nil, count: Int? = nil, session: String? = nil
) -> Event {
  Event(
    schema: 1, at: text.asMoment, gdoc: "v2.14.0", pid: builtPid, run: run, call: call, kind: kind,
    command: command, via: via, session: session, doc: doc, wait: wait, interrupted: interrupted,
    step: step,
    index: index, count: count, title: title, made: made, ok: ok, verified: verified, stop: stop,
    line: line, ms: ms, hold: hold, rule: rule, until: until?.asMoment)
}

extension String {
  /// The moment this string names, in the one format the record writes.
  fileprivate var asMoment: Date {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return formatter.date(from: self) ?? Date(timeIntervalSince1970: 0)
  }

  /// The same moment, a few seconds on, written the way the record writes it.
  fileprivate func secondsLater(_ seconds: TimeInterval) -> String {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    formatter.timeZone = TimeZone(secondsFromGMT: 0)
    return formatter.string(from: asMoment.addingTimeInterval(seconds))
  }
}
