// Whether the pid that wrote a start is still there.
//
// A start with no end means the command is still running, or that it died
// without writing its end. Only the kernel knows which, and kill(pid, 0) is
// the whole question: it signals nothing and answers whether a process by
// that number exists. A number the system has since handed to somebody else
// looks alive, which is an accepted and unmeasured risk inside one day; the
// run id pairs events, so nothing else depends on the number.

import Darwin
import Foundation

public enum Liveness {
  /// True while a process by that number exists.
  ///
  /// EPERM is a yes: the process is there and belongs to another user, which
  /// cannot happen for a gdoc the owner ran but costs nothing to read right.
  public static func isAlive(_ pid: Int32) -> Bool {
    guard pid > 0 else { return false }
    if kill(pid, 0) == 0 { return true }
    return errno == EPERM
  }
}
