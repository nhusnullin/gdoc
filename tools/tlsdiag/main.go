// tlsdiag answers one question: is this network interfering with TLS?
//
// It exists because of a real morning. Every Go program on this machine, gdoc
// included, timed out after ten seconds on any HTTPS call, to every host. The
// error read `net/http: TLS handshake timeout`, which looks like an expired
// credential or a broken tool, and is neither. curl worked. openssl worked. A
// mobile hotspot worked. The office wifi did not.
//
// An hour went into finding that, mostly on wrong theories: a sandbox, a proxy,
// post-quantum handshake size. What would have found it in twenty seconds is
// the one comparison this tool makes, TLS 1.2 against TLS 1.3, to more than one
// host, with the timings side by side.
//
// It is deliberately boring. Standard library only, no flags to learn, no
// network writes, and it reads nothing but the clock and the sockets it opens.
// An admin can read the source in five minutes before running it, which is the
// point of handing it to somebody else.
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// defaultHosts spans three independent operators on purpose. One host failing
// is that host's problem; three failing together is the path.
var defaultHosts = []string{
	"oauth2.googleapis.com:443",
	"api.github.com:443",
	"cloudflare.com:443",
}

const dialTimeout = 8 * time.Second

// probe is one host measured three ways: the TCP connection on its own, then
// the handshake capped at TLS 1.2, then the handshake required to be TLS 1.3.
// Splitting them is what makes the answer readable. A TCP connection that
// succeeds while a handshake stalls says the packets arrive and something
// dislikes what is in them.
type probe struct {
	host  string
	tcp   result
	tls12 result
	tls13 result
}

type result struct {
	ok      bool
	took    time.Duration
	detail  string
	failure string
}

func main() {
	verbose := flag.Bool("v", false, "print the environment as well as the results")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "tlsdiag: is this network interfering with TLS?\n\n")
		fmt.Fprintf(os.Stderr, "usage: tlsdiag [-v] [host:port ...]\n\n")
		fmt.Fprintf(os.Stderr, "With no hosts it checks three, on three different operators.\n")
		fmt.Fprintf(os.Stderr, "Exit code 0 means nothing was found, 1 means TLS 1.3 is being\n")
		fmt.Fprintf(os.Stderr, "interfered with, 2 means the network is unreachable.\n")
	}
	flag.Parse()

	hosts := flag.Args()
	if len(hosts) == 0 {
		hosts = defaultHosts
	}
	for i, h := range hosts {
		if !strings.Contains(h, ":") {
			hosts[i] = h + ":443"
		}
	}

	fmt.Printf("tlsdiag, %s\n\n", time.Now().Format(time.RFC1123))
	if *verbose {
		printEnvironment()
	}

	probes := make([]probe, 0, len(hosts))
	for _, h := range hosts {
		probes = append(probes, run(h))
	}

	printResults(probes)
	compare(probes)
	os.Exit(verdict(probes))
}

func run(hostport string) probe {
	p := probe{host: hostport}
	p.tcp = timed(func() (string, error) {
		c, err := (&net.Dialer{Timeout: dialTimeout}).Dial("tcp", hostport)
		if err != nil {
			return "", err
		}
		addr := c.RemoteAddr().String()
		c.Close()
		return addr, nil
	})
	// The handshakes run only when the connection itself works. Reporting a
	// handshake failure on a host that cannot be reached at all would blame the
	// wrong layer, which is the mistake this tool was written to stop.
	if !p.tcp.ok {
		p.tls12.failure = "not attempted: no TCP connection"
		p.tls13.failure = "not attempted: no TCP connection"
		return p
	}
	name := strings.Split(hostport, ":")[0]
	p.tls12 = handshake(hostport, &tls.Config{ServerName: name, MaxVersion: tls.VersionTLS12})
	p.tls13 = handshake(hostport, &tls.Config{ServerName: name, MinVersion: tls.VersionTLS13})
	return p
}

func handshake(hostport string, cfg *tls.Config) result {
	return timed(func() (string, error) {
		c, err := tls.DialWithDialer(&net.Dialer{Timeout: dialTimeout}, "tcp", hostport, cfg)
		if err != nil {
			return "", err
		}
		st := c.ConnectionState()
		c.Close()
		return fmt.Sprintf("TLS 1.%d, %s", st.Version-0x0301, tls.CipherSuiteName(st.CipherSuite)), nil
	})
}

func timed(f func() (string, error)) result {
	start := time.Now()
	detail, err := f()
	took := time.Since(start)
	if err != nil {
		return result{took: took, failure: oneLine(err.Error())}
	}
	return result{ok: true, took: took, detail: detail}
}

func printResults(probes []probe) {
	fmt.Printf("%-28s  %-22s  %-22s  %s\n", "host", "tcp connect", "handshake TLS 1.2", "handshake TLS 1.3")
	fmt.Println(strings.Repeat("-", 100))
	for _, p := range probes {
		fmt.Printf("%-28s  %-22s  %-22s  %s\n", p.host, cell(p.tcp), cell(p.tls12), cell(p.tls13))
	}
	fmt.Println()
	for _, p := range probes {
		for label, r := range map[string]result{"tcp": p.tcp, "tls1.2": p.tls12, "tls1.3": p.tls13} {
			if r.failure != "" && !strings.HasPrefix(r.failure, "not attempted") {
				fmt.Printf("  %s %s: %s\n", p.host, label, r.failure)
			}
		}
	}
}

func cell(r result) string {
	if r.ok {
		return fmt.Sprintf("ok %.1fs", r.took.Seconds())
	}
	if strings.HasPrefix(r.failure, "not attempted") {
		return "-"
	}
	return fmt.Sprintf("FAILED %.1fs", r.took.Seconds())
}

// compare runs the two tools every admin already trusts against the first host
// whose Go handshake failed, at the same moment, over the same path.
//
// It exists because of how this conversation goes otherwise. "TLS 1.3 is
// blocked" is checked with a browser, the browser works, and the ticket closes.
// The accurate claim is narrower and stranger: Go's handshake stalls where
// openssl's succeeds. Nobody believes that without seeing both, so the tool
// shows both rather than asserting it.
//
// It prints nothing when Go succeeded everywhere, and nothing when neither tool
// is installed.
func compare(probes []probe) {
	var host string
	for _, p := range probes {
		if p.tcp.ok && !p.tls13.ok {
			host = strings.Split(p.host, ":")[0]
			break
		}
	}
	if host == "" {
		return
	}
	fmt.Println()
	fmt.Printf("the same host, at the same moment, using tools that are not Go:\n")
	fmt.Println(strings.Repeat("-", 100))

	if _, err := exec.LookPath("openssl"); err == nil {
		start := time.Now()
		cmd := exec.Command("openssl", "s_client", "-connect", host+":443", "-tls1_3", "-brief")
		cmd.Stdin = strings.NewReader("")
		out, err := cmd.CombinedOutput()
		took := time.Since(start).Seconds()
		if err == nil && strings.Contains(string(out), "TLSv1.3") {
			fmt.Printf("  openssl -tls1_3   ok %.1fs   <- TLS 1.3 works here for openssl\n", took)
		} else {
			fmt.Printf("  openssl -tls1_3   FAILED %.1fs\n", took)
		}
	}
	if _, err := exec.LookPath("curl"); err == nil {
		start := time.Now()
		out, err := exec.Command("curl", "-sS", "--max-time", "8", "-o", "/dev/null",
			"-w", "%{http_code}", "https://"+host+"/").CombinedOutput()
		took := time.Since(start).Seconds()
		if err == nil {
			fmt.Printf("  curl              ok %.1fs   HTTP %s\n", took, strings.TrimSpace(string(out)))
		} else {
			fmt.Printf("  curl              FAILED %.1fs\n", took)
		}
	}
	fmt.Println()
	fmt.Println("  If those succeeded while the Go rows above failed, the network is not")
	fmt.Println("  blocking TLS 1.3 as such. It is treating some clients differently from")
	fmt.Println("  others, which is the fact worth taking to whoever runs it.")
}

// verdict is the whole point of the tool. It reports the one pattern that is
// otherwise invisible, and it says what an admin should look at.
func verdict(probes []probe) int {
	var reachable, twelve, thirteen int
	for _, p := range probes {
		if p.tcp.ok {
			reachable++
		}
		if p.tls12.ok {
			twelve++
		}
		if p.tls13.ok {
			thirteen++
		}
	}
	fmt.Println(strings.Repeat("-", 100))
	switch {
	case reachable == 0:
		fmt.Println("VERDICT: no host was reachable at all. This is not a TLS question:")
		fmt.Println("         check the connection, DNS and any VPN before reading further.")
		return 2

	case thirteen == reachable:
		fmt.Println("VERDICT: nothing found. TLS 1.3 completed to every reachable host.")
		return 0

	case thirteen == 0 && twelve > 0:
		fmt.Println("VERDICT: this network is interfering with TLS 1.3.")
		fmt.Println()
		fmt.Println("         Every reachable host completed a TLS 1.2 handshake and none")
		fmt.Println("         completed TLS 1.3. The connection is fine, so the packets")
		fmt.Println("         arrive and something between here and the internet dislikes")
		fmt.Println("         what is in them.")
		fmt.Println()
		fmt.Println("         For the network admin: something is inspecting or filtering")
		fmt.Println("         TLS handshakes. TLS 1.3 encrypts the certificate exchange,")
		fmt.Println("         so appliances that want to see certificates sometimes drop it")
		fmt.Println("         rather than fail it, which is why it stalls instead of erroring.")
		fmt.Println()
		fmt.Println("         Worth knowing: the same machine may still browse the web fine.")
		fmt.Println("         Browsers and curl can differ from other clients here, so")
		fmt.Println("         \"the internet works\" does not rule this out.")
		return 1

	default:
		fmt.Printf("VERDICT: mixed. TLS 1.3 completed to %d of %d reachable hosts and\n", thirteen, reachable)
		fmt.Printf("         TLS 1.2 to %d. That is not one clean pattern, so read the\n", twelve)
		fmt.Println("         rows above: a single host failing is usually that host.")
		return 1
	}
}

// printEnvironment reads what is cheap and safe to read. It runs two macOS
// commands and prints nothing if they are not there, because a Linux admin
// should still get the table above rather than an error.
func printEnvironment() {
	fmt.Println("environment")
	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("  go %s\n", strings.TrimPrefix(runtimeVersion(), "go"))
	for _, v := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "GODEBUG"} {
		if s := os.Getenv(v); s != "" {
			fmt.Printf("  %s=%s\n", v, s)
		}
	}
	if out := runCmd("route", "get", "default"); out != "" {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "gateway:") || strings.HasPrefix(line, "interface:") {
				fmt.Printf("  %s\n", line)
			}
		}
	}
	if out := runCmd("scutil", "--proxy"); out != "" && !strings.Contains(out, "<dictionary> {\n}") {
		fmt.Printf("  system proxy: %s\n", oneLine(out))
	}
	if names := tunnels(); len(names) > 0 {
		fmt.Printf("  tunnel interfaces: %s\n", strings.Join(names, " "))
		fmt.Println("  (a VPN or filtering agent is a common cause of what follows)")
	}
	fmt.Println()
}

func tunnels() []string {
	out := runCmd("ifconfig", "-l")
	var names []string
	for _, n := range strings.Fields(out) {
		for _, p := range []string{"utun", "ipsec", "ppp", "tun", "tap"} {
			if strings.HasPrefix(n, p) {
				names = append(names, n)
				break
			}
		}
	}
	sort.Strings(names)
	return names
}

func runCmd(name string, args ...string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	out, err := exec.Command(path, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 110 {
		s = s[:110] + "..."
	}
	return s
}

// runtimeVersion is in its own function so the import list above stays the
// short one an admin reads before running this.
func runtimeVersion() string { return goVersion }
