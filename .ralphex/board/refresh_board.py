#!/usr/bin/env python3
"""The run board for a ralphex milestone: one HTML page, rebuilt from the logs.

The page is published as a Claude Code artifact and republished on every task
iteration and every review round, so Nail follows the run on his phone instead
of reading status lines in chat. Nothing here talks to ralphex; it reads what
ralphex already writes.

Sources, all read-only:
  - the ralphex logs (task phase, then one or more review runs)
  - the plan file on the branch: task titles, checkboxes, Files blocks
  - `git log main..<branch>`: one commit per task, then the review fixes
  - the pid, to tell "running" from "stopped"

First run, once per milestone:

    python3 .ralphex/board/refresh_board.py --init \
        --plan docs/plans/2026-09-08-gdoc-v2-m5-generator-parity.md \
        --branch gdoc-v2-m5-generator-parity --heading "Milestone 5: Generator parity" \
        --html /path/to/scratch/m5-run-board.html \
        --logs ralphex-m5-tasks.log ralphex-m5-review.log ralphex-m5-review2.log \
        --warns warns.json --started 09:53 --pid <pid>

Every refresh after that: the same command without --init (the page remembers
plan, branch, logs and warns in a sidecar JSON beside the HTML). Then publish
the HTML with the Artifact tool, same file path, label = the line this prints.

The task list comes from the plan's `### Task N: title` headings. The one-line
"what" under each title is the task's first checkbox unless a tasks JSON
(`--tasks tasks.json`, {"1": {"what": "...", "pkg": "..."}}) says it better;
writing those by hand is worth the five minutes, they are what Nail reads.
"""
import argparse, json, os, re, subprocess, sys, time
from pathlib import Path

ANSI = re.compile(r"\x1b\[[0-9;]*m")
TASK_COMMIT_PREFIXES = ("feat", "fix", "test", "docs", "refactor", "chore")


def sh(*args, cwd):
    return subprocess.run(args, cwd=cwd, capture_output=True, text=True).stdout


def alive(pid):
    if pid <= 0:
        return False
    try:
        os.kill(pid, 0)
        return True
    except OSError:
        return False


def read_logs(paths):
    return [ANSI.sub("", p.read_text(errors="replace")) for p in paths if p.exists()]


def plan_text(repo, branch, plan_rel):
    done_rel = str(Path(plan_rel).parent / "completed" / Path(plan_rel).name)
    for ref in (f"{branch}:{plan_rel}", f"{branch}:{done_rel}"):
        out = sh("git", "show", ref, cwd=repo)
        if out:
            return out
    for p in (repo / plan_rel, repo / done_rel):
        if p.exists():
            return p.read_text()
    return ""


def parse_tasks(plan):
    """Task number, title, first checkbox line, first Files path, checkbox counts."""
    tasks, cur = [], None
    for line in plan.splitlines():
        m = re.match(r"^### Task (\d+): (.*)$", line)
        if m:
            cur = {"n": int(m.group(1)), "name": m.group(2).replace("`", "").strip(), "what": "", "pkg": "", "done": 0, "total": 0}
            tasks.append(cur)
            continue
        if cur is None:
            continue
        fm = re.match(r"^- (?:Create|Modify): `([^`]+)`", line)
        if fm and not cur["pkg"]:
            cur["pkg"] = str(Path(fm.group(1)).parent).replace("go/internal/", "internal/")
        cm = re.match(r"^- \[([ x])\] (.*)$", line)
        if cm:
            cur["total"] += 1
            if cm.group(1) == "x":
                cur["done"] += 1
            if not cur["what"]:
                what = re.sub(r"\s+", " ", cm.group(2)).strip()
                cur["what"] = what[:140] + ("…" if len(what) > 140 else "")
    return tasks


def commits(repo, branch):
    out = sh("git", "log", f"main..{branch}", "--format=%h|%ad|%s", "--date=format:%H:%M", cwd=repo)
    return [tuple(l.split("|", 2)) for l in out.splitlines()]


def build_snapshot(cfg, repo, pid):
    logs = read_logs([Path(p) for p in cfg["logs"]])
    log = "\n".join(logs)
    review_log = "\n".join(logs[1:])
    running = alive(pid)
    tasks_meta = parse_tasks(plan_text(repo, cfg["branch"], cfg["plan"]))
    overrides = cfg.get("tasks", {})
    cms = commits(repo, cfg["branch"])
    n_tasks = len(tasks_meta)

    iterations = len(re.findall(r"^--- task iteration \d+ ---", log, re.M))
    ext_rounds = len(re.findall(r"^--- (codex|custom) review iteration \d+", log, re.M | re.I))
    tasks_done = bool(re.search(r"ALL_TASKS_DONE|task execution completed successfully", log))
    in_external = bool(re.search(r"^--- (codex|custom)[^\n]*review|starting external review", log, re.M | re.I))
    finished = bool(re.search(r"^completed in |review .*completed successfully", review_log, re.M | re.I))
    failed = bool(re.search(r"TASK_FAILED|max iterations reached|^error:", log, re.I | re.M)) and not running
    limited = bool(re.search(r"session limit|usage limit", log, re.I))

    task_commits = [c for c in cms if c[2].startswith(TASK_COMMIT_PREFIXES) and "review findings" not in c[2]]
    tasks, current_found = [], False
    for t in tasks_meta:
        o = overrides.get(str(t["n"]), {})
        commit = ""
        if t["n"] - 1 < len(task_commits):
            h, at, _ = task_commits[t["n"] - 1]
            commit = f"{h} · {at}"
        if commit or (tasks_done and t["total"] and t["done"] == t["total"]):
            state = "done"
        elif not current_found and running and not tasks_done:
            state, current_found = "active", True
        else:
            state = "pending"
        tasks.append({"n": t["n"], "name": o.get("name", t["name"]), "what": o.get("what", t["what"]),
                      "pkg": o.get("pkg", t["pkg"]), "done": t["done"], "total": t["total"], "s": state, "commit": commit})

    if finished:
        phases, state_text = ["done", "done", "done"], "Finished"
    elif in_external:
        phases, state_text = ["done", "active", "pending"], "Running · external review (revmux)"
    elif tasks_done:
        phases, state_text = ["done", "pending", "pending"], "Tasks done · waiting for ralphex --external-only"
    else:
        phases, state_text = ["active", "pending", "pending"], "Running · task phase"
    if not running and not finished:
        if in_external:
            state_text = "Stopped · external review exited before finishing"
        elif not tasks_done:
            state_text = "Stopped" + (" · failed" if failed else "")
        if limited:
            state_text += " · Claude session limit, resumes at the reset"

    fix_commits = len(cms) - len(task_commits)
    return {
        "refreshed": time.strftime("%H:%M"),
        "started": cfg.get("started", ""),
        "state": state_text,
        "iterations": {"used": iterations, "max": 50},
        "reached": sum(1 for s in phases if s != "pending"),
        "phases": [
            {"n": "Command 1", "t": "Tasks", "d": f"ralphex --tasks-only: {n_tasks} tasks, one iteration and one commit each. Tests written first.", "s": phases[0]},
            {"n": "Command 2", "t": "External review", "d": "ralphex --external-only: revmux rounds until clean or the cap of ten, then critical/major checks.", "s": phases[1]},
            {"n": "Then", "t": "Finish", "d": "Plan moved to docs/plans/completed/. Branch pushed, PR opened, live tests run.", "s": phases[2]},
        ],
        "tasks": tasks,
        "reviews": [
            {"t": "In-house Opus rounds", "d": "Not run in the task phase: review is revmux only, Nail's decision after M2.", "s": "pending", "v": "skipped"},
            {"t": "revmux · rounds", "d": "bugs+impl, arch+quality, docs+tests (Opus) and adversarial (Codex), then synthesis and verification.",
             "s": "active" if in_external and not finished else ("done" if finished else "pending"), "v": f"{ext_rounds} rounds" if ext_rounds else "waiting"},
            {"t": "Post-external check", "d": "Critical/major Claude review after the revmux loop, repeated until clean.", "s": "done" if finished else "pending", "v": "done" if finished else "waiting"},
            {"t": "Fix commits after reviews", "d": "Commits on the branch beyond the task commits.", "s": "done" if fix_commits else "pending", "v": str(max(0, fix_commits))},
        ],
        "warns": cfg.get("warns", []),
        "n_tasks": n_tasks,
    }


def write_html(html_path, template, cfg, snapshot):
    page = html_path.read_text() if html_path.exists() else template.read_text()
    page = (page.replace("{{TITLE}}", cfg["title"]).replace("{{PLAN}}", Path(cfg["plan"]).name)
                .replace("{{HEADING}}", cfg["heading"]).replace("{{BRANCH}}", cfg["branch"]))
    start = page.index("const snapshot = ")
    end = page.index("// ---- render ----")
    n_tasks = snapshot.pop("n_tasks")
    body = json.dumps(snapshot, ensure_ascii=False, indent=2)
    page = page[:start] + "const snapshot = " + body + ";\n" + page[end:]
    html_path.write_text(page)
    return n_tasks


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--html", required=True, help="the board page to write (and its sidecar .json)")
    ap.add_argument("--pid", type=int, default=0)
    ap.add_argument("--init", action="store_true", help="create the page from the template and save the config sidecar")
    ap.add_argument("--plan"); ap.add_argument("--branch"); ap.add_argument("--heading"); ap.add_argument("--title")
    ap.add_argument("--logs", nargs="*"); ap.add_argument("--warns"); ap.add_argument("--tasks"); ap.add_argument("--started")
    ap.add_argument("--repo", default=str(Path(__file__).resolve().parents[2]))
    a = ap.parse_args()

    html_path = Path(a.html)
    side = html_path.with_suffix(".json")
    template = Path(__file__).parent / "board.html"
    if a.init:
        for need in ("plan", "branch", "heading", "logs"):
            if not getattr(a, need):
                sys.exit(f"--init needs --{need}")
        cfg = {"plan": a.plan, "branch": a.branch, "heading": a.heading,
               "title": a.title or a.heading.split(":")[0].replace("Milestone ", "M") + " Run Board",
               "logs": [str(Path(p).resolve()) for p in a.logs], "started": a.started or time.strftime("%H:%M")}
        if a.warns:
            cfg["warns"] = json.loads(Path(a.warns).read_text())
        if a.tasks:
            cfg["tasks"] = json.loads(Path(a.tasks).read_text())
        if html_path.exists():
            html_path.unlink()
        side.write_text(json.dumps(cfg, indent=2))
    else:
        if not side.exists():
            sys.exit(f"{side} not found: run once with --init")
        cfg = json.loads(side.read_text())
        if a.started:
            cfg["started"] = a.started
        if a.warns:
            cfg["warns"] = json.loads(Path(a.warns).read_text())
        side.write_text(json.dumps(cfg, indent=2))

    snapshot = build_snapshot(cfg, Path(a.repo), a.pid)
    n_tasks = write_html(html_path, template, cfg, snapshot)
    done_n = sum(1 for t in snapshot["tasks"] if t["s"] == "done")
    print(f"{snapshot['refreshed']} {snapshot['state']} | tasks done {done_n}/{n_tasks} | "
          f"iterations {snapshot['iterations']['used']} | alive={alive(a.pid)}")


if __name__ == "__main__":
    main()
